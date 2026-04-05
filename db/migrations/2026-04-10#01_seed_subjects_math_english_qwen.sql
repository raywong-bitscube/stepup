-- 科目：数学、英语；AI 模型：qwen（Dashscope 兼容模式，默认未激活）
SET NAMES utf8mb4;

INSERT INTO subject
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT '数学', '默认科目：数学', 1, NOW(), 0, NOW(), 0, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM subject WHERE name = '数学' AND is_deleted = 0
);

INSERT INTO subject
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT '英语', '默认科目：英语', 1, NOW(), 0, NOW(), 0, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM subject WHERE name = '英语' AND is_deleted = 0
);

INSERT INTO ai_model
  (name, url, model, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'qwen',
  'https://coding.dashscope.aliyuncs.com/v1/chat/completions',
  'qwen3.5-plus',
  '__REPLACE_WITH_DASHSCOPE_API_KEY__',
  0,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM ai_model
  WHERE name = 'qwen'
    AND url = 'https://coding.dashscope.aliyuncs.com/v1/chat/completions'
    AND is_deleted = 0
);
