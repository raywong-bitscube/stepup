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
│ stage_id    │       │ updated_at  │       │ updated_at  │
│ status      │       └─────────────┘       └─────────────┘
│ created_at  │   
│ updated_at  │       ┌─────────────┐       ┌─────────────┐
└─────────────┘       │  AIModel    │       │   Prompt    │
       │              ├─────────────┤       ├─────────────┤
       │              │ id          │       │ id          │
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

### 9. Admin (管理员)

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

### 10. VerificationCode (验证码)

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

### 11. AuditLog (审计日志)

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
| PaperAnalysis | 快照 | `ai_model_snapshot` 记录本次分析所用模型信息（至少含 `name`、`url`；无外键；不落盘密钥） |
| AuditLog → User | N:1 | 日志关联操作用户 |

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
