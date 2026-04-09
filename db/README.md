# 数据库脚本（MySQL）

所有与 **DDL / 迁移 / 开发种子数据** 相关的 SQL 统一放在本目录，避免分散在 `docs/`、`scripts/` 等多处。

## 目录

| 路径 | 用途 |
|------|------|
| **`schema/`** | **基线建库**：新开环境时整份执行，当前为 `mysql_schema_v0.1_260403.sql`（含全部表定义）。 |
| **`migrations/`** | **增量迁移**：已有库在基线之后追加的变更。**命名**：`yyyy-mm-dd#NN_简述.sql`：**`yyyy-mm-dd` 为撰写/合入当日的真实日历**，**`#NN` 为同一自然日内的序号**（从 `01` 起），勿用虚构日期「凑」字典序；**按文件名字典序**执行；旧库已跑过的脚本不必重跑，只执行新增文件。 |
| **`seed/`** | **开发/测试种子数据**（如默认管理员、科目样例）；**勿**把生产密钥提交进仓库。教材目录示例：`seed/textbook_yuedu_physics_required_2019.sql`（粤教版 2019 物理必修；需先建表并建议执行 `2026-04-08#02_…`，使 `chapter`/`section` 含 `status` 且无序号唯一约束，与当前基线一致）。章节幻灯片示例（需已建 `slide_deck` 表）：`seed/slide_deck_sample_yuedu_physics_ch2_sec1.sql`。 |

## 推荐执行顺序

1. 创建空库与用户后：  
   `mysql ... < db/schema/mysql_schema_v0.1_260403.sql`
2. （仅当库在引入某迁移**之前**就已建好）按需执行：  
   `mysql ... < "db/migrations/yyyy-mm-dd#NN_描述.sql"`（路径含 `#`，请在 shell 中加引号）
3. 开发/联调可选：  
   `mysql ... < db/seed/dev_seed.sql`

Docker Compose 与升级步骤见 **`docs/core/deployment_guide_v0.1_260403.md`**、**`docs/releases/20260404#01_DEPLOY_AND_UPGRADE.md`**（命令中的路径已指向 `db/`）。

## 与文档的对应关系

- 发版请先读 [**docs/README.md**](../docs/README.md) 与 [**docs/releases/**](../docs/releases/) 下当期增量说明。
- 需求、API、部署等基线说明在 **`docs/core/`**；表字段语义以 **`db/schema`** 为准。
- 若新增迁移，请同时：更新 **`db/migrations`**、并在 **`db/schema`** 的基线文件中合并相同 DDL（方便全新环境一条脚本到位），或在 README 中注明「仅增量、未回填 schema」的例外（不推荐长期并存）。
