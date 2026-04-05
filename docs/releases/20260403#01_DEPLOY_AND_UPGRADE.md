# StepUp v0.1 增量：多图试卷、科目与 Qwen 种子、迁移归并、发版文档命名

**发版批次**: `20260403#01`  
**日期**: 2026-04-03  
**适用**: 在已按 [20260404#02](./20260404%2302_DEPLOY_AND_UPGRADE.md)（或等效能力：识图、`UPLOAD_DIR`、`prompt_template`、`ai_call_log` 含 `request_body`/`response_body`）就绪的基础上，合入本批代码与数据库脚本。  
**关联**: [文档索引](../README.md)、[`db/README.md`](../../db/README.md)、[API 学生试卷上传](../core/api_v0.1_260403.md)

---

## 1. 本轮变更摘要（评审 / 运维）

| 主题 | 说明 |
|------|------|
| **多图上传** | `POST /api/v1/student/papers` 支持 `multipart/form-data` 字段 **`files`**（多文件），兼容原单字段 **`file`**。首张写入 `exam_paper.file_url`，其余 `/uploads/...` 路径以 JSON 数组写入 **`extra_file_urls`**（须执行 §2 DDL）。单文件 ≤ **25MB**，整表 **`multipart`** 上限约 **120MB**。 |
| **视觉分析** | `studentpaper` 将多图一并参与 HTTP 视觉分析（与 #02 OpenAI 兼容多模态链路一致）。 |
| **学生前端** | 科目入口与多图选择 UI；更新 **`frontend-student/app.js`**、**`styles.css`**。 |
| **种子与科目** | 迁移 **`2026-04-05#02_seed_subjects_math_english_qwen.sql`**：插入科目 **数学**、**英语**；`ai_model` 增加 Dashscope 兼容 **qwen** 行（**默认未激活**，需替换 `app_secret` 后按需 `status=1`）。 |
| **基线 schema** | **`db/schema/mysql_schema_v0.1_260403.sql`** 已含 **`extra_file_urls`**；**`db/seed/dev_seed.sql`** 与仓库内文档交叉链接随本轮调整。 |
| **迁移归并** | 移除旧文件名 `2026-04-06#01`、`2026-04-07#01`、`2026-04-08#01`、`2026-04-10#01`；对应能力并入 **`2026-04-04#03`–`#05`** 与 **`2026-04-05#02`–`#03`**。已向目标库执行过旧名脚本的环境，一般只需补 **`2026-04-05#03`**（若尚无 `extra_file_urls`）及未覆盖的 **`2026-04-05#02`** 条件插入；请以脚本内 `IF NOT EXISTS` / 条件为准，避免重复破坏性语句。 |
| **发版文档** | **`docs/releases`** 下说明文件命名为 **`YYYYMMDD#NN_DEPLOY_AND_UPGRADE.md`**；日期为发文当日 **`YYYYMMDD`**（与 `db/migrations` 文件名中的日期可以不同）；同级已登记 [20260404#01](./20260404%2301_DEPLOY_AND_UPGRADE.md)、[20260404#02](./20260404%2302_DEPLOY_AND_UPGRADE.md) 与本文件。Markdown 链到带 `#` 的文件名时用 **`%23`**。 |

---

## 2. 数据库升级（已有库）

路径中含 **`#`** 时，请在 shell 中对文件路径 **加引号**。

在 **已完成 [20260404#02 §2](./20260404%2302_DEPLOY_AND_UPGRADE.md)**（`2026-04-04#03`–`#05`，以及按需 `2026-04-05#01`）的前提下，继续执行：

```bash
mysql … < "db/migrations/2026-04-05#02_seed_subjects_math_english_qwen.sql"
mysql … < "db/migrations/2026-04-05#03_exam_paper_extra_file_urls.sql"
```

**尚未执行 #02 中 `2026-04-04#03`–`#05`** 的，须先按 #02 文档跑完，再执行上两行。

**全新建库**：导入当前 **`db/schema/mysql_schema_v0.1_260403.sql`** 与 **`db/seed/dev_seed.sql`**；若种子已覆盖科目与模型，**`2026-04-05#02`** 可按团队约定跳过（以脚本幂等性为准）。

---

## 3. 应用与配置

- 重新构建并部署 **backend**；部署 **frontend-student** 静态资源（多图与科目相关 UI）。  
- **`UPLOAD_DIR`**、**`ANALYSIS_ADAPTER=http`**、**`CORS_ALLOWED_ORIGINS`** 等与 #02 一致。  
- 多图识图耗时更高，**`AI_REQUEST_TIMEOUT_SECONDS`** 建议保持 **180**（勿低于 **120**）。

---

## 4. 验收建议

1. **`GET .../readyz`** → `200`。  
2. 学生端在同一科目下选择 **多张图片** 提交 → 返回成功；数据库 **`exam_paper.extra_file_urls`** 在多于一张时为非空 JSON。  
3. **AI 调用日志** 中 **`paper_analyze`** 可核对多图请求（正文仍按既有脱敏规则）。  
4. 单文件 **`file`** 上传路径仍可用，行为与 #02 一致。

---

## 5. 文档与基线

- **接口约定**：[`api_v0.1_260403.md`](../core/api_v0.1_260403.md) 中 **`POST /api/v1/student/papers`** 多图说明。  
- **本文件** 仅描述相对 **[20260404#02](./20260404%2302_DEPLOY_AND_UPGRADE.md)** 的增量；前置 checklist 见 [**20260404#01**](./20260404%2301_DEPLOY_AND_UPGRADE.md)、[**20260404#02**](./20260404%2302_DEPLOY_AND_UPGRADE.md)。
