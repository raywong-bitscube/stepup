-- 章节幻灯片：slide_deck 挂载教材 section，同一节仅一条 deck_status=active（由应用层事务保证）
-- 设计见 docs/core/slide_deck_design_v0.1_260403.md
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS slide_deck (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  section_id BIGINT UNSIGNED NOT NULL COMMENT '挂载教材节',
  title VARCHAR(200) NOT NULL DEFAULT '' COMMENT '版本/说明，如 新课标 2024',
  deck_status VARCHAR(20) NOT NULL DEFAULT 'draft' COMMENT 'draft, active, archived',
  schema_version INT NOT NULL DEFAULT 1 COMMENT '与 JSON 内 schemaVersion 对齐',
  content JSON NOT NULL COMMENT 'Slide Deck JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_slide_deck_section (section_id),
  KEY idx_slide_deck_lookup (section_id, deck_status, is_deleted),
  CONSTRAINT fk_slide_deck_section FOREIGN KEY (section_id) REFERENCES section (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
