package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    log "github.com/sirupsen/logrus"
)

// Logger 日志中间件
func Logger(skippers ...SkipperFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        if SkipHandler(c, skippers...) {
            c.Next()
            return
        }

        // 开始时间
        start := time.Now()

        // 处理请求
        c.Next()
        // 结束时间
        end := time.Now()
        // 参数记录
        latency := end.Sub(start)
        path := c.Request.URL.Path
        clientIP := c.ClientIP()
        method := c.Request.Method
        statusCode := c.Writer.Status()

        log.Infof("| %3d | %10v | %10s | %-4s | %s | %10v",
            statusCode,
            latency,
            clientIP,
            method,
            path,
            c.Request.Header,
        )
    }
}
