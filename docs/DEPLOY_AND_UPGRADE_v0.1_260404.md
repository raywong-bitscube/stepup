# StepUp v0.1 增量：设计说明与部署 / 升级指南

**日期**: 2026-04-04  
**适用**: 自本提交起准备发往 **测试 / QA** 或同级环境时对照执行。  
**关联**: [文档索引](./README.md)、[部署指南](./deployment_guide_v0.1_260403.md)、[`db/README.md`](../db/README.md)

---

## 1. 本次增量包含什么（给评审 / 运维）

| 主题 | 说明 |
|------|------|
| **AI 调用日志** | 学生上传试卷、`createDB` 同步分析结束后写入表 **`ai_call_log`**；记录适配器类型、HTTP 状态、耗时、错误阶段与摘要、是否回退 mock、关联试卷与学生等；**不落库** API Key 与完整请求/响应体。 |
| **管理端** | 新菜单 **「AI 调用日志」**；接口 **`GET /api/v1/admin/ai-call-logs`**（筛选与分页，需管理员 Token）。 |
| **后端** | `AnalysisAdapter` 返回 **`AnalyzeResult`**（含 **`AnalyzeTrace`**）；HTTP 适配器覆盖成功 / 超时 / 网络 / 解析失败等路径。 |
| **SQL 目录** | 所有 MySQL 脚本统一到仓库 **`db/`**（`schema/`、`migrations/`、`seed/`）；**不再**使用 `docs/*.sql`、`scripts/dev_seed.sql` 等旧路径。 |
| **前端（部署拓扑）** | 学生 / 管理静态与 API 分端口时：`app.js` 内约定 **页面端口 7010 / 7011 → 同主机 API 7012**（可用 `?api=` / `localStorage` 覆盖）；管理登录页 **API 根地址** 可选。 |

详细需求与字段语义见 [**ai_model_log_v0.1_260403.md**](./ai_model_log_v0.1_260403.md)；接口见 [**api_v0.1_260403.md**](./api_v0.1_260403.md) §3.12。

---

## 2. 设计要点（精简）

### 2.1 AI 调用日志

- **写入时机**：`exam_paper` 插入成功并取得 **`paper_id`** 之后，与 **`paper_analysis`** 写入同一业务流程内；`ailog.Writer` 短超时执行，**失败不阻断**上传。  
- **无库模式**：未配置 `DB_DSN` 时不写表。  
- **无表环境**：写入静默失败；需先执行 DDL（见 §3）。  

### 2.2 数据库脚本位置

- **新库**：`db/schema/mysql_schema_v0.1_260403.sql`（含 **`ai_call_log`**，§13）。  
- **旧库**：若此前已按旧版 schema 建库且 **没有** `ai_call_log`，执行  
  `db/migrations/20260404_ai_call_log.sql`  
  （仅 `CREATE TABLE IF NOT EXISTS`，**不**含 `USE`；适合与 Compose 中 **`MYSQL_DATABASE`** 任意库名对齐。）

### 2.3 基线 schema 与库名

- `db/schema/mysql_schema_v0.1_260403.sql` 含 **`CREATE DATABASE`** / **`USE stepup`**。  
- 若测试库名 **不是** `stepup`（例如 `stepup_qa`）：用 CLI 指定库执行时，**仍会被脚本 `USE stepup` 切走**。建议二选一：  
  - **优先**：老库用 **`db/migrations/20260404_ai_call_log.sql`** 只补表；或  
  - 临时在导入前编辑 schema，使 `USE` 与目标库名一致（团队自行规范，勿提交真实环境特有修改）。

---

## 3. 升级 checklist（已有测试库 / 已有部署）

按顺序执行；**前一阶段成功再进下一阶段**。

1. **代码**  
   - 拉取含本功能的 **tag / commit**；重新构建 **backend** 镜像或二进制。  
   - 部署 **frontend-admin** 静态资源（须含「AI 调用日志」与 `app.js` 更新）。**frontend-student** 若使用分端口 + 7012 API，请一并更新。

2. **数据库**  
   - 在目标库执行（库名按环境替换，示例为 Compose 变量）：  
     ```bash
     docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
       < db/migrations/20260404_ai_call_log.sql
     ```  
   - **新搭测试库**：可直接执行 `db/schema/...` 再 `db/seed/...`，并注意 §2.3 库名问题。

3. **配置**  
   - `ANALYSIS_ADAPTER=http`（若要用真实模型）。  
   - **`CORS_ALLOWED_ORIGINS`** 含学生端、管理端 **浏览器 Origin**（含协议与端口）。  
   - 管理端 **AI 模型** 中激活模型 URL、`app_secret`（DeepSeek 等）已配置。

4. **验证**  
   - `GET .../readyz` → `200`。  
   - 管理端登录 → **AI 调用日志** 打开无报错（空表正常）。  
   - 学生端上传一份试卷 → 日志中出现 **`paper_analyze`**；DeepSeek 正常时 **`result_status`** 多为 `success`，异常时可见 `fallback_mock` 与 **`error_*`** 字段。

5. **回滚**（仅需知悉）  
   - 回滚应用版本后，**可不删除** `ai_call_log` 表（旧代码忽略即可）。若必须删表，在业务低峰执行 `DROP TABLE ai_call_log`（注意外键与备份）。

---

## 4. 全新测试环境（空库）最短路径

与 [部署指南 §3](./deployment_guide_v0.1_260403.md) 一致，核心三条：

```bash
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < db/schema/mysql_schema_v0.1_260403.sql
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < db/seed/dev_seed.sql
```

若 `MYSQL_DATABASE` ≠ `stepup`，请先阅读上文 **§2.3**，必要时仅用 migration 补表或调整 `USE`。

---

## 5. 文档维护说明

- **本文件**：记录 **与该发版相关的增量设计与升级步骤**；大段落基础设施仍以 **deployment_guide / architecture** 为准。  
- **以后发版**：可复制本文件为新日期版本，或在本文件追加 §6「历史」；索引入口始终为 [**docs/README.md**](./README.md)。
