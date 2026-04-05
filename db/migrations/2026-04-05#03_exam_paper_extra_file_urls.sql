-- 多图试卷：除首张外其余文件 URL 存 JSON 数组（首张仍在 exam_paper.file_url）
SET NAMES utf8mb4;

ALTER TABLE exam_paper
  ADD COLUMN extra_file_urls JSON NULL COMMENT 'additional /uploads paths for multi-image batch; primary in file_url' AFTER file_url;
