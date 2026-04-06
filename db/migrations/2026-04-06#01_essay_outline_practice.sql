-- 作文提纲练习：落库每次「提交点评」记录；Prompt 模板供管理端可编辑
-- 命名：发文日 2026-04-06，当日批次 #01（见 db/README.md）
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS essay_outline_practice (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  student_id BIGINT UNSIGNED NOT NULL,
  subject_id BIGINT UNSIGNED NULL COMMENT '语文等，可空',
  topic_text TEXT NOT NULL,
  topic_label VARCHAR(128) NOT NULL COMMENT '如 议论文 · 材料作文 或 自定义',
  topic_source VARCHAR(32) NOT NULL COMMENT 'ai_category | custom_text | ocr_image',
  genre VARCHAR(32) NULL,
  task_type VARCHAR(32) NULL,
  outline_text LONGTEXT NOT NULL,
  review_json JSON NULL COMMENT 'summary, stars, suggestions, highlights',
  raw_review_response LONGTEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by BIGINT UNSIGNED NOT NULL,
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  deleted_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_essay_outline_student (student_id),
  KEY idx_essay_outline_created (created_at),
  KEY idx_essay_outline_is_deleted (is_deleted),
  CONSTRAINT fk_essay_outline_student FOREIGN KEY (student_id) REFERENCES student (id),
  CONSTRAINT fk_essay_outline_subject FOREIGN KEY (subject_id) REFERENCES subject (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO prompt_template (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (
  'essay_outline_generate_topic',
  '作文提纲-按文体与命题方式生成题目（占位符 %genre %task_type）',
  '你是一名有10年高中语文教学经验的资深教师，熟悉高考作文命题趋势。\n用户选择的文体形式为：%genre；命题方式为：%task_type。\n请生成1道符合近年高考趋势的作文题目。要求：题目需明确文体/命题类型，内容贴合高中生认知，具有思辨性或情感表达空间，避免偏题怪题。\n请严格用一行输出，格式为：{题目全文} | {文体/命题类型标签}。不要其它说明或换行。',
  1,
  NOW(),
  1,
  NOW(),
  1,
  0
) ON DUPLICATE KEY UPDATE
  description = VALUES(description),
  content = VALUES(content),
  updated_at = NOW();

INSERT INTO prompt_template (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (
  'essay_outline_review',
  '作文提纲-AI点评（占位符 %topic_text %outline_text）',
  '你是一名高考作文阅卷专家，请对用户的作文提纲进行专业点评。\n题目为：%topic_text\n用户提纲为：%outline_text\n请从以下维度分析：1.题目匹配度（是否紧扣文体/命题要求）；2.结构合理性（层次是否清晰，逻辑是否连贯）；3.素材适配性（素材是否典型、支撑中心）。\n请严格用一段连续文本输出三段，段与段之间用英文竖线 | 分隔，格式如下：\n{总体评价}|{维度评分：匹配度X星/结构X星/素材X星}|{详细建议：1.xxx；2.xxx}\n其中 X 为 1-5 的整数。不要 markdown 代码围栏。',
  1,
  NOW(),
  1,
  NOW(),
  1,
  0
) ON DUPLICATE KEY UPDATE
  description = VALUES(description),
  content = VALUES(content),
  updated_at = NOW();

INSERT INTO prompt_template (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (
  'essay_outline_ocr_topic',
  '作文提纲-从题目图片 OCR 提取正文（无占位符或后续扩展）',
  '请识别图片中的作文题目或材料内容，只输出应作为「题目文本」交给学生看的正文本身；不要加「题目：」等前缀，不要解释。若材料为多段，保留合理换行。',
  1,
  NOW(),
  1,
  NOW(),
  1,
  0
) ON DUPLICATE KEY UPDATE
  description = VALUES(description),
  content = VALUES(content),
  updated_at = NOW();
