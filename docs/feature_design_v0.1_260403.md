# 拾级 StepUp - 功能清单与设计实现（精简版）v0.1

**日期**: 2026-04-03  
**版本**: v0.1  
**关联文档**:
- `user_requirement_v0.1_260403.md`
- `entity_analyze_v0.1_260403.md`
- `mysql_schema_v0.1_260403.sql`

---

## 1. 目标与范围

v0.1 目标：完成「后台管理 + 浏览器学生端」最小可用闭环。

- 学生端：登录、上传试卷、触发分析、查看分析结果与改进计划
- 后台端：管理员登录、会话管理、学生管理、AI 模型配置、Prompt 配置、审计日志查询
- 数据层：软删除、审计字段、会话表、AI 模型快照字段落库

不在 v0.1 范围：
- 小程序端
- 家长端
- 学生练习闭环（题库做题、错题本、练习评分闭环）

---

## 2. 全量功能清单（v0.1）

| 模块 | 功能 | v0.1 | 优先级 |
|------|------|------|--------|
| 后台鉴权 | 管理员账号密码登录 | 必做 | P0 |
| 后台鉴权 | 后台登录会话创建/续期/退出 | 必做 | P0 |
| 后台鉴权 | 会话失效校验（过期、停用） | 必做 | P0 |
| 学生账户 | 学生手机号/邮箱验证码注册与密码登录 | 必做 | P0 |
| 学生账户 | 学生状态校验（active/inactive） | 必做 | P0 |
| 学生学习 | 上传试卷（PDF/图片） | 必做 | P0 |
| 学生学习 | 创建分析任务并保存结果 | 必做 | P0 |
| 学生学习 | 查看分析结果 | 必做 | P0 |
| 学生学习 | 查看改进计划 | 必做 | P0 |
| 后台管理 | 学生 CRUD（软删除） | 必做 | P1 |
| 后台管理 | 科目配置 | 必做 | P1 |
| 后台管理 | 阶段配置 | 必做 | P1 |
| 后台管理 | AI 模型配置（多配置单激活） | 必做 | P0 |
| 后台管理 | Prompt 配置（按 key） | 必做 | P0 |
| 审计 | 全系统操作日志记录与查询 | 必做 | P0 |
| 通用 | 统一软删除与审计字段规范 | 必做 | P0 |

---

## 3. 架构与分层（简版）

- 后端：`Go + Gin + GORM + MySQL`
- 分层建议：
  - `handler`（HTTP 入参/出参）
  - `service`（业务流程）
  - `repository`（数据库访问）
  - `middleware`（鉴权、审计、错误处理）
  - `domain/model`（实体与 DTO）
- 前端（v0.1）：以服务端渲染页面或轻量交互为主，优先实现流程可用性

---

## 4. 功能设计与实现（按模块）

### 4.1 后台管理登录与会话

**功能点**
- 管理员用户名+密码登录
- 登录成功创建 `admin_session`
- 每次请求校验 session 是否有效
- 退出登录使 session 失效（`status=0`）

**核心数据**
- `admin`
- `admin_session`

**流程（简版）**
1. 登录接口校验 `admin.username + password`
2. 校验 `admin.status=1` 且未软删
3. 生成 session token（建议保存 hash）
4. 写入 `admin_session`（`expires_at/last_seen_at/ip/user_agent`）
5. 返回 session token（cookie 或 header）

**实现要点**
- middleware 统一校验 session：存在、未过期、`status=1`、未软删
- 可采用滑动续期：请求成功时更新 `last_seen_at`（可选）
- 管理员密码必须 bcrypt

---

### 4.2 学生登录与注册

**功能点**
- 学生手机号/邮箱验证码验证
- 首次验证通过后设置密码并创建学生
- 之后使用密码登录

**核心数据**
- `student`
- `verification_code`

**流程（简版）**
1. 输入手机号或邮箱请求验证码
2. 写入 `verification_code`（含过期时间）
3. 验证成功后：若无学生记录则创建；有记录则继续登录流程
4. 密码登录时校验 `student.status=1`

**实现要点**
- 验证码必须校验：未过期、未使用、类型匹配
- 使用成功后置 `is_used=1`
- `phone/email` 允许二选一，但至少一个存在

---

### 4.3 试卷上传与分析

**功能点**
- 学生上传 PDF/图片试卷
- 创建分析任务并落库分析结果
- 单试卷仅一份分析（`paper_id unique`）

**核心数据**
- `exam_paper`
- `paper_analysis`
- `improvement_plan`

