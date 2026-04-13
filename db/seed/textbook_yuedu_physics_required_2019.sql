-- 粤教版（2019）高中物理 必修第一册 / 必修第二册 — 章节目录
-- 依赖：k12_subject 中已有「物理」、执行顺序在 dev_seed 之后（或确保 textbook 表已存在）
-- 可重复执行：textbook 依赖 uk_textbook_name_version；章/节用 NOT EXISTS（无 number 唯一约束）

-- ---------------------------------------------------------------------------
-- textbook（两册）
-- ---------------------------------------------------------------------------
INSERT INTO textbook (name, version, subject, category, k12_subject_id, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (
  '物理 必修 第一册',
  '粤教版 2019',
  '物理',
  '必修',
  (SELECT id FROM k12_subject WHERE name = '物理' AND is_deleted = 0 LIMIT 1),
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
)
ON CONFLICT (name, version) DO UPDATE SET
  subject = EXCLUDED.subject,
  category = EXCLUDED.category,
  k12_subject_id = EXCLUDED.k12_subject_id,
  status = EXCLUDED.status,
  updated_at = EXCLUDED.updated_at,
  updated_by = EXCLUDED.updated_by;

INSERT INTO textbook (name, version, subject, category, k12_subject_id, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (
  '物理 必修 第二册',
  '粤教版 2019',
  '物理',
  '必修',
  (SELECT id FROM k12_subject WHERE name = '物理' AND is_deleted = 0 LIMIT 1),
  1,
  NOW(),
  0,
  NOW(),
  0,
  0
)
ON CONFLICT (name, version) DO UPDATE SET
  subject = EXCLUDED.subject,
  category = EXCLUDED.category,
  k12_subject_id = EXCLUDED.k12_subject_id,
  status = EXCLUDED.status,
  updated_at = EXCLUDED.updated_at,
  updated_by = EXCLUDED.updated_by;

-- ---------------------------------------------------------------------------
-- 必修第一册 — chapters
-- ---------------------------------------------------------------------------
INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 1, '运动的描述', '第一章 运动的描述', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 1 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 2, '匀变速直线运动', '第二章 匀变速直线运动', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 2 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 3, '相互作用', '第三章 相互作用', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 3 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 4, '牛顿运动定律', '第四章 牛顿运动定律', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 4 AND c.is_deleted = 0)
LIMIT 1;

-- ---------------------------------------------------------------------------
-- 必修第二册 — chapters
-- ---------------------------------------------------------------------------
INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 1, '抛体运动', '第一章 抛体运动', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 1 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 2, '圆周运动', '第二章 圆周运动', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 2 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 3, '万有引力定律', '第三章 万有引力定律', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 3 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 4, '机械能及其守恒定律', '第四章 机械能及其守恒定律', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 4 AND c.is_deleted = 0)
LIMIT 1;

INSERT INTO textbook_chapter (textbook_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT t.id, 5, '牛顿力学的局限性与相对论初步', '第五章 牛顿力学的局限性与相对论初步', 1, NOW(), 0, NOW(), 0, 0
FROM textbook t
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND t.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_chapter c WHERE c.textbook_id = t.id AND c.number = 5 AND c.is_deleted = 0)
LIMIT 1;

