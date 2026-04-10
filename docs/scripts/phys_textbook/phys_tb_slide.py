#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
命令行批量或按节生成 slide_deck（课件 JSON），逻辑对齐后端 adminslidegen + HTTP chat/completions。

与 POST /api/v1/admin/sections/{sectionId}/slide-decks/generate-ai 等价：读节上下文、
拼默认提示词、调用 ai_model 最新一条（status=1）、校验 JSON、插入 slide_deck（draft）、
写入 ai_call_log。

用法（仓库根目录）::

    pip install -r docs/scripts/phys_textbook/requirements-textbook-seed.txt
    export DATABASE_URL='postgres://...'
    export STEPUP_CREATED_BY=1   # 与 admin id 一致，写入 slide_deck / ai_call_log
    # 单节
    python docs/scripts/phys_textbook/phys_tb_slide.py --section-id 123
    # 某教材下全部节（可选只处理尚无课件的节）
    python docs/scripts/phys_textbook/phys_tb_slide.py --textbook-id 5 --only-without-deck

  --execute-db 0  仅打印将执行的 INSERT 与摘要日志，不写库；1 为正常写入。
  环境变量 EXECUTE_DB=0|1 可覆盖命令行默认值。

可选: LOG_LEVEL, --quiet, --timeout, --prompt-file, --limit
"""

from __future__ import annotations

import argparse
import json
import logging
import os
import sys
import time
from typing import Any
from urllib.parse import urlsplit, urlunsplit

import psycopg2
from psycopg2.extras import Json
import requests

log = logging.getLogger("phys_tb_slide")

# 与 backend/internal/service/adminslidegen/service.go slideGenMaxOutputTokens 一致
SLIDE_GEN_MAX_OUTPUT_TOKENS = 32768

# 与 backend/internal/service/ailog/body_limit.go 一致
MAX_STORED_LOG_BODY_BYTES = 400 * 1024


def setup_logging() -> None:
    level_name = (os.environ.get("LOG_LEVEL") or "INFO").strip().upper()
    level = getattr(logging, level_name, logging.INFO)
    logging.basicConfig(
        level=level,
        format="%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
        stream=sys.stdout,
        force=True,
    )


def redact_dsn(dsn: str) -> str:
    dsn = dsn.strip()
    try:
        p = urlsplit(dsn)
        netloc = p.netloc
        if "@" in netloc:
            userinfo, hostport = netloc.rsplit("@", 1)
            if ":" in userinfo:
                user = userinfo.split(":", 1)[0]
                netloc = f"{user}:***@{hostport}"
            else:
                netloc = f"{userinfo}@{hostport}"
        return urlunsplit((p.scheme, netloc, p.path, p.query, p.fragment))
    except Exception:
        return "<无法解析的 DSN>"


def connect_dsn() -> str:
    dsn = (os.environ.get("DATABASE_URL") or os.environ.get("PG_DSN") or "").strip()
    if not dsn:
        print(
            "缺少 DATABASE_URL 或 PG_DSN",
            file=sys.stderr,
        )
        sys.exit(2)
    return dsn


def mogrify_sql(cur, query: str, params: tuple[Any, ...]) -> str:
    raw = cur.mogrify(query.strip(), params)
    return raw.decode("utf-8").strip()


def print_sql_block(title: str, sql: str) -> None:
    print(f"\n-- ===== {title} =====")
    print(sql.rstrip(";") + ";")


def clip_text(s: str, head: int = 600, tail: int = 400) -> str:
    s = s.strip()
    if len(s) <= head + tail + 20:
        return s
    return f"{s[:head]}\n…（省略 {len(s) - head - tail} 字符）…\n{s[-tail:]}"


def preview_json_obj(obj: Any, max_len: int = 2000) -> str:
    try:
        t = json.dumps(obj, ensure_ascii=False, indent=2)
    except Exception:
        t = str(obj)
    if len(t) <= max_len:
        return t
    return t[: max_len - 50] + "\n…(truncated)…"


def trunc_body(s: str) -> str:
    b = s.encode("utf-8")
    if len(b) <= MAX_STORED_LOG_BODY_BYTES:
        return s
    return b[:MAX_STORED_LOG_BODY_BYTES].decode("utf-8", errors="replace") + "\n…[truncated]"


def trunc_title_runes(s: str, max_runes: int = 200) -> str:
    if max_runes <= 0:
        return s
    chars = list(s)
    if len(chars) <= max_runes:
        return s
    return "".join(chars[:max_runes])


# ---------------------------------------------------------------------------
# AI model（对齐 adminslidegen resolveAdapter：最新一条激活模型）
# ---------------------------------------------------------------------------


def load_active_ai_model(cur) -> tuple[int | None, str, str, str, str]:
    cur.execute(
        """
        SELECT id, name, url, model, app_secret
        FROM ai_model
        WHERE status = 1 AND is_deleted = 0
        ORDER BY id DESC
        LIMIT 1
        """
    )
    row = cur.fetchone()
    if not row:
        raise RuntimeError(
            "未找到 ai_model：请配置 status=1 且 is_deleted=0 的记录（与后台幻灯片生成一致，取 id 最大一条）"
        )
    mid, name, url, model, secret = row[0], row[1] or "", row[2] or "", row[3] or "", row[4] or ""
    url = url.strip()
    secret = (secret or "").strip()
    model = (model or "").strip()
    name = (name or "").strip()
    if not url or not secret:
        raise RuntimeError("ai_model 中 url 或 app_secret 为空")
    return int(mid) if mid is not None else None, name, url, model or "gpt-4o-mini", secret


def normalize_chat_completions_url(api_base: str) -> str:
    s = api_base.strip().rstrip("/")
    low = s.lower()
    if low.endswith("chat/completions"):
        return s
    if low.endswith("/v1") or "/v1/" in low:
        return s + "/chat/completions" if not low.endswith("/chat/completions") else s
    return s + "/v1/chat/completions"


def is_dashscope_host(url: str) -> bool:
    h = urlsplit(url).netloc.lower()
    return "dashscope" in h or "aliyuncs.com" in h


def chat_completions_user_only(
    url: str,
    bearer: str,
    model: str,
    user_content: str,
    timeout: int,
    max_tokens: int,
) -> tuple[str, dict[str, Any]]:
    """单条 user 消息，对齐 HTTPAnalysisAdapter.analyzeChatCompletions（幻灯片路径）。"""
    payload: dict[str, Any] = {
        "model": model,
        "messages": [{"role": "user", "content": user_content}],
        "max_tokens": max_tokens,
        "temperature": 0.2,
    }
    if is_dashscope_host(url):
        payload["enable_thinking"] = False

    host = urlsplit(url).netloc or ""
    t0 = time.monotonic()
    log.info(
        "chat/completions: host=%s model=%s max_tokens=%s timeout=%ss dashscope_thinking_off=%s user_chars=%s",
        host,
        model,
        max_tokens,
        timeout,
        payload.get("enable_thinking", False) is False and is_dashscope_host(url),
        len(user_content),
    )
    log.debug("user 提示词预览:\n%s", clip_text(user_content, head=1200, tail=600))

    r = requests.post(
        url,
        headers={
            "Authorization": f"Bearer {bearer}",
            "Content-Type": "application/json",
        },
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        timeout=timeout,
    )
    latency_ms = int((time.monotonic() - t0) * 1000)

    trace: dict[str, Any] = {
        "adapter_kind": "http_chat_completions",
        "endpoint_host": host,
        "chat_model": model,
        "http_status": r.status_code,
        "latency_ms": latency_ms,
        "result_status": "",
        "error_phase": "",
        "error_message": "",
    }
    req_log = trunc_body(
        json.dumps(
            {**payload, "messages": [{"role": "user", "content": f"<{len(user_content)} chars>"}]},
            ensure_ascii=False,
        )
    )
    trace["request_body"] = req_log

    log.info("AI HTTP %s latency_ms=%s", r.status_code, latency_ms)
    if r.status_code >= 400:
        trace["result_status"] = "http_error"
        trace["error_phase"] = "http_status"
        trace["error_message"] = f"HTTP {r.status_code}"
        trace["response_body"] = trunc_body(r.text)
        log.error("AI 错误响应: %s", r.text[:1200])
        raise RuntimeError(f"chat/completions HTTP {r.status_code}: {r.text[:500]}")

    data = r.json()
    trace["response_body"] = trunc_body(json.dumps(data, ensure_ascii=False)[:])
    usage = data.get("usage")
    if usage:
        log.info("AI usage: %s", usage)

    choices = data.get("choices") or []
    if not choices:
        trace["result_status"] = "parse_error"
        trace["error_phase"] = "empty_choices"
        trace["error_message"] = "no choices"
        raise RuntimeError("响应无 choices")

    msg0 = choices[0].get("message") or {}
    content = msg0.get("content")
    content = str(content).strip() if content is not None else ""
    log.info(
        "choice[0] finish_reason=%s content_len=%s",
        choices[0].get("finish_reason"),
        len(content),
    )
    if not content:
        trace["result_status"] = "parse_error"
        trace["error_phase"] = "empty_body"
        trace["error_message"] = "empty assistant content"
        raise RuntimeError("模型返回空 content")

    log.info("assistant 正文预览:\n%s", clip_text(content, head=800, tail=400))
    trace["result_status"] = "success"
    return content, trace


# ---------------------------------------------------------------------------
# Section 上下文与默认提示词（对齐 adminslidegen.DefaultPrompt）
# ---------------------------------------------------------------------------


def load_section_context(cur, section_id: int) -> dict[str, Any]:
    cur.execute(
        """
        SELECT s.id, s.number, s.title, s.full_title, ch.number, ch.title, t.name, t.version, t.subject
        FROM section s
        JOIN chapter ch ON ch.id = s.chapter_id AND ch.is_deleted = 0
        JOIN textbook t ON t.id = ch.textbook_id AND t.is_deleted = 0
        WHERE s.id = %s AND s.is_deleted = 0
        """,
        (section_id,),
    )
    row = cur.fetchone()
    if not row:
        raise RuntimeError(f"section id={section_id} 不存在或已删除")
    full = row[3]
    ft = (full or "").strip() if full is not None else ""
    if not ft:
        ft = (row[2] or "").strip()
    return {
        "section_id": int(row[0]),
        "sec_num": int(row[1]),
        "section_title": (row[2] or "").strip(),
        "section_full": ft,
        "chapter_num": int(row[4]),
        "chapter_title": (row[5] or "").strip(),
        "textbook_name": (row[6] or "").strip(),
        "textbook_version": (row[7] or "").strip(),
        "subject": (row[8] or "").strip(),
    }


def default_prompt(c: dict[str, Any]) -> str:
    ft = c["section_full"].strip() or c["section_title"]
    return f"""你是 StepUp 课件结构生成助手。根据以下教材节信息，输出**仅一个**合法 JSON 对象，符合 slide schemaVersion 1。
