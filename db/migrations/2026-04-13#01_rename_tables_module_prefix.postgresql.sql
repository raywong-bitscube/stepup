-- One-time upgrade: legacy table names -> module-prefixed names + merged sys_session + ai_call_log columns.
-- Target: PostgreSQL databases created from the pre-2026-04-13 baseline (stage, student, exam_paper, …).
-- If you created the DB from the updated postgresql_schema_v0.1_260403.sql (already using k12_grade, sys_user, …), skip this file.
-- Usage:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "db/migrations/2026-04-13#01_rename_tables_module_prefix.postgresql.sql"

BEGIN;

-- 1) Core renames
DO $$
BEGIN
  IF to_regclass('public.stage') IS NOT NULL AND to_regclass('public.k12_grade') IS NULL THEN
    ALTER TABLE stage RENAME TO k12_grade;
  END IF;

  IF to_regclass('public.subject') IS NOT NULL AND to_regclass('public.k12_subject') IS NULL THEN
    ALTER TABLE subject RENAME TO k12_subject;
  END IF;

  IF to_regclass('public.admin') IS NOT NULL AND to_regclass('public.sys_admin_user') IS NULL THEN
    ALTER TABLE admin RENAME TO sys_admin_user;
  END IF;

  IF to_regclass('public.ai_model') IS NOT NULL AND to_regclass('public.ai_provider_model') IS NULL THEN
    ALTER TABLE ai_model RENAME TO ai_provider_model;
  END IF;

  IF to_regclass('public.prompt_template') IS NOT NULL AND to_regclass('public.ai_prompt_template') IS NULL THEN
    ALTER TABLE prompt_template RENAME TO ai_prompt_template;
  END IF;

  IF to_regclass('public.verification_code') IS NOT NULL AND to_regclass('public.sys_verification_code') IS NULL THEN
    ALTER TABLE verification_code RENAME TO sys_verification_code;
  END IF;

  IF to_regclass('public.student') IS NOT NULL AND to_regclass('public.sys_user') IS NULL THEN
    ALTER TABLE student RENAME TO sys_user;
    ALTER TABLE sys_user RENAME COLUMN stage_id TO k12_grade_id;
    ALTER TABLE sys_user DROP CONSTRAINT IF EXISTS fk_student_stage;
    ALTER TABLE sys_user ADD CONSTRAINT fk_sys_user_k12_grade FOREIGN KEY (k12_grade_id) REFERENCES k12_grade(id);
  END IF;

  IF to_regclass('public.exam_paper') IS NOT NULL AND to_regclass('public.student_exam_paper') IS NULL THEN
    ALTER TABLE exam_paper RENAME TO student_exam_paper;
    ALTER TABLE student_exam_paper RENAME COLUMN student_id TO sys_user_id;
    ALTER TABLE student_exam_paper RENAME COLUMN subject_id TO k12_subject_id;
    ALTER TABLE student_exam_paper DROP CONSTRAINT IF EXISTS fk_exam_paper_student;
    ALTER TABLE student_exam_paper DROP CONSTRAINT IF EXISTS fk_exam_paper_subject;
    ALTER TABLE student_exam_paper ADD CONSTRAINT fk_student_exam_paper_sys_user FOREIGN KEY (sys_user_id) REFERENCES sys_user(id);
    ALTER TABLE student_exam_paper ADD CONSTRAINT fk_student_exam_paper_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject(id);
  END IF;

  IF to_regclass('public.paper_analysis') IS NOT NULL AND to_regclass('public.student_paper_analysis') IS NULL THEN
    ALTER TABLE paper_analysis RENAME TO student_paper_analysis;
    ALTER TABLE student_paper_analysis DROP CONSTRAINT IF EXISTS fk_paper_analysis_paper;
    ALTER TABLE student_paper_analysis ADD CONSTRAINT fk_student_paper_analysis_paper FOREIGN KEY (paper_id) REFERENCES student_exam_paper(id);
  END IF;

  IF to_regclass('public.improvement_plan') IS NOT NULL AND to_regclass('public.student_improvement_plan') IS NULL THEN
    ALTER TABLE improvement_plan RENAME TO student_improvement_plan;
    ALTER TABLE student_improvement_plan DROP CONSTRAINT IF EXISTS fk_improvement_plan_paper;
    ALTER TABLE student_improvement_plan ADD CONSTRAINT fk_student_improvement_plan_paper FOREIGN KEY (paper_id) REFERENCES student_exam_paper(id);
  END IF;

  IF to_regclass('public.essay_outline_practice') IS NOT NULL AND to_regclass('public.student_essay_outline_practice') IS NULL THEN
    ALTER TABLE essay_outline_practice RENAME TO student_essay_outline_practice;
    ALTER TABLE student_essay_outline_practice RENAME COLUMN student_id TO sys_user_id;
    ALTER TABLE student_essay_outline_practice RENAME COLUMN subject_id TO k12_subject_id;
    ALTER TABLE student_essay_outline_practice DROP CONSTRAINT IF EXISTS fk_essay_outline_student;
    ALTER TABLE student_essay_outline_practice DROP CONSTRAINT IF EXISTS fk_essay_outline_subject;
    ALTER TABLE student_essay_outline_practice ADD CONSTRAINT fk_student_essay_outline_sys_user FOREIGN KEY (sys_user_id) REFERENCES sys_user(id);
    ALTER TABLE student_essay_outline_practice ADD CONSTRAINT fk_student_essay_outline_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject(id);
  END IF;

  IF to_regclass('public.slide_deck') IS NOT NULL AND to_regclass('public.textbook_slide_deck') IS NULL THEN
    ALTER TABLE slide_deck RENAME TO textbook_slide_deck;
    ALTER TABLE textbook_slide_deck DROP CONSTRAINT IF EXISTS fk_slide_deck_section;
    ALTER TABLE textbook_slide_deck ADD CONSTRAINT fk_textbook_slide_deck_section FOREIGN KEY (section_id) REFERENCES textbook_section(id);
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'textbook' AND column_name = 'subject_id'
  ) THEN
    ALTER TABLE textbook RENAME COLUMN subject_id TO k12_subject_id;
    ALTER TABLE textbook DROP CONSTRAINT IF EXISTS fk_textbook_subject;
    ALTER TABLE textbook ADD CONSTRAINT fk_textbook_k12_subject FOREIGN KEY (k12_subject_id) REFERENCES k12_subject(id) ON DELETE SET NULL;
  END IF;
