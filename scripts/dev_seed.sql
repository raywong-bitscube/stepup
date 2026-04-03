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

-- Admin bootstrap account
-- default password is plain text for current scaffold auth flow.
-- switch to bcrypt and hash-compare in production.
INSERT INTO admin
  (username, password, role, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES
  ('admin', 'admin123', 'super_admin', 1, NOW(), 0, NOW(), 0, 0)
ON DUPLICATE KEY UPDATE
  role = VALUES(role),
  status = VALUES(status),
  updated_at = NOW(),
  updated_by = 0;
