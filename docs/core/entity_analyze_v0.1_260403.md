# 拾级 StepUp - Entity 分析 v0.1

**日期**: 2026-04-03  
**版本**: v0.1  
**来源需求**: `user_requirement_v0.1_260403.md`  
**工程文档入口**: [docs/README.md](../README.md)

---

## 实体关系图 (ER Diagram)

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   Student   │       │   Subject   │       │    Stage    │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id          │       │ id          │       │ id          │
│ phone       │       │ name        │       │ name        │
│ email       │       │ description │       │ description │
│ password    │       │ is_active   │       │ is_active   │
│ name        │       │ created_at  │       │ created_at  │
│ stage_id    │       │ updated_at  │       │ updated_at  │
│ status      │       └─────────────┘       └─────────────┘
│ created_at  │   
│ updated_at  │       ┌─────────────┐       ┌─────────────┐
└─────────────┘       │  AIModel    │       │   Prompt    │
       │              ├─────────────┤       ├─────────────┤
       │              │ id          │       │ id          │
       │              │ name        │       │ key         │
       │              │ url         │       │ description │
       │              │ model       │       │ content     │
       │              │ app_secret  │       │ is_active   │
       │              │ is_active   │       │ created_at  │
       │              │ created_at  │       │ updated_at  │
       │              │ updated_at  │       └─────────────┘
       │              └─────────────┘
       │
       ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  ExamPaper  │──────▶│ PaperAnalysis│─────▶│ImprovementPlan│
├─────────────┘       ├─────────────┤       ├─────────────┤
│ id          │       │ id          │       │ id          │
│ student_id  │       │ paper_id    │       │ paper_id    │
│ subject_id  │       │ ai_model_   │       │ plan_content│
│ file_url    │       │ snapshot    │       │ weak_points │
│ file_type   │       │ raw_content │       │ created_at  │
│ score       │       │ ai_response │       │ updated_at  │
│ exam_date   │       │ status      │       └─────────────┘
│ created_at  │       │ created_at  │
│ updated_at  │       │ updated_at  │
└─────────────┘       └─────────────┘
       │
       ▼
┌─────────────┐       ┌─────────────┐
│    Admin    │       │ Verification│
├─────────────┤       │    Code     │
│ id          │       ├─────────────┤
│ username    │       │ id          │
│ password    │       │ identifier  │
│ role        │       │ code        │
│ status      │       │ type        │
│ created_at  │       │ expires_at  │
│ updated_at  │       │ is_used     │
└─────────────┘       │ created_at  │
                      └─────────────┘

┌─────────────┐
│  AuditLog   │
├─────────────┤
│ id          │
│ user_id     │
│ user_type   │
│ action      │
│ entity_type │
│ entity_id   │
│ snapshot    │
│ ip_address  │
│ created_at  │
└─────────────┘

        教材目录（可选，用于考点/前端目录树；表 textbook_chapter / textbook_section）