-- ---------------------------------------------------------------------------
-- 必修第一册 — sections（按章批量插入）
-- ---------------------------------------------------------------------------
INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '质点 参考系 时间' AS title, '第一节 质点 参考系 时间' AS full_title
  UNION ALL SELECT 2, '位置 位移', '第二节 位置 位移'
  UNION ALL SELECT 3, '速度', '第三节 速度'
  UNION ALL SELECT 4, '测量直线运动物体的瞬时速度', '第四节 测量直线运动物体的瞬时速度'
  UNION ALL SELECT 5, '加速度', '第五节 加速度'
) v
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND c.number = 1 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '匀变速直线运动的特点' AS title, '第一节 匀变速直线运动的特点' AS full_title
  UNION ALL SELECT 2, '匀变速直线运动的规律', '第二节 匀变速直线运动的规律'
  UNION ALL SELECT 3, '测量匀变速直线运动的加速度', '第三节 测量匀变速直线运动的加速度'
  UNION ALL SELECT 4, '自由落体运动', '第四节 自由落体运动'
  UNION ALL SELECT 5, '匀变速直线运动与汽车安全行驶', '第五节 匀变速直线运动与汽车安全行驶'
) v
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND c.number = 2 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '重力' AS title, '第一节 重力' AS full_title
  UNION ALL SELECT 2, '弹力', '第二节 弹力'
  UNION ALL SELECT 3, '摩擦力', '第三节 摩擦力'
  UNION ALL SELECT 4, '力的合成', '第四节 力的合成'
  UNION ALL SELECT 5, '力的分解', '第五节 力的分解'
  UNION ALL SELECT 6, '共点力的平衡条件及其应用', '第六节 共点力的平衡条件及其应用'
) v
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND c.number = 3 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '牛顿第一定律' AS title, '第一节 牛顿第一定律' AS full_title
  UNION ALL SELECT 2, '加速度与力、质量之间的关系', '第二节 加速度与力、质量之间的关系'
  UNION ALL SELECT 3, '牛顿第二定律', '第三节 牛顿第二定律'
  UNION ALL SELECT 4, '牛顿第三定律', '第四节 牛顿第三定律'
  UNION ALL SELECT 5, '牛顿运动定律的应用', '第五节 牛顿运动定律的应用'
  UNION ALL SELECT 6, '失重和超重', '第六节 失重和超重'
  UNION ALL SELECT 7, '力学单位', '第七节 力学单位'
) v
WHERE t.name = '物理 必修 第一册' AND t.version = '粤教版 2019' AND c.number = 4 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

-- ---------------------------------------------------------------------------
-- 必修第二册 — sections
-- ---------------------------------------------------------------------------
INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '曲线运动' AS title, '第一节 曲线运动' AS full_title
  UNION ALL SELECT 2, '运动的合成与分解', '第二节 运动的合成与分解'
  UNION ALL SELECT 3, '平抛运动', '第三节 平抛运动'
  UNION ALL SELECT 4, '生活和生产中的抛体运动', '第四节 生活和生产中的抛体运动'
) v
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND c.number = 1 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '匀速圆周运动' AS title, '第一节 匀速圆周运动' AS full_title
  UNION ALL SELECT 2, '向心力与向心加速度', '第二节 向心力与向心加速度'
  UNION ALL SELECT 3, '生活中的圆周运动', '第三节 生活中的圆周运动'
  UNION ALL SELECT 4, '离心现象及其应用', '第四节 离心现象及其应用'
) v
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND c.number = 2 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '认识天体运动' AS title, '第一节 认识天体运动' AS full_title
  UNION ALL SELECT 2, '认识万有引力定律', '第二节 认识万有引力定律'
  UNION ALL SELECT 3, '万有引力定律的应用', '第三节 万有引力定律的应用'
  UNION ALL SELECT 4, '宇宙速度与航天', '第四节 宇宙速度与航天'
) v
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND c.number = 3 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '功' AS title, '第一节 功' AS full_title
  UNION ALL SELECT 2, '功率', '第二节 功率'
  UNION ALL SELECT 3, '动能 动能定理', '第三节 动能 动能定理'
  UNION ALL SELECT 4, '势能', '第四节 势能'
  UNION ALL SELECT 5, '机械能守恒定律', '第五节 机械能守恒定律'
  UNION ALL SELECT 6, '验证机械能守恒定律', '第六节 验证机械能守恒定律'
  UNION ALL SELECT 7, '生产和生活中的机械能守恒', '第七节 生产和生活中的机械能守恒'
) v
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND c.number = 4 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);

INSERT INTO textbook_section (chapter_id, number, title, full_title, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT c.id, v.n, v.title, v.full_title, 1, NOW(), 0, NOW(), 0, 0
FROM textbook_chapter c
JOIN textbook t ON t.id = c.textbook_id
JOIN (
  SELECT 1 AS n, '牛顿力学的成就与局限性' AS title, '第一节 牛顿力学的成就与局限性' AS full_title
  UNION ALL SELECT 2, '相对论时空观', '第二节 相对论时空观'
  UNION ALL SELECT 3, '宇宙起源和演化', '第三节 宇宙起源和演化'
) v
WHERE t.name = '物理 必修 第二册' AND t.version = '粤教版 2019' AND c.number = 5 AND t.is_deleted = 0 AND c.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM textbook_section s WHERE s.chapter_id = c.id AND s.number = v.n AND s.is_deleted = 0);
