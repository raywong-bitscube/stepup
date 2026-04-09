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

-- Subject: 物理 / 语文 / 数学 / 英语
INSERT INTO subject
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES
  ('物理', '默认科目：物理', 1, NOW(), 0, NOW(), 0, 0),
  ('语文', '默认科目：语文', 1, NOW(), 0, NOW(), 0, 0),
  ('数学', '默认科目：数学', 1, NOW(), 0, NOW(), 0, 0),
  ('英语', '默认科目：英语', 1, NOW(), 0, NOW(), 0, 0)
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

-- Kimi / Moonshot（OpenAI 兼容 chat/completions）。ANALYSIS_ADAPTER=http；model 为上游 model 字段，app_secret 为 API Key。
-- 国内常用 https://api.moonshot.cn/v1/chat/completions；国际可用 https://api.moonshot.ai/v1/chat/completions。
-- 执行前替换 app_secret；model 可按控制台套餐改为 moonshot-v1-8k、moonshot-v1-32k、kimi-k2.5 等。
INSERT INTO ai_model
  (name, url, model, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
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

-- Qwen（阿里云 Dashscope OpenAI 兼容）。默认未激活；将 status 改为 1 前需填写真实 API Key。
-- HTTP 适配器将 POST 到 url 所示完整路径，须含 /chat/completions。
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

-- Prompt: 试卷分析 user 模板（占位符 %subject %stage %file_name）；与 db/migrations/2026-04-04#03_prompt_paper_analyze_template.sql 一致
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

-- 作文提纲练习 Prompt（与 db/migrations/2026-04-06#01_essay_outline_practice.sql 一致；仅种子环境未跑迁移时补齐）
INSERT INTO prompt_template
  (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'essay_outline_generate_topic',
  '作文提纲-按文体与命题方式生成题目（占位符 %genre %task_type）',
  '你是一名有10年高中语文教学经验的资深教师，熟悉高考作文命题趋势。\n用户选择的文体形式为：%genre；命题方式为：%task_type。\n请生成1道符合近年高考趋势的作文题目。要求：题目需明确文体/命题类型，内容贴合高中生认知，具有思辨性或情感表达空间，避免偏题怪题。\n请严格用一行输出，格式为：{题目全文} | {文体/命题类型标签}。不要其它说明或换行。',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM prompt_template
  WHERE `key` = 'essay_outline_generate_topic' AND is_deleted = 0
);

INSERT INTO prompt_template
  (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'essay_outline_review',
  '作文提纲-AI点评（占位符 %topic_text %outline_text）',
  '你是一名高考作文阅卷专家，请对用户的作文提纲进行专业点评。\n题目为：%topic_text\n用户提纲为：%outline_text\n请从以下维度分析：1.题目匹配度（是否紧扣文体/命题要求）；2.结构合理性（层次是否清晰，逻辑是否连贯）；3.素材适配性（素材是否典型、支撑中心）。\n请严格用一段连续文本输出三段，段与段之间用英文竖线 | 分隔，格式如下：\n{总体评价}|{维度评分：匹配度X星/结构X星/素材X星}|{详细建议：1.xxx；2.xxx}\n其中 X 为 1-5 的整数。不要 markdown 代码围栏。\n仅输出上述三段中文正文；不要输出思考过程、英文推演或「Thinking」类内容。',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM prompt_template
  WHERE `key` = 'essay_outline_review' AND is_deleted = 0
);

INSERT INTO prompt_template
  (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'essay_outline_ocr_topic',
  '作文提纲-从题目图片 OCR 提取正文（无占位符或后续扩展）',
  '请识别图片中的作文题目或材料内容，只输出应作为「题目文本」交给学生看的正文本身；不要加「题目：」等前缀，不要解释。若材料为多段，保留合理换行。',
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM prompt_template
  WHERE `key` = 'essay_outline_ocr_topic' AND is_deleted = 0
);

-- 教材目录（粤教版 2019 物理必修一/二）：在跑完本文件且已执行 2026-04-08#01_textbook_chapter_section.sql 与 2026-04-08#02_chapter_section_status_drop_number_unique.sql（或基线 schema 已合并等价 DDL）后，可选：
--   mysql ... < db/seed/textbook_yuedu_physics_required_2019.sql
