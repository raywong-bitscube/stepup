-- exam_source module: source exam papers, per-question extraction, and file library.
-- Storage is abstracted via storage_provider + bucket/object key, so the same schema works for:
-- - local filesystem (provider='local', object_key can be relative path)
-- - AWS S3          (provider='s3')
-- - Alibaba OSS     (provider='oss')

CREATE EXTENSION IF NOT EXISTS vector;

-- 1) source paper header
CREATE TABLE IF NOT EXISTS exam_source_paper (
  id BIGSERIAL PRIMARY KEY,
  paper_code VARCHAR(64) NULL,
  title VARCHAR(255) NOT NULL,
  source_region VARCHAR(64) NULL,
  source_school VARCHAR(128) NULL,
  exam_year INT NULL,
  term VARCHAR(32) NULL,
  grade_label VARCHAR(32) NULL,
  k12_grade_id BIGINT NULL,
  k12_subject_id BIGINT NOT NULL,
  paper_type VARCHAR(32) NOT NULL DEFAULT 'mock_exam',
  total_score NUMERIC(6,2) NULL,
  duration_minutes INT NULL,
  page_count INT NOT NULL DEFAULT 0,
  question_count INT NOT NULL DEFAULT 0,
  remarks VARCHAR(500) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_paper_code UNIQUE (paper_code),
  CONSTRAINT ck_exam_source_paper_exam_year CHECK (exam_year IS NULL OR (exam_year >= 2000 AND exam_year <= 2100)),
  CONSTRAINT ck_exam_source_paper_duration CHECK (duration_minutes IS NULL OR duration_minutes > 0),
  CONSTRAINT fk_exam_source_paper_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject (id),
  CONSTRAINT fk_exam_source_paper_k12_grade FOREIGN KEY (k12_grade_id) REFERENCES k12_grade (id)
);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_subject ON exam_source_paper (k12_subject_id);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_grade ON exam_source_paper (k12_grade_id);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_exam_year ON exam_source_paper (exam_year);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_status ON exam_source_paper (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_is_deleted ON exam_source_paper (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_paper_updated_at ON exam_source_paper;
CREATE TRIGGER trg_exam_source_paper_updated_at BEFORE UPDATE ON exam_source_paper FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 2) file library (works for local / s3 / oss)
CREATE TABLE IF NOT EXISTS exam_source_file (
  id BIGSERIAL PRIMARY KEY,
  storage_provider VARCHAR(16) NOT NULL,
  bucket_name VARCHAR(255) NULL,
  object_key VARCHAR(1024) NOT NULL,
  public_url VARCHAR(2048) NULL,
  original_filename VARCHAR(255) NULL,
  content_type VARCHAR(128) NULL,
  file_ext VARCHAR(16) NULL,
  size_bytes BIGINT NULL,
  sha256 CHAR(64) NULL,
  etag VARCHAR(128) NULL,
  image_width INT NULL,
  image_height INT NULL,
  metadata JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT ck_exam_source_file_provider CHECK (storage_provider IN ('local', 's3', 'oss')),
  CONSTRAINT ck_exam_source_file_size CHECK (size_bytes IS NULL OR size_bytes >= 0),
  CONSTRAINT ck_exam_source_file_width CHECK (image_width IS NULL OR image_width > 0),
  CONSTRAINT ck_exam_source_file_height CHECK (image_height IS NULL OR image_height > 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_exam_source_file_storage_path
  ON exam_source_file (storage_provider, COALESCE(bucket_name, ''), object_key);
CREATE INDEX IF NOT EXISTS idx_exam_source_file_sha256 ON exam_source_file (sha256);
CREATE INDEX IF NOT EXISTS idx_exam_source_file_status ON exam_source_file (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_file_is_deleted ON exam_source_file (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_file_updated_at ON exam_source_file;
CREATE TRIGGER trg_exam_source_file_updated_at BEFORE UPDATE ON exam_source_file FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 3) paper page to file mapping
CREATE TABLE IF NOT EXISTS exam_source_paper_page (
  id BIGSERIAL PRIMARY KEY,
  paper_id BIGINT NOT NULL,
  page_no INT NOT NULL,
  file_id BIGINT NOT NULL,
  ocr_text TEXT NULL,
  ocr_engine VARCHAR(32) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_paper_page_no UNIQUE (paper_id, page_no),
  CONSTRAINT ck_exam_source_paper_page_no CHECK (page_no > 0),
  CONSTRAINT fk_exam_source_paper_page_paper FOREIGN KEY (paper_id) REFERENCES exam_source_paper (id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_source_paper_page_file FOREIGN KEY (file_id) REFERENCES exam_source_file (id)
);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_page_file ON exam_source_paper_page (file_id);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_page_status ON exam_source_paper_page (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_paper_page_is_deleted ON exam_source_paper_page (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_paper_page_updated_at ON exam_source_paper_page;
CREATE TRIGGER trg_exam_source_paper_page_updated_at BEFORE UPDATE ON exam_source_paper_page FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 4) per-question structured record
CREATE TABLE IF NOT EXISTS exam_source_question (
  id BIGSERIAL PRIMARY KEY,
  paper_id BIGINT NOT NULL,
  question_no VARCHAR(32) NOT NULL,
  question_order INT NOT NULL DEFAULT 0,
  section_no VARCHAR(32) NULL,
  question_type VARCHAR(32) NOT NULL DEFAULT 'unknown',
  score NUMERIC(6,2) NULL,
  stem_text TEXT NULL,
  answer_text TEXT NULL,
  explanation_text TEXT NULL,
  page_from INT NULL,
  page_to INT NULL,
  metadata JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_question_no UNIQUE (paper_id, question_no),
  CONSTRAINT ck_exam_source_question_order CHECK (question_order >= 0),
  CONSTRAINT ck_exam_source_question_page_from CHECK (page_from IS NULL OR page_from > 0),
  CONSTRAINT ck_exam_source_question_page_to CHECK (page_to IS NULL OR page_to > 0),
  CONSTRAINT ck_exam_source_question_page_range CHECK (
    page_from IS NULL OR page_to IS NULL OR page_to >= page_from
  ),
  CONSTRAINT fk_exam_source_question_paper FOREIGN KEY (paper_id) REFERENCES exam_source_paper (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_order ON exam_source_question (paper_id, question_order);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_type ON exam_source_question (question_type);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_status ON exam_source_question (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_is_deleted ON exam_source_question (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_question_updated_at ON exam_source_question;
CREATE TRIGGER trg_exam_source_question_updated_at BEFORE UPDATE ON exam_source_question FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 5) question to file mapping (one question can have multiple images)
CREATE TABLE IF NOT EXISTS exam_source_question_file (
  id BIGSERIAL PRIMARY KEY,
  question_id BIGINT NOT NULL,
  file_id BIGINT NOT NULL,
  role VARCHAR(32) NOT NULL DEFAULT 'stem',
  sort_no INT NOT NULL DEFAULT 1,
  page_no INT NULL,
  bbox_norm JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_question_file UNIQUE (question_id, file_id, role),
  CONSTRAINT ck_exam_source_question_file_sort_no CHECK (sort_no > 0),
  CONSTRAINT ck_exam_source_question_file_page_no CHECK (page_no IS NULL OR page_no > 0),
  CONSTRAINT fk_exam_source_question_file_question FOREIGN KEY (question_id) REFERENCES exam_source_question (id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_source_question_file_file FOREIGN KEY (file_id) REFERENCES exam_source_file (id)
);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_file_question_order ON exam_source_question_file (question_id, sort_no);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_file_file ON exam_source_question_file (file_id);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_file_status ON exam_source_question_file (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_file_is_deleted ON exam_source_question_file (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_source_question_file_updated_at ON exam_source_question_file;
CREATE TRIGGER trg_exam_source_question_file_updated_at BEFORE UPDATE ON exam_source_question_file FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 6) question-level embeddings (pgvector)
CREATE TABLE IF NOT EXISTS exam_source_question_embedding (
  id BIGSERIAL PRIMARY KEY,
  question_id BIGINT NOT NULL,
  embedding_model VARCHAR(64) NOT NULL,
  embedding_dim INT NOT NULL DEFAULT 1536,
  content_text TEXT NOT NULL,
  embedding VECTOR(1536) NOT NULL,
  metadata JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_question_embedding UNIQUE (question_id, embedding_model),
  CONSTRAINT ck_exam_source_question_embedding_dim CHECK (embedding_dim = 1536),
  CONSTRAINT fk_exam_source_question_embedding_question FOREIGN KEY (question_id) REFERENCES exam_source_question (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_embedding_status ON exam_source_question_embedding (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_embedding_is_deleted ON exam_source_question_embedding (is_deleted);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_embedding_model ON exam_source_question_embedding (embedding_model);
CREATE INDEX IF NOT EXISTS idx_exam_source_question_embedding_vec
  ON exam_source_question_embedding USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
DROP TRIGGER IF EXISTS trg_exam_source_question_embedding_updated_at ON exam_source_question_embedding;
CREATE TRIGGER trg_exam_source_question_embedding_updated_at BEFORE UPDATE ON exam_source_question_embedding FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 7) page/block-level embeddings for OCR chunks (pgvector)
CREATE TABLE IF NOT EXISTS exam_source_page_chunk_embedding (
  id BIGSERIAL PRIMARY KEY,
  paper_page_id BIGINT NOT NULL,
  chunk_no INT NOT NULL DEFAULT 1,
  chunk_text TEXT NOT NULL,
  embedding_model VARCHAR(64) NOT NULL,
  embedding_dim INT NOT NULL DEFAULT 1536,
  embedding VECTOR(1536) NOT NULL,
  metadata JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_exam_source_page_chunk UNIQUE (paper_page_id, chunk_no, embedding_model),
  CONSTRAINT ck_exam_source_page_chunk_no CHECK (chunk_no > 0),
  CONSTRAINT ck_exam_source_page_chunk_dim CHECK (embedding_dim = 1536),
  CONSTRAINT fk_exam_source_page_chunk_page FOREIGN KEY (paper_page_id) REFERENCES exam_source_paper_page (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_exam_source_page_chunk_status ON exam_source_page_chunk_embedding (status);
CREATE INDEX IF NOT EXISTS idx_exam_source_page_chunk_is_deleted ON exam_source_page_chunk_embedding (is_deleted);
CREATE INDEX IF NOT EXISTS idx_exam_source_page_chunk_model ON exam_source_page_chunk_embedding (embedding_model);
CREATE INDEX IF NOT EXISTS idx_exam_source_page_chunk_vec
  ON exam_source_page_chunk_embedding USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
DROP TRIGGER IF EXISTS trg_exam_source_page_chunk_updated_at ON exam_source_page_chunk_embedding;
CREATE TRIGGER trg_exam_source_page_chunk_updated_at BEFORE UPDATE ON exam_source_page_chunk_embedding FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();