END $$;

-- 2) ai_call_log columns + FK
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'ai_call_log' AND column_name = 'ai_model_id'
  ) THEN
    ALTER TABLE ai_call_log DROP CONSTRAINT IF EXISTS fk_ai_call_log_model;
    ALTER TABLE ai_call_log RENAME COLUMN ai_model_id TO ai_provider_model_id;
    ALTER TABLE ai_call_log ADD CONSTRAINT fk_ai_call_log_ai_provider_model
      FOREIGN KEY (ai_provider_model_id) REFERENCES ai_provider_model(id) ON DELETE SET NULL;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'ai_call_log' AND column_name = 'student_id'
  ) THEN
    ALTER TABLE ai_call_log RENAME COLUMN student_id TO sys_user_id;
  END IF;
END $$;

UPDATE ai_call_log SET ref_table = 'student_exam_paper' WHERE ref_table = 'exam_paper';

-- 3) Merge admin_session + student_session -> sys_session
DO $$
BEGIN
  IF to_regclass('public.sys_session') IS NULL AND to_regclass('public.admin_session') IS NOT NULL THEN
    CREATE TABLE sys_session (
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
    CREATE INDEX idx_sys_session_user ON sys_session (user_type, user_id);
    CREATE INDEX idx_sys_session_expires_at ON sys_session (expires_at);
    CREATE INDEX idx_sys_session_status ON sys_session (status);
    CREATE INDEX idx_sys_session_is_deleted ON sys_session (is_deleted);

    INSERT INTO sys_session (
      user_type, user_id, session_token, expires_at, last_seen_at, ip_address, user_agent,
      status, created_at, created_by, updated_at, updated_by, is_deleted, deleted_at, deleted_by
    )
    SELECT 'admin', admin_id, session_token, expires_at, last_seen_at, ip_address, user_agent,
           status, created_at, created_by, updated_at, updated_by, is_deleted, deleted_at, deleted_by
    FROM admin_session;

    IF to_regclass('public.student_session') IS NOT NULL THEN
      INSERT INTO sys_session (
        user_type, user_id, session_token, expires_at, last_seen_at, ip_address, user_agent,
        status, created_at, created_by, updated_at, updated_by, is_deleted, deleted_at, deleted_by
      )
      SELECT 'student', student_id, session_token, expires_at, last_seen_at, ip_address, user_agent,
             status, created_at, created_by, updated_at, updated_by, is_deleted, deleted_at, deleted_by
      FROM student_session;
      DROP TABLE student_session;
    END IF;

    DROP TABLE admin_session;

    CREATE TRIGGER trg_sys_session_updated_at BEFORE UPDATE ON sys_session FOR EACH ROW EXECUTE PROCEDURE stepup_touch_updated_at();
  END IF;
END $$;

COMMIT;
