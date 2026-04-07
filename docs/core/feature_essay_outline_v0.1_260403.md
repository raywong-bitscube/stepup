# 作文提纲练习（学生端）— 需求与设计 v0.1

**日期**: 2026-04-06  
**入口**: 学生前端 → 科目「语文」→「作文提纲练习」  
**后端**: `POST` / `GET /api/v1/student/essay-outline/*`（需登录与 MySQL；列表与详情见 `api_v0.1_260403.md` §5.5.4）

---

## 1. 产品目标

为高中学生提供 **作文提纲专项练习**：支持 AI 按文体/命题方式生成题目、自拟文本或图片 OCR 录题；学生提交提纲后由 AI 给出 **结构化点评**（总体评价、三维星级、分条建议），帮助掌握不同文体下的提纲设计。

---

## 2. 核心流程

1. 进入功能页，选择 **分类选题** 或 **自定义题目**。
2. **分类选题**：选择文体（记叙文 / 议论文 / 散文 / 应用文 / 说明文）与命题方式（命题作文 / 材料作文 / 话题作文 / 任务驱动型作文），点击「生成题目」，调用 AI 返回题目与标签。
3. **自定义题目**：在「题目正文」中直接输入或粘贴；或上传 **JPG/PNG**，调用多模态模型 OCR 提取题目文本。
4. 在「我的提纲」多行文本框中撰写提纲，点击「提交点评」。
5. 服务端调用点评 AI，解析为结构化 JSON，写入 **`essay_outline_practice`**，并将 `review` 返回前端展示（总体评价、1–5 星评分、建议列表）。
6. 学生端该功能页底部展示 **练习记录** 列表（按时间倒序），点击可查看题目、提纲与完整点评详情；进入「作文提纲练习」时 **不展示** 与试卷上传相关的「我的试卷」区块，避免与提纲练习混淆。

---

## 3. 数据设计

### 3.1 表 `essay_outline_practice`

| 字段 | 说明 |
|------|------|
| student_id | 学生 |
| subject_id | 关联「语文」科目 id，可空 |
| topic_text / topic_label | 题目正文与展示标签（如 `议论文 · 材料作文` 或 `自定义`） |
| topic_source | `ai_category` \| `custom_text` \| `ocr_image` |
| genre / task_type | 分类选题时记录；自定义/OCR 为空 |
| outline_text | 学生提纲 |
| review_json | 解析后的点评：summary、stars(match/structure/material)、suggestions、highlights |
| raw_review_response | 模型原始文本（便于溯源） |

迁移：`db/migrations/2026-04-06#01_essay_outline_practice.sql`  
基线：`db/schema/mysql_schema_v0.1_260403.sql`（`essay_outline_practice` 段）

### 3.2 Prompt 模板（`prompt_template`）

| key | 用途 |
|-----|------|
| `essay_outline_generate_topic` | 出题；占位符 `%genre` `%task_type` |
| `essay_outline_review` | 点评；`%topic_text` `%outline_text` |
| `essay_outline_ocr_topic` | 识图题目（vision user 文本） |

---

## 4. AI 与可观测性

- 使用与试卷分析相同的 **HTTP Chat Completions** 适配器（激活 `ai_model` / `ANALYSIS_ADAPTER=http`）。
- **`ai_call_log.action`**：`essay_outline_generate_topic`、`essay_outline_ocr_topic`、`essay_outline_review`（无 `paper_id`，含 `student_id`）。
- 未配置上游时回退 **mock**，便于无网联调。

---

## 5. 解析约定

- **出题**：模型输出推荐 `题目全文 | 文体/命题标签`（`|` 分隔）。
- **点评**：`总体评价|维度评分行（含 匹配度X星/结构X星/素材X星）|详细建议（分号或换行分条）`。
- 服务端容错解析星级与建议条目，写入 `review_json`；异常时字段可能部分缺省。

---

## 6. 前端交互要点

- **分类 / 自定义** Tab 切换，无需整页刷新即可切换区块。
- 题目区、标签可编辑；提交前同步 DOM 到 `state`。
- 点评区：卡片展示总体评价、三维度星级、建议列表。

---

## 7. 部署注意

1. 执行迁移 SQL，确保存在三条 Prompt（或通过管理端 Prompt 页维护同 key）。
2. 确认学生库中存在名为 **「语文」** 的科目（`subject_id` 可自动关联；缺失则 `subject_id` 为空）。
3. 与既有 CORS、学生鉴权配置一致。
