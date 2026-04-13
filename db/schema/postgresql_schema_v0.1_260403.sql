-- StepUp v0.1 PostgreSQL baseline (module-prefixed tables + pgvector for embeddings)
-- Usage: psql "postgres://USER:PASS@HOST:5432/stepup?sslmode=disable" -f db/schema/postgresql_schema_v0.1_260403.sql
-- If the app connects as a different role than the one that ran this file, grant privileges:
--   see db/schema/postgresql_grants_app_role.sql
SET client_encoding = 'UTF8';

CREATE EXTENSION IF NOT EXISTS vector;

CREATE OR REPLACE FUNCTION stepup_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1) k12_grade (was: stage)
CREATE TABLE IF NOT EXISTS k12_grade (
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
  CONSTRAINT uk_k12_grade_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS idx_k12_grade_status ON k12_grade (status);
CREATE INDEX IF NOT EXISTS idx_k12_grade_is_deleted ON k12_grade (is_deleted);
DROP TRIGGER IF EXISTS trg_k12_grade_updated_at ON k12_grade;
CREATE TRIGGER trg_k12_grade_updated_at BEFORE UPDATE ON k12_grade FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 2) k12_subject (was: subject)
CREATE TABLE IF NOT EXISTS k12_subject (
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
  CONSTRAINT uk_k12_subject_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS idx_k12_subject_status ON k12_subject (status);
CREATE INDEX IF NOT EXISTS idx_k12_subject_is_deleted ON k12_subject (is_deleted);
DROP TRIGGER IF EXISTS trg_k12_subject_updated_at ON k12_subject;
CREATE TRIGGER trg_k12_subject_updated_at BEFORE UPDATE ON k12_subject FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 3) sys_user (was: student; future parent/teacher rows may use same table with profile/discriminator)
CREATE TABLE IF NOT EXISTS sys_user (
  id BIGSERIAL PRIMARY KEY,
  phone VARCHAR(32) NULL,
  email VARCHAR(255) NULL,
  password VARCHAR(255) NOT NULL,
  name VARCHAR(128) NOT NULL,
  k12_grade_id BIGINT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by BIGINT NOT NULL,
  is_deleted SMALLINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by BIGINT NULL,
  CONSTRAINT uk_sys_user_phone UNIQUE (phone),
  CONSTRAINT uk_sys_user_email UNIQUE (email),
  CONSTRAINT fk_sys_user_k12_grade FOREIGN KEY (k12_grade_id) REFERENCES k12_grade (id)
);
CREATE INDEX IF NOT EXISTS idx_sys_user_k12_grade_id ON sys_user (k12_grade_id);
CREATE INDEX IF NOT EXISTS idx_sys_user_status ON sys_user (status);
CREATE INDEX IF NOT EXISTS idx_sys_user_is_deleted ON sys_user (is_deleted);
DROP TRIGGER IF EXISTS trg_sys_user_updated_at ON sys_user;
CREATE TRIGGER trg_sys_user_updated_at BEFORE UPDATE ON sys_user FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 4) sys_admin_user (was: admin)
CREATE TABLE IF NOT EXISTS sys_admin_user (
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
  CONSTRAINT uk_sys_admin_user_username UNIQUE (username)
);
CREATE INDEX IF NOT EXISTS idx_sys_admin_user_status ON sys_admin_user (status);
CREATE INDEX IF NOT EXISTS idx_sys_admin_user_is_deleted ON sys_admin_user (is_deleted);
DROP TRIGGER IF EXISTS trg_sys_admin_user_updated_at ON sys_admin_user;
CREATE TRIGGER trg_sys_admin_user_updated_at BEFORE UPDATE ON sys_admin_user FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 5) ai_provider_model (was: ai_model)
CREATE TABLE IF NOT EXISTS ai_provider_model (
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
CREATE INDEX IF NOT EXISTS idx_ai_provider_model_status ON ai_provider_model (status);
CREATE INDEX IF NOT EXISTS idx_ai_provider_model_is_deleted ON ai_provider_model (is_deleted);
DROP TRIGGER IF EXISTS trg_ai_provider_model_updated_at ON ai_provider_model;
CREATE TRIGGER trg_ai_provider_model_updated_at BEFORE UPDATE ON ai_provider_model FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 6) ai_prompt_template (was: prompt_template)
CREATE TABLE IF NOT EXISTS ai_prompt_template (
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
  CONSTRAINT uk_ai_prompt_template_key UNIQUE ("key")
);
CREATE INDEX IF NOT EXISTS idx_ai_prompt_template_status ON ai_prompt_template (status);
CREATE INDEX IF NOT EXISTS idx_ai_prompt_template_is_deleted ON ai_prompt_template (is_deleted);
DROP TRIGGER IF EXISTS trg_ai_prompt_template_updated_at ON ai_prompt_template;
CREATE TRIGGER trg_ai_prompt_template_updated_at BEFORE UPDATE ON ai_prompt_template FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 7) student_exam_paper (was: exam_paper)
CREATE TABLE IF NOT EXISTS student_exam_paper (
  id BIGSERIAL PRIMARY KEY,
  sys_user_id BIGINT NOT NULL,
  k12_subject_id BIGINT NOT NULL,
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
  CONSTRAINT fk_student_exam_paper_sys_user FOREIGN KEY (sys_user_id) REFERENCES sys_user (id),
  CONSTRAINT fk_student_exam_paper_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject (id)
);
CREATE INDEX IF NOT EXISTS idx_student_exam_paper_sys_user_id ON student_exam_paper (sys_user_id);
CREATE INDEX IF NOT EXISTS idx_student_exam_paper_k12_subject_id ON student_exam_paper (k12_subject_id);
CREATE INDEX IF NOT EXISTS idx_student_exam_paper_is_deleted ON student_exam_paper (is_deleted);
DROP TRIGGER IF EXISTS trg_student_exam_paper_updated_at ON student_exam_paper;
CREATE TRIGGER trg_student_exam_paper_updated_at BEFORE UPDATE ON student_exam_paper FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 8) student_paper_analysis (was: paper_analysis)
CREATE TABLE IF NOT EXISTS student_paper_analysis (
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
  CONSTRAINT uk_student_paper_analysis_paper_id UNIQUE (paper_id),
  CONSTRAINT fk_student_paper_analysis_paper FOREIGN KEY (paper_id) REFERENCES student_exam_paper (id)
);
CREATE INDEX IF NOT EXISTS idx_student_paper_analysis_status ON student_paper_analysis (status);
CREATE INDEX IF NOT EXISTS idx_student_paper_analysis_is_deleted ON student_paper_analysis (is_deleted);
DROP TRIGGER IF EXISTS trg_student_paper_analysis_updated_at ON student_paper_analysis;
CREATE TRIGGER trg_student_paper_analysis_updated_at BEFORE UPDATE ON student_paper_analysis FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 9) student_improvement_plan (was: improvement_plan)
CREATE TABLE IF NOT EXISTS student_improvement_plan (
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
  CONSTRAINT uk_student_improvement_plan_paper_id UNIQUE (paper_id),
  CONSTRAINT fk_student_improvement_plan_paper FOREIGN KEY (paper_id) REFERENCES student_exam_paper (id)
);
CREATE INDEX IF NOT EXISTS idx_student_improvement_plan_is_deleted ON student_improvement_plan (is_deleted);
DROP TRIGGER IF EXISTS trg_student_improvement_plan_updated_at ON student_improvement_plan;
CREATE TRIGGER trg_student_improvement_plan_updated_at BEFORE UPDATE ON student_improvement_plan FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 10) sys_session (merged admin_session + student_session; user_type distinguishes principals)
CREATE TABLE IF NOT EXISTS sys_session (
  id BIGSERIAL PRIMARY KEY,
  user_type VARCHAR(32) NOT NULL,
  user_id BIGINT NOT NULL,
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
  CONSTRAINT uk_sys_session_token UNIQUE (session_token)
);
CREATE INDEX IF NOT EXISTS idx_sys_session_user ON sys_session (user_type, user_id);
CREATE INDEX IF NOT EXISTS idx_sys_session_expires_at ON sys_session (expires_at);
CREATE INDEX IF NOT EXISTS idx_sys_session_status ON sys_session (status);
CREATE INDEX IF NOT EXISTS idx_sys_session_is_deleted ON sys_session (is_deleted);
DROP TRIGGER IF EXISTS trg_sys_session_updated_at ON sys_session;
CREATE TRIGGER trg_sys_session_updated_at BEFORE UPDATE ON sys_session FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 11) sys_verification_code (was: verification_code)
CREATE TABLE IF NOT EXISTS sys_verification_code (
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
CREATE INDEX IF NOT EXISTS idx_sys_verification_identifier ON sys_verification_code (identifier);
CREATE INDEX IF NOT EXISTS idx_sys_verification_expires_at ON sys_verification_code (expires_at);
CREATE INDEX IF NOT EXISTS idx_sys_verification_is_used ON sys_verification_code (is_used);
CREATE INDEX IF NOT EXISTS idx_sys_verification_is_deleted ON sys_verification_code (is_deleted);
DROP TRIGGER IF EXISTS trg_sys_verification_code_updated_at ON sys_verification_code;
CREATE TRIGGER trg_sys_verification_code_updated_at BEFORE UPDATE ON sys_verification_code FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 12) student_essay_outline_practice (was: essay_outline_practice)
CREATE TABLE IF NOT EXISTS student_essay_outline_practice (
  id BIGSERIAL PRIMARY KEY,
  sys_user_id BIGINT NOT NULL,
  k12_subject_id BIGINT NULL,
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
  CONSTRAINT fk_student_essay_outline_sys_user FOREIGN KEY (sys_user_id) REFERENCES sys_user (id),
  CONSTRAINT fk_student_essay_outline_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject (id)
);
CREATE INDEX IF NOT EXISTS idx_student_essay_outline_sys_user ON student_essay_outline_practice (sys_user_id);
CREATE INDEX IF NOT EXISTS idx_student_essay_outline_created ON student_essay_outline_practice (created_at);
CREATE INDEX IF NOT EXISTS idx_student_essay_outline_is_deleted ON student_essay_outline_practice (is_deleted);
DROP TRIGGER IF EXISTS trg_student_essay_outline_practice_updated_at ON student_essay_outline_practice;
CREATE TRIGGER trg_student_essay_outline_practice_updated_at BEFORE UPDATE ON student_essay_outline_practice FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

