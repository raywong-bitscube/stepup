-- 教材目录：textbook（书） / chapter（章） / section（节）
-- 与 db/schema 同步；种子数据见 db/seed/textbook_yuedu_physics_required_2019.sql
SET NAMES utf8mb4;

-- =====================================
-- textbook（物理/语文等各版本教材元数据）
-- =====================================
CREATE TABLE IF NOT EXISTS textbook (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(50) NOT NULL COMMENT '书籍名称，如 物理 必修 第一册',
  version VARCHAR(50) NOT NULL COMMENT '教材版本，如 粤教版 2019',
  subject VARCHAR(20) NOT NULL COMMENT '所属学科展示名，如 物理',
  category VARCHAR(20) NOT NULL COMMENT '必修、选择性必修 等',
  subject_id BIGINT UNSIGNED NULL COMMENT '可选，关联 subject.id，与 subject 字段冗余便于报表',
  status TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1=active,0=inactive',
  remarks VARCHAR(255) NULL COMMENT '备注',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_textbook_name_version (name, version),
  KEY idx_textbook_subject_id (subject_id),
  KEY idx_textbook_category (category),
  KEY idx_textbook_status (status),
  KEY idx_textbook_is_deleted (is_deleted),
  CONSTRAINT fk_textbook_subject FOREIGN KEY (subject_id) REFERENCES subject (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- chapter（章；隶属于 textbook）
-- =====================================
CREATE TABLE IF NOT EXISTS chapter (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  textbook_id BIGINT UNSIGNED NOT NULL,
  number INT UNSIGNED NOT NULL COMMENT '章序号，从 1 起；需兼容「第一章上」时可改为 VARCHAR 迁移',
  title VARCHAR(100) NOT NULL COMMENT '短标题，如 运动的描述',
  full_title VARCHAR(150) NULL COMMENT '完整标题，如 第一章 运动的描述',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_chapter_textbook_number (textbook_id, number),
  KEY idx_chapter_textbook (textbook_id),
  KEY idx_chapter_is_deleted (is_deleted),
  CONSTRAINT fk_chapter_textbook FOREIGN KEY (textbook_id) REFERENCES textbook (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================
-- section（节；隶属于 chapter）
-- =====================================
CREATE TABLE IF NOT EXISTS section (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  chapter_id BIGINT UNSIGNED NOT NULL,
  number INT UNSIGNED NOT NULL COMMENT '节序号，从 1 起',
  title VARCHAR(100) NOT NULL COMMENT '短标题，如 质点 参考系 时间',
  full_title VARCHAR(150) NULL COMMENT '完整标题，如 第一节 质点 参考系 时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_section_chapter_number (chapter_id, number),
  KEY idx_section_chapter (chapter_id),
  KEY idx_section_is_deleted (is_deleted),
  CONSTRAINT fk_section_chapter FOREIGN KEY (chapter_id) REFERENCES chapter (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
