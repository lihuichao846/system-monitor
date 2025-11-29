# System Monitor & Security Awareness Dashboard

面向服务器 / 工作站的实时监控与网络安全态势感知仪表盘。后端使用 **Go + Gin + gopsutil** 采集系统指标及网络连接信息，前端采用 **Vue 3 + Vite + Element Plus + ECharts** 渲染动态大屏，支持 Docker Compose 一键部署。

## 功能亮点

- **全方位实时采集**：
  - 基础资源：CPU、内存、磁盘 I/O、文件系统使用率。
  - 网络流量：实时上下行速率、总流量统计。
  - **安全态势**：
    - **深度审计**：实时关联活跃连接的 **进程名称**、**远程 IP**、**地理位置** (国家/城市) 及端口信息。
    - **威胁感知**：流量突发或定时触发连接快照，有效捕获潜在的恶意通信（如 C2 心跳）。
    - **可视化热力图**：通过 **GeoIP** 绘制全球连接热力分布，直观定位威胁来源。
  - 进程监控：Top CPU/Memory 进程实时追踪。
- **实时数据流 (SSE)**：采用 Server-Sent Events (SSE) 技术，实现秒级数据推送，告别传统轮询，降低延迟。
- **智能告警工作台**：
  - 可配置的 CPU/内存/磁盘 阈值。
  - 告警历史记录与分页查询。
  - **网络审计日志**：详细记录异常流量时刻的连接快照。
- **可视化大屏**：
  - 动态仪表盘：CPU/内存/磁盘 仪表盘（支持动画与渐变效果）。
  - 趋势图表：ECharts 绘制流量趋势、负载变化。
  - **地理视图**：全球地图热力分布，直观展示外部连接活跃区域。

## 项目结构

```
.
├─backend/                 # Go 后端（Gin + gopsutil + GeoIP2）
│  ├─main.go               # 入口：提供 REST API 及 SSE 流
│  ├─metrics/              # 指标采集与告警逻辑
│  ├─GeoLite2-City.mmdb    # (需自行下载) MaxMind GeoIP 数据库
│  └─Dockerfile            # 多阶段构建
├─frontend/
│  └─system-monitor-web/   # Vue 3 + Vite 前端
│     ├─src/               # 前端源码 (App.vue, components, assets)
│     ├─Dockerfile         # Node 构建 + Nginx 托管
│     └─nginx.conf         # 生产环境反向代理配置
└─docker-compose.yml       # 前后端一键编排
```

## 快速开始

### 1. 准备 GeoIP 数据库 (必需)
本项目使用 MaxMind GeoLite2 数据库进行 IP 地理定位。
1. 访问 [MaxMind 官网](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 注册并下载 `GeoLite2-City.mmdb`。
2. 将 `.mmdb` 文件放置在 `backend/` 目录下，或在运行时通过环境变量指定路径。

### 2. 本地开发

#### 后端启动
```bash
cd backend
# 安装依赖
go mod download
# 设置环境变量（可选，默认值如下）
export GEOIP_DB_PATH="./GeoLite2-City.mmdb"
export ALERT_CPU_THRESHOLD=80
export ALERT_MEM_THRESHOLD=80
export ALERT_DISK_THRESHOLD=90
# 运行
go run .
```
后端默认监听 `:8080`。

#### 前端启动
```bash
cd frontend/system-monitor-web
npm install
npm run dev
```
前端默认访问 `http://localhost:5173`。开发环境下 `vite.config.js` 已配置代理，将 `/api` 请求转发至本地后端。

### 3. Docker / Compose 部署

根目录提供 `docker-compose.yml`，一键启动所有服务。

```bash
# 1. 确保 GeoLite2-City.mmdb 已放入 backend/ 目录

# 2. 构建并启动
docker compose up -d --build

# 3. 查看日志
docker compose logs -f
```

访问 `http://<服务器IP>/` 即可查看仪表盘。

## 配置说明 (环境变量)

支持通过环境变量调整系统行为（可在 `docker-compose.yml` 或 shell 中设置）：

| 变量名 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `PORT` | `8080` | 后端服务端口 |
| `GEOIP_DB_PATH` | `./GeoLite2-City.mmdb` | GeoIP 数据库文件路径 |
| `ALERT_CPU_THRESHOLD` | `80` | CPU 告警阈值 (%) |
| `ALERT_MEM_THRESHOLD` | `80` | 内存告警阈值 (%) |
| `ALERT_DISK_THRESHOLD` | `90` | 磁盘告警阈值 (%) |

## API 接口

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| **GET** | `/api/stream` | **SSE** 实时数据流，推送完整监控数据 |
| **GET** | `/api/dashboard` | 获取当前快照数据 (JSON) |
| **GET** | `/api/alerts` | 获取分页告警历史 (Query: `page`, `size`) |

## 技术栈

- **Backend**: Go, Gin, gopsutil, geoip2-golang
- **Frontend**: Vue 3, Vite, Element Plus, ECharts 5, Axios
- **Deployment**: Docker, Docker Compose, Nginx
