# 发版与增量说明

本目录存放 **按批次 / 日期** 的上线说明：本轮改了什么、如何升级数据库与应用、如何验收。**不改变** [`docs/core/`](../core/) 里基线文档的日期。

发版说明文件名：**`YYYYMMDD#NN_DEPLOY_AND_UPGRADE.md`**，其中 **`YYYYMMDD` 为发文当日**（可与 `db/migrations` 前缀日期不同）；同一自然日多批次则 **`#NN` 递增**。链到带 `#` 的文件时 Markdown 建议使用 **`%23`**（例：`20260404%2301_DEPLOY_AND_UPGRADE.md`、`20260405%2301_DEPLOY_AND_UPGRADE.md`、`20260405%2302_DEPLOY_AND_UPGRADE.md`）。

| 文件 | 说明 |
|------|------|
| [20260404#01_DEPLOY_AND_UPGRADE.md](./20260404%2301_DEPLOY_AND_UPGRADE.md) | AI 调用日志、`db/` 目录、文档结构、前端分端口等增量与升级 checklist |
| [20260404#02_DEPLOY_AND_UPGRADE.md](./20260404%2302_DEPLOY_AND_UPGRADE.md) | 识图上传与 `UPLOAD_DIR`、Prompt 模板与 `paper_analyze_chat_user`、AI 日志 `request_body`/`response_body` 与 admin 表格优化、Kimi 迁移可选 |
| [20260405#01_DEPLOY_AND_UPGRADE.md](./20260405%2301_DEPLOY_AND_UPGRADE.md) | 多图试卷与 `extra_file_urls`、数学/英语与 Qwen 种子、迁移文件归并、`releases` 文档按发文日 `YYYYMMDD#NN` 命名 |
| [20260405#02_DEPLOY_AND_UPGRADE.md](./20260405%2302_DEPLOY_AND_UPGRADE.md) | CORS 默认 `*`、LAN 双 Origin、Go+Nginx 示例与 OPTIONS 预检、学生端 API 基址与失败重挂修复、管理端错误提示与部署文档 |
| [20260408#01_DEPLOY_AND_UPGRADE.md](./20260408%2301_DEPLOY_AND_UPGRADE.md) | 教材目录表与迁移 `#02`（章/节 `status`、去序号唯一）、粤教版种子、管理端目录 API 与 `frontend-admin` 独立视图 |
| [20260413#01_DEPLOY_AND_UPGRADE.md](./20260413%2301_DEPLOY_AND_UPGRADE.md) | PostgreSQL 表模块前缀、`sys_session` 合并会话、移除 MySQL 基线、`ai_call_log` 列重命名、升级与 MySQL 向量能力说明 |

新增发版时：在**发文当日**新建 **`YYYYMMDD#(N+1)_DEPLOY_AND_UPGRADE.md`**，并在本表登记（不必与 migration 文件日期对齐）。

返回 [**文档总索引**](../README.md) · [**core 基线文档**](../core/README.md)。
