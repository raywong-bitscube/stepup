# StepUp v0.1 增量：识图分析、Prompt 模板、AI 日志正文与界面优化

**日期**: 2026-04-06  
**适用**: 在已具备 [260404 增量](./DEPLOY_AND_UPGRADE_v0.1_260404.md)（或等效：库内已有 `ai_call_log`）的基础上，部署本批次代码与数据库脚本。  
**关联**: [文档索引](../README.md)、[`db/README.md`](../../db/README.md)、[API §3.10/3.12](../core/api_v0.1_260403.md)

---

## 1. 本轮变更摘要（评审 / 运维）

| 主题 | 说明 |
|------|------|
| **学生试卷 + 视觉模型** | 上传文件落盘至 **`UPLOAD_DIR`**（默认 `data/uploads`，Compose 常为 `/srv/uploads`）；`POST /api/v1/student/papers` 读全文件后调用分析；HTTP 适配器对 Kimi/Moonshot 等发送 **OpenAI 兼容多模态** `image_url`（`data:…;base64`）；请求日志中的 **inline base64 会脱敏** 后再写入 `ai_call_log`。 |
| **静态下载** | 后端挂载 **`GET /uploads/`**，与库中 `file_url`（`/uploads/...`）一致。 |
| **Prompt 模板** | 表 **`prompt_template`** 预置 key **`paper_analyze_chat_user`**（迁移/种子）；分析时的 user 文案由库中 `content` 驱动，占位符 **`%subject` `%stage` `%file_name`**（识图不依赖 `%file_content`）。 |
| **管理端 Prompt** | **仅可编辑**，不提供新增/删除；已移除 **`POST /api/v1/admin/prompts`**；`PATCH` 不修改 `key`。 |
| **AI 调用日志** | 表增加 **`request_body` / `response_body`**（截断约 400KB）；列表接口返回 **`outcome`**（合并业务结果与 HTTP），**不**在 `items` 中返回 `ai_model_id`、`paper_id`、`student_id`、`http_status`；管理端表格 **折叠** 展示请求/响应/Meta。 |
| **默认 AI 模型** | 可选迁移 **`2026-04-05#01_ai_model_kimi_moonshot.sql`**：切至 Moonshot/Kimi 端点（按环境替换密钥）。 |

---

## 2. 数据库升级（已有库，按顺序执行）

在目标库执行（示例与 260404 相同，库名按环境替换）：

```bash
# 迁移文件名含「#」，请在 shell 中对路径加引号。
# 若尚未执行 Kimi 默认模型切换（可选）
mysql … < "db/migrations/2026-04-05#01_ai_model_kimi_moonshot.sql"

# 预置试卷分析 Prompt（无则插入）
mysql … < "db/migrations/2026-04-06#01_prompt_paper_analyze_template.sql"

# 若曾插入含 %file_content 的旧版 Prompt，对齐为当前默认文案（可选，见脚本条件）
mysql … < "db/migrations/2026-04-07#01_prompt_paper_analyze_no_file_content.sql"

# AI 日志：请求/响应正文列（本轮必须，否则新后端列表查询会失败）
mysql … < "db/migrations/2026-04-08#01_ai_call_log_request_response_body.sql"
```

**全新建库**：直接使用当前 **`db/schema/mysql_schema_v0.1_260403.sql`**（已含 `ai_call_log` 新列与结构），再按需 **`db/seed/dev_seed.sql`**；上述迁移中 **仅** `2026-04-05#01` / `2026-04-06#01` 在种子已覆盖时可跳过（仍以团队约定为准）。

---

## 3. 应用与配置

- **Docker Compose**：`backend` 已增加 **`UPLOAD_DIR`**（默认 `/srv/uploads`）与卷 **`stepup_uploads`**；部署后确认容器内目录可写。  
- **非容器**：设置 **`UPLOAD_DIR`** 为持久化目录，并保证进程有写权限。  
- **分析**：`ANALYSIS_ADAPTER=http`，视觉模型在管理端 **`ai_model.model`** 中配置（如 `kimi-k2.5`）。**`AI_REQUEST_TIMEOUT_SECONDS`** 默认 **180**（识图建议不低于 120）。已有库若仍为 `app_key` 列，执行 `db/migrations/2026-04-04#02_ai_model_rename_app_key_to_model.sql`（路径请加引号）。  
- **管理端静态**：须更新 **`frontend-admin`**（Prompt 页、AI 调用日志页）。

---

## 4. 验收建议

1. `readyz` → `200`。  
2. 管理端 → **Prompt 模板** 可见 **`paper_analyze_chat_user`**，仅能编辑保存。  
3. 学生端上传 **图片试卷** → 分析成功；**AI 调用日志** 中可展开 **请求 JSON**（图片处为占位符）、**响应原文**；**状态** 列展示 **`outcome`**。  
4. **`GET /uploads/<文件名>`** 能下载已上传文件（与 `file_url` 一致）。

---

## 5. 文档与基线

- **API**：[`api_v0.1_260403.md`](../core/api_v0.1_260403.md) §3.10、§3.12 已随代码更新。  
- **本文件**：仅描述 **260404 之后至本轮** 的增量；260404 的 checklist 仍可作为前置阅读。
