package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"system-monitor/metrics"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	metrics.StartCollector() // 启动数据采集

	r := gin.Default()

	// 提供前端请求的 API
	r.GET("/api/dashboard", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		metrics.Mu.RLock()
		data := metrics.Latest
		metrics.Mu.RUnlock()
		c.JSON(http.StatusOK, data)
	})

	r.GET("/api/alerts", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")
		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)
		items, total := metrics.GetAlerts(limit, offset)
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	r.GET("/api/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		ctx := c.Request.Context()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics.Mu.RLock()
				data := metrics.Latest
				metrics.Mu.RUnlock()
				b, _ := json.Marshal(data)
				fmt.Fprintf(c.Writer, "event: dashboard\n")
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
				c.Writer.Flush()
			}
		}
	})

	r.Run(":8080") // 启动服务
}
