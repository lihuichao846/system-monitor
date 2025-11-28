package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	mem "github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
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

// NetLogEntry 表示一条网络流量日志
type NetLogEntry struct {
	Time       string        `json:"time"`       // 时间戳 HH:MM:SS
	Rx         float64       `json:"rx"`         // 总下行 KB/s
	Tx         float64       `json:"tx"`         // 总上行 KB/s
	Interfaces []NetworkInfo `json:"interfaces"` // 当时各网卡明细
}

var (
	Latest DashboardData
	Mu     sync.RWMutex

	lastDiskIO map[string]disk.IOCountersStat
	lastNetIO  map[string]net.IOCountersStat
	lastTime   time.Time

	alertLog []AlertInfo
	netLog   []NetLogEntry
)

func StartCollector() {
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
	if perf.CPUUsage > 80 {
		alerts = append(alerts, AlertInfo{
			Level: "warn",
			Text:  fmt.Sprintf("CPU 使用率过高：%.1f%%", perf.CPUUsage),
			Time:  now.Format("15:04:05"),
		})
	}
	if perf.MemUsedPercent > 90 {
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

	// append to net log: 仅当总流量较大时记录（大于约 1 MB/s）
	if totalRx+totalTx > 1024 {
		netLog = append(netLog, NetLogEntry{
			Time:       now.Format("15:04:05"),
			Rx:         totalRx,
			Tx:         totalTx,
			Interfaces: nets,
		})
		if len(netLog) > 300 {
			netLog = netLog[len(netLog)-300:]
		}
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
		System:  SystemInfo{Hostname: hostStat.Hostname, OS: hostStat.OS, Platform: hostStat.Platform, BootTime: hostStat.BootTime},
		Perf:    perf,
		Alerts:  alertLog,
		Current: alerts,
		NetLog:  netLog,
		Timestamp: now.Unix(),
	}
	Mu.Unlock()

	lastNetIO = newNet
	lastDiskIO = newDiskIO
	lastTime = now
}
