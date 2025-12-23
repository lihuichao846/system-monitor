# System Monitor & Security Awareness Dashboard

面向服务器 / 工作站的实时监控与网络安全态势感知仪表盘。后端使用 **Go + Gin + gopsutil** 采集系统指标及网络连接信息，前端采用 **Vue 3 + Vite + Element Plus + ECharts** 渲染动态大屏。

## 功能亮点

- 全方位实时采集：CPU、内存、磁盘 I/O、分区使用率，以及实时上下行网络速率。
- 安全审计：记录外部活跃连接的远程 IP、端口、协议、进程名称，并可选地解析地理位置。
- 实时推送：基于 Server-Sent Events (SSE) 秒级推送仪表盘数据，降本增效。
- 告警工作台：可配置 CPU/内存阈值，自动生成告警并保留历史分页查询；在流量突发或定时周期生成网络审计快照。
- 可视化大屏：性能仪表、趋势图表与地理热力视图（GeoIP 可选）。

## API 接口

- `GET /api/dashboard`：返回最新一次采集的仪表盘数据（含 CPU/内存/磁盘/网络/告警/地理热力）。
- `GET /api/alerts?limit=20&offset=0`：分页返回历史告警。
- `GET /api/stream`：SSE 数据流，事件名 `dashboard`，每秒推送一次当前仪表盘数据。

## 开发启动

- 后端（默认端口 `8080`）：
  ```bash
  cd backend
  # 可选：启用地理解析（需先准备 mmdb 文件）
  # Windows PowerShell:
  # $env:GEOIP_DB_PATH = "./GeoLite2-City.mmdb"
  # Linux/macOS:
  # export GEOIP_DB_PATH=./GeoLite2-City.mmdb
  go run ./main.go
  # 或构建后运行：
  # go build -o system-monitor-backend ./main.go && ./system-monitor-backend
  ```
- 前端（Vite 开发服务器）：
  ```bash
  cd frontend/system-monitor-web
  npm install
  npm run dev
  # 打开 http://localhost:5173
  ```
  前端会优先请求同源的 `/api/*`，访问失败时回退到 `http://localhost:8040/api/*`。

## 部署指南

### Windows 一键部署（推荐）

本脚本专为**完全空白的 Windows 环境**设计（例如刚安装好的 Windows Server）。您无需手动安装任何语言环境，脚本会自动完成一切。

1) **准备工作**：
   - 将本项目代码下载到目标机器（例如 `C:\system-monitor`）。
   - **（可选）GeoIP 数据库**：若需要 IP 地图功能，将 `GeoLite2-City.mmdb` 放入项目根目录。

2) **执行脚本**：
   右键点击开始菜单，选择 **Windows PowerShell (管理员)**，然后运行：
   ```powershell
   cd C:\system-monitor
   Set-ExecutionPolicy Bypass -Scope Process -Force
   ./bootstrap.ps1 -BackendPort 8040 -FrontendPort 8041
   ```

3) **脚本会自动执行以下操作**：
   - ✅ 自动安装包管理器 **Chocolatey**。
   - ✅ 自动安装 **Git**, **Go (Golang)**, **Node.js**, **NSSM** (服务管理器)。
   - ✅ 自动配置环境变量（PATH, GOPROXY 等），无需人工干预。
   - ✅ 自动编译后端 Go 代码并注册为 Windows 服务。
   - ✅ 自动编译前端 Vue 代码并托管为 Windows 服务。
   - ✅ 自动配置防火墙放行端口。

4) **验证访问**：
   - 后端 API：`http://localhost:8040`
   - 前端大屏：`http://localhost:8041`

### Linux 部署（手动构建）

由于 Docker 容器环境可能无法完美获取宿主机的全部硬件指标（如准确的磁盘分区、进程列表等），**推荐直接在宿主机编译运行**。

1) **准备环境**：确保已安装 Go 1.23+ 和 Node.js 18+。
2) **后端**：
   ```bash
   cd backend
   go build -o system-monitor-backend .
   # 可选：设置 GeoIP 路径
   export GEOIP_DB_PATH=/path/to/GeoLite2-City.mmdb
   nohup ./system-monitor-backend > backend.log 2>&1 &
   ```
3) **前端**：
   ```bash
   cd frontend/system-monitor-web
   npm install && npm run build
   # 使用 serve 或 nginx 托管 dist 目录
   npm i -g serve
   nohup serve -s dist -l 8041 > frontend.log 2>&1 &
   ```

> **注意**：虽然提供了 `docker-compose.yml`，但为了获得最准确的系统监控数据（尤其是局域网扫描和底层硬件信息），建议优先采用上述直接部署方式。

## 环境变量

- `PORT`：后端监听端口，默认 `8080`。
- `GEOIP_DB_PATH`：GeoIP 数据库文件路径；未设置时地理解析功能关闭。
- `ALERT_CPU_WARN`：CPU 告警阈值（百分比，默认 `80`）。
- `ALERT_MEM_WARN`：内存告警阈值（百分比，默认 `90`）。
- （容器部署）`HOST_PROC`、`HOST_SYS`、`HOST_ETC`、`HOST_ROOT`、`HOST_HOSTNAME`、`HOST_OS`：用于在容器中读取宿主机信息，已在 `docker-compose.yml` 提供样例。

## 项目结构

- `backend/`：Go 后端服务（Gin 路由、指标采集、SSE 推送、GeoIP 可选）。
- `frontend/system-monitor-web/`：Vue3 前端（Element Plus + ECharts）。
- `docker-compose.yml`：前后端容器编排，含健康检查与反向代理配置。
- `bootstrap.ps1`：Windows 一键安装与服务化脚本。
- `deploy.sh`：Linux 环境一键构建与启动脚本。

## 技术栈

- Backend：Go、Gin、gopsutil、geoip2-golang
- Frontend：Vue 3、Vite、Element Plus、ECharts 6、Axios
