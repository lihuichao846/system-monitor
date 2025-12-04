package lan

import (
	"fmt"
	"net"
	"time"
)

func checkMonitorPort(ip string, port int) bool {
	timeout := 200 * time.Millisecond
	target := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
