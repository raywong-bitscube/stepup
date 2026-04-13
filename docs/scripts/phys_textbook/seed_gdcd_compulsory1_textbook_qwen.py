#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 StepUp 的 PostgreSQL 读取「通义千问 / Qwen」ai_model 配置，调用 OpenAI 兼容
chat/completions，按 --textbook-name / --version 等生成「广东省普通高中」对应分册的
章/小节目录，写入 textbook、textbook_chapter、textbook_section 表。

用法示例（仓库根目录）::

    pip install -r docs/scripts/phys_textbook/requirements-textbook-seed.txt
    export DATABASE_URL='postgres://stepup_user:pass@127.0.0.1:5432/stepup?sslmode=disable'
    python docs/scripts/phys_textbook/seed_gdcd_compulsory1_textbook_qwen.py --subject 物理 \\
        --textbook-name '物理（必修第一册）' --version '人教版2019' --category 必修

加 --execute-db 0 时只打印 INSERT SQL（mogrify），不写库；默认 1 为真实 INSERT。

textbook.subject_id：未传 --subject-id 时，会按 subject.name 等于 --subject 在库中解析并写入；
若无匹配科目则 subject_id 为 NULL（需先种子 subject 或显式传入 --subject-id）。

说明与大模型生成内容可能不完全与纸书一致，导入后请在管理端核对。

环境变量:
  DATABASE_URL 或 PG_DSN — PostgreSQL 连接串（与后端 DB_DSN 形式一致）
可选:
  STEPUP_CREATED_BY — 写入 created_by/updated_by，默认 0
  LOG_LEVEL — 日志级别，默认 INFO；设为 DEBUG 可输出更长请求/响应片段
  EXECUTE_DB — 设为 0/1 时覆盖命令行 --execute-db（仅当需要统一用环境变量控制时）