**流程（简版）**
1. 上传文件，创建 `exam_paper`
2. 读取当前激活 `ai_model` 与对应 prompt
3. 执行 OCR/AI 分析
4. 写入 `paper_analysis`（含 `ai_model_snapshot`）
5. 生成并写入 `improvement_plan`

**实现要点**
- `paper_analysis.status`：`pending/processing/completed/failed`
- `ai_model_snapshot` 至少存 `name/url`，不落盘密钥
- 分析失败时写失败状态并记录错误日志（可在 `ai_response` 或独立日志）

---

### 4.4 分析结果与改进计划展示

**功能点**
- 学生端查看分析结论
- 学生端查看改进计划
- 后台可查看对应结果

**核心数据**
- `paper_analysis.ai_response`
- `improvement_plan.plan_content`

**实现要点**
- 返回结构可直接透传 JSON，再由前端渲染
- 建议定义统一输出格式（summary/weak_points/plan）

---

### 4.5 后台基础配置（科目/阶段/模型/Prompt）

#### A. 科目配置（Subject）
- 管理科目列表（当前物理、语文）
- 支持启停（`status`）

#### B. 阶段配置（Stage）
- 管理阶段（当前高中）
- 支持启停（`status`）

#### C. AI 模型配置（AIModel）
- 可维护多个模型配置
- 仅允许一个 `status=1`（激活）

#### D. Prompt 配置（Prompt）
- 通过唯一 `key` 管理 prompt 模板
- 支持启停与版本替换（v0.1 可直接覆盖）

**实现要点**
- 启用模型时，需将其它模型置为 inactive（事务处理）
- Prompt key 冲突时返回明确错误

---

### 4.6 学生管理（后台）

**功能点**
- 学生列表查询
- 学生信息修改
- 学生停用/启用
- 软删除

**实现要点**
- 所有删除行为均做软删除
- 查询默认过滤 `is_deleted=0`
- 敏感字段（密码）不在列表接口返回

---

### 4.7 审计日志（AuditLog）

**功能点**
- 记录 login/create/update/delete/password_change 等操作
- 后台支持按时间、用户、实体类型检索

**实现要点**
- update/delete 记录前镜像 `snapshot`（敏感字段脱敏）
- 建议通过 middleware + service 钩子统一写日志，避免漏记

---

## 5. 接口草案（精简）

### 5.1 后台鉴权
- `POST /admin/auth/login`
- `POST /admin/auth/logout`
- `GET /admin/auth/me`

### 5.2 学生鉴权
- `POST /student/auth/send-code`
- `POST /student/auth/verify-code`
- `POST /student/auth/set-password`
- `POST /student/auth/login`

### 5.3 学生学习
- `POST /student/papers`
- `GET /student/papers`
- `GET /student/papers/{paperId}/analysis`
- `GET /student/papers/{paperId}/plan`

### 5.4 后台管理
- `GET/POST/PATCH /admin/students`
- `GET/POST/PATCH /admin/subjects`
- `GET/POST/PATCH /admin/stages`
- `GET/POST/PATCH /admin/ai-models`
- `GET/POST/PATCH /admin/prompts`
- `GET /admin/audit-logs`

---

## 6. 数据与状态约定

- 软删除：`is_deleted=1` 表示删除，不做物理删除
- 启停状态：`status` 统一 `1=active, 0=inactive`
- 审计字段：
  - 必有：`created_at`, `created_by`
  - 可修改记录必有：`updated_at`, `updated_by`
- 分析唯一性：
  - `paper_analysis.paper_id` 唯一
  - `improvement_plan.paper_id` 唯一

---

## 7. 开发优先级建议（直接开工顺序）

1. 公共底座：配置、DB连接、迁移、日志、错误码、鉴权中间件、审计中间件  
2. 后台登录与 session：`admin + admin_session`  
3. 学生登录：验证码 + 密码登录  
4. 试卷上传与分析落库：`exam_paper -> paper_analysis -> improvement_plan`  
5. 后台配置：模型与 Prompt（先保证分析链路可跑）  
6. 后台查询：学生列表、试卷与分析查看、审计日志  

---

## 8. 验收标准（v0.1）

- 管理员可登录后台，并保持有效 session
- 学生可完成登录并上传试卷
- 每份试卷可生成一份分析与改进计划并可查看
- 后台可配置激活模型与 prompt
- 关键操作均可在审计日志查询到
- 所有关键业务数据满足软删除与审计字段规范

---

*本文件为 v0.1 开发启动用精简设计文档，后续可按迭代补充详细接口字段与时序图。*
