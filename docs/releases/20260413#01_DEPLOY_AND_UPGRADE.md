# 2026-04-13 — PostgreSQL 表模块前缀、`sys_session` 合并、移除 MySQL 基线

## 背景

- 核心表增加 **功能模块前缀**（如 `k12_grade`、`sys_user`、`student_exam_paper`），便于扩展与避免同名歧义。
- **管理员会话**与**学生会话**合并为 **`sys_session`**，用 **`user_type`**（如 `admin`、`student`）区分；后续家长、教师等可共用同表结构。
- **`verification_code`** → **`sys_verification_code`**；**`ai_model`** → **`ai_provider_model`**。
- **`ai_call_log`** 列：**`ai_model_id`** → **`ai_provider_model_id`**，**`student_id`** → **`sys_user_id`**；业务引用 **`ref_table = 'student_exam_paper'`**（原 `exam_paper`）。
- 仓库 **删除 `db/schema/mysql_schema_v0.1_260403.sql`**；**仅支持 PostgreSQL**（pgvector 用于后续 embedding）。

## 应用与前端

- 后端已改为新表名；管理端 AI 调用日志筛选参数优先 **`ai_provider_model_id`**（兼容旧查询参数 **`ai_model_id`** 一轮）。
- 日志 JSON 中展示 **`sys_user_id`**、**`ai_provider_model_id`**、**`ref_table`** / **`ref_id`**。

## 你本地/测试环境如何更新

### 方案 A：空库或数据可丢弃（推荐测试环境）

1. 备份（如有需要）：`pg_dump … > backup.sql`
2. 删库重建或 `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`（慎用）
3. 导入基线：  
   `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/schema/postgresql_schema_v0.1_260403.sql`
4. 种子：  
   `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seed/dev_seed.sql`
5. 按需：`db/seed/textbook_yuedu_physics_required_2019.sql`、`db/seed/slide_deck_sample_yuedu_physics_ch2_sec1.sql`

### 方案 B：保留现有数据（从旧表名升级）

1. **先备份**：`pg_dump … > backup.sql`
2. 在停写或维护窗口执行迁移（路径含 `#` 请加引号）：  
   `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "db/migrations/2026-04-13#01_rename_tables_module_prefix.postgresql.sql"`
3. 若迁移报错，用备份回滚后再对照错误信息处理（例如部分表已手工改过名会导致「已存在」类问题，需个案调整）。

迁移脚本会：

- 将 `stage` → `k12_grade`、`subject` → `k12_subject`、`student` → `sys_user`（`stage_id` → `k12_grade_id`）等；
- 将 `exam_paper` → `student_exam_paper`（`student_id`/`subject_id` → `sys_user_id`/`k12_subject_id`）及分析与计划表；
- 将 `slide_deck` → `textbook_slide_deck`；`textbook.subject_id` → `k12_subject_id`；
- 更新 `ai_call_log` 外键列名；`ref_table` 中 `exam_paper` → `student_exam_paper`；
- 从 `admin_session` / `student_session` 导入 **`sys_session`** 后删除旧表。

### Docker Compose

- 仍使用 **`pgvector/pgvector`** 镜像；**无需 MySQL 服务**。
- 重建后端镜像并重启栈；确认 **`DB_DSN`** 指向 PostgreSQL。

## MySQL 与向量能力（结论）

- **MySQL 8.x** 本身**没有**与 **pgvector** 同等成熟、通用的向量索引与生态。
- **MySQL 9.x** 起提供原生 **`VECTOR`** 类型与距离函数，但索引与工具链仍较新；**RDS/托管版本与 ORM 支持**未必与 PostgreSQL + pgvector 一致。
- 若你需要 **HNSW/IVFFlat、混合检索、与现有 SQL 同一套迁移**，**PostgreSQL + pgvector** 仍是更稳妥的默认选择；不必因熟悉 MySQL 而回退，除非运维强制要求且能接受 MySQL 9+ 向量特性的限制。

## 后续

- 试卷题库与 **embedding** 表结构将在上述命名空间稳定后单独设计（`exam_source_*` 等）。
