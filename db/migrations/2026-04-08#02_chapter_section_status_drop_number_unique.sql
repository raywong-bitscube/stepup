-- 章/节：增加 status（停用）；去掉 (textbook_id,number)/(chapter_id,number) 唯一约束，改为普通索引（序号可重复、可调整）
SET NAMES utf8mb4;

ALTER TABLE chapter
  ADD COLUMN status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive' AFTER full_title,
  DROP INDEX uk_chapter_textbook_number,
  ADD KEY idx_chapter_textbook_number (textbook_id, number);

ALTER TABLE section
  ADD COLUMN status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive' AFTER full_title,
  DROP INDEX uk_section_chapter_number,
  ADD KEY idx_section_chapter_number (chapter_id, number);
