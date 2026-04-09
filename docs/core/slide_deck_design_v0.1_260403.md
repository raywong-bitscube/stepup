# 章节幻灯片（Slide Deck）需求与设计 v0.1

**日期**: 2026-04-03  
**版本**: `schemaVersion` **1**（JSON 结构与本文同步）  
**状态**: 设计已定稿；**初版已实现**：`slide_deck` 表、管理端/学生端 API、`frontend-student` 的 `slideRenderer.js` + `slides.css` + 物理科目「章节互动课件」入口（迁移 **`db/migrations/2026-04-10#01_slide_deck.sql`**，示例种子 **`db/seed/slide_deck_sample_yuedu_physics_ch2_sec1.sql`**）。管理端 JSON 编辑 UI 可后续再做。

**关联文档**: [功能清单与设计](./feature_design_v0.1_260403.md)、[API](./api_v0.1_260403.md)、教材 `textbook` → `chapter` → `section`

---

## 1. 目标与场景

### 1.1 业务目标

- 按**教材节（推荐首选挂载点）**或章，配置**一套或多套**幻灯片 JSON；**同一挂载点仅允许一套为当前生效（active）**。
- 学生端：**主区域**用统一组件 **SlideRenderer** 解析 JSON，支持**上一页 / 下一页**、**步进揭示**（点击「下一步」逐步显示内容）、**单选/多选**等互动。
- **右侧**为与系统 AI 的对话区；学生在学习过程中可持续提问。**AI 侧应能获得**当前 `deckId` / `slideId`、**当前步**、**已作答情况**等结构化摘要（由前端或 BFF 组装上下文，具体接口实现另文补充）。

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| 模版 + 组件流 | **不写绝对坐标**；用 `layoutTemplate` 选版式，用 `role` 绑定语义槽位；**前端为每个 role 写死 CSS**（或 CSS 变量 + 主题）。 |
| 扁平 `elements` | 单张幻灯片内所有块放在 **`elements[]`**；用 **`step`（整数，从 1 起）** 控制「第几次点击后出现」。同一步多条元素 **同时出现**。 |
| 主题与版式正交 | `meta.theme` 管色板/字体（如 `dark-physics`）；`layoutTemplate` 管网格区域；二者独立。 |
| 可演进 | 顶层保留 **`schemaVersion`**；服务端或构建 pipeline 可对 `layoutTemplate` × `role` × `type` 做校验。 |

### 1.3 非目标（v0.1 实现阶段可剔除）

- PPT 导入、时间轴动画曲线编辑器。
- 任意拖拽排版编辑器。

---

## 2. JSON 结构（schemaVersion 1）

### 2.1 顶层

```json
{
  "schemaVersion": 1,
  "meta": {
    "title": " deck 级标题（可选，可与业务侧 deck.title 重复）",
    "theme": "dark-physics"
  },
  "slides": [ ]
}
```

- **`schemaVersion`**：必填，当前仅 **1**。解析器遇到未知大版本应拒绝或走降级策略。
- **`meta.theme`**：主题 id 字符串；前端维护 **theme → CSS 变量/class** 映射表；未识别时可回退默认主题。
- **`slides`**：按播放顺序排列；学生端「下一页」即 `slideIndex + 1`。

### 2.2 单张幻灯片 `slides[]`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 幻灯片稳定 id（日志、AI 上下文引用）；建议 `slide-1` 或语义化 slug。 |
| `layoutTemplate` | string | **模版库枚举**，见第 3 节。 |
| `elements` | array | 扁平列表，见 2.3。 |

### 2.3 元素 `elements[]`

公共字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | `text` \| `latex` \| `image` \| `question`（后续可扩展 `divider` 等）。 |
| `role` | string | **语义槽位**，须落在当前 `layoutTemplate` 允许的集合内；前端按 `role` 选用组件与 class。 |
| `step` | integer | **≥ 1**。揭示序号；`currentStep` 为该页状态时，渲染所有 `step <= currentStep` 的元素。 |

**按 `type` 的专用字段（v1）**

| type | 字段 | 说明 |
|------|------|------|
| `text` | `content` | 正文字符串；渲染侧约定支持 **Markdown 子集**（标题/加粗/列表/换行），避免任意 HTML。 |
| `latex` | `content` | LaTeX 源码字符串；`display` 可选：`block`（默认）\| `inline`。 |
| `image` | `src`（必填）、`alt`、`caption` | `alt` 建议必填以利于无障碍；`caption` 小号字显示在图下。 |
| `question` | `mode`、`data` | 见下表。 |

**`question`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `mode` | string | **`single`** \| **`multi`**，必填。 |
| `data` | object | `text`（题干，可与 Markdown 子集一致）、`options`：`[{ "id": "A", "text": "…" }, …]`。选项 `id` 须稳定，便于 AI 与用户态引用。 |

**答案与解析（存 JSON 策略）**

