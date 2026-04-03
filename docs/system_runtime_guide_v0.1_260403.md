# StepUp 系统运行与代码架构说明（给 Java/PHP/Python 开发者）

**日期**: 2026-04-03  
**目标读者**: 熟悉 Java / PHP / Python Web 开发，但不熟 Go 与当前项目实现细节。  
**版本范围**: 当前仓库 `v0.1`（Go backend + 两个静态前端占位页 + MySQL + 可选 mock-ai）。

---

## 1. 先讲结论：这个项目本质上是什么

你可以把它理解成一个标准三层后端应用：

- **入口层**：`net/http` 路由 + handler（类似 Spring Controller / Laravel Controller / FastAPI endpoint）
- **服务层**：`internal/service/*`（业务规则 + DB 操作 + 少量回退逻辑）
- **基础设施层**：MySQL 连接、配置加载、中间件、审计写入

前端目前不是完整业务前端（不是 SPA，也不是完整 HTMX 页面），`frontend-admin` / `frontend-student` 只是**静态占位入口页**，方便部署时占位和页面演示。业务联调主要看后端 API。

---

## 2. 代码目录怎么读（建议阅读顺序）

从后端开始：

1. `backend/cmd/server/main.go`  
   程序入口，调用 `app.Run()`
2. `backend/internal/app/app.go`  
   读取配置、尝试连接 DB、组装路由、启动 HTTP 服务、优雅退出
3. `backend/internal/router/router.go`  
   统一注册所有 API 路由 + 注入 middleware + 注入 service/handler 依赖
4. `backend/internal/handler/**`  
   请求解析、参数校验、调用 service、返回 JSON
5. `backend/internal/service/**`  
   业务核心（登录、试卷、模型、审计、管理端 CRUD）
6. `backend/internal/middleware/**`  
   CORS、Bearer 鉴权、把 session 注入 context
7. `backend/internal/config/config.go`  
   环境变量到配置对象映射

---

## 3. 启动过程（程序生命周期）

`app.Run()` 的流程可以类比成 Spring Boot `main` + bean 初始化：

1. `config.Load()` 读环境变量
2. 如果有 `DB_DSN`，调用 `database.OpenMySQL()` 建连接池
3. `router.New(cfg, db)` 组装所有 handler/service/middleware
4. 启动 `http.Server`
5. 接收 `SIGINT/SIGTERM` 后优雅关闭

关键点：

- **有 DB**：走 DB 持久化逻辑（session、试卷、审计都落库）
- **无 DB**：很多能力会回退到内存实现（便于本地快速跑）

---

## 4. 路由与“依赖注入”怎么工作的

`router.registerAPIRoutes()` 里集中构造依赖（手工 DI）：

- 先创建 service：`adminauth.New(cfg, db)`、`studentpaper.New(cfg, db)` 等
- 再创建 handler：`admin.NewAuthHandler(service, auditWriter)` 等
- 最后把 handler 挂到 `ServeMux`

你可以把这种写法理解为：

- Java 里不用 Spring IoC 容器，手动 `new` 出对象并装配
- Python 里手动构造依赖后注册到 Flask/FastAPI route

鉴权路由用 middleware 包装，例如：

- `RequireAdminAuth(adminAuthService, adminStudentsHandler.List)`
- `RequireStudentAuth(studentAuthService, studentPaperHandler.Create)`

middleware 会校验 token，并把 session/identifier 放进 `context.Context`，后续 handler 直接读取。

---

## 5. 核心业务流：数据和程序怎么走

### 5.1 Admin 登录流

1. `POST /api/v1/admin/auth/login`
2. `admin/auth_handler.go` 解析 JSON 后调用 `adminauth.Service.Login`
3. service：
   - 有 DB：查 `admin` 表 + bcrypt 校验 + 写 `admin_session`
   - 无 DB：用 `ADMIN_BOOTSTRAP_*` 做内存登录
4. 返回 token，后续请求放 `Authorization: Bearer <token>`
5. 同时写一条 `audit_log`（如果 DB 可用）

### 5.2 Student 登录流

路径是验证码 + 设置密码 + 登录：

1. `send-code`：发验证码（开发阶段会在响应里返回 code）
2. `verify-code`
3. `set-password`（bcrypt）
4. `login`（得到 student token）
5. `student_session`（DB 模式）或内存 session（无 DB）

### 5.3 学生上传试卷 + AI 分析流（最关键）

入口：`POST /api/v1/student/papers`（multipart: `subject`, `stage`, `file`）

执行链路：

1. `student/paper_handler.go#Create` 校验上传
2. 调用 `studentpaper.Service.Create`
3. `Create` 若有 DB，走 `createDB`：
   - 找 student / subject
   - 调用 `resolveAdapter()` 决定分析适配器
   - 插入 `exam_paper`
   - 插入 `paper_analysis`（含 `ai_model_snapshot`、summary/weak_points）
   - 插入 `improvement_plan`
4. 记录 `audit_log`（student 创建试卷）
5. `GET /analysis` / `GET /plan` 再从表里取结果

### 5.4 AI 模型解析优先级（`ANALYSIS_ADAPTER=http`）

`studentpaper.resolveAdapter()` 的顺序：

1. 如果 `ANALYSIS_ADAPTER != http`，直接 mock
2. 若有 DB：优先查激活模型 `ai_model(status=1,is_deleted=0)` 最新一条的 `url`
3. 若 DB 未命中：回退 `AI_ENDPOINT`
4. 再没有就 mock

