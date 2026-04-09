SET NAMES utf8mb4;

ALTER TABLE slide_deck
  ADD COLUMN generation_prompt LONGTEXT NULL COMMENT '最近一次 AI 生成时使用的完整 prompt' AFTER content;
