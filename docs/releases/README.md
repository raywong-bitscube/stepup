# 发版与增量说明

本目录存放 **按批次 / 日期** 的上线说明：本轮改了什么、如何升级数据库与应用、如何验收。**不改变** [`docs/core/`](../core/) 里基线文档的日期。

命名与 `db/migrations` 一致：**`YYYYMMDD#NN`**（同一自然日多批次递增），链到带 `#` 的文件时 Markdown 建议使用 `%23`（例：`20260404%2301_DEPLOY_AND_UPGRADE.md`）。

| 文件 | 说明 |
|------|------|
| [20260404#01_DEPLOY_AND_UPGRADE.md](./20260404%2301_DEPLOY_AND_UPGRADE.md) | AI 调用日志、`db/` 目录、文档结构、前端分端口等增量与升级 checklist |
| [20260404#02_DEPLOY_AND_UPGRADE.md](./20260404%2302_DEPLOY_AND_UPGRADE.md) | 识图上传与 `UPLOAD_DIR`、Prompt 模板与 `paper_analyze_chat_user`、AI 日志 `request_body`/`response_body` 与 admin 表格优化、Kimi 迁移可选 |
| [20260404#03_DEPLOY_AND_UPGRADE.md](./20260404%2303_DEPLOY_AND_UPGRADE.md) | 多图试卷与 `extra_file_urls`、数学/英语与 Qwen 种子、迁移文件归并、`releases` 文档 `YYYYMMDD#NN` 命名 |

新增发版时：在同一自然日新建 **`YYYYMMDD#(N+1)_DEPLOY_AND_UPGRADE.md`**（或与 `db/migrations` 同日序号对齐），并在本表登记。

返回 [**文档总索引**](../README.md) · [**core 基线文档**](../core/README.md)。
