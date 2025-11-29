package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	stdnet "net"

	"github.com/oschwald/geoip2-golang"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	mem "github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type DashboardData struct {
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	Network   []NetworkInfo `json:"network"`
	System    SystemInfo    `json:"system"`
	Perf      PerfInfo      `json:"perf"`
	Alerts    []AlertInfo   `json:"alerts"`         // 历史告警日志
	Current   []AlertInfo   `json:"current_alerts"` // 当前采样周期内的告警
	NetLog    []NetLogEntry `json:"net_log"`        // 网络流量日志
	GeoHeat   []GeoPoint    `json:"geo_heat"`
	Timestamp int64         `json:"timestamp"`
}

type CPUInfo struct {
	Usage     float64   `json:"usage"`
	PerCore   []float64 `json:"per_core"`
	Load1     float64   `json:"load1"`
	Load5     float64   `json:"load5"`
	Load15    float64   `json:"load15"`
	Cores     int       `json:"cores"`
	ModelName string    `json:"model_name"`
	Mhz       float64   `json:"mhz"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapTotal   uint64  `json:"swap_total"`
}

type DiskInfo struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	ReadSpeed   float64 `json:"read_speed"`
	WriteSpeed  float64 `json:"write_speed"`
}

type NetworkInfo struct {
	Interface string  `json:"interface"`
	RX        float64 `json:"rx"`
	TX        float64 `json:"tx"`
}

type SystemInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Platform string `json:"platform"`
	BootTime uint64 `json:"boot_time"`
}

type PerfInfo struct {
	CPUUsage       float64 `json:"cpu_usage"`        // 当前 CPU 使用率
	Load1          float64 `json:"load1"`            // 1 分钟负载
	Load5          float64 `json:"load5"`            // 5 分钟负载
	Load15         float64 `json:"load15"`           // 15 分钟负载
	MemUsedPercent float64 `json:"mem_used_percent"` // 内存使用率
	SwapUsed       uint64  `json:"swap_used"`
	SwapTotal      uint64  `json:"swap_total"`
	NetRxKBps      float64 `json:"net_rx_kbps"`     // 汇总下行 KB/s
	NetTxKBps      float64 `json:"net_tx_kbps"`     // 汇总上行 KB/s
	DiskReadKBps   float64 `json:"disk_read_kbps"`  // 所有磁盘总读速
	DiskWriteKBps  float64 `json:"disk_write_kbps"` // 所有磁盘总写速
	CPUTemp        float64 `json:"cpu_temp"`        // CPU 温度（可能为 0，视平台支持情况）
}

type AlertInfo struct {
	Level string `json:"level"` // ok / warn / critical
	Text  string `json:"text"`  // 描述，例如 "CPU 使用率过高：95.2%"
	Time  string `json:"time"`  // 触发时间，格式 HH:MM:SS
}

// NetLogEntry 表示一条网络流量日志（含安全审计信息）
type NetLogEntry struct {
	Time        string           `json:"time"`        // 时间戳 HH:MM:SS
	Rx          float64          `json:"rx"`          // 总下行 KB/s
	Tx          float64          `json:"tx"`          // 总上行 KB/s
	Interfaces  []NetworkInfo    `json:"interfaces"`  // 当时各网卡明细
	Connections []ConnectionInfo `json:"connections"` // 活跃连接快照
}

type ConnectionInfo struct {
	RemoteIP   string `json:"remote_ip"`
	RemotePort uint32 `json:"remote_port"`
	LocalPort  uint32 `json:"local_port"`
	Protocol   string `json:"protocol"` // TCP/UDP
	Status     string `json:"status"`   // ESTABLISHED, etc
	Process    string `json:"process"`  // Process Name
	Country    string `json:"country"`
	City       string `json:"city"`
}

type GeoPoint struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Count   int     `json:"count"`
	Country string  `json:"country"`
	City    string  `json:"city"`
}

var (
	Latest DashboardData
	Mu     sync.RWMutex

	lastDiskIO map[string]disk.IOCountersStat
	lastNetIO  map[string]net.IOCountersStat
	lastTime   time.Time

	alertLog   []AlertInfo
	netLog     []NetLogEntry
	logCounter int
)

var geoReader *geoip2.Reader

type Config struct {
	CPUWarn float64
	MemWarn float64
}

var cfg Config

func InitConfigFromEnv() {
	cfg.CPUWarn = 80
	cfg.MemWarn = 90
	if v := os.Getenv("ALERT_CPU_WARN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.CPUWarn = f
		}
	}
	if v := os.Getenv("ALERT_MEM_WARN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.MemWarn = f
		}
	}
}

func InitGeo() {
	path := os.Getenv("GEOIP_DB_PATH")
	if path == "" {
		return
	}
	r, err := geoip2.Open(path)
	if err != nil {
		return
	}
	geoReader = r
}

func StartCollector() {
	InitConfigFromEnv()
	InitGeo()
	lastTime = time.Now()
	lastDiskIO, _ = disk.IOCounters()
	lastNetIO = netSliceToMap() // 🔥 正确初始化

	go func() {
		for {
			collect()
			time.Sleep(time.Second)
		}
	}()
}

// 🔥 将 []net.IOCountersStat 转换成 map[string]net.IOCountersStat
func netSliceToMap() map[string]net.IOCountersStat {
	m := make(map[string]net.IOCountersStat)
	list, _ := net.IOCounters(true)
	for _, v := range list {
		m[v.Name] = v
	}
	return m
}

func collect() {
	now := time.Now()
	delta := now.Sub(lastTime).Seconds()

	// CPU
	perCore, _ := cpu.Percent(0, true)
	totalCPU, _ := cpu.Percent(0, false)
	cpuInfo, _ := cpu.Info()
	loadAvg, _ := load.Avg()

	// Memory
	memStat, _ := mem.VirtualMemory()
	swapStat, _ := mem.SwapMemory()

	// Disk
	partitions, _ := disk.Partitions(true)
	newDiskIO, _ := disk.IOCounters()

	var disks []DiskInfo
	var totalDiskRead, totalDiskWrite float64
	for _, p := range partitions {
		usage, _ := disk.Usage(p.Mountpoint)

		var rSpeed, wSpeed float64
		if old, ok := lastDiskIO[p.Device]; ok {
			cur := newDiskIO[p.Device]
			rSpeed = float64(cur.ReadBytes-old.ReadBytes) / 1024 / delta
			wSpeed = float64(cur.WriteBytes-old.WriteBytes) / 1024 / delta
		}

		totalDiskRead += rSpeed
		totalDiskWrite += wSpeed

		disks = append(disks, DiskInfo{
			Path:        p.Mountpoint,
			Total:       usage.Total,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
			ReadSpeed:   rSpeed,
			WriteSpeed:  wSpeed,
		})
	}

	// Network
	newNet := netSliceToMap()

	var nets []NetworkInfo
	var totalRx, totalTx float64
	for name, cur := range newNet {
		if old, ok := lastNetIO[name]; ok {
			rx := float64(cur.BytesRecv-old.BytesRecv) / 1024 / delta
			tx := float64(cur.BytesSent-old.BytesSent) / 1024 / delta
			totalRx += rx
			totalTx += tx
			nets = append(nets, NetworkInfo{
				Interface: name,
				RX:        rx,
				TX:        tx,
			})
		}
	}

	// System
	hostStat, _ := host.Info()

	// Sensors / temperature (best-effort, 部分平台可能不支持)
	var cpuTemp float64
	if temps, err := host.SensorsTemperatures(); err == nil {
		for _, t := range temps {
			key := strings.ToLower(t.SensorKey)
			if strings.Contains(key, "cpu") || strings.Contains(key, "package") {
				cpuTemp = t.Temperature
				break
			}
		}
	}

	// Perf summary
	perf := PerfInfo{
		CPUUsage:       totalCPU[0],
		Load1:          loadAvg.Load1,
		Load5:          loadAvg.Load5,
		Load15:         loadAvg.Load15,
		MemUsedPercent: memStat.UsedPercent,
		SwapUsed:       swapStat.Used,
		SwapTotal:      swapStat.Total,
		NetRxKBps:      totalRx,
		NetTxKBps:      totalTx,
		DiskReadKBps:   totalDiskRead,
		DiskWriteKBps:  totalDiskWrite,
		CPUTemp:        cpuTemp,
	}

	// Alerts (current snapshot)
	var alerts []AlertInfo
	if perf.CPUUsage > cfg.CPUWarn {
		alerts = append(alerts, AlertInfo{
			Level: "warn",
			Text:  fmt.Sprintf("CPU 使用率过高：%.1f%%", perf.CPUUsage),
			Time:  now.Format("15:04:05"),
		})
	}
	if perf.MemUsedPercent > cfg.MemWarn {
		alerts = append(alerts, AlertInfo{
			Level: "warn",
			Text:  fmt.Sprintf("内存使用率过高：%.1f%%", perf.MemUsedPercent),
			Time:  now.Format("15:04:05"),
		})
	}

	// append to alert log (simple in-memory ring, 保留最近 200 条)
	if len(alerts) > 0 {
		alertLog = append(alertLog, alerts...)
		if len(alertLog) > 200 {
			alertLog = alertLog[len(alertLog)-200:]
		}
	}

	// append to net log: 流量 > 100KB/s 或 每 10 秒强制检查一次活跃连接
	logCounter++
	shouldLog := false
	if totalRx+totalTx > 100 {
		shouldLog = true
	}
	if logCounter >= 10 {
		shouldLog = true
		logCounter = 0
	}

	if shouldLog {
		auditConns := collectAuditConnections()
		// 只有当确实有外部连接 或 流量确实大时才记录
		if len(auditConns) > 0 || totalRx+totalTx > 100 {
			netLog = append(netLog, NetLogEntry{
				Time:        now.Format("15:04:05"),
				Rx:          totalRx,
				Tx:          totalTx,
				Interfaces:  nets,
				Connections: auditConns,
			})
			if len(netLog) > 300 {
				netLog = netLog[len(netLog)-300:]
			}
		}
	}

	var geoPoints []GeoPoint
	if geoReader != nil {
		geoPoints = collectGeoPoints()
	}

	// Update global data
	Mu.Lock()
	Latest = DashboardData{
		CPU: CPUInfo{
			Usage:     totalCPU[0],
			PerCore:   perCore,
			Load1:     loadAvg.Load1,
			Load5:     loadAvg.Load5,
			Load15:    loadAvg.Load15,
			Cores:     len(perCore),
			ModelName: cpuInfo[0].ModelName,
			Mhz:       cpuInfo[0].Mhz,
		},
		Memory: MemoryInfo{
			Total:       memStat.Total,
			Used:        memStat.Used,
			Free:        memStat.Free,
			UsedPercent: memStat.UsedPercent,
			SwapUsed:    swapStat.Used,
			SwapTotal:   swapStat.Total,
		},
		Disk:      disks,
		Network:   nets,
		System:    SystemInfo{Hostname: hostStat.Hostname, OS: hostStat.OS, Platform: hostStat.Platform, BootTime: hostStat.BootTime},
		Perf:      perf,
		Alerts:    alertLog,
		Current:   alerts,
		NetLog:    netLog,
		GeoHeat:   geoPoints,
		Timestamp: now.Unix(),
	}
	Mu.Unlock()

	lastNetIO = newNet
	lastDiskIO = newDiskIO
	lastTime = now
}

func GetAlerts(limit, offset int) ([]AlertInfo, int) {
	Mu.RLock()
	defer Mu.RUnlock()
	total := len(alertLog)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	items := make([]AlertInfo, end-offset)
	copy(items, alertLog[offset:end])
	return items, total
}

func isPrivateIP(ip string) bool {
	p := stdnet.ParseIP(ip)
	if p == nil {
		return true
	}
	if p.IsLoopback() || p.IsMulticast() || p.IsUnspecified() {
		return true
	}
	if p.To4() == nil {
		return false
	}
	b := p.To4()
	if b[0] == 10 {
		return true
	}
	if b[0] == 172 && b[1] >= 16 && b[1] <= 31 {
		return true
	}
	if b[0] == 192 && b[1] == 168 {
		return true
	}
	return false
}

func collectAuditConnections() []ConnectionInfo {
	conns, err := net.Connections("inet")
	if err != nil {
		return nil
	}

	var out []ConnectionInfo
	// Cache for pid -> name to avoid duplicate lookups in same batch
	procNames := make(map[int32]string)

	for _, c := range conns {
		// 只关注 ESTABLISHED 且有远程地址的
		if c.Status != "ESTABLISHED" || c.Raddr.IP == "" {
			continue
		}
		if isPrivateIP(c.Raddr.IP) {
			continue
		}

		// Process Name
		name, ok := procNames[c.Pid]
		if !ok {
			if c.Pid > 0 {
				if p, err := process.NewProcess(c.Pid); err == nil {
					name, _ = p.Name()
				}
			}
			if name == "" {
				name = "unknown"
			}
			procNames[c.Pid] = name
		}

		// Geo
		country, city := "-", "-"
		if geoReader != nil {
			if rec, err := geoReader.City(stdnet.ParseIP(c.Raddr.IP)); err == nil && rec != nil {
				if n, ok := rec.Country.Names["zh-CN"]; ok {
					country = n
				} else if n, ok := rec.Country.Names["en"]; ok {
					country = n
				}
				if n, ok := rec.City.Names["zh-CN"]; ok {
					city = n
				} else if n, ok := rec.City.Names["en"]; ok {
					city = n
				}
			}
		}

		proto := "TCP"
		if c.Type == 2 { // UDP
			proto = "UDP"
		}

		out = append(out, ConnectionInfo{
			RemoteIP:   c.Raddr.IP,
			RemotePort: c.Raddr.Port,
			LocalPort:  c.Laddr.Port,
			Protocol:   proto,
			Status:     c.Status,
			Process:    name,
			Country:    country,
			City:       city,
		})

		if len(out) >= 20 { // Limit max entries per snapshot to avoid bloat
			break
		}
	}
	return out
}

func collectGeoPoints() []GeoPoint {
	conns, err := net.Connections("inet")
	if err != nil {
		return nil
	}
	type key struct{ lat, lon float64 }
	agg := make(map[key]GeoPoint)
	for _, c := range conns {
		ip := c.Raddr.IP
		if ip == "" || isPrivateIP(ip) {
			continue
		}
		rec, err := geoReader.City(stdnet.ParseIP(ip))
		if err != nil || rec == nil {
			continue
		}
		lat := rec.Location.Latitude
		lon := rec.Location.Longitude
		k := key{lat: lat, lon: lon}
		g := agg[k]
		g.Lat = lat
		g.Lon = lon
		g.Count++
		if rec.Country.Names != nil {
			if v, ok := rec.Country.Names["zh-CN"]; ok {
				g.Country = v
			} else {
				g.Country = rec.Country.Names["en"]
			}
		}
		if rec.City.Names != nil {
			if v, ok := rec.City.Names["zh-CN"]; ok {
				g.City = v
			} else {
				g.City = rec.City.Names["en"]
			}
		}
		agg[k] = g
	}
	out := make([]GeoPoint, 0, len(agg))
	for _, v := range agg {
		out = append(out, v)
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}
