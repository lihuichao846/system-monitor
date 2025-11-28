# System Monitor

面向服务器 / 工作站的实时监控仪表盘。后端使用 **Go + Gin + gopsutil** 采集系统指标，前端采用 **Vue 3 + Vite + Element Plus + ECharts** 渲染大屏图表，支持 Docker Compose 一键部署。

## 功能亮点

- **实时采集**：每秒采集 CPU、内存、磁盘 I/O、网络流量、系统信息等关键指标，采集逻辑位于 `backend/metrics`（@backend/metrics/collector.go#17-299）。
- **告警日志**：CPU/内存超阈值时生成告警，保留最近 200 条历史。
- **网络流量记录**：带宽超过约 1 MB/s 即记录当时各网卡的上下行速率。
- **前端大屏**：Vue 3 + Element Plus + ECharts 构建的动态仪表盘（@frontend/system-monitor-web/src/App.vue#250-340）。

## 项目结构

```
.
├─backend/                 # Go 后端（Gin + gopsutil）
│  ├─main.go               # 入口：提供 /api/dashboard（@backend/main.go#1-22）
│  ├─metrics/              # 指标采集逻辑
│  └─Dockerfile            # 多阶段构建
├─frontend/
│  └─system-monitor-web/   # Vue 3 + Vite 前端
│     ├─src/               # 前端源码
│     ├─Dockerfile         # Node 构建 + Nginx 托管
│     └─nginx.conf         # 将 /api/* 代理至 backend（@frontend/system-monitor-web/nginx.conf#1-18）
└─docker-compose.yml       # 前后端一键编排
```

## 本地开发

### 环境要求
- Go 1.23+
- Node.js 18+（含 npm）
- 可选：Docker 24+、Docker Compose V2

### 启动后端
```bash
cd backend
go mod download
go run .            # 默认监听 :8080
```

### 启动前端
```bash
cd frontend/system-monitor-web
npm install
npm run dev        # 默认 http://localhost:5173
```

开发态下，Vite 在 `vite.config.js` 中配置了 `/api` 代理到 `http://localhost:8080`（@frontend/system-monitor-web/vite.config.js#1-15），可直接联调。

## Docker / Compose 部署

根目录提供 `docker-compose.yml`，结合现有 Dockerfile 可一键构建、运行：

```bash
# 构建镜像
docker compose build

# 后台启动
docker compose up -d

# 查看日志
docker compose logs -f

# 停止并清理
docker compose down
```

- `backend` 镜像由 `backend/Dockerfile` 构建，暴露 `8080`（@backend/Dockerfile#1-30）。
- `frontend` 镜像由 `frontend/system-monitor-web/Dockerfile` 构建，Nginx 监听 `80`，并将 `/api/*` 转发到 `backend:8080`。
- Compose V2 会提示 `version` 字段已弃用，如无需要可删除该字段，不影响运行。

访问 `http://<服务器IP>/` 即可查看仪表盘（默认映射宿主 80 端口）。

## API 摘要

| 方法 | 路径             | 描述         |
| ---- | ---------------- | ------------ |
| GET  | `/api/dashboard` | 返回实时指标 |

响应结构定义在 `backend/metrics/collector.go` 的 `DashboardData`，包含 CPU、内存、磁盘、网络、告警、网络日志等数据。

## 参考

- [Gin](https://gin-gonic.com/)
- [gopsutil](https://github.com/shirou/gopsutil)
- [Vue 3 / Vite](https://vitejs.dev/)
- [ECharts](https://echarts.apache.org/)
