# StepUp API 文档 v0.1（已实现部分）

**日期**: 2026-04-03  
**Base URL（本地）**: `http://localhost:8080`

---

## 1) 通用说明

- 所有请求/响应均为 JSON（除上传接口使用 `multipart/form-data`）。
- 鉴权采用 `Authorization: Bearer <token>`。
- `admin` 和 `student` token 不通用。
- **试卷 HTTP 分析**（环境变量 `ANALYSIS_ADAPTER=http`）：在 **`DB_DSN` 已配置** 时，分析请求发往 MySQL **`ai_model`** 中 **当前激活** 模型（`status=1`、`is_deleted=0`，按 `id` 取最新）的 **`url`**；若无激活模型则用 **`AI_ENDPOINT`**；再无可则用 mock。`paper_analysis` 中保存的模型信息为 `name` + `url`，不含密钥。
- **审计**：同上，仅在 **`DB_DSN` 已配置** 时向 **`audit_log`** 追加记录；`GET /api/v1/admin/audit-logs` 为只读查询。快照字段不落密码 / `app_secret` 正文；AI 模型 PATCH 若包含 `app_secret` 更新，动作为 **`credential_change`**；改学生密码为 **`password_change`**。

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

- `GET /api/v1/admin/prompts` — 列表（含 `key`、`content`）
- `POST /api/v1/admin/prompts` — `key`、`content` 必填；`description`、`status` 可选
- `PATCH /api/v1/admin/prompts/{promptId}` — 局部更新；`key` 冲突返回 `409`

### 3.11 审计日志（管理端）

- `GET /api/v1/admin/audit-logs?limit=100` — 只读列表，`limit` 默认 `100`，最大 `500`，按 `id` 降序。
- **写入范围（v0.1）**：管理员登录、学生登录、学生创建试卷（上传）、管理端对上述学生 / 科目 / 阶段 / AI 模型 / Prompt 资源的 **POST 创建** 与 **PATCH 更新**（含密码 / secret 类事件的特殊 `action`，见 §1）。需数据库；无 `DB_DSN` 时不写审计表。

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
