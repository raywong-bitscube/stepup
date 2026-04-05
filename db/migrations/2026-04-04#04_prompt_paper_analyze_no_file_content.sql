-- 已部署旧版模板（含 %file_content 占位）时，对齐为「识图仅走多模态、不靠文本槽位」的默认文案。
UPDATE prompt_template
SET
  description = '学生试卷分析：发给大模型的 user 文案（OpenAI 兼容 chat/completions）。占位符：%subject %stage %file_name；识图走消息的 image 部分。',
  content = '试卷上传元信息：科目=%subject，阶段=%stage，原始文件名=%file_name。
请只输出一段合法 JSON（不要用 markdown 代码围栏），严格符合下列键：summary (string)、weak_points (string 数组)、improvement_plan (string 数组)、raw_content (string，可为试卷要点摘录或空字符串)。
内容针对中国学生试卷分析场景，用语简洁专业。',
  updated_at = NOW(),
  updated_by = 0
WHERE `key` = 'paper_analyze_chat_user'
  AND is_deleted = 0
  AND content LIKE '%file_content%';
