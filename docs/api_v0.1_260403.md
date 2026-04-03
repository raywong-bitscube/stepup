# StepUp API 文档 v0.1（已实现部分）

**日期**: 2026-04-03  
**Base URL（本地）**: `http://localhost:8080`

---

## 1) 通用说明

- 所有请求/响应均为 JSON（除上传接口使用 `multipart/form-data`）。
- 鉴权采用 `Authorization: Bearer <token>`。
- `admin` 和 `student` token 不通用。

---

## 2) 健康检查

### GET `/healthz`

响应：

```json
{"status":"ok"}
```

### GET `/readyz`

响应：

```json
{"status":"ready"}
```

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
- `NOT_IMPLEMENTED`（尚未完成的接口）

---

## 7) 说明

- 当前是 v0.1 开发阶段接口文档，后续将补充 OpenAPI 规范。
- 部分功能有「DB 实现 + 内存回退」双模式，用于在无数据库时持续联调。
- 支持 `ANALYSIS_ADAPTER=http` 对接本地 mock-ai（`AI_ENDPOINT=http://mock-ai:8090/analyze`）。
