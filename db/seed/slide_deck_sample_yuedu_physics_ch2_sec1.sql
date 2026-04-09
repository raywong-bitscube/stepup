-- 示例：为「物理 必修 第一册 / 第二章 匀变速直线运动 / 第 1 节」挂载一套 active 幻灯片（若该节尚无 slide_deck 则插入）
-- 依赖：已执行 textbook 粤教版种子；已存在 slide_deck 表（2026-04-10#01_slide_deck.sql 或基线 schema）
USE stepup;

INSERT INTO slide_deck (section_id, title, deck_status, schema_version, content, created_by, updated_by, is_deleted)
SELECT
  s.id,
  '示例：匀变速（开发预览）',
  'active',
  1,
  CAST('{"schemaVersion":1,"meta":{"title":"匀变速直线运动","theme":"dark-physics"},"slides":[{"id":"slide-1","layoutTemplate":"cover-image","elements":[{"type":"text","role":"title","content":"匀变速直线运动","step":1},{"type":"text","role":"subtitle","content":"探索速度与时间的关系","step":2}]},{"id":"slide-2","layoutTemplate":"split-left-right","elements":[{"type":"latex","role":"main-formula","content":"v = v_0 + at","step":1},{"type":"text","role":"body","content":"公式描述速度与时间的关系。","step":2}]},{"id":"slide-3","layoutTemplate":"quiz-center","elements":[{"type":"question","role":"main-content","mode":"single","data":{"text":"关于加速度，下列说法正确的是？","options":[{"id":"A","text":"加速度是矢量"},{"id":"B","text":"速度为 0 则加速度一定为 0"}]},"step":1,"answer":{"correctOptionIds":["A"]}}]}]}' AS JSON),
  0,
  0,
  0
FROM section s
INNER JOIN chapter ch ON ch.id = s.chapter_id AND ch.is_deleted = 0
INNER JOIN textbook t ON t.id = ch.textbook_id AND t.is_deleted = 0
WHERE t.name = '物理 必修 第一册'
  AND t.version = '粤教版 2019'
  AND ch.number = 2
  AND s.number = 1
  AND s.is_deleted = 0
  AND NOT EXISTS (SELECT 1 FROM slide_deck d WHERE d.section_id = s.id AND d.is_deleted = 0)
LIMIT 1;

-- 若库中该节已有一条 slide_deck，本脚本不会插入；可将已有 deck PATCH 为 archived 后再执行，或改 WHERE NOT EXISTS 条件做测试。
