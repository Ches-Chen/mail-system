package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    log "github.com/sirupsen/logrus"

    "msps/internal/app/model/common"
)

// Recovery 异常中间件
func Recovery(handle gin.RecoveryFunc) gin.HandlerFunc {
    if handle == nil {
        handle = func(c *gin.Context, err any) {
            log.Errorf("[%s %s] recover:%+v", c.Request.Method, c.Request.URL.String(), err)
            c.AbortWithStatusJSON(http.StatusInternalServerError, common.NewResponse(common.WithSuccess(false)))
        }
    }

    return gin.RecoveryWithWriter(gin.DefaultErrorWriter, handle)
}
