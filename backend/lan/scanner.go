package lan

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Host struct {
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	Latency    string `json:"latency"` // e.g. "2ms"
	HasMonitor bool   `json:"has_monitor"`
}

type ScanResult struct {
	LocalIP string `json:"local_ip"`
	Subnet  string `json:"subnet"` // e.g. "192.168.1.0/24"
	Hosts   []Host `json:"hosts"`
}

var (
	lastResult ScanResult
	lastScan   time.Time
	mu         sync.RWMutex
	isScanning bool
)

// GetTopology returns the cached topology or triggers a new scan
func GetTopology() ScanResult {
	mu.RLock()
	// Cache valid for 60 seconds
	if time.Since(lastScan) < 60*time.Second && len(lastResult.Hosts) > 0 {
		defer mu.RUnlock()
		return lastResult
	}
	mu.RUnlock()

	mu.Lock()
	if isScanning {
		mu.Unlock()
		// Return old data (or empty) while scanning
		return lastResult
	}
	isScanning = true
	mu.Unlock()

	// Start scan in background
	go func() {
		res := performScan()
		mu.Lock()
		lastResult = res
		lastScan = time.Now()
		isScanning = false
		mu.Unlock()
	}()

	// Return current state immediately (might be empty on first run)
	mu.RLock()
	defer mu.RUnlock()
	return lastResult
}

func performScan() ScanResult {
	localIP, subnet, err := getLocalIPAndSubnet()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		return ScanResult{}
	}

	hosts := scanSubnet(subnet, localIP)

	// [MOCK] Add some fake devices for testing topology view
	// 假设子网是 192.168.1.x，我们随机生成几个同网段 IP
	// 如果是其他网段，这里仅作演示，IP 可能看起来不匹配，但逻辑上能通
	baseIP := "192.168.1."
	if parts := strings.Split(localIP, "."); len(parts) == 4 {
		baseIP = fmt.Sprintf("%s.%s.%s.", parts[0], parts[1], parts[2])
	}

	hosts = append(hosts, Host{IP: baseIP + "55", Hostname: "iPhone-14-Pro", Latency: "25ms", HasMonitor: false})
	hosts = append(hosts, Host{IP: baseIP + "101", Hostname: "HP-LaserJet-M102", Latency: "4ms", HasMonitor: false})
	hosts = append(hosts, Host{IP: baseIP + "200", Hostname: "NAS-Synology", Latency: "1ms", HasMonitor: true})
	hosts = append(hosts, Host{IP: baseIP + "88", Hostname: "Guest-Laptop", Latency: "120ms", HasMonitor: false})

	return ScanResult{
		LocalIP: localIP,
		Subnet:  subnet,
		Hosts:   hosts,
	}
}

func getLocalIPAndSubnet() (string, string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// Found IPv4
				ones, _ := ipnet.Mask.Size()
				// Only scan /24 for performance in this demo
				if ones < 24 {
					// If mask is too large (e.g. /16), we still treat it as /24 based on current IP
					// to avoid scanning 65535 hosts
					ones = 24
				}
				// Construct base subnet
				ip := ipnet.IP.To4()
				// Simple approach: assume /24 and just get first 3 octets
				// Correct approach: Apply mask
				masked := ip.Mask(net.CIDRMask(ones, 32))
				subnet := fmt.Sprintf("%s/%d", masked.String(), ones)
				return ip.String(), subnet, nil
			}
		}
	}
	return "", "", fmt.Errorf("no suitable interface found")
}

func scanSubnet(subnet string, localIP string) []Host {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		s := ip.String()
		// Skip network address and broadcast (simple heuristic)
		if strings.HasSuffix(s, ".0") || strings.HasSuffix(s, ".255") {
			continue
		}
		ips = append(ips, s)
	}

	// Limit to 255 IPs to be safe
	if len(ips) > 255 {
		ips = ips[:255]
	}

	var wg sync.WaitGroup
	// Semaphore to limit concurrency
	sem := make(chan struct{}, 50) // 50 concurrent pings
	var foundHosts []Host
	var hostsMu sync.Mutex

	for _, target := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(t string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Check if it's me
			if t == localIP {
				hostsMu.Lock()
				foundHosts = append(foundHosts, Host{
					IP:         t,
					Hostname:   "System Monitor (Server)",
					Latency:    "0ms",
					HasMonitor: true,
				})
				hostsMu.Unlock()
				return
			}

			alive, lat := ping(t)
			if alive {
				// Check monitor port (default 8041 for frontend)
				hasMon := checkMonitorPort(t, 8041)

				// Resolve hostname
				hostname := ""
				names, _ := net.LookupAddr(t)
				if len(names) > 0 {
					hostname = strings.TrimSuffix(names[0], ".")
				}

				hostsMu.Lock()
				foundHosts = append(foundHosts, Host{
					IP:         t,
					Hostname:   hostname,
					Latency:    lat,
					HasMonitor: hasMon,
				})
				hostsMu.Unlock()
			}
		}(target)
	}

	wg.Wait()
	return foundHosts
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func ping(ip string) (bool, string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// -n 1: count 1, -w 200: timeout 200ms
		cmd = exec.Command("ping", "-n", "1", "-w", "500", ip)
	} else {
		// -c 1: count 1, -W 1: timeout 1s (Linux ping usually uses seconds for -W)
		// Some busybox ping uses -w for seconds too.
		// We use -W 1 for standard iputils-ping
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return false, ""
	}

	// Format latency
	ms := float64(duration.Microseconds()) / 1000.0
	lat := fmt.Sprintf("%.1fms", ms)

	return true, lat
}