- **管理端 / 作者工具**：可在同一 `question` 下增加 **`answer`**（如新窗口扩展：`correctOptionIds: ["A"]`、`explanation`），用于自动判分与后台预览。
- **学生端接口**：由后端决定是否把 `answer` 下发给浏览器；未下发时由服务端提交答案接口判分，或仅收集选项用于 AI 上下文。

完整示例见 **第 5 节**。

### 2.4 步进语义（与 AI 「下一步」）

- 每页维护 **`currentStep`**（整数，建议初始 **0** 表示尚未揭示任何 `step`，或初始 **1** 若首屏就要显示 `step <= 1` —— **产品定议：推荐初始 0，第一次「下一步」后变为 1**，这样 `step: 1` 为首屏内容与 PPT 习惯一致；若希望一进页就显示 `step: 1`，可在进入页时把 `currentStep` 设为 1。文档实现时**择一写死并在 SlideRenderer 中单处实现**）。
- **推荐约定（本文采用）**：进入幻灯片时 `currentStep = 1`，即默认展示所有 `step <= 1` 的元素；每次「下一步」`currentStep++`，直到 `currentStep >= maxStepOnSlide`，再点「下一步」可视为切到下一页（或按钮禁用，由产品定）。
- AI 「说下一步」若对接语音/指令：**等价于触发一次与 UI 相同的 `currentStep++` 或 `nextSlide`**，由客户端统一调度，避免双逻辑。

---

## 3. 模版库（layoutTemplate）— 先写死 8 个

以下为 **v1 固定枚举**；前端为每个模版提供 **根级布局 class**（如 `slide-tpl-split-left-right`）及 **子区域 Grid/Flex**。**`role` 必须在对应模版白名单内**，否则构建/保存时校验失败或降级为 `body` 并告警。

| layoutTemplate | 用途简述 | 布局思路（实现提示） | 允许 `role`（建议白名单） |
|----------------|----------|----------------------|---------------------------|
| `cover-image` | 封面，大标题 + 副标题，可衬底图 | 全屏单列，`min-height` 满屏；背景可由 `theme` 或首个 `type:image` + `role:background` 铺底 | `background`（可选）, `title`, `subtitle`, `tagline` |
| `title-body` | 标题 + 正文讲义 | 上区标题、下区可滚动正文 | `title`, `body`, `callout` |
| `formula-focus` | 公式为主，配短注 | 垂直居中栈：标题可选、公式大块、注释 | `title`, `main-formula`, `annotation` |
| `split-left-right` | 左公式/左图 + 右文，或互换 | CSS Grid：`grid-template-columns: 1fr 1fr`；首行可 `title` 跨两列 | `title`, `main-formula`, `illustration`, `body` |
| `split-top-bottom` | 上图下文 | 上行占高比例放图，下行正文 | `title`, `illustration`, `body` |
| `quiz-center` | 习题页 | 居中卡片，宽度 `min(640px, 100%)` | `title`（可选）, `main-content`（**须**为 `type: question`） |
| `bullet-steps` | 分步要点列表 | 标题 + 列表区；每项可用不同 `step` 逐步出现 | `title`, `bullet`（`type: text`） |
| `two-column-text` | 双栏对比/并列概念 | 两列等宽边距；标题跨列 | `title`, `column-left`, `column-right` |

**`type` 与 `role` 的约束（强建议）**

- `quiz-center` 中 **`main-content` 必须是 `question`**。
- `formula-focus` 的 **`main-formula` 必须为 `latex`**。
- `cover-image` 的 **`background` 若为图片则用 `type: image`**（仅作底图样式由 `role: background` 控制 object-fit cover）。
- 其余组合以「模版说明」为准，校验器可按表维护 **允许矩阵**。

---

## 4. SlideRenderer 组件（前端职责）

**技术栈**：与主站一致（React 或 Vue）；本文仅定义**职责与数据契约**。

### 4.1 输入（props）

- **`deck`**: 符合本文 JSON 的对象（已由接口拉取并 `JSON.parse`）。
- **`slideIndex`**: 当前页下标。
- **`currentStep`**: 当前页步进值（见 2.4）。
- **回调**: `onSlideChange(index)`、`onStepChange(step)`、`onQuestionAnswer({ slideId, elementRef, mode, selectedIds })` 等，便于父级同步到 AI 上下文或提交服务器。

### 4.2 渲染流程

1. 校验 `deck.schemaVersion === 1`（或兼容范围）；失败则展示降级 UI。
2. 取 `slides[slideIndex]`，读取 `layoutTemplate`，挂载对应 **模版根组件**（或单一组件 + `switch(template)`）。
3. 计算 `maxStep = max(elements[].step)`（空数组则为 0）。
4. 过滤 `elements.filter(el => el.step <= currentStep)`，按 `type` 分发子组件；子组件根据 **`role`** 应用 class（如 `data-role="title"` + `.slide-role-title`）。
5. **控制条**：上一页 / 下一步 / 下一页；在 `currentStep < maxStep` 时「下一步」步进；否则「下一步」可切换到下一页（若产品如此约定）。

### 4.3 可访问性与键盘

- 建议：`→` / `Space` 下一步，`←` 上一步（可选）；焦点管理保证题目选项可键盘操作。

