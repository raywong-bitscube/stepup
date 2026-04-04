# StepUp API 文档 v0.1（已实现部分）

**日期**: 2026-04-03  
**Base URL（本地）**: `http://localhost:8080`  
**部署 / 升级**: [文档索引](../README.md)、[DEPLOY_AND_UPGRADE_v0.1_260404.md](../releases/DEPLOY_AND_UPGRADE_v0.1_260404.md)

---

## 1) 通用说明

- 所有请求/响应均为 JSON（除上传接口使用 `multipart/form-data`）。
- 鉴权采用 `Authorization: Bearer <token>`。
- `admin` 和 `student` token 不通用。
- **试卷 HTTP 分析**（环境变量 `ANALYSIS_ADAPTER=http`）：在 **`DB_DSN` 已配置** 时，分析请求发往 MySQL **`ai_model`** 中 **当前激活** 模型（`status=1`、`is_deleted=0`，按 `id` 取最新）的 **`url`**；若无激活模型则用 **`AI_ENDPOINT`**；再无可则用 mock。`paper_analysis` 中保存的模型信息为 `name` + `url`，不含密钥。
- **审计**：同上，仅在 **`DB_DSN` 已配置** 时向 **`audit_log`** 追加记录；`GET /api/v1/admin/audit-logs` 为只读查询。快照字段不落密码 / `app_secret` 正文；AI 模型 PATCH 若包含 `app_secret` 更新，动作为 **`credential_change`**；改学生密码为 **`password_change`**。
- **AI 调用轨迹**：在 **`DB_DSN` 已配置** 且已建 **`ai_call_log`** 表时，学生上传触发 **`paper_analyze`** 后写入一条调用记录（适配器类型、HTTP 状态、耗时、错误摘要、`endpoint` 主机、`chat` 模型 id、是否回退 mock、关联 `paper_id`/`student_id` 等）；**不落** API Key 与完整请求/响应正文。管理端 `GET /api/v1/admin/ai-call-logs` 支持筛选与分页；详见 [`ai_model_log_v0.1_260403.md`](./ai_model_log_v0.1_260403.md)。

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
      "created_at": "2026-04-03T12:00:00Z"
    }
  ]
}
```

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
- `POST /api/v1/admin/ai-models`：请求体 `name`、`url`、`app_key`、`app_secret` 必填；`status` 可选 `0|1`。将某一模型设为 `status=1` 时，会先将其余模型置为 `0`（同一时间仅一个激活模型）。
- `PATCH /api/v1/admin/ai-models/{modelId}`：可选更新上述字段（至少一项）。将 `status` 设为 `1` 时同样会 deactivate 其他模型。

### 3.10 Prompt 模板（管理端）

模板行由迁移/种子预置，**不支持**通过 API 新增或删除；管理端仅可 **编辑** `description`、`content`、`status`。

- `GET /api/v1/admin/prompts` — 列表（含 `key`、`content`），按 `key` 升序。
- `PATCH /api/v1/admin/prompts/{promptId}` — 可选更新 `description`、`content`、`status`（至少一项）；**不可**修改 `key`。

系统密钥 **`paper_analyze_chat_user`**：学生试卷走 HTTP 分析时，发给大模型的 **user** 文案取自该模板（`status=1`）。占位符由后端替换：`%subject`、`%stage`、`%file_name`。附带试卷图片时由请求中的多模态 **`image_url`** 传入像素内容，由模型自行识图/OCR 与推理，**不**通过占位符预填 OCR 结果。缺行或不可读时回退至后端内置默认模板。

### 3.11 审计日志（管理端）

- `GET /api/v1/admin/audit-logs?limit=100` — 只读列表，`limit` 默认 `100`，最大 `500`，按 `id` 降序。
- **写入范围（v0.1）**：管理员登录、学生登录、学生创建试卷（上传）、管理端对上述学生 / 科目 / 阶段 / AI 模型的 **POST 创建** 与 **PATCH 更新**、Prompt 模板仅 **PATCH 更新**（含密码 / secret 类事件的特殊 `action`，见 §1）。需数据库；无 `DB_DSN` 时不写审计表。

### 3.12 AI 调用日志（管理端）

需 **`DB_DSN`**、已执行建表（见 `db/schema/mysql_schema_v0.1_260403.sql` 第 13 节；历史环境另执行 `db/migrations/20260408_ai_call_log_request_response_body.sql` 等增量脚本）。

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
- `file`：PDF/JPG/PNG

示例：

```bash
curl -X POST "http://localhost:8080/api/v1/student/papers" \
  -H "Authorization: Bearer <student_token>" \
  -F "subject=物理" \
  -F "stage=高中" \
  -F "file=@/path/to/paper.pdf"
```

### 5.2 试卷列表

`GET /api/v1/student/papers`

### 5.3 试卷分析结果

`GET /api/v1/student/papers/{paperId}/analysis`

### 5.4 改进计划

`GET /api/v1/student/papers/{paperId}/plan`

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
- `INVALID_MULTIPART`
- `NOT_FOUND`
- `CONFLICT`
- `DATABASE_REQUIRED`（管理端依赖 MySQL 的接口在未配置数据库时）
- `DATABASE_UNAVAILABLE` / `DATABASE_UNREACHABLE`（`/readyz` 在配置 `DB_DSN` 时）
- `NOT_IMPLEMENTED`（尚未完成的接口）

---

## 7) 说明

- 当前是 v0.1 开发阶段接口文档，后续将补充 OpenAPI 规范。
- 部分功能有「DB 实现 + 内存回退」双模式，用于在无数据库时持续联调。
- `ANALYSIS_ADAPTER=http` 时：有数据库且存在激活 AI 模型则优先用库中 `url`；否则可用 `AI_ENDPOINT`（例如本地 `http://mock-ai:8090/analyze`）；再无则 mock。
- 审计与 `audit_log` 表仅在配置 MySQL 时参与；管理端列表接口不改变审计内容。
