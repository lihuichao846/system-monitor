package main

import (
	"system-monitor/metrics"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	metrics.StartCollector() // 启动数据采集

	r := gin.Default()

	// 提供前端请求的 API
	r.GET("/api/dashboard", func(c *gin.Context) {
		metrics.Mu.RLock()
		data := metrics.Latest
		metrics.Mu.RUnlock()
		c.JSON(http.StatusOK, data)
	})

	r.Run(":8080") // 启动服务
}
