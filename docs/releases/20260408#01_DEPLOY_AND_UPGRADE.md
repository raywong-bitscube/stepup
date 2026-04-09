# 2026-04-08#01 — 教材目录表（textbook / chapter / section）、章/节 status 与 admin 目录 API/UI

**范围**: 新增三张业务表；章/节增加 `status`、去掉序号唯一约束以便管理端调整序号；粤教版（2019）物理必修可选种子；**管理端**在科目编辑中提供 **目录** 独立视图（教材 → 章 → 节，仅 PATCH）。

---

## 数据库

1. **已有库（基线早于本变更）**  
   按字典序执行增量迁移（路径含 `#` 请加引号）：

   ```bash
   mysql -u… -p… stepup < "db/migrations/2026-04-08#01_textbook_chapter_section.sql"
   mysql -u… -p… stepup < "db/migrations/2026-04-08#02_chapter_section_status_drop_number_unique.sql"
   ```

2. **全新库**  
   若已更新并执行当前 `db/schema/mysql_schema_v0.1_260403.sql`，表结构已含 `#01`+`#02` 合并结果，仅需按需跑种子。

3. **种子（开发/联调）**  
   建议在 `db/seed/dev_seed.sql` 之后执行（需已存在 `subject.name = 物理`，以便填充 `textbook.subject_id`）：

   ```bash
   mysql -u… -p… stepup < db/seed/textbook_yuedu_physics_required_2019.sql
   ```

   教材行仍用 `ON DUPLICATE KEY UPDATE`；章/节以 `NOT EXISTS` 保证可重复执行。不引入生产密钥。

---

## 应用

- **后端**: 6 个 admin 教材目录接口（见 **`docs/core/api_v0.1_260403.md`** §3.7 教材目录）。任意已登录 admin 可调；须 `DB_DSN` 且库结构含 `chapter.status` / `section.status`。
- **frontend-admin**: 科目列表 → **编辑** → 若该 `subject_id` 下已有教材则显示 **目录** → 全屏独立视图（侧栏隐藏）；教材表 **编辑** / **章节** → 章表 **编辑** / **小节** → 节表 **编辑**。

---

## 表名与约定

- 物理表名：`textbook`、`chapter`、`section`（与仓库其余表一致，采用单数实体名）。  
- 业务字段：`name` / `version` / `subject` / `category`（书）；`number` + `title` + `full_title`；章/节 **`status`**（0 停用 / 1 启用）。管理端对目录的停用**只改** `status`，不改 `is_deleted`。  
- 系统字段：`id`、审计四元组、软删除，与 **`db/schema`** 中其它核心表一致。

---

## 验收

- `SHOW TABLES` 含 `textbook`、`chapter`、`section`；`DESCRIBE chapter` 含 `status`，且无 `uk_chapter_textbook_number`（仅有 `idx_chapter_textbook_number`）。  
- 种子执行后：`SELECT COUNT(*) FROM textbook WHERE version='粤教版 2019';` 为 `2`；`section` 行数大于 `40`（两册全部小节）。  
- 管理端：物理科目编辑弹窗出现 **目录**；进入后可 PATCH 教材字段并下钻章、节。

---

## 文档

- API：**`docs/core/api_v0.1_260403.md`**（§3.7 教材目录）。  
- 实体：**`docs/core/entity_analyze_v0.1_260403.md`**（§12–§14）。  
- 种子：**`db/README.md`**。

返回 [**releases 索引**](./README.md) · [**docs 总索引**](../README.md)。
