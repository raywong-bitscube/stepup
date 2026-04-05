-- 已存在 v0.1 库时追加 AI 调用日志表（与 db/schema/mysql_schema_v0.1_260403.sql 第 13 节一致）
-- 用法示例（在仓库根目录）：
--   mysql -u... -p... stepup < "db/migrations/2026-04-04#01_ai_call_log.sql"

SET NAMES utf8mb4;

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
