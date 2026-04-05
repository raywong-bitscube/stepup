-- Legacy: 将 ai_model.app_key 重命名为 model（与 OpenAI chat 的 model 字段一致；密钥仍在 app_secret）。
-- 若基线 schema 已含 model 列，本过程为空操作。须在 `2026-04-05#01_ai_model_kimi_moonshot.sql` 之前执行（按文件名排序：日期递增，同日 #NN 递增）。
SET NAMES utf8mb4;

DELIMITER //

CREATE PROCEDURE stepup_rename_ai_model_app_key_to_model()
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'ai_model'
      AND COLUMN_NAME = 'app_key'
  ) THEN
    ALTER TABLE ai_model
      CHANGE COLUMN app_key model VARCHAR(255) NOT NULL COMMENT 'OpenAI-compatible chat model id';
  END IF;
END//

DELIMITER ;

CALL stepup_rename_ai_model_app_key_to_model();
DROP PROCEDURE stepup_rename_ai_model_app_key_to_model;