-- 13) ai_call_log
CREATE TABLE IF NOT EXISTS ai_call_log (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ai_provider_model_id BIGINT NULL,
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
  sys_user_id BIGINT NULL,
  ref_table VARCHAR(64) NULL,
  ref_id BIGINT NULL,
  request_meta JSONB NULL,
  response_meta JSONB NULL,
  request_body TEXT NULL,
  response_body TEXT NULL,
  CONSTRAINT fk_ai_call_log_ai_provider_model FOREIGN KEY (ai_provider_model_id) REFERENCES ai_provider_model (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_created ON ai_call_log (created_at);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_model ON ai_call_log (ai_provider_model_id);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_action ON ai_call_log (action);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_status ON ai_call_log (result_status);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_adapter ON ai_call_log (adapter_kind);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_sys_user ON ai_call_log (sys_user_id);
CREATE INDEX IF NOT EXISTS idx_ai_call_log_ref ON ai_call_log (ref_table, ref_id);

-- 14) audit_log
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

-- 15) textbook / chapter / section / textbook_slide_deck
CREATE TABLE IF NOT EXISTS textbook (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  version VARCHAR(50) NOT NULL,
  subject VARCHAR(20) NOT NULL,
  category VARCHAR(20) NOT NULL,
  k12_subject_id BIGINT NULL,
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
  CONSTRAINT fk_textbook_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_textbook_k12_subject_id ON textbook (k12_subject_id);
CREATE INDEX IF NOT EXISTS idx_textbook_category ON textbook (category);
CREATE INDEX IF NOT EXISTS idx_textbook_status ON textbook (status);
CREATE INDEX IF NOT EXISTS idx_textbook_is_deleted ON textbook (is_deleted);
DROP TRIGGER IF EXISTS trg_textbook_updated_at ON textbook;
CREATE TRIGGER trg_textbook_updated_at BEFORE UPDATE ON textbook FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS textbook_chapter (
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
  CONSTRAINT fk_textbook_chapter_textbook FOREIGN KEY (textbook_id) REFERENCES textbook (id)
);
CREATE INDEX IF NOT EXISTS idx_textbook_chapter_textbook ON textbook_chapter (textbook_id);
CREATE INDEX IF NOT EXISTS idx_textbook_chapter_textbook_number ON textbook_chapter (textbook_id, number);
CREATE INDEX IF NOT EXISTS idx_textbook_chapter_is_deleted ON textbook_chapter (is_deleted);
DROP TRIGGER IF EXISTS trg_textbook_chapter_updated_at ON textbook_chapter;
CREATE TRIGGER trg_textbook_chapter_updated_at BEFORE UPDATE ON textbook_chapter FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS textbook_section (
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
  CONSTRAINT fk_textbook_section_chapter FOREIGN KEY (chapter_id) REFERENCES textbook_chapter (id)
);
CREATE INDEX IF NOT EXISTS idx_textbook_section_chapter ON textbook_section (chapter_id);
CREATE INDEX IF NOT EXISTS idx_textbook_section_chapter_number ON textbook_section (chapter_id, number);
CREATE INDEX IF NOT EXISTS idx_textbook_section_is_deleted ON textbook_section (is_deleted);
DROP TRIGGER IF EXISTS trg_textbook_section_updated_at ON textbook_section;
CREATE TRIGGER trg_textbook_section_updated_at BEFORE UPDATE ON textbook_section FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();

CREATE TABLE IF NOT EXISTS textbook_slide_deck (
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
  CONSTRAINT fk_textbook_slide_deck_section FOREIGN KEY (section_id) REFERENCES textbook_section (id)
);
CREATE INDEX IF NOT EXISTS idx_textbook_slide_deck_section ON textbook_slide_deck (section_id);
CREATE INDEX IF NOT EXISTS idx_textbook_slide_deck_lookup ON textbook_slide_deck (section_id, deck_status, is_deleted);
DROP TRIGGER IF EXISTS trg_textbook_slide_deck_updated_at ON textbook_slide_deck;
CREATE TRIGGER trg_textbook_slide_deck_updated_at BEFORE UPDATE ON textbook_slide_deck FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();
