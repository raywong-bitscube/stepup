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

-- Prompt: 试卷分析 user 模板（占位符 %subject %stage %file_name）；与 db/migrations/20260406_prompt_paper_analyze_template.sql 一致
INSERT INTO prompt_template
  (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'paper_analyze_chat_user',
  '学生试卷分析：发给大模型的 user 文案。占位符：%subject %stage %file_name；识图走多模态 image。',
  '试卷上传元信息：科目=%subject，阶段=%stage，原始文件名=%file_name。
请只输出一段合法 JSON（不要用 markdown 代码围栏），严格符合下列键：summary (string)、weak_points (string 数组)、improvement_plan (string 数组)、raw_content (string，可为试卷要点摘录或空字符串)。
内容针对中国学生试卷分析场景，用语简洁专业。',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM prompt_template
  WHERE `key` = 'paper_analyze_chat_user' AND is_deleted = 0
);
