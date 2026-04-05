# 拾级 StepUp - 架构与部署设计（v0.1）

**日期**: 2026-04-03  
**版本**: v0.1  
**适用阶段**: 开发环境 + 生产首发  
**文档导航**: [docs/README.md](../README.md) · [部署与升级说明（20260404#01）](../releases/20260404%2301_DEPLOY_AND_UPGRADE.md) · [部署指南](./deployment_guide_v0.1_260403.md)

---

## 1. 架构结论（先给结论）

采用以下常规生产方案：

- **一个后端 API 服务**（统一业务与数据访问）
- **两个前端应用**
  - `frontend-admin`（后台管理）
  - `frontend-student`（浏览器学生端）
- **一个 MySQL 数据库**
- **一个反向代理网关（Nginx）**，统一域名入口与 HTTPS

该方案是典型的「BFF-less 多端前端 + 统一后端」架构，适合当前 v0.1 和后续扩展（如小程序）。

---

## 2. 系统组件与职责

### 2.1 backend（API 服务）

职责：
- 统一提供 `admin` 与 `student` 的 API
- 鉴权与会话管理（含 `admin_session`）
- 业务处理（上传试卷、AI 分析、改进计划）
- 审计日志记录（`audit_log`）
- 数据访问（MySQL）

建议路由分组：
- `/api/v1/admin/*`
- `/api/v1/student/*`

### 2.2 frontend-admin（后台管理）

职责：
- 管理员登录、会话维持
- 学生管理
- 科目/阶段/AI 模型/Prompt 管理
- 审计日志查询

### 2.3 frontend-student（学生端）

职责：
- 学生登录注册
- 试卷上传
- 分析结果与改进计划查看

### 2.4 MySQL

职责：
- 保存核心业务数据
- 软删除与审计字段规范统一落地

### 2.5 Nginx（网关）

职责：
- HTTPS 终止
- 路由转发到两个前端和后端
- CORS 与基础安全头
- 可选：静态资源缓存

---

## 3. 端口与域名规划（建议）

### 3.1 本地开发端口

- `frontend-student`: `http://localhost:3000`
- `frontend-admin`: `http://localhost:3001`
- `backend`: `http://localhost:8080`
- `mysql`: `localhost:3306`

### 3.2 生产域名建议

- 学生端：`https://app.stepup.xxx`
- 后台端：`https://admin.stepup.xxx`
- API：`https://api.stepup.xxx`

说明：
- 前后端分域名更清晰，权限边界更明确
- 若后续接小程序，可直接复用 `api.stepup.xxx`

---

## 4. 鉴权与会话策略

## 4.1 admin 端

- 登录成功后创建 `admin_session`
- 会话字段至少包括：
  - `session_token`（建议仅保存 hash）
  - `expires_at`
  - `last_seen_at`
  - `ip_address`
  - `user_agent`
  - `status`
- 中间件校验：存在、未过期、未失效、账户 active

### 4.2 student 端

- v0.1 使用验证码 + 密码登录
- 后续可替换为 JWT + refresh token，或同样 session 表模式

### 4.3 权限模型

- 后端必须严格按路由组隔离：
  - `admin` token 不可访问 `student` 受限写操作（除管理视角接口）
  - `student` token 不可访问 `admin` API

---

## 5. 安全基线（v0.1 必做）

- 全量 HTTPS（生产）
- 密码使用 bcrypt
- 严格 CORS 白名单（仅允许对应前端域名）
- 接口限流（至少登录与验证码发送）
- 审计日志脱敏（snapshot 不含密钥、密码、验证码）
- 文件上传白名单（MIME/后缀/大小限制）
- 统一错误码，不回传内部堆栈

---

## 6. 环境变量设计（建议）

后端环境变量：

- `APP_ENV=dev|staging|prod`
- `HTTP_HOST=0.0.0.0`
- `HTTP_PORT=8080`
- `DB_DSN=...`
- `SESSION_TTL_MINUTES=30`（管理端与学生端会话；未设时可使用 legacy `ADMIN_SESSION_TTL_HOURS`）
- `CORS_ALLOWED_ORIGINS=https://app.xxx,https://admin.xxx`
- `AI_TIMEOUT_SECONDS=60`
- `UPLOAD_MAX_MB=20`

AI 模型配置建议：
- 运行时从 DB 读取激活模型（`ai_model`）
- `paper_analysis` 保存 `ai_model_snapshot`（name/url）

---

## 7. Docker 化设计

## 7.1 服务清单

- `mysql`
- `backend`
- `frontend-student`
- `frontend-admin`
- （可选）`nginx`

### 7.2 Dockerfile（建议）

- `backend/Dockerfile`：多阶段构建 Go 二进制
- `frontend-student/Dockerfile`：构建产物 + Nginx 或 Node 静态服务
- `frontend-admin/Dockerfile`：同上

---

## 8. docker-compose（本地开发建议）

建议文件：`docker-compose.yml`

建议编排：
- MySQL 容器映射 3306
- Backend 容器映射 8080，依赖 MySQL 健康检查
- 两个前端容器分别映射 3000/3001
- 同一 bridge network，服务间用 service name 通信

最小原则：
- 使用 `.env` 管理端口和账号密码
- MySQL 挂载 volume 保留本地数据

---

## 9. CI/CD 与部署建议

### 9.1 CI（每次 PR）

- 后端：`go test ./...`
- 前端：lint + build
- 文档：可选 markdown lint

### 9.2 CD（主干部署）

- 构建镜像并打 tag（commit SHA）
- 推送镜像仓库
- 生产环境滚动更新（backend/frontend-admin/frontend-student）
- 保留最近 N 个版本可回滚

### 9.3 数据库迁移

- 建议引入迁移工具（如 golang-migrate）
- 迁移脚本版本化
- 生产禁止手工改表

---

## 10. 可观测性与运维

v0.1 最低要求：
- 健康检查：`/healthz`、`/readyz`
- 结构化日志（request_id, user_id, route, latency, status）
- 错误日志聚合
- 慢查询日志开启（MySQL）

推荐后续：
- Metrics（Prometheus）
- Dashboard（Grafana）
- 异常追踪（Sentry 等）

---

## 11. 分阶段落地计划（建议）

### 阶段 A：骨架与联通
- 三个服务跑起来（backend + 2 frontends）
- 打通前后端联调链路

### 阶段 B：核心闭环
- admin 登录 + session 持久化
- student 登录
- 试卷上传与分析结果落库

### 阶段 C：管理能力
- 模型配置、Prompt 配置、学生管理、审计查询

### 阶段 D：上线前强化
- 限流、审计补齐、监控告警、备份策略、压测与故障演练

---

## 12. 风险与规避

- 风险：两个前端接口契约不一致  
  规避：统一 OpenAPI 或接口文档，接口评审前置

- 风险：会话逻辑混乱导致越权  
  规避：admin/student 鉴权中间件完全分离

- 风险：AI 接口超时影响体验  
  规避：分析任务异步化 + 状态轮询（v0.1 可先同步，尽快升级异步）

- 风险：上传文件安全问题  
  规避：大小、类型、扫描和存储路径隔离

---

## 13. 本文档对应的最终建议

你当前的判断是正确的：  
**分开前后端 + 一个后端 + 两个前端** 是合理、常规、且可直接上生产的方案。  
下一步应按本文档推进容器化与端口拆分，避免后续改动成本。

