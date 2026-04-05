-- Seed system prompt for student exam paper AI analysis (chat completions user message).
-- Placeholders: %subject, %stage, %file_name （有图时由多模态 image_url 传入，模型自行识图，无需文本槽位）

INSERT INTO prompt_template
  (`key`, description, content, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
  'paper_analyze_chat_user',
  '学生试卷分析：发给大模型的 user 文案（OpenAI 兼容 chat/completions）。占位符：%subject %stage %file_name；识图走消息的 image 部分。',
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
