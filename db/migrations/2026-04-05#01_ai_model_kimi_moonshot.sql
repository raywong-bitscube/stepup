-- 将当前激活模型切换为 Kimi（Moonshot，OpenAI 兼容 /v1/chat/completions）
-- 执行前将下方占位密钥替换为开放平台 API Key，或导入后在管理端「AI 模型」中修改。
-- 国际站可将 url 改为 https://api.moonshot.ai/v1/chat/completions（与控制台一致即可）。
-- 模型名 `model` 可按套餐调整，例如 moonshot-v1-8k、moonshot-v1-32k、kimi-k2.5 等。
--
-- 用法（在仓库根目录，库名按环境替换）：
--   mysql -u... -p... your_db < "db/migrations/2026-04-05#01_ai_model_kimi_moonshot.sql"

SET NAMES utf8mb4;

-- 仅保留一个「对外激活」模型：先全部置为未激活
UPDATE ai_model
SET status = 0, updated_at = NOW(), updated_by = 0
WHERE is_deleted = 0;

-- 尚无该端点时插入一行（避免重复执行产生多条同 URL）
INSERT INTO ai_model
  (name, url, model, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'Kimi (Moonshot)',
  'https://api.moonshot.cn/v1/chat/completions',
  'kimi-k2.5',
  '__REPLACE_WITH_MOONSHOT_API_KEY__',
  0,
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

-- 统一名称与模型 id（已存在行也更新）
UPDATE ai_model
SET
  name = 'Kimi (Moonshot)',
  model = 'kimi-k2.5',
  updated_at = NOW(),
  updated_by = 0
WHERE url = 'https://api.moonshot.cn/v1/chat/completions'
  AND is_deleted = 0;

-- 仅激活「该 URL 下 id 最大」的一行（与后端按激活行解析一致）
UPDATE ai_model m
JOIN (
  SELECT MAX(id) AS id FROM ai_model
  WHERE url = 'https://api.moonshot.cn/v1/chat/completions'
    AND is_deleted = 0
) t ON m.id = t.id
SET m.status = 1, m.updated_at = NOW(), m.updated_by = 0;
