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

-- DeepSeek（OpenAI 兼容 chat/completions）。运行后端请设 ANALYSIS_ADAPTER=http；app_key 为模型名，app_secret 为 API Key。
-- 执行前将 app_secret 占位符替换为真实 key（勿提交到 git）；若密钥曾出现在仓库历史中，请到 DeepSeek 控制台轮换。
INSERT INTO ai_model
  (name, url, app_key, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'DeepSeek',
  'https://api.deepseek.com/v1/chat/completions',
  'deepseek-chat',
  '__REPLACE_WITH_DEEPSEEK_API_KEY__',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM ai_model
  WHERE name = 'DeepSeek'
    AND url = 'https://api.deepseek.com/v1/chat/completions'
    AND is_deleted = 0
);