"""

from __future__ import annotations

import argparse
import json
import logging
import os
import sys
from typing import Any
from urllib.parse import urlsplit, urlunsplit

import psycopg2
import requests

log = logging.getLogger("seed_textbook")

# 全流程主步骤数（用于 [i/N] 输出）
STEP_TOTAL = 7

# execute_db=0 时，打印用的占位 id（新建 textbook 时尚无真实 id）
PLACEHOLDER_TEXTBOOK_ID = 999_900
PLACEHOLDER_CHAPTER_ID_BASE = 888_000


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


def step_msg(i: int, msg: str) -> str:
    return f"[步骤 {i}/{STEP_TOTAL}] {msg}"


def redact_dsn(dsn: str) -> str:
    """控制台打印用：隐藏密码。"""
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


def mogrify_sql(cur, query: str, params: tuple[Any, ...]) -> str:
    """把参数嵌入为服务端可读的 SQL 字面量（仅用于打印/测试）。"""
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

# ---------------------------------------------------------------------------
# DB helpers
# ---------------------------------------------------------------------------


def connect_dsn() -> str:
    dsn = (os.environ.get("DATABASE_URL") or os.environ.get("PG_DSN") or "").strip()
    if not dsn:
        print(
            "缺少 DATABASE_URL 或 PG_DSN（与后端 DB_DSN 相同，postgres://...）",
            file=sys.stderr,
        )
        sys.exit(2)
    return dsn


def resolve_subject_id(
    cur, subject_display: str, explicit_id: int | None
) -> int | None:
    """
    textbook.subject_id：优先使用命令行 --subject-id（须存在于 subject 表）；
    否则按 subject.name 精确匹配 --subject（去空格），取 status=1 且未删除的一条。
    """
    if explicit_id is not None:
        cur.execute(
            """
            SELECT id FROM subject
            WHERE id = %s AND is_deleted = 0
            LIMIT 1
            """,
            (explicit_id,),
        )
        row = cur.fetchone()
        if not row:
            raise RuntimeError(
                f"--subject-id={explicit_id} 在 subject 表中不存在或已删除，请核对"
            )
        log.info("subject_id 使用显式参数: id=%s", explicit_id)
        return int(row[0])

    name = (subject_display or "").strip()
    if not name:
        log.warning("学科名为空，无法解析 subject_id，textbook.subject_id 将为 NULL")
        return None

    cur.execute(
        """
        SELECT id, status FROM subject
        WHERE name = %s AND is_deleted = 0
        ORDER BY (status = 1) DESC, id DESC
        LIMIT 1
        """,
        (name,),
    )
    row = cur.fetchone()
    if not row:
        log.warning(
            "subject 表中无 name=%s 的记录，textbook.subject_id 将为 NULL；"
            "请先插入 subject 或传入 --subject-id",
            name,
        )
        return None
    sid, st = int(row[0]), int(row[1])
    if st != 1:
        log.warning("subject id=%s name=%s 的 status=%s（非 1），仍作为外键使用", sid, name, st)
    else:
        log.info("subject_id 已由学科名解析: name=%s -> id=%s", name, sid)
    return sid


def load_qwen_model(cur) -> tuple[str, str, str, int | None]:
    """
    返回 (api_url, bearer_token, model_name, ai_model_id)。
    选取 status=1 且 name/model/url 之一含 qwen（不区分大小写）的最新一条。
    """
    cur.execute(
        """
        SELECT id, url, model, app_secret
        FROM ai_model
        WHERE status = 1 AND is_deleted = 0
          AND (
            LOWER(COALESCE(name, '')) LIKE %s
            OR LOWER(COALESCE(model, '')) LIKE %s
            OR LOWER(COALESCE(url, '')) LIKE %s
          )
        ORDER BY id DESC
        LIMIT 1
        """,
        ("%qwen%", "%qwen%", "%qwen%"),
    )
    row = cur.fetchone()
    if not row:
        raise RuntimeError(
            "未找到可用的 Qwen 模型行：请在 ai_model 中配置 status=1 且名称/型号/URL 含 qwen 的记录"
        )
    mid, url, model, secret = row[0], (row[1] or "").strip(), (row[2] or "").strip(), (
        row[3] or ""
    ).strip()
    if not url or not secret:
        raise RuntimeError("ai_model 中 url 或 app_secret 为空，无法调用接口")
    chat_url = normalize_chat_completions_url(url)
    return chat_url, secret, model or "qwen-plus", mid


def normalize_chat_completions_url(api_base: str) -> str:
    """将库内常见 base URL 补全为 .../chat/completions（与后端 HTTP 适配器习惯一致）。"""
    s = api_base.strip().rstrip("/")
    low = s.lower()
    if low.endswith("chat/completions"):
        return s
    if low.endswith("/v1") or "/v1/" in low:
        return s + "/chat/completions" if not low.endswith("/chat/completions") else s
    return s + "/v1/chat/completions"


# ---------------------------------------------------------------------------
# HTTP — OpenAI-compatible chat completions (DashScope / Qwen)
# ---------------------------------------------------------------------------


def is_dashscope_host(url: str) -> bool:
    h = urlsplit(url).netloc.lower()
    return "dashscope" in h or "aliyuncs.com" in h


def chat_completions(
    url: str,
    bearer: str,
    model: str,
    system: str,
    user: str,
    timeout: int = 180,
) -> str:
    payload: dict[str, Any] = {
        "model": model,
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ],
        "temperature": 0.2,
        "max_tokens": 8192,
    }
    if is_dashscope_host(url):
        payload["enable_thinking"] = False

    log.info(
        "AI 请求 chat/completions: url=%s model=%s timeout=%ss dashscope=%s "
        "enable_thinking=%s system_chars=%s user_chars=%s",
        url,
        model,
        timeout,
        is_dashscope_host(url),
        payload.get("enable_thinking", "<n/a>"),
        len(system),
        len(user),
    )
    log.info("AI 请求 system 全文:\n%s", system)
    log.info("AI 请求 user 摘要 (前 800 字):\n%s", user[:800] + ("…" if len(user) > 800 else ""))
    log.debug(
        "AI 请求 JSON 体(无密钥): %s",
        json.dumps(
            {**payload, "messages_preview": [m["role"] + ":" + str(len(str(m.get("content", "")))) for m in payload["messages"]]},
            ensure_ascii=False,
        ),
    )

    r = requests.post(
        url,
        headers={
            "Authorization": f"Bearer {bearer}",
            "Content-Type": "application/json",
        },
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        timeout=timeout,
    )

    log.info("AI 响应 HTTP %s Content-Length=%s", r.status_code, r.headers.get("Content-Length", "?"))

    if r.status_code >= 400:
        log.error("AI 响应错误体 (前 1200 字): %s", r.text[:1200])
        raise RuntimeError(f"chat/completions HTTP {r.status_code}: {r.text[:800]}")

    data = r.json()
    usage = data.get("usage")
    if usage:
        log.info("AI 响应 usage: %s", usage)
    else:
        log.info("AI 响应 usage: (上游未返回)")

    log.debug("AI 响应 JSON 摘要: %s", preview_json_obj({k: data[k] for k in data if k != "choices"}, 1500))

    choices = data.get("choices") or []
    if not choices:
        log.error("AI 响应无 choices，原始片段: %s", json.dumps(data, ensure_ascii=False)[:1200])
        raise RuntimeError(f"无 choices: {json.dumps(data, ensure_ascii=False)[:800]}")

    msg0 = choices[0].get("message") or {}
    finish = choices[0].get("finish_reason")
    log.info("AI 响应 choice[0] finish_reason=%s message_keys=%s", finish, list(msg0.keys()))

    content = msg0.get("content") or ""
    content = str(content).strip()
    if not content:
        raise RuntimeError("模型返回空 content")

    log.info("AI 解析出的 assistant 正文长度=%s 字符", len(content))
    log.info("AI assistant 正文预览:\n%s", clip_text(content, head=700, tail=400))
    log.debug("AI assistant 正文全文:\n%s", content)

    return content


# ---------------------------------------------------------------------------
# JSON 从模型输出中剥离
# ---------------------------------------------------------------------------


def extract_json_object(text: str) -> dict[str, Any]:
    raw_for_log = text
    text = text.strip()
    if text.startswith("```"):
        log.info("模型输出外包了 markdown 代码围栏，已剥离外层")
        lines = text.split("\n")
        if lines and lines[0].strip().startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        text = "\n".join(lines).strip()
    start = text.find("{")
    if start < 0:
        log.error("无法从模型输出中提取 JSON，原始前 500 字: %s", raw_for_log[:500])
        raise ValueError("输出中未找到 JSON 对象起始")
    depth = 0
    for i in range(start, len(text)):
        if text[i] == "{":
            depth += 1
        elif text[i] == "}":
            depth -= 1
            if depth == 0:
                blob = text[start : i + 1]
                log.info("从模型输出中切出 JSON 对象长度=%s 字符", len(blob))
                try:
                    return json.loads(blob)
                except json.JSONDecodeError as e:
                    log.error("JSON 解析失败: %s 片段: %s", e, blob[:400])
                    raise
    log.error("JSON 花括号不平衡，原始尾部 300 字: %s", raw_for_log[-300:])
    raise ValueError("JSON 花括号不平衡")


# ---------------------------------------------------------------------------
# 截断（对齐 schema 长度）
# ---------------------------------------------------------------------------

TEXTBOOK_NAME_MAX = 50
TEXTBOOK_VERSION_MAX = 50
TEXTBOOK_SUBJECT_MAX = 20
TEXTBOOK_CATEGORY_MAX = 20
TITLE_MAX = 100
FULL_TITLE_MAX = 150


def trunc(s: str | None, n: int) -> str | None:
    if s is None:
        return None
    s = str(s).strip()
    if not s:
        return None
    if len(s) <= n:
        return s
    return s[: n - 1] + "…"


def number_to_chinese_ordinal(n: int) -> str:
    """
    将正整数转为汉字数字，用于「第一章」「第一节」中与种子 SQL 一致的写法（1–99）。
    大于 99 时退回阿拉伯数字字符串。
    """
    if n <= 0:
        return str(n)
    if n > 99:
        return str(n)
    digits = "零一二三四五六七八九"
    if n < 10:
        return digits[n]
    if n == 10:
        return "十"
    if n < 20:
        return "十" + (digits[n % 10] if n > 10 else "")
    tens, ones = divmod(n, 10)
    head = digits[tens] + "十"
    if ones:
        head += digits[ones]
    return head


def resolve_chapter_full_title(cnum: int, ctitle: str, raw: Any) -> str:
    """
    与 db/seed 中 textbook_chapter 格式一致：full_title 优先模型返回值，否则「第{n}章 {title}」。
    """
    ft = trunc(raw, FULL_TITLE_MAX) if raw is not None else None
    if ft:
        return ft
    cn = number_to_chinese_ordinal(cnum)
    base = f"第{cn}章 {ctitle}".strip()
    if len(base) > FULL_TITLE_MAX:
        return base[: FULL_TITLE_MAX - 1] + "…"
    return base


def resolve_section_full_title(snum: int, stitle: str, raw: Any) -> str:
    """
    与 db/seed 中 textbook_section 格式一致：否则「第{n}节 {title}」。
    """
    ft = trunc(raw, FULL_TITLE_MAX) if raw is not None else None
    if ft:
        return ft
    cn = number_to_chinese_ordinal(snum)
    base = f"第{cn}节 {stitle}".strip()
    if len(base) > FULL_TITLE_MAX:
        return base[: FULL_TITLE_MAX - 1] + "…"
    return base


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------


def build_prompt(subject_display: str, textbook_label: str, edition_note: str) -> str:
    return f"""你是熟悉中国大陆普通高中课程与教材目录的编辑。

