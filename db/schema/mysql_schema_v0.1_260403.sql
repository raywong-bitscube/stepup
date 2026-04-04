-- StepUp v0.1 MySQL Schema
-- 路径：db/schema/（全量基线）；增量变更另见 db/migrations/
-- Generated from:
--   docs/core/user_requirement_v0.1_260403.md
--   docs/core/entity_analyze_v0.1_260403.md
--
-- Conventions:
-- 1) Core business data uses soft delete: is_deleted/deleted_at/deleted_by
-- 2) Status for active/inactive uses: active = 1, inactive = 0
-- 3) Each record includes created_at/created_by
-- 4) Updatable records include updated_at/updated_by

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE DATABASE IF NOT EXISTS stepup
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE stepup;

-- =====================================
-- 1) stage
-- =====================================
CREATE TABLE IF NOT EXISTS stage (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_stage_name (name),
  KEY idx_stage_status (status),
  KEY idx_stage_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 2) subject
-- =====================================
CREATE TABLE IF NOT EXISTS subject (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_subject_name (name),
  KEY idx_subject_status (status),
  KEY idx_subject_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 3) student
-- =====================================
CREATE TABLE IF NOT EXISTS student (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  phone VARCHAR(32) NULL,
  email VARCHAR(255) NULL,
  password VARCHAR(255) NOT NULL,
  name VARCHAR(128) NOT NULL,
  stage_id BIGINT UNSIGNED NOT NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_student_phone (phone),
  UNIQUE KEY uk_student_email (email),
  KEY idx_student_stage_id (stage_id),
  KEY idx_student_status (status),
  KEY idx_student_is_deleted (is_deleted),
  CONSTRAINT fk_student_stage
    FOREIGN KEY (stage_id) REFERENCES stage (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 4) admin
-- =====================================
CREATE TABLE IF NOT EXISTS admin (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(32) NOT NULL COMMENT 'super_admin/admin/operator',
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_admin_username (username),
  KEY idx_admin_status (status),
  KEY idx_admin_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 5) ai_model
-- =====================================
CREATE TABLE IF NOT EXISTS ai_model (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  url VARCHAR(512) NOT NULL,
  app_key VARCHAR(255) NOT NULL,
  app_secret VARCHAR(255) NOT NULL,
  status TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_ai_model_status (status),
  KEY idx_ai_model_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 6) prompt_template
-- =====================================
CREATE TABLE IF NOT EXISTS prompt_template (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `key` VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  content TEXT NOT NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_prompt_key (`key`),
  KEY idx_prompt_status (status),
  KEY idx_prompt_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 7) exam_paper
-- =====================================
CREATE TABLE IF NOT EXISTS exam_paper (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  student_id BIGINT UNSIGNED NOT NULL,
  subject_id BIGINT UNSIGNED NOT NULL,
  file_url VARCHAR(1024) NOT NULL,
  file_type VARCHAR(16) NOT NULL COMMENT 'pdf/image',
  score INT NULL,
  exam_date DATE NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_exam_paper_student_id (student_id),
  KEY idx_exam_paper_subject_id (subject_id),
  KEY idx_exam_paper_is_deleted (is_deleted),
  CONSTRAINT fk_exam_paper_student
    FOREIGN KEY (student_id) REFERENCES student (id),
  CONSTRAINT fk_exam_paper_subject
    FOREIGN KEY (subject_id) REFERENCES subject (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 8) paper_analysis
-- =====================================
CREATE TABLE IF NOT EXISTS paper_analysis (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  paper_id BIGINT UNSIGNED NOT NULL,
  ai_model_snapshot JSON NOT NULL COMMENT 'at least {name,url}, no secret/key persisted',
  raw_content LONGTEXT NULL,
  ai_response LONGTEXT NULL COMMENT 'JSON string from AI',
  status TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0=pending,1=processing,2=completed,3=failed',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_paper_analysis_paper_id (paper_id),
  KEY idx_paper_analysis_status (status),
  KEY idx_paper_analysis_is_deleted (is_deleted),
  CONSTRAINT fk_paper_analysis_paper
    FOREIGN KEY (paper_id) REFERENCES exam_paper (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 9) improvement_plan
-- =====================================
CREATE TABLE IF NOT EXISTS improvement_plan (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  paper_id BIGINT UNSIGNED NOT NULL,
  plan_content LONGTEXT NOT NULL COMMENT 'JSON/Markdown',
  weak_points JSON NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_improvement_plan_paper_id (paper_id),
  KEY idx_improvement_plan_is_deleted (is_deleted),
  CONSTRAINT fk_improvement_plan_paper
    FOREIGN KEY (paper_id) REFERENCES exam_paper (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 10) admin_session
-- =====================================
CREATE TABLE IF NOT EXISTS admin_session (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  admin_id BIGINT UNSIGNED NOT NULL,
  session_token VARCHAR(255) NOT NULL COMMENT 'store token hash in production',
  expires_at DATETIME NOT NULL,
  last_seen_at DATETIME NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(512) NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_admin_session_token (session_token),
  KEY idx_admin_session_admin_id (admin_id),
  KEY idx_admin_session_expires_at (expires_at),
  KEY idx_admin_session_status (status),
  KEY idx_admin_session_is_deleted (is_deleted),
  CONSTRAINT fk_admin_session_admin
    FOREIGN KEY (admin_id) REFERENCES admin (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 11) student_session
-- =====================================
CREATE TABLE IF NOT EXISTS student_session (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  student_id BIGINT UNSIGNED NOT NULL,
  session_token VARCHAR(255) NOT NULL COMMENT 'store token hash in production',
  expires_at DATETIME NOT NULL,
  last_seen_at DATETIME NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(512) NULL,
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_student_session_token (session_token),
  KEY idx_student_session_student_id (student_id),
  KEY idx_student_session_expires_at (expires_at),
  KEY idx_student_session_status (status),
  KEY idx_student_session_is_deleted (is_deleted),
  CONSTRAINT fk_student_session_student
    FOREIGN KEY (student_id) REFERENCES student (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 12) verification_code
-- =====================================
CREATE TABLE IF NOT EXISTS verification_code (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  identifier VARCHAR(255) NOT NULL COMMENT 'phone or email',
  code VARCHAR(16) NOT NULL,
  type VARCHAR(16) NOT NULL COMMENT 'login/register',
  expires_at DATETIME NOT NULL,
  is_used TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_verification_identifier (identifier),
  KEY idx_verification_expires_at (expires_at),
  KEY idx_verification_is_used (is_used),
  KEY idx_verification_is_deleted (is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 13) ai_call_log（AI 外部调用轨迹，不含密钥与完整请求体）
-- =====================================
CREATE TABLE IF NOT EXISTS ai_call_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ai_model_id BIGINT UNSIGNED NULL COMMENT '当时他行 id，模型删除后可空',
  model_name_snapshot VARCHAR(128) NOT NULL DEFAULT '',
  action VARCHAR(64) NOT NULL DEFAULT 'paper_analyze',
  adapter_kind VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'mock_builtin, http_chat_completions, http_mock_ai_protocol, http_unconfigured',
  result_status VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'success, mock_only, fallback_mock',
  http_status INT NULL,
  latency_ms INT UNSIGNED NULL,
  error_phase VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'timeout, network, http_status, decode, parse, empty_body, ...',
  error_message VARCHAR(512) NOT NULL DEFAULT '',
  endpoint_host VARCHAR(255) NOT NULL DEFAULT '',
  chat_model VARCHAR(128) NOT NULL DEFAULT '',
  fallback_to_mock TINYINT(1) NOT NULL DEFAULT 0,
  paper_id BIGINT UNSIGNED NULL,
  student_id BIGINT UNSIGNED NULL,
  request_meta JSON NULL COMMENT 'subject, stage, file_name',
  response_meta JSON NULL COMMENT 'summary_len, weak_points_n, ...',
  PRIMARY KEY (id),
  KEY idx_ai_call_log_created (created_at),
  KEY idx_ai_call_log_model (ai_model_id),
  KEY idx_ai_call_log_action (action),
  KEY idx_ai_call_log_status (result_status),
  KEY idx_ai_call_log_adapter (adapter_kind),
  KEY idx_ai_call_log_paper (paper_id),
  KEY idx_ai_call_log_student (student_id),
  CONSTRAINT fk_ai_call_log_model FOREIGN KEY (ai_model_id) REFERENCES ai_model (id) ON DELETE SET NULL,
  CONSTRAINT fk_ai_call_log_paper FOREIGN KEY (paper_id) REFERENCES exam_paper (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- 14) audit_log
-- =====================================
CREATE TABLE IF NOT EXISTS audit_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NULL,
  user_type VARCHAR(16) NOT NULL COMMENT 'student/admin',
  action VARCHAR(32) NOT NULL COMMENT 'login/create/update/delete/password_change...',
  entity_type VARCHAR(64) NOT NULL,
  entity_id BIGINT UNSIGNED NULL,
  snapshot JSON NULL COMMENT 'before-image for update/delete, excludes sensitive data',
  ip_address VARCHAR(64) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (id),
  KEY idx_audit_log_user_id (user_id),
  KEY idx_audit_log_entity_type (entity_type),
  KEY idx_audit_log_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;
