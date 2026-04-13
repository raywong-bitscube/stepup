-- PostgreSQL：与 2026-04-12#01_rename_textbook_chapter_section.sql 等价（供单独维护 PG 的环境使用）
-- 列名 chapter_id、slide_deck.section_id 不变；仅表重命名及 ai_call_log 业务表名字符串。
-- 若已由 db/schema/postgresql_schema_v0.1_260403.sql 直接建库（表名已是 textbook_*），请勿执行。

UPDATE ai_call_log SET ref_table = 'textbook_section' WHERE ref_table = 'section';

ALTER TABLE chapter RENAME TO textbook_chapter;
ALTER TABLE section RENAME TO textbook_section;

ALTER TRIGGER trg_chapter_updated_at ON textbook_chapter RENAME TO trg_textbook_chapter_updated_at;
ALTER TRIGGER trg_section_updated_at ON textbook_section RENAME TO trg_textbook_section_updated_at;

ALTER INDEX idx_chapter_textbook RENAME TO idx_textbook_chapter_textbook;
ALTER INDEX idx_chapter_textbook_number RENAME TO idx_textbook_chapter_textbook_number;
ALTER INDEX idx_chapter_is_deleted RENAME TO idx_textbook_chapter_is_deleted;

ALTER INDEX idx_section_chapter RENAME TO idx_textbook_section_chapter;
ALTER INDEX idx_section_chapter_number RENAME TO idx_textbook_section_chapter_number;
ALTER INDEX idx_section_is_deleted RENAME TO idx_textbook_section_is_deleted;

ALTER TABLE textbook_chapter RENAME CONSTRAINT fk_chapter_textbook TO fk_textbook_chapter_textbook;
ALTER TABLE textbook_section RENAME CONSTRAINT fk_section_chapter TO fk_textbook_section_chapter;