请列出：**广东省普通高中**，学科为「{subject_display}」，教材为「{textbook_label}」
（{edition_note}）的 **全部章（chapter）与每章下各小节（section）** 目录。

严格要求：
1. 结构应接近国内主流新课标教材（如人教版等）中与**上述教材册次、知识进度**相匹配的常见编排；若有多套主流版本差异，以使用面最广的一种为准，并在小节标题中可略体现知识点名称。
2. 每章包含字段：number（整数，从 1 起全书连续）、title（短标题）、full_title（建议填写如「第一章 运动的描述」，与 title 呼应；若省略脚本会按此格式自动补全）。
3. 每章下 sections 数组：每节 number 从 1 起在本章内连续；full_title 建议如「第一节 xxx」；若省略脚本会补全。
4. **只输出一个 JSON 对象**，不要 markdown 代码围栏，不要任何前言或结语。

JSON 形状示例（仅示意结构）：
{{"chapters":[{{"number":1,"title":"……","full_title":"第一章 ……","sections":[{{"number":1,"title":"……","full_title":"第一节 ……"}}]}}]}}
"""


def load_catalog_via_qwen(
    chat_url: str,
    bearer: str,
    model: str,
    subject_display: str,
    textbook_label: str,
    edition_note: str,
    timeout: int = 180,
) -> dict[str, Any]:
    system = "你只输出合法 JSON，键用英文，不输出 markdown。"
    user = build_prompt(subject_display, textbook_label, edition_note)
    raw = chat_completions(
        chat_url, bearer, model, system, user, timeout=timeout
    )
    parsed = extract_json_object(raw)
    chlist = parsed.get("chapters")
    nch = len(chlist) if isinstance(chlist, list) else 0
    nsec = 0
    if isinstance(chlist, list):
        for ch in chlist:
            if isinstance(ch, dict) and isinstance(ch.get("sections"), list):
                nsec += len(ch["sections"])
    log.info(
        "API 结果解析: 顶层键=%s chapters=%s 小节总数(模型给出)=%s",
        list(parsed.keys()),
        nch,
        nsec,
    )
    log.debug("解析后 JSON 预览:\n%s", preview_json_obj(parsed, 3000))
    return parsed


def ensure_textbook(
    cur,
    name: str,
    version: str,
    subject: str,
    category: str,
    subject_id: int | None,
    created_by: int,
    append_to_existing: bool,
    execute_db: bool,
) -> int:
    name = trunc(name, TEXTBOOK_NAME_MAX) or "教材"
    version = trunc(version, TEXTBOOK_VERSION_MAX) or "必修一"
    subject_dis = trunc(subject, TEXTBOOK_SUBJECT_MAX) or "综合"
    category = trunc(category, TEXTBOOK_CATEGORY_MAX) or "必修"
    cur.execute(
        """
        SELECT id FROM textbook
        WHERE name = %s AND version = %s AND is_deleted = 0
        LIMIT 1
        """,
        (name, version),
    )
    row = cur.fetchone()
    if row:
        tid = int(row[0])
        if not append_to_existing:
            raise RuntimeError(
                f"textbook 已存在 id={tid}（name+version 唯一）。"
                "请更换 --textbook-name / --version，或使用 --append-to-existing 向该教材追加章/节（注意可能重复）。"
            )
        log.info(
            "textbook 已存在，将在其下追加章节: id=%s name=%s version=%s",
            tid,
            name,
            version,
        )
        return tid
    log.info(
        "将新建 textbook: name=%s version=%s subject=%s category=%s subject_id=%s",
        name,
        version,
        subject_dis,
        category,
        subject_id,
    )
    ins_q = """
        INSERT INTO textbook
          (name, version, subject, category, subject_id, status, remarks,
           created_at, created_by, updated_at, updated_by, is_deleted)
        VALUES (%s, %s, %s, %s, %s, 1, NULL, NOW(), %s, NOW(), %s, 0)
        RETURNING id
        """
    ins_params = (
        name,
        version,
        subject_dis,
        category,
        subject_id,
        created_by,
        created_by,
    )
    if not execute_db:
        print_sql_block(
            "INSERT textbook（预览；execute_db=0 未执行）",
            mogrify_sql(cur, ins_q, ins_params),
        )
        print(
            f"\n-- 后续 INSERT chapter 若为新教材，将使用占位 textbook_id={PLACEHOLDER_TEXTBOOK_ID}",
            f"--（正式上线请替换为上一句 RETURNING 的实际 id）\n",
        )
        log.info(
            "execute_db=0：跳过执行 INSERT textbook，返回占位 id=%s 用于生成 chapter SQL",
            PLACEHOLDER_TEXTBOOK_ID,
        )
        return PLACEHOLDER_TEXTBOOK_ID

    cur.execute(ins_q, ins_params)
    new_id = int(cur.fetchone()[0])
    log.info("已插入新 textbook id=%s", new_id)
    return new_id


def insert_chapters_sections(
    cur,
    textbook_id: int,
    chapters: list[dict[str, Any]],
    created_by: int,
    execute_db: bool,
) -> tuple[int, int]:
    """返回 (写入章数, 写入小节数)。execute_db=0 时仅打印 SQL，返回将写入的条数统计。"""
    chapter_rows = 0
    section_rows = 0
    ch_q = """
            INSERT INTO textbook_chapter
              (textbook_id, number, title, full_title, status,
               created_at, created_by, updated_at, updated_by, is_deleted)
            VALUES (%s, %s, %s, %s, 1, NOW(), %s, NOW(), %s, 0)
            RETURNING id
            """
    sec_q = """
                INSERT INTO textbook_section
                  (chapter_id, number, title, full_title, status,
                   created_at, created_by, updated_at, updated_by, is_deleted)
                VALUES (%s, %s, %s, %s, 1, NOW(), %s, NOW(), %s, 0)
                """

    for idx, ch in enumerate(chapters, start=1):
        cnum = int(ch.get("number") or 0)
        ctitle = trunc(ch.get("title"), TITLE_MAX) or f"第{cnum}章"
        cfull = resolve_chapter_full_title(cnum, ctitle, ch.get("full_title"))
        ch_params = (textbook_id, cnum, ctitle, cfull, created_by, created_by)

        if not execute_db:
            print_sql_block(
                f"INSERT chapter [{idx}/{len(chapters)}] {ctitle}",
                mogrify_sql(cur, ch_q, ch_params),
            )
            chapter_id = PLACEHOLDER_CHAPTER_ID_BASE + idx
            log.info(
                "execute_db=0：章 [%s/%s] 使用占位 chapter_id=%s 生成 section SQL",
                idx,
                len(chapters),
                chapter_id,
            )
        else:
            cur.execute(ch_q, ch_params)
            chapter_id = int(cur.fetchone()[0])
            log.info(
                "已写入章 [%s/%s]: chapter_id=%s number=%s title=%s",
                idx,
                len(chapters),
                chapter_id,
                cnum,
                ctitle,
            )

        chapter_rows += 1
        sections = ch.get("sections") or []
        sec_count = 0
        if not isinstance(sections, list):
            log.warning("章 %s (%s) 的 sections 非数组，已跳过小节", idx, ctitle)
            continue
        for sec in sections:
            if not isinstance(sec, dict):
                continue
            snum = int(sec.get("number") or 0)
            stitle = trunc(sec.get("title"), TITLE_MAX) or f"第{snum}节"
            sfull = resolve_section_full_title(snum, stitle, sec.get("full_title"))
            sec_params = (chapter_id, snum, stitle, sfull, created_by, created_by)
            if not execute_db:
                print_sql_block(
                    f"  INSERT section 章{idx} §{snum} {stitle}",
                    mogrify_sql(cur, sec_q, sec_params),
                )
            else:
                cur.execute(sec_q, sec_params)
            section_rows += 1
            sec_count += 1

        log.info(
            "章 [%s/%s] number=%s title=%s 小节数=%s",
            idx,
            len(chapters),
            cnum,
            ctitle,
            sec_count,
        )
    return chapter_rows, section_rows


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description="用库内 Qwen 配置生成广东高中必修一目录并写入 textbook/chapter/section",
    )
    p.add_argument(
        "--subject",
        required=True,
        help="学科展示名，写入 textbook.subject，如：物理、语文、数学",
    )
    p.add_argument(
        "--textbook-name",
        required=True,
        help="教科书名称简称，写入 textbook.name（≤50 字符，将自动截断）",
    )
    p.add_argument(
        "--version",
        default="人教版2019",
        help="版本/册别说明，写入 textbook.version（默认：人教版2019）",
    )
    p.add_argument(
        "--category",
        default="必修",
        help="教材类别，写入 textbook.category（默认：必修）",
    )
    p.add_argument(
        "--edition-note",
        default="依据普通高中课程标准，按本教材书名与版本所对应分册的常见教学编排",
        help="写入提示词括号内说明；若分册与默认口径不符请自行改写",
    )
    p.add_argument(
        "--textbook-label",
        default="",
        help="传给模型的教材描述（教材为「…」）；默认「textbook-name · version · category」",
    )
    p.add_argument(
        "--subject-id",
        type=int,
        default=None,
        help="可选：直接指定 subject id；省略时按 subject.name = --subject 自动解析",
    )
    p.add_argument(
        "--dry-run",
        action="store_true",
        help="只请求模型并打印 JSON，不写库",
    )
    p.add_argument(
        "--print-model",
        action="store_true",
        help="启动时打印所选 ai_model 的 id、url、model（脱敏 app_secret）",
    )
    p.add_argument(
        "--timeout",
        type=int,
        default=180,
        help="HTTP 超时秒数",
    )
    p.add_argument(
        "--append-to-existing",
        action="store_true",
        help="若 name+version 已存在，向其追加章节（可能产生重复序号，请自行清理）",
    )
    p.add_argument(
        "--quiet",
        action="store_true",
        help="仅警告及以上（简化控制台输出）；默认 INFO 详见步骤与 AI 摘要",
    )
    p.add_argument(
        "--execute-db",
        type=int,
        default=1,
        choices=[0, 1],
        metavar="0|1",
        help="1=执行 INSERT 并提交；0=仅打印将执行的 SQL（仍会连库读 ai_model、SELECT textbook）。可用环境变量 EXECUTE_DB=0|1 覆盖默认值",
    )
    return p.parse_args()


def resolve_execute_db(arg_val: int) -> bool:
    env = (os.environ.get("EXECUTE_DB") or "").strip().lower()
    if env in ("0", "false", "no"):
        return False
    if env in ("1", "true", "yes"):
        return True
    return bool(arg_val)


def main() -> None:
    args = parse_args()
    setup_logging()
    if args.quiet:
        logging.getLogger().setLevel(logging.WARNING)

    execute_db = resolve_execute_db(args.execute_db)

    log.info(step_msg(1, "开始：解析参数与连接信息"))
    log.info(
        "参数 subject=%s textbook_name=%s version=%s category=%s dry_run=%s execute_db=%s append_to_existing=%s subject_id=%s created_by(env)=%s",
        args.subject,
        args.textbook_name,
        args.version,
        args.category,
        args.dry_run,
        int(execute_db),
        args.append_to_existing,
        args.subject_id,
        int(os.environ.get("STEPUP_CREATED_BY", "0")),
    )

    dsn = connect_dsn()
    log.info(step_msg(1, f"已读取 DSN（脱敏）: {redact_dsn(dsn)}"))
    created_by = int(os.environ.get("STEPUP_CREATED_BY", "0"))

    textbook_label = (args.textbook_label or "").strip()
    if not textbook_label:
        textbook_label = (
            f"{args.textbook_name.strip()} · {args.version.strip()} · {args.category.strip()}"
        )
    log.info("模型侧教材描述 textbook_label=%s", textbook_label)

    log.info(step_msg(2, "连接 PostgreSQL…"))
    conn = psycopg2.connect(dsn)
    log.info("数据库连接成功")

    try:
        with conn.cursor() as cur:
            log.info(step_msg(3, "查询 ai_model 中激活的 Qwen 配置…"))
            chat_url, bearer, model, mid = load_qwen_model(cur)
            log.info(
                "已选中 ai_model: id=%s chat_url=%s model=%s app_secret=已加载(长度=%s)",
                mid,
                chat_url,
                model,
                len(bearer),
            )
            if args.print_model:
                log.info("(兼容 --print-model) 与上方相同，密钥已脱敏")

            log.info(step_msg(4, "调用大模型 chat/completions 生成目录…"))
            data = load_catalog_via_qwen(
                chat_url,
                bearer,
                model,
                args.subject.strip(),
                textbook_label,
                args.edition_note,
                timeout=args.timeout,
            )

            chapters = data.get("chapters")
            log.info(step_msg(5, "校验并统计模型返回的 chapters…"))
            if not isinstance(chapters, list) or not chapters:
                raise RuntimeError("模型返回 JSON 中缺少非空 chapters 数组")

            nsec_model = sum(
                len(ch["sections"])
                for ch in chapters
                if isinstance(ch, dict) and isinstance(ch.get("sections"), list)
            )
            log.info(
                "校验通过: 章数=%s 模型给出小节数=%s",
                len(chapters),
                nsec_model,
            )

            if args.dry_run:
                log.info(step_msg(6, "dry-run：跳过数据库写入，打印完整 JSON 到 stdout"))
                print(json.dumps(data, ensure_ascii=False, indent=2))
                log.info(step_msg(7, "结束（dry-run，未提交事务）"))
                return

            subject_id = resolve_subject_id(cur, args.subject.strip(), args.subject_id)

            if not execute_db:
                log.info(
                    step_msg(
                        6,
                        "execute_db=0：仅打印 SQL，不执行 INSERT；开始处理 textbook…",
                    )
                )
                print(
                    "\n-- ########## SQL 预览（execute_db=0，以下语句未在数据库执行）##########\n"
                )
            else:
                log.info(step_msg(6, "处理 textbook（执行 INSERT）…"))

            tid = ensure_textbook(
                cur,
                args.textbook_name.strip(),
                args.version.strip(),
                args.subject.strip(),
                args.category.strip(),
                subject_id,
                created_by,
                args.append_to_existing,
                execute_db,
            )
            if tid == PLACEHOLDER_TEXTBOOK_ID:
                log.info(
                    "textbook_id=%s（占位，仅用于预览 chapter/section SQL）",
                    tid,
                )
            else:
                log.info("textbook_id=%s（真实 id）", tid)

            log.info(step_msg(7, "处理 chapter / section…"))
            ch_written, sec_written = insert_chapters_sections(
                cur, tid, chapters, created_by, execute_db
            )
            if execute_db:
                conn.commit()
                log.info(
                    "完成：textbook_id=%s 写入章行=%s 小节行=%s（模型原始章数=%s）",
                    tid,
                    ch_written,
                    sec_written,
                    len(chapters),
                )
            else:
                conn.rollback()
                log.info(
                    "SQL 预览结束：本应写入 章=%s 小节=%s（未提交，已 rollback）",
                    ch_written,
                    sec_written,
                )
    except Exception:
        conn.rollback()
        log.exception("执行失败，已 rollback")
        raise
    finally:
        conn.close()
        log.info("数据库连接已关闭")


if __name__ == "__main__":
    main()
