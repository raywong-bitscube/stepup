# StepUp 文档索引

**建议入口**：发版 / 测试部署请先读 [**部署与升级说明（v0.1 增量）**](./DEPLOY_AND_UPGRADE_v0.1_260404.md)，再按需展开下列文档。

---

## 按用途

| 你想… | 建议阅读 |
|--------|-----------|
| **把本版本发到测试 / 预发** | [DEPLOY_AND_UPGRADE_v0.1_260404.md](./DEPLOY_AND_UPGRADE_v0.1_260404.md) → [deployment_guide_v0.1_260403.md](./deployment_guide_v0.1_260403.md) |
| **接 API、联调前端** | [api_v0.1_260403.md](./api_v0.1_260403.md) |
| **理解运行时与数据流** | [system_runtime_guide_v0.1_260403.md](./system_runtime_guide_v0.1_260403.md) |
| **产品与功能范围** | [user_requirement_v0.1_260403.md](./user_requirement_v0.1_260403.md)、[feature_design_v0.1_260403.md](./feature_design_v0.1_260403.md) |
| **领域与实体** | [entity_analyze_v0.1_260403.md](./entity_analyze_v0.1_260403.md) |
| **拓扑与组件职责** | [architecture_deployment_v0.1_260403.md](./architecture_deployment_v0.1_260403.md) |
| **AI 调用日志（表结构、接口、隐私边界）** | [ai_model_log_v0.1_260403.md](./ai_model_log_v0.1_260403.md) |

---

## 按目录（仓库内）

| 位置 | 内容 |
|------|------|
| **`docs/`** | 本目录：需求、设计、部署、API 说明 |
| **`db/`** | MySQL 基线 schema、增量迁移、开发 seed；说明见 [`db/README.md`](../db/README.md) |
| **`backend/README.md`** | 后端环境变量、本地 / Compose 运行 |

---

## 文档版本约定

- 文件名中的日期（如 `260403`）表示该文档 **基线截稿日**；增量发版说明单独用 **`DEPLOY_AND_UPGRADE_v0.1_260404.md`** 等文件记录，避免大批文档改日期。
- 表结构以 **`db/schema/mysql_schema_v0.1_260403.sql`** 为准；增量以 **`db/migrations/`** 为准。

---

## 速查：测试环境最小路径

1. 配置 `.env` / `.env.qa`，确认 **`CORS_ALLOWED_ORIGINS`**、**`ANALYSIS_ADAPTER`**、数据库账号。  
2. 执行 **`db/schema`**（新库）或 **schema + 必要 migration**（老库），再按需 **`db/seed`**。  
3. `docker compose up -d --build`（或等价编排）。  
4. `readyz` → 管理端登录 → 学生上传 → **AI 调用日志** 有记录。

详细命令与排错见 [部署指南](./deployment_guide_v0.1_260403.md) 与 [升级说明](./DEPLOY_AND_UPGRADE_v0.1_260404.md)。
