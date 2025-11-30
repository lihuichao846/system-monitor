# System Monitor & Security Awareness Dashboard

面向服务器 / 工作站的实时监控与网络安全态势感知仪表盘。后端使用 **Go + Gin + gopsutil** 采集系统指标及网络连接信息，前端采用 **Vue 3 + Vite + Element Plus + ECharts** 渲染动态大屏。

## 功能亮点

- **全方位实时采集**：
  - 基础资源：CPU、内存、磁盘 I/O、**磁盘分区详情 (已用/总量/使用率)**。
  - 网络流量：实时上下行速率、总流量统计。
  - **多平台支持**：完美支持 Windows、Linux、macOS。
- **安全态势**：
  - **深度审计**：实时关联活跃连接的 **进程名称**、**远程 IP**、**地理位置** (国家/城市) 及端口信息。
  - **威胁感知**：流量突发或定时触发连接快照，有效捕获潜在的恶意通信（如 C2 心跳）。
  - **可视化热力图**：通过 **GeoIP** 绘制全球连接热力分布，直观定位威胁来源。
  - 进程监控：Top CPU/Memory 进程实时追踪。
- **实时数据流 (SSE)**：采用 Server-Sent Events (SSE) 技术，实现秒级数据推送，告别传统轮询，降低延迟。
- **智能告警工作台**：
  - 可配置的 CPU/内存 阈值。
  - 告警历史记录与分页查询。
  - **网络审计日志**：详细记录异常流量时刻的连接快照。
- **可视化大屏**：
  - 动态仪表盘：CPU/内存/磁盘 仪表盘（支持动画与渐变效果）。
  - 趋势图表：ECharts 绘制流量趋势、负载变化。
  - **地理视图**：全球地图热力分布，直观展示外部连接活跃区域。

## 部署建议

**推荐使用本地直接运行的方式**，尤其是 Windows 环境。
由于 Docker 的隔离机制，容器内无法直接读取 Windows 宿主机的物理 CPU/内存使用率（只能读取 Docker 虚拟机的资源），且网络流量监控也会受限于虚拟网卡。**本地运行可获得最准确、无损的硬件监控体验。**

## 快速开始

### 1. 准备 GeoIP 数据库 (必需)
本项目使用 MaxMind GeoLite2 数据库进行 IP 地理定位。
1. **官方渠道**：访问 [MaxMind 官网](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 注册并下载 `GeoLite2-City.mmdb`。
2. **备用镜像**（如无法访问官网）：可使用开源社区维护的镜像下载：
   ```bash
   # 在 backend 目录下执行
   curl -L -o GeoLite2-City.mmdb "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-City.mmdb"
   ```
3. 将 `.mmdb` 文件放置在 `backend/` 目录下。

### 2. 启动服务 (推荐)

#### 第一步：启动后端
```bash
cd backend
# 安装依赖
go mod download
# 运行服务
go run .
```
后端默认监听 `:8080`。

#### 第二步：启动前端
```bash
cd frontend/system-monitor-web
npm install
npm run dev
```
访问终端输出的地址（通常为 `http://localhost:5173`）。开发环境下 `vite.config.js` 已配置代理，将 `/api` 请求转发至本地后端。

### 3. Docker 部署 (仅限 Linux 服务器)

如果您是在 Linux 服务器上部署，Docker 也是一个不错的选择。请使用 `host` 网络模式以获取准确指标。

```bash
./deploy.sh
```
或
```bash
docker compose up -d --build
```

## 配置说明 (环境变量)

| 变量名 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `PORT` | `8080` | 后端服务端口 |
| `GEOIP_DB_PATH` | `./GeoLite2-City.mmdb` | GeoIP 数据库文件路径 |
| `ALERT_CPU_WARN` | `80` | CPU 告警阈值 (%) |
| `ALERT_MEM_WARN` | `90` | 内存告警阈值 (%) |

## 技术栈

- **Backend**: Go, Gin, gopsutil, geoip2-golang
- **Frontend**: Vue 3, Vite, Element Plus, ECharts 5, Axios
