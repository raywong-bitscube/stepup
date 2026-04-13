# StepUp API 文档 v0.1（已实现部分）

**日期**: 2026-04-03  
**Base URL（本地）**: `http://localhost:8080`  
**部署 / 升级**: [文档索引](../README.md)、[20260404#01 发版说明](../releases/20260404%2301_DEPLOY_AND_UPGRADE.md)

---

## 1) 通用说明

- 所有请求/响应均为 JSON（除上传接口使用 `multipart/form-data`）。
- 鉴权采用 `Authorization: Bearer <token>`。
- `admin` 和 `student` token 不通用。
- **试卷 HTTP 分析**（环境变量 `ANALYSIS_ADAPTER=http`）：在 **`DB_DSN` 已配置** 时，分析请求发往 MySQL **`ai_model`** 中 **当前激活** 模型（`status=1`、`is_deleted=0`，按 `id` 取最新）的 **`url`**；若无激活模型则用 **`AI_ENDPOINT`**；再无可则用 mock。`paper_analysis` 中保存的模型信息为 `name` + `url`，不含密钥。
- **审计**：同上，仅在 **`DB_DSN` 已配置** 时向 **`audit_log`** 追加记录；`GET /api/v1/admin/audit-logs` 为只读查询。快照字段不落密码 / `app_secret` 正文；AI 模型 PATCH 若包含 `app_secret` 更新，动作为 **`credential_change`**；改学生密码为 **`password_change`**。
- **AI 调用轨迹**：在 **`DB_DSN` 已配置** 且已建 **`ai_call_log`** 表时，各 AI 入口（如 **`paper_analyze`**、幻灯片 **`slide_deck_generate_ai`**、作文提纲等）写入调用记录（适配器类型、HTTP 状态、耗时、错误摘要、`endpoint` 主机、`chat` 模型 id、是否回退 mock、可选 **`student_id`**、业务关联 **`ref_table` + `ref_id`** 等）；**不落** API Key 与完整请求/响应正文。管理端 `GET /api/v1/admin/ai-call-logs` 支持筛选与分页；详见 [`ai_model_log_v0.1_260403.md`](./ai_model_log_v0.1_260403.md)。

---

## 2) 健康检查

### GET `/healthz`

响应：

```json
{"status":"ok"}
```

### GET `/readyz`

就绪探测：

- 未设置 `DB_DSN`：始终 `200`，`{"status":"ready"}`。
- 已设置 `DB_DSN` 但进程未持有可用连接池（例如连接打开失败）：`503`，`{"status":"not_ready","code":"DATABASE_UNAVAILABLE"}`。
- 已持有连接池但 `Ping` 失败：`503`，`{"status":"not_ready","code":"DATABASE_UNREACHABLE"}`。
- 数据库可达：`200`，`{"status":"ready"}`。

### GET `/api/v1/catalog`

**无需鉴权**（供学生端下拉框等使用）。需已配置 `DB_DSN`；无数据库时 `503`，`{"code":"DATABASE_REQUIRED"}`。

成功示例：

```json
{
  "subjects": [{ "id": 1, "name": "物理" }],
  "stages": [{ "id": 1, "name": "高中" }]
}
```

仅返回 `status = 1` 且未软删的科目与阶段。

---

## 3) Admin 鉴权

### 3.1 登录

`POST /api/v1/admin/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

成功响应：

```json
{
  "token": "<admin_token>",
  "expires_at": "2026-04-04T10:00:00Z",
  "user": {
    "username": "admin",
    "role": "super_admin"
  }
}
```

### 3.2 当前登录信息

`GET /api/v1/admin/auth/me`

Header:

```text
Authorization: Bearer <admin_token>
```

### 3.3 退出登录

`POST /api/v1/admin/auth/logout`

Header:

```text
Authorization: Bearer <admin_token>
```

### 3.4 学生列表（管理端）

`GET /api/v1/admin/students`

Header:

```text
Authorization: Bearer <admin_token>
```

成功响应示例：

```json
{
  "items": [
    {
      "id": 1,
      "phone": "13800138000",
      "email": null,
      "name": "示例学生",
      "stage": "高中",
      "status": 1,
      "created_at": "2026-04-03T12:00:00Z"
    }
  ]
}
```

说明：

- 需要已配置 `DB_DSN`（无数据库时返回 `503`，`{"code":"DATABASE_REQUIRED"}`）。

### 3.5 创建学生（管理端）

`POST /api/v1/admin/students`

Header:

```text
Authorization: Bearer <admin_token>
```

请求：

```json
{
  "identifier": "13800138000",
  "password": "12345678",
  "name": "示例学生",
  "stage": "高中"
}
```

成功响应：

```json
{"status":"ok"}
```

说明：

- `identifier` 支持手机号或邮箱。
- 学生密码按 bcrypt 存储。
- `identifier` 冲突返回 `409`，`{"code":"CONFLICT"}`。

### 3.6 更新学生（管理端）

`PATCH /api/v1/admin/students/{studentId}`

Header:

```text
Authorization: Bearer <admin_token>
```

请求（可选字段，至少一个）：

```json
{
  "name": "新名字",
  "stage": "高中",
  "status": 1,
  "password": "newpass123"
}
```

成功响应：

```json
{"status":"ok"}
```

### 3.6.1 学生试卷与分析（管理端，只读）

Header: `Authorization: Bearer <admin_token>`

- `GET /api/v1/admin/students/{studentId}/papers` — 返回 `items`：`id`、`subject`、`stage`、`file_url`、`file_name`、`created_at`。学生不存在时 `404` `NOT_FOUND`。
- `GET /api/v1/admin/students/{studentId}/papers/{paperId}/analysis` — 返回 `analysis`（与学生端分析结构一致，含摘要、薄弱点、模型快照、改进计划摘要等）。
- `GET /api/v1/admin/students/{studentId}/papers/{paperId}/plan` — 返回 `paper_id`、`plan`（数组）、`updated`（与改进计划表一致）。

均需 `DB_DSN`。

### 3.7 科目（管理端）

#### 列表

`GET /api/v1/admin/subjects`

Header: `Authorization: Bearer <admin_token>`

响应示例：

```json
{
  "items": [
    {
      "id": 1,
      "name": "物理",
      "description": "默认科目：物理",
      "status": 1,
      "textbook_count": 1,
      "created_at": "2026-04-03T12:00:00Z"
    }
  ]
}
```

`textbook_count`：该科目下未软删教材数量；管理端列表在 `textbook_count > 0` 时显示「目录」入口。

#### 创建

`POST /api/v1/admin/subjects`

请求：

```json
{
  "name": "化学",
  "description": "可选说明"
}
```

#### 更新

`PATCH /api/v1/admin/subjects/{subjectId}`

请求（可选字段，至少一项）：

```json
{
  "name": "化学（新课标）",
  "description": "",
  "status": 1
}
```

说明：`description` 传空字符串表示置为 `NULL`。

#### 教材目录（管理端，只改不增删）

用于维护已落库的 `textbook` / `textbook_chapter` / `textbook_section`：**仅 PATCH**，不提供创建与删除接口。学生端科目下**至少绑定一本教材**时，管理端「科目」列表行内显示 **目录**，进入盖在主内容上的全屏目录区，**左侧保留管理菜单**，且「科目」菜单项保持高亮（`frontend-admin`）。

均需 `Authorization: Bearer <admin_token>` 与 `DB_DSN`；`textbook_chapter`/`textbook_section` 需已执行迁移（含 `status` 字段、章/节序号无唯一约束及表重命名 **`db/migrations/2026-04-12#01_rename_textbook_chapter_section.sql`**），或当前基线 schema。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/admin/subjects/{subjectId}/textbooks` | 科目须存在。返回该 `subject_id` 下未软删教材 `items`：`id`、`name`、`version`、`subject`、`category`、`remarks`、`status`、`updated_at`。 |
| `PATCH` | `/api/v1/admin/textbooks/{textbookId}` | 可选字段（至少一项）：`name`、`version`、`subject`、`remarks`、`status`（0/1）。**不可**改 `category` / `subject_id`。`remarks` 空串置 `NULL`。`name+version` 与现唯一键冲突时 `409` `CONFLICT`。 |
| `GET` | `/api/v1/admin/textbooks/{textbookId}/chapters` | 教材须存在。`items`：`id`、`textbook_id`、`number`、`title`、`full_title`、`status`、`updated_at`。 |
| `PATCH` | `/api/v1/admin/chapters/{chapterId}` | 可选：`number`、`title`、`full_title`、`status`（至少一项）。`full_title` 空串置 `NULL`。 |
| `GET` | `/api/v1/admin/chapters/{chapterId}/sections` | 章须存在。`items`：`id`、`chapter_id`、`number`、`title`、`full_title`、`status`、`slide_deck_count`（该节未删除的 `slide_deck` 条数）、`updated_at`。 |
| `PATCH` | `/api/v1/admin/sections/{sectionId}` | 可选：`number`、`title`、`full_title`、`status`（至少一项）。 |

#### 章节幻灯片 Slide Deck（管理端）

表 **`slide_deck`**：挂载 **`section_id`**（指向 **`textbook_section.id`**）；`deck_status` 为 `draft` | `active` | `archived`；同一节仅一条 `active`（将某套设为 `active` 时，其余同节 `active` 自动改为 `archived`）。`content` 为 JSON，须含 `schemaVersion: 1` 与 `slides` 数组（结构见 **`docs/core/slide_deck_design_v0.1_260403.md`**）。可选列 **`generation_prompt`**：最近一次 AI 生成使用的完整提示词（由 **`generate-ai`** 写入）。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/admin/sections/{sectionId}/slide-decks` | 节须存在。返回 `items[]`：`id`、`section_id`、`title`、`deck_status`、`schema_version`、`updated_at`（**不含**正文 `content`）。 |
| `POST` | `/api/v1/admin/sections/{sectionId}/slide-decks` | 体：`title`（必填）、`content`（必填，合法 JSON 对象）、`deck_status`（可选，默认 `draft`）、`schema_version`（可选，默认 1）、`generation_prompt`（可选）。成功：`201` + `{ "id", "status":"ok" }`。 |
| `GET` | `/api/v1/admin/sections/{sectionId}/slide-generate/default-prompt` | 节须存在。返回 `{ "prompt": "<默认提示词>" }`，供管理端「生成幻灯片」弹窗预填（可再经本地缓存覆盖）。 |
| `POST` | `/api/v1/admin/sections/{sectionId}/slide-decks/generate-ai` | 体：`{ "prompt": "..." }`。调用当前环境配置的 chat 模型生成合法 slide deck JSON，校验后以 **`draft`** 入库并写入 **`generation_prompt`**；同时追加 **`ai_call_log`**（`action`：`slide_deck_generate_ai`，`ref_table`：`textbook_section`，`ref_id`：节 id）。成功：`201` + `{ "id", "status":"ok" }`；模型返回非合法 JSON 时 `400` `AI_SLIDE_JSON_INVALID`。 |
| `GET` | `/api/v1/admin/slide-decks/{deckId}` | 单条含完整 `content`；若存在则含 **`generation_prompt`**。 |
| `PATCH` | `/api/v1/admin/slide-decks/{deckId}` | 可选：`title`、`content`、`deck_status`、`generation_prompt`（至少一项）。设为 `active` 时同节其他 `active` 归档。 |

均需 `Authorization: Bearer <admin_token>` 与 `DB_DSN`；需已执行 **`db/migrations/2026-04-10#01_slide_deck.sql`** 及后续幻灯片/AI 日志相关迁移（或当前基线 schema 已含 **`slide_deck.generation_prompt`** 与 **`ai_call_log.ref_table`/`ref_id`**）。

### 3.8 阶段（管理端）

与科目相同模式：

- `GET /api/v1/admin/stages`
- `POST /api/v1/admin/stages`
- `PATCH /api/v1/admin/stages/{stageId}`

创建请求示例：

```json
{
  "name": "初中",
  "description": "可选说明"
}
```

科目 / 阶段接口均需 `DB_DSN`；名称唯一冲突返回 `409` `CONFLICT`。

### 3.9 AI 模型（管理端）

列表、创建、更新均需 `Authorization: Bearer <admin_token>` 与 `DB_DSN`。

- `GET /api/v1/admin/ai-models`：返回 `items`（**不含** `app_secret`）。
- `POST /api/v1/admin/ai-models`：请求体 `name`、`url`、**`model`**、`app_secret` 必填；`status` 可选 `0|1`。为兼容旧客户端，仍可传 **`app_key`**（与 `model` 同源，优先 `model`）。将某一模型设为 `status=1` 时，会先将其余模型置为 `0`（同一时间仅一个激活模型）。
- `PATCH /api/v1/admin/ai-models/{modelId}`：可选更新上述字段（至少一项）。将 `status` 设为 `1` 时同样会 deactivate 其他模型。

### 3.10 Prompt 模板（管理端）

模板行由迁移/种子预置，**不支持**通过 API 新增或删除；管理端仅可 **编辑** `description`、`content`、`status`。

- `GET /api/v1/admin/prompts` — 列表（含 `key`、`content`），按 `key` 升序。
- `PATCH /api/v1/admin/prompts/{promptId}` — 可选更新 `description`、`content`、`status`（至少一项）；**不可**修改 `key`。

系统密钥 **`paper_analyze_chat_user`**：学生试卷走 HTTP 分析时，发给大模型的 **user** 文案取自该模板（`status=1`）。占位符由后端替换：`%subject`、`%stage`、`%file_name`。附带试卷图片时由请求中的多模态 **`image_url`** 传入像素内容，由模型自行识图/OCR 与推理，**不**通过占位符预填 OCR 结果。缺行或不可读时回退至后端内置默认模板。

### 3.11 审计日志（管理端）

- `GET /api/v1/admin/audit-logs?limit=100&offset=0` — 只读列表，`limit` 默认 `100`，最大 `500`；`offset` 分页偏移，默认 `0`。响应含 `items`、`limit`、`offset`，按 `id` 降序。
- **写入范围（v0.1）**：管理员登录、学生登录、学生创建试卷（上传）、管理端对上述学生 / 科目 / 阶段 / AI 模型的 **POST 创建** 与 **PATCH 更新**、Prompt 模板仅 **PATCH 更新**（含密码 / secret 类事件的特殊 `action`，见 §1）。需数据库；无 `DB_DSN` 时不写审计表。

### 3.12 AI 调用日志（管理端）

需 **`DB_DSN`**、已执行建表（见 `db/schema/mysql_schema_v0.1_260403.sql` 第 13 节；历史环境另执行 `db/migrations/2026-04-04#05_ai_call_log_request_response_body.sql` 等增量脚本）。

`GET /api/v1/admin/ai-call-logs`

| Query | 说明 |
|--------|------|
| `limit` | 默认 `50`，最大 `200` |
| `offset` | 分页偏移，默认 `0` |
| `ai_model_id` | 精确匹配 `ai_model.id` |
| `action` | 精确匹配，当前多为 `paper_analyze` |
| `result_status` | `success` \| `fallback_mock` \| `mock_only` |
| `adapter_kind` | 如 `http_chat_completions`、`mock_builtin` 等 |
| `from` | 起始时间：`RFC3339` 或日期 `2006-01-02`（按本地日边界 00:00） |
| `to` | 结束时间：日期形式时包含当日直到 **23:59:59.999** |

响应 `items[]` 字段（管理端列表已精简显示项；筛选仍可用 `ai_model_id` 等 query，但响应里不返回试卷/学生 id）：`id`、`created_at`、`model_name_snapshot`、`action`、`adapter_kind`、**`outcome`**（`result_status` 与 HTTP 合并展示：`success` 且 HTTP 200 时仅 `success`，否则含 `· HTTP xxx`）、`latency_ms`、`error_phase`、`error_message`、`endpoint_host`、`chat_model`、`fallback_to_mock`、`request_meta`、`response_meta`、**`request_body`**（出站 chat JSON，**内联图片 base64 已脱敏**）、**`response_body`**（上游响应原文，可截断存储）。

无库或表不存在时：查询可能返回 `503` / `500`；写入侧在表缺失时静默跳过（不影响上传主流程）。

---

## 4) Student 鉴权

### 4.1 发送验证码

`POST /api/v1/student/auth/send-code`

请求：

```json
{
  "identifier": "13800138000"
}
```

> 开发阶段响应会返回验证码（便于本地联调）。

### 4.2 校验验证码

`POST /api/v1/student/auth/verify-code`

请求：

```json
{
  "identifier": "13800138000",
  "code": "123456"
}
```

### 4.3 设置密码

`POST /api/v1/student/auth/set-password`

请求：

```json
{
  "identifier": "13800138000",
  "password": "12345678"
}
```

### 4.4 登录

`POST /api/v1/student/auth/login`

请求：

```json
{
  "identifier": "13800138000",
  "password": "12345678"
}
```

成功响应：

```json
{
  "token": "<student_token>",
  "expires_at": "2026-04-04T10:00:00Z",
  "user": {
    "identifier": "13800138000"
  }
}
```

### 4.5 当前登录信息

`GET /api/v1/student/auth/me`

Header:

```text
Authorization: Bearer <student_token>
```

### 4.6 退出登录

`POST /api/v1/student/auth/logout`

Header:

```text
Authorization: Bearer <student_token>
```

---

## 5) Student 试卷分析流程

> 以下接口都要求 student token。

分析引擎由服务端 **`ANALYSIS_ADAPTER`** 决定：`mock` 为本地占位；`http` 时实际请求 URL 按 §1「试卷 HTTP 分析」优先级解析（激活 `ai_model` → `AI_ENDPOINT` → mock）。

### 5.1 上传试卷

`POST /api/v1/student/papers`

Header:

```text
Authorization: Bearer <student_token>
Content-Type: multipart/form-data
```

表单字段：
- `subject`：例如 `物理`
- `stage`：例如 `高中`
- **`files`**：可重复字段，**最多 10 个**；**全部为图片**时视为同一套试卷多页，模型将收到多张 `image_url`。**仅允许 1 个 PDF**，且 PDF **不可与其他文件混传**。（推荐学生端统一用 `files`。）
- **`file`**：兼容旧客户端的单个文件，与 `files` 二选一（优先使用 `files` 段）。

单文件 ≤ **25MB**；整表 `multipart` 上限约 **120MB**。多图时其余文件路径写入 `exam_paper.extra_file_urls`（需已执行迁移 `db/migrations/2026-04-05#03_exam_paper_extra_file_urls.sql` 或当前基线 schema）。

示例（多图）：

```bash
curl -X POST "http://localhost:8080/api/v1/student/papers" \
  -H "Authorization: Bearer <student_token>" \
  -F "subject=物理" \
  -F "stage=高中" \
  -F "files=@/path/to/p1.jpg" \
  -F "files=@/path/to/p2.jpg"
```

### 5.2 试卷列表

`GET /api/v1/student/papers`

### 5.3 试卷分析结果

`GET /api/v1/student/papers/{paperId}/analysis`

### 5.4 改进计划

`GET /api/v1/student/papers/{paperId}/plan`

### 5.5 作文提纲练习（语文）

需 **`DB_DSN`**（落库 `essay_outline_practice`）及已执行迁移 `db/migrations/2026-04-06#01_essay_outline_practice.sql`（或当前基线 schema）。AI 行为与 §1 试卷分析相同优先级（`ANALYSIS_ADAPTER`、`ai_model` 激活项）。详见 [`feature_essay_outline_v0.1_260403.md`](./feature_essay_outline_v0.1_260403.md)。

#### 5.5.1 生成题目

`POST /api/v1/student/essay-outline/generate-topic`

```json
{ "genre": "议论文", "task_type": "材料作文" }
```

文体枚举：`记叙文` `议论文` `散文` `应用文` `说明文`。命题枚举：`命题作文` `材料作文` `话题作文` `任务驱动型作文`。

成功：

```json
{ "topic_text": "……", "label": "议论文 · 材料作文", "raw": "……" }
```

#### 5.5.2 图片 OCR 题目

`POST /api/v1/student/essay-outline/ocr-topic`，`multipart/form-data` 字段 **`file`**（JPG/PNG，建议 ≤12MB）。

成功：`topic_text`、`label`（默认可为 `自定义`）。

#### 5.5.3 提交提纲点评

`POST /api/v1/student/essay-outline/review`

```json
{
  "topic_text": "题目全文",
  "topic_label": "议论文 · 材料作文",
  "topic_source": "ai_category",
  "genre": "议论文",
  "task_type": "材料作文",
  "outline_text": "提纲全文"
}
```

`topic_source`：`ai_category`（需与合法 genre/task 一致）| `custom_text` | `ocr_image`。

成功：

```json
{
  "id": 1,
  "review": {
    "summary": "……",
    "stars": { "match": 4, "structure": 3, "material": 4 },
    "suggestions": ["……"],
    "highlights": ["……"]
  },
  "raw_review": "……"
}
```

#### 5.5.4 练习记录列表与详情

- `GET /api/v1/student/essay-outline/practices?limit=50` — 当前学生已保存的练习（`essay_outline_practice`，未删除），`limit` 默认 50、最大 100。响应 `items[]`：`id`、`created_at`、`topic_label`、`topic_source`、`topic_preview`（题目摘要）。另含 **`meta`**：`row_count_total`（该 `student_id` 总行数）、`row_count_active`（`is_deleted=0` 行数），便于排查「库里有行但列表为空」（多为已软删）。
- `GET /api/v1/student/essay-outline/practices/{practiceId}` — 单条详情（含 `topic_text`、`outline_text`、`review`、`raw_review` 等）。非本人或不存在：`404` + `NOT_FOUND`。

#### 5.5.5 章节幻灯片（当前 active deck）

- `GET /api/v1/student/sections/{sectionId}/slide-deck` — 返回该节 **`deck_status = active`** 且未删除的一套幻灯片。响应：`id`、`section_id`、`title`、`schema_version`、`updated_at`、`content`（完整 deck JSON）。**`content` 内 `type: question` 节点会移除 `answer` 字段**（若存在），避免暴露标答。节不存在、或无 active deck：`404` + `NOT_FOUND`。

`ai_call_log.action`：`essay_outline_generate_topic` / `essay_outline_ocr_topic` / `essay_outline_review`。

---

## 6) 错误码（当前）

- `INVALID_JSON`
- `INVALID_INPUT`
- `UNAUTHORIZED`
- `CODE_INVALID`
- `CODE_EXPIRED`
- `CODE_USED`
- `PASSWORD_UNSET`
- `FILE_REQUIRED`
- `TOO_MANY_FILES`（多于 10 个文件）
- `PDF_REQUIRES_SINGLE_FILE`（PDF 与多文件混用或多项 PDF）
- `INVALID_IMAGE_BATCH`（多图 batch 中存在非可分析图片或超 10MB 等）
- `INVALID_MULTIPART`
- `NOT_FOUND`
- `CONFLICT`
- `DATABASE_REQUIRED`（管理端依赖 MySQL 的接口在未配置数据库时；`essay-outline/review` 亦同）
- `GENERATE_FAILED` / `OCR_FAILED` / `REVIEW_FAILED`（作文提纲 AI 或落库异常）
- `INVALID_IMAGE`（OCR 接口非图片或非法输入）
- `DATABASE_UNAVAILABLE` / `DATABASE_UNREACHABLE`（`/readyz` 在配置 `DB_DSN` 时）
- `NOT_IMPLEMENTED`（尚未完成的接口）

---

## 7) 说明

- 当前是 v0.1 开发阶段接口文档，后续将补充 OpenAPI 规范。
- 部分功能有「DB 实现 + 内存回退」双模式，用于在无数据库时持续联调。
- `ANALYSIS_ADAPTER=http` 时：有数据库且存在激活 AI 模型则优先用库中 `url`；否则可用 `AI_ENDPOINT`（例如本地 `http://mock-ai:8090/analyze`）；再无则 mock。
- 审计与 `audit_log` 表仅在配置 MySQL 时参与；管理端列表接口不改变审计内容。
