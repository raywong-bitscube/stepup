-- Full (redacted/truncated) HTTP request/response text for admin AI call logs.
ALTER TABLE ai_call_log
  ADD COLUMN request_body LONGTEXT NULL COMMENT 'chat请求JSON（图片base64已脱敏）' AFTER response_meta,
  ADD COLUMN response_body LONGTEXT NULL COMMENT '上游响应原文（可截断）' AFTER request_body;
