-- exam_source: external import review records (step-1 analyze snapshots)

CREATE TABLE IF NOT EXISTS exam_source_import_record (
  id VARCHAR(64) PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  source_dir VARCHAR(1024) NULL,
  image_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
  analyze_snapshot JSONB NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  paper_id BIGINT NULL,
  remarks VARCHAR(500) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT ck_exam_source_import_status CHECK (status IN ('pending', 'created', 'rejected')),
  CONSTRAINT ck_exam_source_import_deleted CHECK (is_deleted IN (0, 1)),
  CONSTRAINT fk_exam_source_import_paper FOREIGN KEY (paper_id) REFERENCES exam_source_paper (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_exam_source_import_status ON exam_source_import_record (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_import_created_at ON exam_source_import_record (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exam_source_import_is_deleted ON exam_source_import_record (is_deleted);
CREATE INDEX IF NOT EXISTS idx_exam_source_import_paper_id ON exam_source_import_record (paper_id);

DROP TRIGGER IF EXISTS trg_exam_source_import_record_updated_at ON exam_source_import_record;
CREATE TRIGGER trg_exam_source_import_record_updated_at
BEFORE UPDATE ON exam_source_import_record
FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();
