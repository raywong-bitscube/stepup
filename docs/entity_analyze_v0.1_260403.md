# 拾级 StepUp - Entity 分析 v0.1

**日期**: 2026-04-03  
**版本**: v0.1  
**来源需求**: `user_requirement_v0.1_260403.md`

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
│ stage_id    │──┐    │ updated_at  │       │ updated_at  │
│ status      │  │    └─────────────┘       └─────────────┘
│ created_at  │  │
│ updated_at  │  │    ┌─────────────┐       ┌─────────────┐
└─────────────┘  │    │  AIModel    │       │   Prompt    │
       │         │    ├─────────────┤       ├─────────────┤
       │         └───▶│ id          │       │ id          │
       │              │ name        │       │ key         │
       │              │ url         │       │ description │
       │              │ app_key     │       │ content     │
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
│ subject_id  │       │ raw_content │       │ plan_content│
│ file_url    │       │ ai_response │       │ created_at  │
│ file_type   │       │ status      │       │ updated_at  │
│ score       │       │ created_at  │       └─────────────┘
│ exam_date   │       │ updated_at  │
│ created_at  │       └─────────────┘
│ updated_at  │
└─────────────┘
       │
       ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  Question   │──────▶│KnowledgePoint│      │  Practice   │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id          │       │ id          │       │ id          │
│ subject_id  │       │ name        │       │ student_id  │
│ content     │       │ subject_id  │       │ question_id │
│ answer      │       │ description │       │ answer      │
│ difficulty  │       │ created_at  │       │ is_correct  │
│ knowledge_  │       │ updated_at  │       │ score       │
│   point_id  │       └─────────────┘       │ created_at  │
│ created_at  │                              │ updated_at  │
│ updated_at  │                              └─────────────┘
└─────────────┘

┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    Admin    │       │ Verification│       │  AuditLog   │
├─────────────┤       │    Code     │       ├─────────────┤
│ id          │       ├─────────────┤       │ id          │
│ username    │       │ id          │       │ user_id     │
│ password    │       │ identifier  │       │ user_type   │
│ role        │       │ code        │       │ action      │
│ status      │       │ type        │       │ entity_type │
│ created_at  │       │ expires_at  │       │ entity_id   │
│ updated_at  │       │ is_used     │       │ snapshot    │
└─────────────┘       │ created_at  │       │ ip_address  │
                      └─────────────┘       │ created_at  │
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
| app_key | string | API Key |
| app_secret | string | API Secret |
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

### 9. Question (题目)

**说明**: 题库中的题目，用于推送针对性练习。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| subject_id | uint | 所属科目 |
| content | text | 题目内容 |
| answer | text | 标准答案 |
| difficulty | enum | easy/medium/hard |
| knowledge_point_id | uint | 关联知识点 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: subject_id, knowledge_point_id

---

### 10. KnowledgePoint (知识点)

**说明**: 学科知识点，用于定位学生薄弱环节。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 知识点名称 |
| subject_id | uint | 所属科目 |
| description | string | 知识点描述 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

**索引**: subject_id

---

### 11. Practice (练习记录)

**说明**: 学生完成的练习记录，包含答案和批改结果。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| student_id | uint | 所属学生 |
| question_id | uint | 题目 ID |
| answer | text | 学生答案 |
| is_correct | bool | 是否正确 |
| score | int | 得分 |
| created_at | time | 答题时间 |
| updated_at | time | 更新时间 |

**索引**: student_id, question_id

---

### 12. Admin (管理员)

**说明**: 后台管理系统用户。

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

### 13. VerificationCode (验证码)

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

### 14. AuditLog (审计日志)

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
| PaperAnalysis → AIModel | N:1 | 分析使用某个 AI 模型 |
| Question → Subject | N:1 | 题目属于某个科目 |
| Question → KnowledgePoint | N:1 | 题目关联某个知识点 |
| Practice → Student | N:1 | 练习记录属于某个学生 |
| Practice → Question | N:1 | 练习记录关联某个题目 |
| AuditLog → User | N:1 | 日志关联操作用户 |

---

## 待确认事项

1. **家长模块**: v0.1 暂不实现，但需预留 `parent_id` 字段或单独建表
2. **微信小程序**: v0.1 暂不实现，实体设计需考虑后续扩展
3. **文件存储**: `file_url` 字段需确定存储方案（本地/云存储）
4. **AI 响应格式**: `ai_response` 和 `plan_content` 的 JSON 结构需进一步定义

---

*本文档基于 `user_requirement_v0.1_260403.md` 分析生成，后续可能根据开发需求调整。*