## 输出格式（硬性）
- 只输出**一个** UTF-8 JSON 对象；第一个非空白字符必须是「{{」；**禁止**在 JSON 前写开场白、标题或「如下」等说明。
- 若你无法避免使用 Markdown 代码围栏，须保证围栏（三个反引号）成对闭合；更推荐**直接输出裸 JSON**。
- **禁止**输出未闭合的 JSON（如中途被截断）。若长度紧张，可减少页数或缩短 explanation 文字，但必须保持 schemaVersion、slides 与每个对象括号、引号、逗号语法完整。

## 体量（必须遵守）
- 总页数：**10～20 页**（建议 **14～18 页**）；硬上限 **20 页**，硬下限 **10 页**。内容要充实，禁止只做提纲式几页。

## 教学深度（必须遵守）
- **把学生讲懂为首要目标**：除封面与小结外，多数页面要有足够正文；用「概念→关键词→典型情景→易错点→巩固」的节奏铺陈。
- **例子与例题要多**：至少 **4～6 组**有头有尾的例子或演算/分析（可分布在多页）。每组尽量包含：**条件复述、关键步骤、简短总结**；必要时用 bullet-steps、split-left-right、formula-focus 等模版呈现对比（对错、有无、变式）。
- **变式与反例**：至少 **2 处**「常见错误」「易混辨析」或反例说明（可用 callout 区或独立页）。
- 适用处使用 **LaTeX**（type: latex，role 符合模版）与 **文字中的 $行内公式$**，保证公式与叙述同屏可读。

## JSON 结构（与之前一致）
- 顶层：schemaVersion:1，meta:{{ "title": string, "theme":"dark-physics" }}，slides: 数组
- 每页：id, layoutTemplate（cover-image | title-body | formula-focus | split-left-right | split-top-bottom | quiz-center | bullet-steps | two-column-text），elements: 扁平数组；每项含 type(text|latex|image|question)、role、step（从 1 起的整数）
- question：mode 为 single 或 multi；data:{{ "text", "options":[{{ "id","text" }}] }}

## 题目与答案（硬性要求，缺一即视为不合格输出）
- **每一个 type 为 question 的元素**，除 data 外**必须在同一 JSON 对象上**包含 **answer** 字段（与 data 同级），结构如下，**不得省略**：
  - "answer": {{ "correctOptionIds": ["A"], "explanation": "…" }}
  - correctOptionIds：与 options 的 id 一致；单选仅 1 个；多选可多个。
  - explanation：**多句 Markdown 子集**，必须写清：① 正确选项为何对；② 其它常见错选错在何处；③ 本题考查点/口诀/易错点。**禁止**空字符串或泛泛一句「略」。
- 若某页以测验为主，优先用 layoutTemplate **quiz-center**；题干 data.text 中可含 $公式$。

教材：《{c["textbook_name"]}》 {c["textbook_version"]}，学科 {c["subject"]}
章：第 {c["chapter_num"]} 章 {c["chapter_title"]}
节：第 {c["sec_num"]} 节 {c["section_title"]}（{ft}）

请生成一套**信息量大、例题丰富、每道选择题都有完整 answer** 的课堂幻灯片 JSON。"""


# ---------------------------------------------------------------------------
# slide JSON 规范化（对齐 adminslidegen.normalizeSlideJSON + ValidateSlideJSON）
# ---------------------------------------------------------------------------


def trim_utf8_bom(s: str) -> str:
    s = s.strip()
    if s.startswith("\ufeff"):
        s = s[1:]
    return s


def strip_code_fence(s: str) -> str:
    s = s.strip()
    if not s.startswith("```"):
        return s
    rest = s[3:]
    rest = rest.lstrip()
    if rest.lower().startswith("json"):
        rest = rest[4:].lstrip()
    idx = rest.rfind("```")
    if idx >= 0:
        rest = rest[:idx]
    return rest.strip()


def strip_fence_from_anywhere(s: str) -> str:
    s = s.strip()
    idx = s.find("```")
    if idx < 0:
        return s
    rest = s[idx + 3 :].strip()
    if len(rest) >= 4 and rest[:4].lower() == "json":
        rest = rest[4:].strip()
    close_idx = rest.find("```")
    if close_idx >= 0:
        rest = rest[:close_idx]
    return rest.strip()


def extract_first_json_object(s: str) -> tuple[str, bool]:
    s = trim_utf8_bom(s)
    start = s.find("{")
    if start < 0:
        return "", False
    depth = 0
    in_string = False
    escape = False
    i = start
    while i < len(s):
        c = s[i]
        if escape:
            escape = False
            i += 1
            continue
        if in_string:
            if c == "\\":
                escape = True
            elif c == '"':
                in_string = False
            i += 1
            continue
        if c == '"':
            in_string = True
        elif c == "{":
            depth += 1
        elif c == "}":
            depth -= 1
            if depth == 0:
                return s[start : i + 1].strip(), True
        i += 1
    return "", False


def validate_slide_json_obj(root: dict[str, Any]) -> None:
    sv = root.get("schemaVersion")
    if sv is None:
        raise ValueError("missing schemaVersion")
    if isinstance(sv, float):
        v = int(sv)
    elif isinstance(sv, int):
        v = sv
    else:
        raise ValueError(f"schemaVersion must be number (got {type(sv).__name__})")
    if v != 1:
        raise ValueError("schemaVersion must be 1")
    slides = root.get("slides")
    if slides is None:
        raise ValueError("missing slides")
    if not isinstance(slides, list):
        raise ValueError(f"slides must be a JSON array (got {type(slides).__name__})")


def normalize_slide_json(raw: str) -> str:
    s = trim_utf8_bom(raw.strip())
    seen: set[str] = set()
    cands: list[str] = []

    def add(v: str) -> None:
        v = v.strip()
        if not v or v in seen:
            return
        seen.add(v)
        cands.append(v)

    add(s)
    add(strip_code_fence(s))
    add(strip_fence_from_anywhere(s))
    j, ok = extract_first_json_object(s)
    if ok:
        add(j)
    sf = strip_fence_from_anywhere(s)
    if sf != s:
        add(strip_code_fence(sf))
        j2, ok2 = extract_first_json_object(sf)
        if ok2:
            add(j2)

    last_err: Exception | None = None
    for c in cands:
        try:
            m = json.loads(c)
        except json.JSONDecodeError as e:
            last_err = e
            continue
        if not isinstance(m, dict):
            last_err = ValueError("root not object")
            continue
        try:
            validate_slide_json_obj(m)
        except ValueError as e:
            last_err = e
            continue
        out = json.dumps(m, ensure_ascii=False, separators=(",", ":"))
        return out
    if last_err:
        raise ValueError(f"no valid deck json: {last_err}") from last_err
    raise ValueError("no valid deck json found in model output")


def deck_title_from_json(content: str) -> str:
    try:
        root = json.loads(content)
    except Exception:
        return ""
    meta = root.get("meta")
    if not isinstance(meta, dict):
        return ""
    t = meta.get("title")
    return str(t).strip() if t is not None else ""


# ---------------------------------------------------------------------------
# DB 写入
# ---------------------------------------------------------------------------


def insert_ai_call_log(
    cur,
    *,
    ai_model_id: int | None,
    model_name_snap: str,
    action: str,
    trace: dict[str, Any],
    section_id: int,
    request_meta: dict[str, Any],
    response_meta: dict[str, Any],
) -> None:
    cur.execute(
        """
        INSERT INTO ai_call_log (
          ai_model_id, model_name_snapshot, action, adapter_kind, result_status,
          http_status, latency_ms, error_phase, error_message, endpoint_host, chat_model,
          fallback_to_mock, student_id, ref_table, ref_id, request_meta, response_meta,
          request_body, response_body
        ) VALUES (
          %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
        )
        """,
        (
            ai_model_id,
            model_name_snap[:128] if model_name_snap else "",
            action,
            trace.get("adapter_kind") or "",
            trace.get("result_status") or "",
            trace.get("http_status"),
            trace.get("latency_ms"),
            (trace.get("error_phase") or "")[:32],
            (trace.get("error_message") or "")[:512],
            (trace.get("endpoint_host") or "")[:255],
            (trace.get("chat_model") or "")[:128],
            0,
            None,
            "section",
            section_id,
            Json(request_meta),
            Json(response_meta),
            trace.get("request_body") or "",
            trace.get("response_body") or "",
        ),
    )


def insert_slide_deck_sql_and_params(
    section_id: int,
    title: str,
    content_json_str: str,
    generation_prompt: str,
    admin_id: int,
) -> tuple[str, tuple[Any, ...]]:
    q = """
        INSERT INTO slide_deck
          (section_id, title, deck_status, schema_version, content, generation_prompt,
           created_at, created_by, updated_at, updated_by, is_deleted)
        VALUES (%s, %s, %s, %s, %s::jsonb, %s, NOW(), %s, NOW(), %s, 0)
        RETURNING id
        """
    params = (
        section_id,
        title,
        "draft",
        1,
        content_json_str,
        generation_prompt,
        admin_id,
        admin_id,
    )
    return q, params


def list_sections_for_textbook(
    cur, textbook_id: int, only_without_deck: bool, limit: int | None
) -> list[int]:
    q = """
        SELECT s.id
        FROM section s
        INNER JOIN chapter ch ON ch.id = s.chapter_id AND ch.is_deleted = 0
        WHERE ch.textbook_id = %s AND s.is_deleted = 0
        """
    args: list[Any] = [textbook_id]
    if only_without_deck:
        q += """
          AND NOT EXISTS (
            SELECT 1 FROM slide_deck sd
            WHERE sd.section_id = s.id AND sd.is_deleted = 0
          )
        """
    q += " ORDER BY ch.number, s.number, s.id"
    if limit is not None and limit > 0:
        q += " LIMIT %s"
        args.append(limit)
    cur.execute(q, tuple(args))
    return [int(r[0]) for r in cur.fetchall()]


def resolve_execute_db(arg_val: int) -> bool:
    env = (os.environ.get("EXECUTE_DB") or "").strip().lower()
    if env in ("0", "false", "no"):
        return False
    if env in ("1", "true", "yes"):
        return True
    return bool(arg_val)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description="生成 slide_deck（对齐后台 AI 幻灯片生成逻辑）",
    )
    g = p.add_mutually_exclusive_group(required=True)
    g.add_argument("--section-id", type=int, help="单个 section.id")
    g.add_argument("--textbook-id", type=int, help="教材下所有 section（顺序：章号、节号）")
    p.add_argument(
        "--only-without-deck",
        action="store_true",
        help="仅处理尚无任何 slide_deck 的节（对 --textbook-id 有效）",
    )
    p.add_argument("--limit", type=int, default=0, help="最多处理节数（0 表示不限制）")
    p.add_argument(
        "--no-continue-on-error",
        action="store_true",
        help="批量时某一节失败则立即退出（默认失败则跳过该节继续）",
    )
    p.add_argument(
        "--execute-db",
        type=int,
        default=1,
        choices=[0, 1],
        metavar="0|1",
        help="1=INSERT slide_deck + ai_call_log 并提交；0=只打印 SQL 与日志",
    )
    p.add_argument(
        "--created-by",
        type=int,
        default=-1,
        help="created_by/updated_by；默认读环境变量 STEPUP_CREATED_BY",
    )
    p.add_argument("--timeout", type=int, default=600, help="HTTP 超时（秒），幻灯片 JSON 较大建议放宽")
    p.add_argument("--prompt-file", type=str, default="", help="从文件读入完整提示词（覆盖默认模板）")
    p.add_argument("--quiet", action="store_true", help="仅 WARNING 及以上")
    return p.parse_args()


def main() -> None:
    args = parse_args()
    setup_logging()
    if args.quiet:
        logging.getLogger().setLevel(logging.WARNING)

    execute_db = resolve_execute_db(args.execute_db)
    created_by = args.created_by
    if created_by < 0:
        created_by = int(os.environ.get("STEPUP_CREATED_BY", "0"))

    if execute_db and created_by <= 0:
        log.error("execute_db=1 时需要有效的 --created-by 或 STEPUP_CREATED_BY（>0），与后台 admin id 一致")
        sys.exit(2)

    dsn = connect_dsn()
    log.info("已连接 DSN（脱敏）: %s", redact_dsn(dsn))
    limit = args.limit if args.limit > 0 else None
    continue_on_error = not args.no_continue_on_error

    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor() as cur:
            chat_url: str | None = None
            bearer: str | None = None
            model: str | None = None
            mid: int | None = None
            model_name_snap: str = ""

            def ensure_ai() -> None:
                nonlocal chat_url, bearer, model, mid, model_name_snap
                if chat_url is None:
                    mid_i, name, url, mdl, secret = load_active_ai_model(cur)
                    mid = mid_i
                    model_name_snap = name
                    chat_url = normalize_chat_completions_url(url)
                    bearer = secret
                    model = mdl
                    log.info(
                        "选用 ai_model id=%s name=%s chat_url=%s model=%s",
                        mid,
                        model_name_snap,
                        chat_url,
                        model,
                    )

            if args.textbook_id:
                section_ids = list_sections_for_textbook(
                    cur,
                    args.textbook_id,
                    args.only_without_deck,
                    limit,
                )
                log.info(
                    "教材 textbook_id=%s 待处理 section 数=%s only_without_deck=%s limit=%s",
                    args.textbook_id,
                    len(section_ids),
                    args.only_without_deck,
                    args.limit,
                )
            else:
                section_ids = [args.section_id]
                log.info("单节模式 section_id=%s", section_ids[0])

            if not section_ids:
                log.warning("没有需要处理的 section，退出")
                return

            prompt_override: str | None = None
            if args.prompt_file:
                with open(args.prompt_file, encoding="utf-8") as f:
                    prompt_override = f.read()
                log.info("已加载 --prompt-file，长度=%s 字符", len(prompt_override))

            total = len(section_ids)
            failures: list[tuple[int, str]] = []
            successes = 0

            for n, sid in enumerate(section_ids, start=1):
                log.info("======== [%s/%s] section_id=%s ========", n, total, sid)

                try:
                    ctx = load_section_context(cur, sid)
                    log.info(
                        "节上下文: 书=%s %s 学科=%s 章=%s节 %s 小节=%s节 %s",
                        ctx["textbook_name"],
                        ctx["textbook_version"],
                        ctx["subject"],
                        ctx["chapter_num"],
                        ctx["chapter_title"],
                        ctx["sec_num"],
                        ctx["section_title"],
                    )
                    prompt = (
                        prompt_override.strip()
                        if prompt_override
                        else default_prompt(ctx)
                    )
                    if not prompt:
                        raise RuntimeError("提示词为空")

                    ensure_ai()
                    assert chat_url is not None and bearer is not None and model is not None

                    raw, trace = chat_completions_user_only(
                        chat_url,
                        bearer,
                        model,
                        prompt,
                        timeout=args.timeout,
                        max_tokens=SLIDE_GEN_MAX_OUTPUT_TOKENS,
                    )

                    req_meta = {
                        "section_id": sid,
                        "prompt_chars": len(prompt),
                        "script": "docs/scripts/phys_textbook/phys_tb_slide.py",
                    }

                    try:
                        content_norm = normalize_slide_json(raw)
                    except ValueError as e:
                        resp_meta = {"error": "invalid_slide_json", "detail": str(e)[:800]}
                        trace = {
                            **trace,
                            "result_status": "parse_error",
                            "error_phase": "slide_json_validate",
                            "error_message": str(e)[:512],
                        }
                        if execute_db:
                            insert_ai_call_log(
                                cur,
                                ai_model_id=mid,
                                model_name_snap=model_name_snap,
                                action="slide_deck_generate_ai",
                                trace=trace,
                                section_id=sid,
                                request_meta=req_meta,
                                response_meta=resp_meta,
                            )
                            conn.commit()
                            log.info("JSON 校验失败，已写入 ai_call_log 并 commit")
                        else:
                            log.info(
                                "execute_db=0：跳过 ai_call_log；校验失败摘要 response_meta=%s",
                                preview_json_obj(resp_meta, 1200),
                            )
                        raise RuntimeError(f"幻灯片 JSON 校验失败: {e}") from e

                    title = deck_title_from_json(content_norm)
                    if not title:
                        title = ctx["section_title"] + " · AI草稿"
                    title = trunc_title_runes(title, 200)

                    resp_meta = {"deck_status": "draft", "title_len": len(title)}
                    deck_id_placeholder: int | None = None

                    ins_q, ins_params = insert_slide_deck_sql_and_params(
                        sid, title, content_norm, prompt, created_by
                    )

                    if not execute_db:
                        print_sql_block(
                            f"INSERT slide_deck section_id={sid}（预览，未执行）",
                            mogrify_sql(cur, ins_q, ins_params),
                        )
                        log.info(
                            "execute_db=0：跳过写库；规范化 content 字节长约=%s title=%s",
                            len(content_norm.encode("utf-8")),
                            title,
                        )
                        resp_meta = {**resp_meta, "deck_id": None, "execute_db": 0}
                        log.info(
                            "execute_db=0：将不会在库中写入 ai_call_log；成功轨迹摘要 response_meta=%s",
                            preview_json_obj(resp_meta, 1500),
                        )
                        conn.rollback()
                        log.info("[%s/%s] execute_db=0 结束本节（已 rollback 事务）", n, total)
                    else:
                        cur.execute(ins_q, ins_params)
                        row = cur.fetchone()
                        deck_id_placeholder = int(row[0]) if row else 0
                        resp_meta = {**resp_meta, "deck_id": deck_id_placeholder}
                        log.info(
                            "已插入 slide_deck id=%s section_id=%s title=%s",
                            deck_id_placeholder,
                            sid,
                            title,
                        )
                        insert_ai_call_log(
                            cur,
                            ai_model_id=mid,
                            model_name_snap=model_name_snap,
                            action="slide_deck_generate_ai",
                            trace=trace,
                            section_id=sid,
                            request_meta=req_meta,
                            response_meta=resp_meta,
                        )
                        log.info("已插入 ai_call_log section_id=%s", sid)
                        conn.commit()
                        log.info("[%s/%s] commit 成功 deck_id=%s", n, total, deck_id_placeholder)

                    successes += 1

                except Exception as e:
                    conn.rollback()
                    err_s = str(e)
                    failures.append((sid, err_s))
                    log.exception("[%s/%s] section_id=%s 失败: %s", n, total, sid, err_s)
                    if not continue_on_error:
                        raise
                    continue

            if failures:
                log.error(
                    "结束：成功 %s 节，失败 %s 节。失败详情（前 20 条）: %s",
                    successes,
                    len(failures),
                    failures[:20],
                )
                sys.exit(1)
            log.info("全部成功：共 %s 节", successes)

    finally:
        conn.close()


if __name__ == "__main__":
    main()
