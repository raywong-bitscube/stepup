# StepUp v0.1 增量：CORS / Go+Nginx 部署、学生端稳定性与运维备忘

**发版批次**: `20260405#02`（与文件名 `20260405#02_...` 一致）  
**日期**: 2026-04-05  
**适用**: 在已具备 [20260405#01](./20260405%2301_DEPLOY_AND_UPGRADE.md)（或等效：多图、`extra_file_urls`、相关迁移与前端）的基础上，合入本批 **运维与前端行为** 相关改动；**无新增数据库 migration**。  
**关联**: [文档索引](../README.md)、[部署指南 §3.7 / §8](../core/deployment_guide_v0.1_260403.md)、[`backend/README.md`](../../backend/README.md)、[`docs/deploy/nginx_go_static_split_ports.conf.example`](../deploy/nginx_go_static_split_ports.conf.example)

---

## 1. 本轮变更摘要（评审 / 运维）

| 主题 | 说明 |
|------|------|
| **CORS 默认 `*`** | **`CORS_ALLOWED_ORIGINS`** 在 **未设置环境变量** 时，后端默认列表 **以 `*` 为首项**：对浏览器 **`http://` / `https://` Origin 回显 `Access-Control-Allow-Origin`**，便于 **公网/LAN IP + 分端口**（如 `http://IP:7011` 页面调 `http://IP:7012` API）而无需在每台机器写死白名单。**`docker-compose` 默认** 同样包含 **`*`**。 **公网生产** 务必 **覆盖** 该变量并 **去掉 `*`**，只保留可信 Origin。实现见 `backend/internal/middleware/cors.go`（列表中含 `*` 即启用回显逻辑）。 |
| **白名单做法（可选）** | 若不使用 `*`，则 **`CORS_ALLOWED_ORIGINS`** 须 **同时** 包含 **学生页** 与 **管理页** 的完整 Origin（协议+主机+端口、无路径）：例如 `http://203.0.113.50:7010,http://203.0.113.50:7011`；Compose 静态 **`:3000` / `:3001`** 则用 `http://IP:3000,http://IP:3001`。缺一会导致对应端 **`Failed to fetch` / 预检无 CORS 头**。 |
| **Go 本机 + Nginx** | 后端 **`go run` / 二进制** 监听例如 **`127.0.0.1:8080`**；Nginx **7010**、**7011** 托管 **`frontend-student`** / **`frontend-admin`** 静态资源，**7012** **`proxy_pass`** 到 Go。**关键**：浏览器 **OPTIONS 预检** 必须 **到达 Go**（或由 Nginx 返回 **完整** `Access-Control-*` 头）。**禁止**在 API 的 `server` 里对 `OPTIONS` 单独 `return 204` 却 **不写** CORS 头，否则表现与「后端未开 CORS」相同。 |
| **Nginx 示例** | 仓库 **[`docs/deploy/nginx_go_static_split_ports.conf.example`](../deploy/nginx_go_static_split_ports.conf.example)**：`7010` / `7011` 静态、`7012` 反代、`client_max_body_size` 与超时利于试卷上传；部署前修改其中 **`root` / `upstream`** 端口。 |
| **前端 API 根（LAN + Compose 端口）** | **`frontend-student` / `frontend-admin` `app.js`**：当页面 **不是** `localhost`/`127.0.0.1` 且端口为 **`3000` 或 `3001`** 时，API 基址指向 **同主机 `8080`**（与常见 **`BACKEND_PORT`** 一致），避免误把请求发到静态端口。非常规映射仍可用 **`?api=`**、**`localStorage`**、meta。 |
| **学生端：失败后死循环** | 登录后若 **`refreshPapers` 等失败**，原实现曾在 **`catch` 里再次 `mount()`**，导致 **整页反复重挂、控制台错误暴增**。现改为 **`startMainShell` + 代数丢弃过期回调**，试卷列表失败仅 **局部重试**，不再无限 **`mount`**。 |
| **管理端：`Failed to fetch` 提示** | 仪表盘等加载失败时追加 **CORS/API 地址** 简短说明（仍须后端或 Nginx 配置正确）。 |
| **文档** | [**`docs/core/deployment_guide_v0.1_260403.md`**](../core/deployment_guide_v0.1_260403.md) 新增 **§3.7**（Go + Nginx）、**§8 FAQ** 已更新 CORS / 预检说明；**`.env.example` / `backend/.env.example` / `docker-compose.yml` 注释** 与 **`backend/README.md`** 同步。 |

---

## 2. 数据库

- **本批次不要求** 执行新的 SQL migration。数据库升级仍以 [20260405#01 §2](./20260405%2301_DEPLOY_AND_UPGRADE.md) 及更早发版为准。

---

## 3. 升级 checklist（`go run` + Nginx，无 Docker）

1. **拉代码** → 重新编译/重启 Go 进程（确保包含 **`*` 默认 CORS** 或你已显式设置 `CORS_ALLOWED_ORIGINS`）。  
2. **环境变量**：若 shell/systemd 里曾导出 **不含 `*`** 的旧 `CORS_ALLOWED_ORIGINS`，要么 **取消导出** 使用代码默认，要么 **加上 `*`** 或 **显式写上** `http://<IP>:7010,http://<IP>:7011`（及按需 `:3000`,`:3001`）。  
3. **Nginx**：对照 **[示例配置](../deploy/nginx_go_static_split_ports.conf.example)**，确认 **7012** 对 **`/`** 的 **`proxy_pass`** 会转发 **OPTIONS**；更新静态站点 `root` 下的 **`frontend-admin`** / **`frontend-student`** 构建产物（含 **`app.js`**）。  
4. **`nginx -t` → reload**；重启 backend 后验证预检：对 `OPTIONS http://<API主机>:7012/api/v1/catalog`（带 `Origin: http://<页面>:7011`）应看到 **`Access-Control-Allow-Origin`** 等头来自后端或通过代理完整转发。

---

## 4. 验收建议

1. **管理端**（`7011`）：登录、仪表盘加载无 **CORS / Failed to fetch**。  
2. **学生端**（`7010`）：登录后 API 失败时 **页面不再无限刷新**；**重试** 按钮可恢复列表（backend 恢复后）。  
3. **`readyz`**、上传试卷（多图）、**CORS** 在 **仅 IP、无域名** 场景下可用（默认 `*`）或在你配置的严格白名单下可用。

---

## 5. 文档与基线

- **部署与排错**：[部署指南](../core/deployment_guide_v0.1_260403.md) **§3.6–3.7、§8**。  
- **本文件** 为 **20260405#02**，与 [20260405#01](./20260405%2301_DEPLOY_AND_UPGRADE.md) 同属 **20260405** 批次增量；前置功能说明仍以 **20260404#01 / #02** 与 **20260405#01** 为准。
