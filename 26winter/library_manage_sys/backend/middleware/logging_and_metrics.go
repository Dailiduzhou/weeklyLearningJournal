package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GinLoggerAndMetrics(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method
		ip := c.ClientIP()

		c.Next()

		latency := time.Since(start).Seconds()
		status := c.Writer.Status()

		HttpRequestCountTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
		HttpRequestDurationSeconds.WithLabelValues(method, path).Observe(latency)

		var msg string
		var level func(msg string, fields ...zap.Field)
		if status >= 500 {
			msg = "服务器内部错误"
			level = logger.Error
		} else if status >= 400 {
			msg = "客户端请求错误"
			level = logger.Warn
		} else {
			msg = "请求处理成功"
			level = logger.Info
		}

		level(msg,
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Float64("latency", latency),
			zap.String("ip", ip),
		)
	}
}
