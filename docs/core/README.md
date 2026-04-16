# 核心文档（v0.1 基线）

本目录收录 **需求、设计、API、部署与运维参考** 等**长期有效**的说明，与具体某次发版的增量变更分开（增量见 **[`docs/releases/`](../releases/)**）。

| 文件 | 说明 |
|------|------|
| [user_requirement_v0.1_260403.md](./user_requirement_v0.1_260403.md) | 需求 |
| [entity_analyze_v0.1_260403.md](./entity_analyze_v0.1_260403.md) | 实体与 ER |
| [feature_design_v0.1_260403.md](./feature_design_v0.1_260403.md) | 功能清单与实现要点 |
| [architecture_deployment_v0.1_260403.md](./architecture_deployment_v0.1_260403.md) | 架构与组件 |
| [deployment_guide_v0.1_260403.md](./deployment_guide_v0.1_260403.md) | 部署指南（Compose / 变量 / 初始化） |
| [api_v0.1_260403.md](./api_v0.1_260403.md) | API 文档 |
| [system_runtime_guide_v0.1_260403.md](./system_runtime_guide_v0.1_260403.md) | 运行时与代码结构 |
| [ai_model_log_v0.1_260403.md](./ai_model_log_v0.1_260403.md) | AI 调用日志需求与设计 |
| [feature_essay_outline_v0.1_260403.md](./feature_essay_outline_v0.1_260403.md) | 作文提纲练习（学生端）需求与数据设计 |
| [slide_deck_design_v0.1_260403.md](./slide_deck_design_v0.1_260403.md) | 章节幻灯片 Slide Deck：JSON schema、模版库、AI Prompt、SlideRenderer |

## 文件系统存储（UPLOAD_DIR）与 Nginx 配置

当系统使用本地文件系统保存上传文件时（`UPLOAD_DIR`），后端会把文件 URL 记录为 `/uploads/...`，并在应用内映射该路径。

- 未配置 `UPLOAD_DIR` 时，默认目录是 `data/uploads`（相对后端进程启动目录）。
- 建议线上显式配置绝对路径，例如 `/var/lib/stepup/uploads`，避免工作目录变化导致文件找不到。

如果前面有 Nginx（例如访问端口是 Nginx，而后端在另一个端口），需要确保 `/uploads/` 可访问，常见两种方式：

1) 由 Nginx 转发给后端应用（推荐，配置简单）：

```nginx
location /uploads/ {
    proxy_pass http://127.0.0.1:7012;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

2) 由 Nginx 直接读取磁盘目录：

```nginx
location /uploads/ {
    alias /var/lib/stepup/uploads/;
    try_files $uri =404;
}
```

注意：`alias` 方案中的目录应与 `UPLOAD_DIR` 对应，且通常需要以 `/` 结尾。

## 试卷库（exam_source）大题分组

整卷识别与建卷支持 **大题（`exam_source_question_group`）**：`system_kind` 为系统题型（如 `single_choice` / `multi_choice` 等），`title_label` / `description_text` 存卷面上该大题标题与完整说明；`exam_source_question.group_id` 指向所属大题。`GET /api/v1/admin/exam-source/papers/{paperId}` 返回 `question_groups` 与带 `group_id` 的题目列表；上传分析接口会返回识别到的 `groups` 与题目上的 `group_order`。

返回 [**文档总索引**](../README.md)。
