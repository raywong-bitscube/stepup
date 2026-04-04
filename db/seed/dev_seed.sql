-- StepUp v0.1 development seed
-- Run after schema initialization.

USE stepup;

-- Stage: 高中
INSERT INTO stage
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES
  ('高中', '默认阶段：高中', 1, NOW(), 0, NOW(), 0, 0)
ON DUPLICATE KEY UPDATE
  description = VALUES(description),
  status = VALUES(status),
  updated_at = NOW(),
  updated_by = 0;

-- Subject: 物理 / 语文
INSERT INTO subject
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES
  ('物理', '默认科目：物理', 1, NOW(), 0, NOW(), 0, 0),
  ('语文', '默认科目：语文', 1, NOW(), 0, NOW(), 0, 0)
ON DUPLICATE KEY UPDATE
  description = VALUES(description),
  status = VALUES(status),
  updated_at = NOW(),
  updated_by = 0;

-- Admin bootstrap account (password: admin123, bcrypt cost default)
-- Regenerate: go run scripts/gen_admin_bcrypt.go
INSERT INTO admin
  (username, password, role, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES
  ('admin', '$2a$10$cuSJlDfmvmDFtT/9q68TTuRlwD.ZC/2Ki5ehU5bnrqtjrPVAEaGM2', 'super_admin', 1, NOW(), 0, NOW(), 0, 0)
ON DUPLICATE KEY UPDATE
  password = VALUES(password),
  role = VALUES(role),
  status = VALUES(status),
  updated_at = NOW(),
  updated_by = 0;

-- Kimi / Moonshot（OpenAI 兼容 chat/completions）。ANALYSIS_ADAPTER=http；app_key 为模型名，app_secret 为 API Key。
-- 国内常用 https://api.moonshot.cn/v1/chat/completions；国际可用 https://api.moonshot.ai/v1/chat/completions。
-- 执行前替换 app_secret；模型 id 可按控制台套餐改为 moonshot-v1-8k、moonshot-v1-32k、kimi-k2.5 等。
INSERT INTO ai_model
  (name, url, app_key, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'Kimi (Moonshot)',
  'https://api.moonshot.cn/v1/chat/completions',
  'kimi-k2.5',
  '__REPLACE_WITH_MOONSHOT_API_KEY__',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM ai_model
  WHERE url = 'https://api.moonshot.cn/v1/chat/completions'
    AND is_deleted = 0
);