---

## 5. 完整示例（schemaVersion 1）

```json
{
  "schemaVersion": 1,
  "meta": {
    "title": "匀变速直线运动",
    "theme": "dark-physics"
  },
  "slides": [
    {
      "id": "slide-1",
      "layoutTemplate": "cover-image",
      "elements": [
        {
          "type": "text",
          "role": "title",
          "content": "匀变速直线运动",
          "step": 1
        },
        {
          "type": "text",
          "role": "subtitle",
          "content": "探索速度与时间的关系",
          "step": 2
        }
      ]
    },
    {
      "id": "slide-2",
      "layoutTemplate": "split-left-right",
      "elements": [
        {
          "type": "latex",
          "role": "main-formula",
          "content": "v = v_0 + at",
          "step": 1
        },
        {
          "type": "image",
          "role": "illustration",
          "src": "/static/slides/ch1/v-t-graph.png",
          "alt": "v-t 图像示意",
          "caption": "v-t 图像斜率代表加速度",
          "step": 2
        }
      ]
    },
    {
      "id": "slide-3",
      "layoutTemplate": "quiz-center",
      "elements": [
        {
          "type": "question",
          "role": "main-content",
          "mode": "single",
          "data": {
            "text": "关于加速度，下列说法正确的是？",
            "options": [
              { "id": "A", "text": "加速度是矢量" },
              { "id": "B", "text": "速度为 0 则加速度为 0" }
            ]
          },
          "step": 1
        }
      ]
    }
  ]
}
```

---

## 6. 数据表（存库要点，实现阶段补齐 DDL）

与前期方案一致，建议表名 **`slide_deck`**（或项目命名规范下的等价名）：

- 挂载：`section_id`（优先）与/或 `chapter_id`（二者业务规则在迁移注释中写清）。
- `title`、`status`（`draft` / `active` / `archived`）、**同一挂载点仅一条 `active`**（唯一约束或事务内切换）。
- `schema_version`（整数，与 JSON `schemaVersion` 对齐）。
- `content`：**JSON** 列或 `TEXT` 存序列化字符串。
- 审计字段与软删除策略与项目统一规范一致。

接口与 OpenAPI 在 **`api_v0.1_260403.md`** 后续迭代中追加；本文不展开 HTTP 细节。

---

## 7. AI Prompt：模版定义（供系统提示词 / 工具说明粘贴）

以下内容可整体放入 **系统提示** 或 **RAG 片段**，约束模型只输出合法 JSON，且 **`layoutTemplate` 必须从给定枚举中选取**。

```markdown
你是「物理课件幻灯片」生成器。只输出一个合法 JSON 对象，不要 Markdown 代码围栏外的解释文字。

顶层必须包含：
- "schemaVersion": 1
- "meta": { "title": string, "theme": "dark-physics" | "light-default" }
- "slides": array

每张幻灯片对象必须包含：
- "id": 唯一字符串
- "layoutTemplate": 必须是以下之一：
  cover-image | title-body | formula-focus | split-left-right | split-top-bottom | quiz-center | bullet-steps | two-column-text
- "elements": 数组；每项必须含 "type", "role", "step"（从 1 起的整数）
- type 只能是：text | latex | image | question

元素字段：
- text: "content" (Markdown 子集：粗体、列表、换行)
- latex: "content"；可选 "display": "block" | "inline"
- image: "src", 可选 "alt", "caption"
- question: 必须 "mode": "single" | "multi"；"data": { "text", "options": [{ "id", "text" }] }

layoutTemplate 与 role 白名单（违反则改到合法组合）：
- cover-image: background(optional, image), title, subtitle, tagline
- title-body: title, body, callout
- formula-focus: title, main-formula, annotation  （main-formula 须为 latex）
- split-left-right: title, main-formula, illustration, body
- split-top-bottom: title, illustration, body
- quiz-center: title(optional), main-content（须为 question）
- bullet-steps: title, bullet（text）
- two-column-text: title, column-left, column-right

编写技巧：先选模版，再填 elements；同一 step 的多个元素会同时出现；不同 step 用于分步动画。
```

（按实际主题枚举扩展 `meta.theme` 行即可。）

---

## 8. 校验与测试清单（开发自检）

- [ ] 未知 `layoutTemplate` / `role` / `type` 行为明确（报错或降级）。
- [ ] `schemaVersion` 非 1 的拒绝或提示升级。
- [ ] `quiz-center` + `main-content` 非 `question` 时保存失败或 CI 失败。
- [ ] 8 套模版在 `dark-physics` 与 `light-default` 下视觉可读。
- [ ] `currentStep` 边界与「下一页」衔接符合 2.4 约定。
- [ ] 题目选项变更时，向 AI 提交的上下文包含 `slideId` 与已选 `option id`。

---

## 9. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-04-03 | 初版：模版 + role + 扁平 elements + step；8 模版；AI Prompt 附录；SlideRenderer 职责 |

---

*实现完成后，请将实际 `meta.theme` 列表、管理端 API 路径与本文件交叉引用更新。*
