-- StepUp v0.1 PostgreSQL baseline (MySQL parity + pgvector for future embeddings)
-- Usage: psql "postgres://USER:PASS@HOST:5432/stepup?sslmode=disable" -f db/schema/postgresql_schema_v0.1_260403.sql
SET client_encoding = 'UTF8';

CREATE EXTENSION IF NOT EXISTS vector;

-- Emulate MySQL ON UPDATE CURRENT_TIMESTAMP for updated_at
CREATE OR REPLACE FUNCTION stepup_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1) stage
CREATE TABLE IF NOT EXISTS stage (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_stage_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS idx_stage_status ON stage (status);
CREATE INDEX IF NOT EXISTS idx_stage_is_deleted ON stage (is_deleted);
DROP TRIGGER IF EXISTS trg_stage_updated_at ON stage;
CREATE TRIGGER trg_stage_updated_at BEFORE UPDATE ON stage FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 2) subject
CREATE TABLE IF NOT EXISTS subject (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_subject_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS idx_subject_status ON subject (status);
CREATE INDEX IF NOT EXISTS idx_subject_is_deleted ON subject (is_deleted);
DROP TRIGGER IF EXISTS trg_subject_updated_at ON subject;
CREATE TRIGGER trg_subject_updated_at BEFORE UPDATE ON subject FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 3) student
CREATE TABLE IF NOT EXISTS student (
  id BIGSERIAL PRIMARY KEY,
  phone VARCHAR(32) NULL,
  email VARCHAR(255) NULL,
  password VARCHAR(255) NOT NULL,
  name VARCHAR(128) NOT NULL,
  stage_id BIGINT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_student_phone UNIQUE (phone),
  CONSTRAINT uk_student_email UNIQUE (email),
  CONSTRAINT fk_student_stage FOREIGN KEY (stage_id) REFERENCES stage (id)
);
CREATE INDEX IF NOT EXISTS idx_student_stage_id ON student (stage_id);
CREATE INDEX IF NOT EXISTS idx_student_status ON student (status);
CREATE INDEX IF NOT EXISTS idx_student_is_deleted ON student (is_deleted);
DROP TRIGGER IF EXISTS trg_student_updated_at ON student;
CREATE TRIGGER trg_student_updated_at BEFORE UPDATE ON student FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 4) admin
CREATE TABLE IF NOT EXISTS admin (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(32) NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_admin_username UNIQUE (username)
);
CREATE INDEX IF NOT EXISTS idx_admin_status ON admin (status);
CREATE INDEX IF NOT EXISTS idx_admin_is_deleted ON admin (is_deleted);
DROP TRIGGER IF EXISTS trg_admin_updated_at ON admin;
CREATE TRIGGER trg_admin_updated_at BEFORE UPDATE ON admin FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 5) ai_model
CREATE TABLE IF NOT EXISTS ai_model (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  url VARCHAR(512) NOT NULL,
  model VARCHAR(255) NOT NULL,
  app_secret VARCHAR(255) NOT NULL,
  status SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL
);
CREATE INDEX IF NOT EXISTS idx_ai_model_status ON ai_model (status);
CREATE INDEX IF NOT EXISTS idx_ai_model_is_deleted ON ai_model (is_deleted);
DROP TRIGGER IF EXISTS trg_ai_model_updated_at ON ai_model;
CREATE TRIGGER trg_ai_model_updated_at BEFORE UPDATE ON ai_model FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 6) prompt_template
CREATE TABLE IF NOT EXISTS prompt_template (
  id BIGSERIAL PRIMARY KEY,
  "key" VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  content TEXT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_prompt_key UNIQUE ("key")
);
CREATE INDEX IF NOT EXISTS idx_prompt_status ON prompt_template (status);
CREATE INDEX IF NOT EXISTS idx_prompt_is_deleted ON prompt_template (is_deleted);
DROP TRIGGER IF EXISTS trg_prompt_template_updated_at ON prompt_template;
CREATE TRIGGER trg_prompt_template_updated_at BEFORE UPDATE ON prompt_template FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 7) exam_paper
CREATE TABLE IF NOT EXISTS exam_paper (
  id BIGSERIAL PRIMARY KEY,
  student_id BIGINT NOT NULL,
  subject_id BIGINT NOT NULL,
  file_url VARCHAR(1024) NOT NULL,
  extra_file_urls JSONB NULL,
  file_type VARCHAR(16) NOT NULL,
  score INT NULL,
  exam_date DATE NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT fk_exam_paper_student FOREIGN KEY (student_id) REFERENCES student (id),
  CONSTRAINT fk_exam_paper_subject FOREIGN KEY (subject_id) REFERENCES subject (id)
);
CREATE INDEX IF NOT EXISTS idx_exam_paper_student_id ON exam_paper (student_id);
CREATE INDEX IF NOT EXISTS idx_exam_paper_subject_id ON exam_paper (subject_id);
CREATE INDEX IF NOT EXISTS idx_exam_paper_is_deleted ON exam_paper (is_deleted);
DROP TRIGGER IF EXISTS trg_exam_paper_updated_at ON exam_paper;
CREATE TRIGGER trg_exam_paper_updated_at BEFORE UPDATE ON exam_paper FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 8) paper_analysis
CREATE TABLE IF NOT EXISTS paper_analysis (
  id BIGSERIAL PRIMARY KEY,
  paper_id BIGINT NOT NULL,
  ai_model_snapshot JSONB NOT NULL,
  raw_content TEXT NULL,
  ai_response TEXT NULL,
  status INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_paper_analysis_paper_id UNIQUE (paper_id),
  CONSTRAINT fk_paper_analysis_paper FOREIGN KEY (paper_id) REFERENCES exam_paper (id)
);
CREATE INDEX IF NOT EXISTS idx_paper_analysis_status ON paper_analysis (status);
CREATE INDEX IF NOT EXISTS idx_paper_analysis_is_deleted ON paper_analysis (is_deleted);
DROP TRIGGER IF EXISTS trg_paper_analysis_updated_at ON paper_analysis;
CREATE TRIGGER trg_paper_analysis_updated_at BEFORE UPDATE ON paper_analysis FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 9) improvement_plan
CREATE TABLE IF NOT EXISTS improvement_plan (
  id BIGSERIAL PRIMARY KEY,
  paper_id BIGINT NOT NULL,
  plan_content TEXT NOT NULL,
  weak_points JSONB NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_improvement_plan_paper_id UNIQUE (paper_id),
  CONSTRAINT fk_improvement_plan_paper FOREIGN KEY (paper_id) REFERENCES exam_paper (id)
);
CREATE INDEX IF NOT EXISTS idx_improvement_plan_is_deleted ON improvement_plan (is_deleted);
DROP TRIGGER IF EXISTS trg_improvement_plan_updated_at ON improvement_plan;
CREATE TRIGGER trg_improvement_plan_updated_at BEFORE UPDATE ON improvement_plan FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 10) admin_session
CREATE TABLE IF NOT EXISTS admin_session (
  id BIGSERIAL PRIMARY KEY,
  admin_id BIGINT NOT NULL,
  session_token VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(512) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_admin_session_token UNIQUE (session_token),
  CONSTRAINT fk_admin_session_admin FOREIGN KEY (admin_id) REFERENCES admin (id)
);
CREATE INDEX IF NOT EXISTS idx_admin_session_admin_id ON admin_session (admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_session_expires_at ON admin_session (expires_at);
CREATE INDEX IF NOT EXISTS idx_admin_session_status ON admin_session (status);
CREATE INDEX IF NOT EXISTS idx_admin_session_is_deleted ON admin_session (is_deleted);
DROP TRIGGER IF EXISTS trg_admin_session_updated_at ON admin_session;
CREATE TRIGGER trg_admin_session_updated_at BEFORE UPDATE ON admin_session FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 11) student_session
CREATE TABLE IF NOT EXISTS student_session (
  id BIGSERIAL PRIMARY KEY,
  student_id BIGINT NOT NULL,
  session_token VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(512) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_student_session_token UNIQUE (session_token),
  CONSTRAINT fk_student_session_student FOREIGN KEY (student_id) REFERENCES student (id)
);
CREATE INDEX IF NOT EXISTS idx_student_session_student_id ON student_session (student_id);
CREATE INDEX IF NOT EXISTS idx_student_session_expires_at ON student_session (expires_at);
CREATE INDEX IF NOT EXISTS idx_student_session_status ON student_session (status);
CREATE INDEX IF NOT EXISTS idx_student_session_is_deleted ON student_session (is_deleted);
DROP TRIGGER IF EXISTS trg_student_session_updated_at ON student_session;
CREATE TRIGGER trg_student_session_updated_at BEFORE UPDATE ON student_session FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 12) verification_code
CREATE TABLE IF NOT EXISTS verification_code (
  id BIGSERIAL PRIMARY KEY,
  identifier VARCHAR(255) NOT NULL,
  code VARCHAR(16) NOT NULL,
  type VARCHAR(16) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  is_used SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL
);
CREATE INDEX IF NOT EXISTS idx_verification_identifier ON verification_code (identifier);
CREATE INDEX IF NOT EXISTS idx_verification_expires_at ON verification_code (expires_at);
CREATE INDEX IF NOT EXISTS idx_verification_is_used ON verification_code (is_used);
CREATE INDEX IF NOT EXISTS idx_verification_is_deleted ON verification_code (is_deleted);
DROP TRIGGER IF EXISTS trg_verification_code_updated_at ON verification_code;
CREATE TRIGGER trg_verification_code_updated_at BEFORE UPDATE ON verification_code FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 13) essay_outline_practice
CREATE TABLE IF NOT EXISTS essay_outline_practice (
  id BIGSERIAL PRIMARY KEY,
  student_id BIGINT NOT NULL,
  subject_id BIGINT NULL,
  topic_text TEXT NOT NULL,
  topic_label VARCHAR(128) NOT NULL,
  topic_source VARCHAR(32) NOT NULL,
  genre VARCHAR(32) NULL,
  task_type VARCHAR(32) NULL,
  outline_text TEXT NOT NULL,
  review_json JSONB NULL,
  raw_review_response TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT fk_essay_outline_student FOREIGN KEY (student_id) REFERENCES student (id),
  CONSTRAINT fk_essay_outline_subject FOREIGN KEY (subject_id) REFERENCES subject (id)
);
CREATE INDEX IF NOT EXISTS idx_essay_outline_student ON essay_outline_practice (student_id);
CREATE INDEX IF NOT EXISTS idx_essay_outline_created ON essay_outline_practice (created_at);
CREATE INDEX IF NOT EXISTS idx_essay_outline_is_deleted ON essay_outline_practice (is_deleted);
DROP TRIGGER IF EXISTS trg_essay_outline_practice_updated_at ON essay_outline_practice;
CREATE TRIGGER trg_essay_outline_practice_updated_at BEFORE UPDATE ON essay_outline_practice FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 14) ai_call_log
CREATE TABLE IF NOT EXISTS ai_call_log (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ai_model_id BIGINT NULL,
  model_name_snapshot VARCHAR(128) NOT NULL DEFAULT '',
  action VARCHAR(64) NOT NULL DEFAULT 'paper_analyze',
  adapter_kind VARCHAR(64) NOT NULL DEFAULT '',
  result_status VARCHAR(32) NOT NULL DEFAULT '',
  http_status INT NULL,
  latency_ms BIGINT NULL,
  error_phase VARCHAR(32) NOT NULL DEFAULT '',
  error_message VARCHAR(512) NOT NULL DEFAULT '',
  endpoint_host VARCHAR(255) NOT NULL DEFAULT '',
  chat_model VARCHAR(128) NOT NULL DEFAULT '',
  fallback_to_mock SMALLINT NOT NULL DEFAULT 0,
  student_id BIGINT NULL,
  ref_table VARCHAR(64) NULL,
  ref_id BIGINT NULL,
  request_meta JSONB NULL,
  response_meta JSONB NULL,
  request_body TEXT NULL,
  response_body TEXT NULL,
  CONSTRAINT fk_ai_call_log_model FOREIGN KEY (ai_model_id) REFERENCES ai_model (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_created ON ai_call_log (created_at);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_model ON ai_call_log (ai_model_id);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_action ON ai_call_log (action);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_status ON ai_call_log (result_status);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_adapter ON ai_call_log (adapter_kind);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_student ON ai_call_log (student_id);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_ref ON ai_call_log (ref_table, ref_id);

-- 15) audit_log
CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NULL,
  user_type VARCHAR(16) NOT NULL,
  action VARCHAR(32) NOT NULL,
  entity_type VARCHAR(64) NOT NULL,
  entity_id BIGINT NULL,
  snapshot JSONB NULL,
  ip_address VARCHAR(64) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity_type ON audit_log (entity_type);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log (created_at);

-- 16) textbook / chapter / section / slide_deck
CREATE TABLE IF NOT EXISTS textbook (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  version VARCHAR(50) NOT NULL,
  subject VARCHAR(20) NOT NULL,
  category VARCHAR(20) NOT NULL,
  subject_id BIGINT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  remarks VARCHAR(255) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_textbook_name_version UNIQUE (name, version),
  CONSTRAINT fk_textbook_subject FOREIGN KEY (subject_id) REFERENCES subject (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_textbook_subject_id ON textbook (subject_id);
CREATE INDEX IF NOT EXISTS idx_textbook_category ON textbook (category);
CREATE INDEX IF NOT EXISTS idx_textbook_status ON textbook (status);
CREATE INDEX IF NOT EXISTS idx_textbook_is_deleted ON textbook (is_deleted);
DROP TRIGGER IF EXISTS trg_textbook_updated_at ON textbook;
CREATE TRIGGER trg_textbook_updated_at BEFORE UPDATE ON textbook FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS chapter (
  id BIGSERIAL PRIMARY KEY,
  textbook_id BIGINT NOT NULL,
  number INT NOT NULL,
  title VARCHAR(100) NOT NULL,
  full_title VARCHAR(150) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT fk_chapter_textbook FOREIGN KEY (textbook_id) REFERENCES textbook (id)
);
CREATE INDEX IF NOT EXISTS idx_chapter_textbook ON chapter (textbook_id);
CREATE INDEX IF NOT EXISTS idx_chapter_textbook_number ON chapter (textbook_id, number);
CREATE INDEX IF NOT EXISTS idx_chapter_is_deleted ON chapter (is_deleted);
DROP TRIGGER IF EXISTS trg_chapter_updated_at ON chapter;
CREATE TRIGGER trg_chapter_updated_at BEFORE UPDATE ON chapter FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS section (
  id BIGSERIAL PRIMARY KEY,
  chapter_id BIGINT NOT NULL,
  number INT NOT NULL,
  title VARCHAR(100) NOT NULL,
  full_title VARCHAR(150) NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT fk_section_chapter FOREIGN KEY (chapter_id) REFERENCES chapter (id)
);
CREATE INDEX IF NOT EXISTS idx_section_chapter ON section (chapter_id);
CREATE INDEX IF NOT EXISTS idx_section_chapter_number ON section (chapter_id, number);
CREATE INDEX IF NOT EXISTS idx_section_is_deleted ON section (is_deleted);
DROP TRIGGER IF EXISTS trg_section_updated_at ON section;
CREATE TRIGGER trg_section_updated_at BEFORE UPDATE ON section FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS slide_deck (
  id BIGSERIAL PRIMARY KEY,
  section_id BIGINT NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '',
  deck_status VARCHAR(20) NOT NULL DEFAULT 'draft',
  schema_version INT NOT NULL DEFAULT 1,
  content JSONB NOT NULL,
  generation_prompt TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT fk_slide_deck_section FOREIGN KEY (section_id) REFERENCES section (id)
);
CREATE INDEX IF NOT EXISTS idx_slide_deck_section ON slide_deck (section_id);
CREATE INDEX IF NOT EXISTS idx_slide_deck_lookup ON slide_deck (section_id, deck_status, is_deleted);
DROP TRIGGER IF EXISTS trg_slide_deck_updated_at ON slide_deck;
CREATE TRIGGER trg_slide_deck_updated_at BEFORE UPDATE ON slide_deck FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();
