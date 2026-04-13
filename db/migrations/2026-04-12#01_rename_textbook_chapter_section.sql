-- MySQL：教材目录表重命名为 textbook_chapter / textbook_section；
-- slide_deck.section_id、textbook_section.chapter_id 等列名不变。
-- 执行前请确认已应用 2026-04-08#01、#02 及 slide_deck 相关迁移。
-- 若已由 db/schema/mysql_schema_v0.1_260403.sql 直接建库（表名已是 textbook_*），请勿执行。
SET NAMES utf8mb4;

UPDATE ai_call_log SET ref_table = 'textbook_section' WHERE ref_table = 'section';

RENAME TABLE chapter TO textbook_chapter, section TO textbook_section;