此外：

- 激活模型存在且 `app_secret` 非空时，HTTP adapter 走 OpenAI 兼容 `chat/completions`（Bearer）
- `paper_analysis.ai_model_snapshot` 只保存 `name` + `url`（不保存 secret）

### 5.5 审计日志流

`auditlog.Writer` 在 DB 可用时写 `audit_log`，无 DB 时 no-op。

当前已覆盖的关键动作（v0.1）：

- admin/student 登录
- 学生创建试卷
- 管理端学生/科目/阶段/AI 模型/Prompt 的 create/patch

特点：

- 审计写入有短超时，不阻塞主业务
- `snapshot` 避免敏感字段正文（如密码、`app_secret`）

---

## 6. 数据模型（与你调试最相关的表）

建议重点关注这些表：

- 身份与会话：`admin`, `admin_session`, `student`, `student_session`, `verification_code`
- 业务主链路：`exam_paper`, `paper_analysis`, `improvement_plan`
- 配置：`subject`, `stage`, `ai_model`, `prompt_template`
- 审计：`audit_log`

你可以把 `exam_paper -> paper_analysis -> improvement_plan` 理解成一次“试卷处理流水线”的三段结果快照。

---

## 7. 如何在本地/测试环境跑通（最短路径）

### 7.1 启动

在仓库根目录：

```bash
cp .env.example .env
docker compose up -d --build
```

### 7.2 初始化数据库

```bash
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < docs/mysql_schema_v0.1_260403.sql
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < scripts/dev_seed.sql
```

### 7.3 健康检查

```bash
curl -sS http://localhost:8080/healthz
curl -sS http://localhost:8080/readyz
```

---

## 8. API 冒烟测试脚本（可直接按顺序跑）

### 8.1 管理员登录

```bash
ADMIN_TOKEN=$(curl -sS -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r .token)
echo "$ADMIN_TOKEN"
```

### 8.2 查看学生列表（验证 admin token + DB）

```bash
curl -sS http://localhost:8080/api/v1/admin/students \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### 8.3 学生登录（先走验证码流程）

```bash
# 1) 发验证码（开发环境会返回 code）
CODE=$(curl -sS -X POST http://localhost:8080/api/v1/student/auth/send-code \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"13800138000"}' | jq -r .code)

# 2) 校验验证码
curl -sS -X POST http://localhost:8080/api/v1/student/auth/verify-code \
  -H 'Content-Type: application/json' \
  -d "{\"identifier\":\"13800138000\",\"code\":\"$CODE\"}" | jq

# 3) 设置密码
curl -sS -X POST http://localhost:8080/api/v1/student/auth/set-password \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"13800138000","password":"12345678"}' | jq

# 4) 登录拿 token
STU_TOKEN=$(curl -sS -X POST http://localhost:8080/api/v1/student/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"13800138000","password":"12345678"}' | jq -r .token)
echo "$STU_TOKEN"
```

### 8.4 上传试卷并获取分析/计划

```bash
PAPER_ID=$(curl -sS -X POST http://localhost:8080/api/v1/student/papers \
  -H "Authorization: Bearer $STU_TOKEN" \
  -F "subject=物理" \
  -F "stage=高中" \
  -F "file=@/path/to/sample.pdf" | jq -r .paper.id)

curl -sS "http://localhost:8080/api/v1/student/papers/$PAPER_ID/analysis" \
  -H "Authorization: Bearer $STU_TOKEN" | jq

curl -sS "http://localhost:8080/api/v1/student/papers/$PAPER_ID/plan" \
  -H "Authorization: Bearer $STU_TOKEN" | jq
```

### 8.5 验证审计

```bash
curl -sS "http://localhost:8080/api/v1/admin/audit-logs?limit=50" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

---

## 9. 测试怎么执行（代码层）

当前仓库后端暂未编写大量单测文件，日常验证以“编译检查 + 冒烟链路”为主：

```bash
go test ./...
go vet ./...
```

建议你后续优先补 3 类测试：

1. `studentpaper.resolveAdapter()` 的优先级测试（DB 模型 > AI_ENDPOINT > mock）
2. `adminauth/studentauth` 的会话边界（过期、无效 token）
3. handler 级别的 HTTP 合约测试（状态码 + 错误码）

---

## 10. 你最容易踩的坑（快速排障）

- **`readyz` 返回 503**：`DB_DSN` 不通或 MySQL 未就绪
- **前端跨域报错**：`CORS_ALLOWED_ORIGINS` 没配当前域名
- **分析结果总是 mock**：
  - `ANALYSIS_ADAPTER` 不是 `http`
  - `ai_model` 没有激活行 / URL 为空
  - HTTP 调用失败后自动回退 mock
- **看不到审计日志**：无 DB 或动作未覆盖

---

## 11. 与你熟悉技术栈的映射（速记）

- `handler` ≈ Controller / View Function
- `service` ≈ Service / Domain Service
- `middleware` ≈ Interceptor / Filter / Middleware
- `context.Context` ≈ request-scoped context（携带超时、trace、当前用户）
- `sql.DB` + `QueryRowContext/ExecContext` ≈ 轻量 DAO（当前未引 ORM）

如果你把这个项目当成“无框架但结构化的后端”，理解速度会很快。

