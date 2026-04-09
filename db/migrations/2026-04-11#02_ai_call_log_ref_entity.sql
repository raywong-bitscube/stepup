-- 将 paper_id 泛化为 ref_table + ref_id，便于关联 slide_deck、section 等业务
SET NAMES utf8mb4;

ALTER TABLE ai_call_log
  ADD COLUMN ref_table VARCHAR(64) NULL COMMENT '关联业务表，如 exam_paper、section、student' AFTER student_id,
  ADD COLUMN ref_id BIGINT UNSIGNED NULL COMMENT '关联业务主键' AFTER ref_table;

UPDATE ai_call_log SET ref_table = 'exam_paper', ref_id = paper_id WHERE paper_id IS NOT NULL;

ALTER TABLE ai_call_log DROP FOREIGN KEY fk_ai_call_log_paper;

ALTER TABLE ai_call_log DROP INDEX idx_ai_call_log_paper;

ALTER TABLE ai_call_log DROP COLUMN paper_id;

CREATE INDEX idx_ai_call_log_ref ON ai_call_log (ref_table, ref_id);
