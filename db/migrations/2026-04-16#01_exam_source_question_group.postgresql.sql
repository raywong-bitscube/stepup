-- exam_source: 大题分组（卷面「一、单项选择题…」类说明 + 系统题型）

CREATE TABLE IF NOT EXISTS exam_source_question_group (
  id BIGSERIAL PRIMARY KEY,
  paper_id BIGINT NOT NULL,
  group_order INT NOT NULL DEFAULT 0,
  system_kind VARCHAR(64) NOT NULL DEFAULT 'unknown',
  title_label VARCHAR(255) NULL,
  description_text TEXT NULL,
  page_no INT NULL,
  metadata JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT ck_exam_source_qg_order CHECK (group_order >= 0),
  CONSTRAINT ck_exam_source_qg_page CHECK (page_no IS NULL OR page_no > 0),
  CONSTRAINT fk_exam_source_qg_paper FOREIGN KEY (paper_id) REFERENCES exam_source_paper (id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_exam_source_qg_paper_order
  ON exam_source_question_group (paper_id, group_order)
  WHERE is_deleted = 0;
CREATE INDEX IF NOT EXISTS idx_exam_source_qg_paper ON exam_source_question_group (paper_id);
CREATE INDEX IF NOT EXISTS idx_exam_source_qg_status ON exam_source_question_group (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_qg_is_deleted ON exam_source_question_group (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_question_group_updated_at ON exam_source_question_group;
CREATE TRIGGER trg_exam_source_question_group_updated_at BEFORE UPDATE ON exam_source_question_group FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

ALTER TABLE exam_source_question
  ADD COLUMN IF NOT EXISTS group_id BIGINT NULL;
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_exam_source_question_group'
  ) THEN
    ALTER TABLE exam_source_question
      ADD CONSTRAINT fk_exam_source_question_group
      FOREIGN KEY (group_id) REFERENCES exam_source_question_group (id) ON DELETE SET NULL;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_exam_source_question_group ON exam_source_question (group_id);
