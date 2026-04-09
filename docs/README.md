# StepUp 文档索引

文档分两类：

| 目录 | 用途 |
|------|------|
| **[`docs/core/`](./core/)** | **v0.1 基线**：需求、架构、API、部署指南、运行时、实体与功能说明等（长期对照，随大版本更新文件名或目录）。 |
| **[`docs/releases/`](./releases/)** | **发版增量**：某次上线的变更说明、升级 checklist、评审摘要（按日期/批次追加新文件，不必改基线文档日期）。 |
| **[`docs/deploy/`](./deploy/)** | **部署样例**：与文档配套的 Nginx 等可复制配置（非运行时目录）。 |

**建议入口**：发往测试 / 预发请按序阅读 [**releases/20260404#01**](./releases/20260404%2301_DEPLOY_AND_UPGRADE.md)（AI 日志与 `db/` 等）、[**20260404#02**](./releases/20260404%2302_DEPLOY_AND_UPGRADE.md)（识图、Prompt、日志正文）、[**20260405#01**](./releases/20260405%2301_DEPLOY_AND_UPGRADE.md)（多图上传、种子与迁移归并）、[**20260405#02**](./releases/20260405%2302_DEPLOY_AND_UPGRADE.md)（CORS / Go+Nginx / 学生端稳定性）、[**20260408#01**](./releases/20260408%2301_DEPLOY_AND_UPGRADE.md)（教材目录表与粤教版物理必修种子），并按需展开 **core** 中文档。

---

## 按用途（链接至 core）

| 你想… | 建议阅读 |
|--------|-----------|
| **把本版本发到测试 / 预发** | [20260404#01](./releases/20260404%2301_DEPLOY_AND_UPGRADE.md) → [20260404#02](./releases/20260404%2302_DEPLOY_AND_UPGRADE.md) → [20260405#01](./releases/20260405%2301_DEPLOY_AND_UPGRADE.md) → [20260405#02](./releases/20260405%2302_DEPLOY_AND_UPGRADE.md)；再结合 [core/部署指南](./core/deployment_guide_v0.1_260403.md) |
| **接 API、联调前端** | [core/API](./core/api_v0.1_260403.md) |
| **理解运行时与数据流** | [core/系统运行说明](./core/system_runtime_guide_v0.1_260403.md) |
| **产品与功能范围** | [core/需求](./core/user_requirement_v0.1_260403.md)、[core/功能清单](./core/feature_design_v0.1_260403.md) |
| **领域与实体** | [core/实体分析](./core/entity_analyze_v0.1_260403.md) |
| **拓扑与组件职责** | [core/架构与部署设计](./core/architecture_deployment_v0.1_260403.md) |
| **AI 调用日志（表结构、接口、隐私边界）** | [core/ai_model_log](./core/ai_model_log_v0.1_260403.md) |

**增量说明列表**：见 [**releases/README.md**](./releases/README.md)。

---

## 按仓库目录

| 位置 | 内容 |
|------|------|
| **`docs/`** | 本索引 + `core/` + `releases/` + `deploy/`（示例配置） |
| **`db/`** | MySQL schema / migrations / seed；[`db/README.md`](../db/README.md) |
| **`backend/README.md`** | 后端环境与运行 |

---

## 文档版本约定

- **core** 文件名中的日期（如 `260403`）表示该基线文档 **截稿日**；小功能可在 **releases** 中单开说明，避免整批改日期。
- 表结构以 **`db/schema/mysql_schema_v0.1_260403.sql`** 为准；增量 DDL 以 **`db/migrations/`** 为准。

---

## 速查：测试环境最小路径

1. 配置 `.env` / `.env.qa`，确认 **`CORS_ALLOWED_ORIGINS`**、**`ANALYSIS_ADAPTER`**、数据库账号。  
2. 执行 **`db/schema`**（新库）或 **schema + 必要 migration**（老库），再按需 **`db/seed`**。  
3. `docker compose up -d --build`（或等价编排）。  
4. `readyz` → 管理端登录 → 学生上传 → **AI 调用日志** 有记录。

详细命令与排错见 [core/部署指南](./core/deployment_guide_v0.1_260403.md) 与 [releases/20260404#01](./releases/20260404%2301_DEPLOY_AND_UPGRADE.md)、[20260404#02](./releases/20260404%2302_DEPLOY_AND_UPGRADE.md)、[20260405#01](./releases/20260405%2301_DEPLOY_AND_UPGRADE.md)、[20260405#02](./releases/20260405%2302_DEPLOY_AND_UPGRADE.md)。
