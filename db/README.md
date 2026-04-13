# 数据库脚本（PostgreSQL）

DDL / 迁移 / 开发种子数据统一放在本目录。本项目以 **PostgreSQL** 为唯一目标库（**pgvector** 扩展用于后续向量检索）；不再维护 MySQL 基线。

## 目录

| 路径 | 用途 |
|------|------|
| **`schema/`** | **基线建库**：新开环境整份执行，当前为 `postgresql_schema_v0.1_260403.sql`（含 `CREATE EXTENSION vector` 与全部表定义）。 |
| **`migrations/`** | **增量迁移**：已有库在基线之后追加的变更。命名：`yyyy-mm-dd#NN_简述.sql`：`yyyy-mm-dd` 为撰写/合入当日的真实日历（请在仓库内用 `date +%F` 核对），`#NN` 为同一自然日内序号；按文件名字典序执行。 |
| **`seed/`** | **开发/测试种子**（默认管理员、科目样例等）；勿提交生产密钥。教材目录：`seed/textbook_yuedu_physics_required_2019.sql`；章节幻灯片示例：`seed/slide_deck_sample_yuedu_physics_ch2_sec1.sql`（依赖 `textbook_slide_deck` 表）。 |

## 推荐执行顺序（全新库）

1. 创建数据库与用户后执行基线：  
   `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/schema/postgresql_schema_v0.1_260403.sql`
2. 开发/联调种子：  
   `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seed/dev_seed.sql`
3. 可选：教材与示例幻灯片（顺序见各文件头注释）。

路径中含 `#` 时请在 shell 中加引号。

## 已有测试库升级到模块前缀表名

若当前库仍是旧表名（`stage`、`student`、`exam_paper`、`admin_session` 等），在备份后执行：

`psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "db/migrations/2026-04-13#01_rename_tables_module_prefix.postgresql.sql"`

详见 **`docs/releases/20260413#01_DEPLOY_AND_UPGRADE.md`**。

## 与文档的对应关系

- 部署与 Compose 见 **`docs/core/deployment_guide_v0.1_260403.md`**（若与本文冲突，以本文与当期 release 为准）。
- 表字段语义以 **`db/schema/postgresql_schema_v0.1_260403.sql`** 为准。

## 历史说明

`migrations/` 中部分较早文件为 MySQL 语法，仅作历史记录；**新环境请以 PostgreSQL 基线 + 上述 PG 迁移为准**，勿对 MySQL 再执行。
