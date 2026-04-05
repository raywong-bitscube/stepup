# AI 模型调用日志 — 需求与设计 v0.1

**日期**: 2026-04-04  
**状态**: 已实现（与当前后端 / 管理端对齐）

**关联文档**: [文档索引](../README.md)、[部署与升级说明（20260404#01）](../releases/20260404%2301_DEPLOY_AND_UPGRADE.md)、[`api_v0.1_260403.md`](./api_v0.1_260403.md)、[`db/schema/mysql_schema_v0.1_260403.sql`](../../db/schema/mysql_schema_v0.1_260403.sql)、[`feature_design_v0.1_260403.md`](./feature_design_v0.1_260403.md)、[`db/README.md`](../../db/README.md)

---

## 1. 业务需求（整理）

### 1.1 背景

学生上传试卷后，后端会按配置调用外部 AI（DeepSeek / mock-ai 协议等）生成分析。运维与产品需要可追溯：**调用是否成功、耗时、HTTP 状态、是否回退到内置 mock、错误阶段与摘要**，便于排查密钥、网络、超时与上游格式问题。

### 1.2 功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| R1 | 每次「试卷分析」触发的 AI 调用写一条日志 | 范围：当前实现为 **学生上传创建 `exam_paper` 时** 同步执行的分析路径（`action = paper_analyze`）。 |
| R2 | 记录成功与失败形态 | 含：**直连模型成功**、**从未走 HTTP（纯 mock）**、**HTTP 失败后回退 mock**；记录 HTTP 状态码、耗时（ms）、错误阶段（timeout / http_status / decode / …）、错误摘要（截断）。 |
| R3 | 不记录敏感与过大载荷 | **禁止**落库：`Authorization`、完整 API Key、完整 prompt、完整上游 JSON。允许：`endpoint` **主机名**、`chat` 模型 id、科目/阶段/文件名等 **request_meta**、输出长度类 **response_meta**。 |
| R4 | 管理端可查询 | 支持按 **AI 模型（id）**、**动作**、**结果状态**、**适配器类型**、**时间范围** 筛选；分页（limit/offset）。 |
| R5 | 与业务关联 | 可选字段：`paper_id`、`student_id`、`ai_model_id`（可空）、`model_name_snapshot`（防模型删除后不可读）。 |

### 1.3 非需求（v0.1 不做）

- 不对管理端「改 AI 模型配置」等操作单独记一条「模型管理」类 AI 日志（仍走 **`audit_log`**）。
- 不提供学生端查看该日志。
- 不在日志中存储完整 request/response body（合规与体积）。

---

## 2. 数据设计

### 2.1 表：`ai_call_log`

见 `db/schema/mysql_schema_v0.1_260403.sql` **第 13 节** 与 `db/migrations/2026-04-04#01_ai_call_log.sql`。

核心字段语义：

- `adapter_kind`: `mock_builtin` | `http_unconfigured` | `http_chat_completions` | `http_mock_ai_protocol`
- `result_status`: `success`（上游可用）| `mock_only`（未配置 HTTP 或仅占位 mock）| `fallback_mock`（曾尝试 HTTP 失败后使用内置 mock 结果）
- `error_phase`: 如 `timeout`、`network`、`http_status`、`decode`、`empty_summary`、`empty_choices` 等
- `fallback_to_mock`: 是否与「非内置 HTTP 成功路径」相比发生了回退（与 `result_status` 一致便于筛选）

### 2.2 写入时机

在 `exam_paper` 插入成功并获得 `paper_id` 后、写入 `paper_analysis` 前（或紧邻事务外），由 `studentpaper.Service` 调用 `ailog.Writer` 追加一行。表不存在或写入失败时 **不阻断** 上传主流程。

---

## 3. API 与前端

- **HTTP**: `GET /api/v1/admin/ai-call-logs`（需管理员 Bearer），参数见 [`api_v0.1_260403.md`](./api_v0.1_260403.md) §3.12。
- **管理 UI**: `frontend-admin` 侧栏 **「AI 调用日志」**，表格展示 + 筛选表单 + 简单上下页。

---

## 4. 运维说明

- **新库**：直接使用更新后的 `db/schema/mysql_schema_v0.1_260403.sql` 初始化即可。
- **已有库**：执行 `db/migrations/2026-04-04#01_ai_call_log.sql`（路径含 `#`，shell 中请加引号）。
- 日志增长与索引：`created_at`、`ai_model_id`、`action`、`result_status` 已建索引；后续可按数据量做归档或分区（超出 v0.1）。

---

## 5. 文档修订清单

| 文档 | 修订内容 |
|------|-----------|
| `docs/README.md`、`docs/core/README.md`、`docs/releases/README.md` | 文档索引与目录说明 |
| `docs/releases/YYYYMMDD#NN_DEPLOY_AND_UPGRADE.md` | 具体发版增量与升级步骤 |
| `db/schema/mysql_schema_v0.1_260403.sql` | 新增 `ai_call_log`，原 `audit_log` 顺延为第 14 节（SQL 统一见 [`db/README.md`](../../db/README.md)） |
| `db/migrations/2026-04-04#01_ai_call_log.sql` | 增量建表 |
| `docs/core/api_v0.1_260403.md` | §1 说明、`§3.12` 接口 |
| `docs/core/deployment_guide_v0.1_260403.md` | 初始化 / 升级时提及 `ai_call_log` |
| `docs/core/feature_design_v0.1_260403.md` | 功能表增加「AI 调用日志」|
| `backend/README.md` | 简述 `ai_call_log` 与 `ANALYSIS_ADAPTER` 行为关系 |

---

## 6. 版本

- **v0.1**：仅 `paper_analyze` 一条动作类型；后续若增加「异步分析任务 / 重试 / 其他 AI 入口」，可扩展 `action` 与写入点，并保持本表字段兼容。