┌─────────────┐       ┌───────────────────┐       ┌───────────────────┐
│  Textbook   │ 1:N   │ TextbookChapter   │ 1:N   │ TextbookSection   │
├─────────────┤       ├───────────────────┤       ├───────────────────┤
│ id          │       │ id                │       │ id                │
│ name        │       │ textbook_id       │       │ chapter_id        │
│ version     │       │ number            │       │ number            │
│ subject     │       │ title             │       │ title             │
│ category    │       │ full_title        │       │ full_title        │
│ subject_id  │       │ …audit            │       │ …audit            │
│ …audit      │       └───────────────────┘       └───────────────────┘
└─────────────┘
```

---

## 实体详细定义

### 1. Student (学生)

**说明**: 系统核心用户，通过手机号或邮箱登录，上传试卷并查看分析报告。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| phone | string | 手机号（可登录） |
| email | string | 邮箱（可登录） |
| password | string | 登录密码（bcrypt 加密） |
| name | string | 学生姓名 |
| stage_id | uint | 学生阶段（高中/初中/小学） |
| status | enum | pending/active/inactive |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: phone (unique), email (unique)

---

### 2. Subject (科目)

**说明**: 系统支持的学科科目，如物理、语文等。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 科目名称（物理/语文） |
| description | string | 科目描述 |
| is_active | bool | 是否启用 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

---

### 3. Stage (学生阶段)

**说明**: 学生所在的教育阶段。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 阶段名称（高中/初中/小学） |
| description | string | 阶段描述 |
| is_active | bool | 是否启用 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

---

### 4. ExamPaper (试卷)

**说明**: 学生上传的试卷，包含 PDF 或图片文件。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| student_id | uint | 所属学生 |
| subject_id | uint | 所属科目 |
| file_url | string | 文件存储路径 |
| file_type | enum | pdf/image |
| score | int | 试卷分数（可选） |
| exam_date | date | 考试日期 |
| created_at | time | 上传时间 |
| updated_at | time | 更新时间 |

**索引**: student_id, subject_id

---

### 5. PaperAnalysis (试卷分析结果)

**说明**: AI 对试卷的分析结果，包含原始内容和 AI 响应。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| paper_id | uint | 关联试卷 |
| ai_model_snapshot | json | 使用时的 AI 模型快照（至少包含 name、url，可扩展 provider/version） |
| raw_content | text | OCR 识别的原始内容 |
| ai_response | text | AI 分析结果（JSON） |
| status | enum | pending/processing/completed/failed |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: paper_id (unique)

---

### 6. ImprovementPlan (改进计划)

**说明**: AI 根据试卷分析生成的个性化学习改进计划。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| paper_id | uint | 关联试卷 |
| plan_content | text | 改进计划内容（JSON/Markdown） |
| weak_points | text | 薄弱知识点列表（JSON） |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: paper_id (unique)

---

### 7. AIModel (AI 模型配置)

**说明**: 浏览器应用使用的 AI 大模型配置，可配置多个，一个激活。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 模型名称 |
| url | string | API URL |
| model | string | 上游 OpenAI 兼容 API 的 chat **model** 名（如 `deepseek-chat`） |
| app_secret | string | API Key（Bearer），不落盘到 `paper_analysis` |
| is_active | bool | 是否激活（只能有一个 true） |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

---

### 8. Prompt (Prompt 配置)

**说明**: 系统各处使用的 AI Prompt 模板，通过 key 引用。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| key | string | 唯一标识符（如：paper_analysis_system） |
| description | string | 用途说明 |
| content | text | Prompt 内容 |
| is_active | bool | 是否启用 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: key (unique)

---

### 9. Admin (管理员)

**说明**: 后台管理系统登录用户表，用于后台账号认证和角色权限控制。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| username | string | 用户名 |
| password | string | 密码（bcrypt 加密） |
| role | enum | super_admin/admin/operator |
| status | enum | active/inactive |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: username (unique)

---

### 10. AdminSession (管理员登录会话)

**说明**: 后台管理登录态会话记录，用于 session 续期、失效和退出管理。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| admin_id | uint | 关联管理员 ID |
| session_token | string | 会话 token（建议存 hash） |
| expires_at | time | 过期时间 |
| last_seen_at | time | 最后访问时间 |
| ip_address | string | 登录/访问 IP |
| user_agent | string | 客户端 UA |
| status | enum | active/inactive |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: admin_id, session_token (unique), expires_at

---

### 11. VerificationCode (验证码)

**说明**: 登录/注册时发送的验证码。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| identifier | string | 手机号或邮箱 |
| code | string | 验证码（6 位数字） |
| type | enum | login/register |
| expires_at | time | 过期时间 |
| is_used | bool | 是否已使用 |
| created_at | time | 创建时间 |

**索引**: identifier, expires_at

---

### 12. Textbook（教材书目）

**说明**: 一本具体版本、具体册别的教科书（如「粤教版 2019 · 物理 必修 第一册」）。与系统 `subject` 通过 `subject_id` 可选关联，`subject` 字段保留展示用短名。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string(50) | 书籍名称 |
| version | string(50) | 教材版本 |
| subject | string(20) | 学科展示名 |
| category | string(20) | 必修 / 选择性必修 等 |
| subject_id | uint nullable | 关联 `subject.id`，ON DELETE SET NULL |
| status | tinyint | 1=启用 |
| remarks | string nullable | 备注 |
| created_at / created_by / updated_at / updated_by | | 与全库审计惯例一致 |
| is_deleted / deleted_at / deleted_by | | 软删除 |

**索引**: `(name, version)` 唯一；`subject_id`、`category`、`status`、`is_deleted`。

---

### 13. Chapter（章）

**表名**: `textbook_chapter`。

**说明**: 某本教材下的一章；`title` 不含「第一章」前缀，`full_title` 可选存完整展示文案。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| textbook_id | uint | 外键 → textbook |
| number | uint | 章序号（建议从 1 起；可与同书下其他章重复，由业务约束） |
| title | string(100) | 短标题 |
| full_title | string(150) nullable | 如「第一章 运动的描述」 |
| status | tinyint | 1=启用，0=停用（管理端用状态代替对目录行的软删操作） |
| 审计与软删除 | | 同 textbook |

**索引**: `idx_textbook_chapter_textbook_number (textbook_id, number)` 非唯一，仅查询排序。

---

### 14. Section（节）

**表名**: `textbook_section`。

**说明**: 某一章下的一节；序号与标题规则同章。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| chapter_id | uint | 外键 → `textbook_chapter` |
| number | uint | 节序号（可与同章下其他节重复，由业务约束） |
| title | string(100) | 短标题 |
| full_title | string(150) nullable | 如「第一节 质点 参考系 时间」 |
| status | tinyint | 1=启用，0=停用 |
| 审计与软删除 | | 同 textbook |

**索引**: `idx_textbook_section_chapter_number (chapter_id, number)` 非唯一。

---

### 15. AuditLog (审计日志)

**说明**: 记录所有用户操作，支持数据变更快照。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 操作用户 ID |
| user_type | enum | student/admin |
| action | enum | login/create/update/delete |
| entity_type | string | 被操作实体类型（如：Student/ExamPaper） |
| entity_id | uint | 被操作实体 ID |
| snapshot | json | 操作前数据快照（update/delete 时） |
| ip_address | string | 操作 IP |
| created_at | time | 操作时间 |

**索引**: user_id, entity_type, created_at

---

## 实体关系说明

| 关系 | 类型 | 说明 |
|------|------|------|
| Student → Stage | N:1 | 学生属于某个阶段 |
| Student → ExamPaper | 1:N | 学生可上传多份试卷 |
| ExamPaper → Subject | N:1 | 试卷属于某个科目 |
| ExamPaper → PaperAnalysis | 1:1 | 每份试卷对应一个分析 |
| ExamPaper → ImprovementPlan | 1:1 | 每份试卷对应一个改进计划 |
| Admin → AdminSession | 1:N | 管理员可有多个登录会话 |
| PaperAnalysis | 快照 | `ai_model_snapshot` 记录本次分析所用模型信息（至少含 `name`、`url`；无外键；不落盘密钥） |
| AuditLog → User | N:1 | 日志关联操作用户 |
| Textbook → Subject | N:1 | `subject_id` 可空；展示字段 `subject` 与科目表可并存 |
| Textbook → Chapter | 1:N | 一本书下多章 |
| Chapter → Section | 1:N | 一章下多节 |

---

## 待确认事项

1. **家长模块**: v0.1 暂不实现，但需预留 `parent_id` 字段或单独建表
2. **微信小程序**: v0.1 暂不实现，实体设计需考虑后续扩展
3. **文件存储**: `file_url` 字段需确定存储方案（本地/云存储）
4. **AI 响应格式**: `ai_response` 和 `plan_content` 的 JSON 结构需进一步定义
5. **分析模型快照**: v0.1 不做 `PaperAnalysis → AIModel` 外键，仅在分析结果中保留 `ai_model_snapshot`（至少包含 `name`、`url`）
6. **分析多版本范围**: v0.1 上传一次只生成一份分析结果（`PaperAnalysis/ImprovementPlan` 以 `paper_id` 做 1:1 并保持 `unique`）

---

*本文档基于 `user_requirement_v0.1_260403.md` 分析生成，后续可能根据开发需求调整。*
