package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	"path"
	"strings"

	"msps/internal/app/model/common"
)

// SkipperFunc 定义中间件跳过函数
type SkipperFunc func(*gin.Context) bool

// NoRoute 路径不存在
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, common.NewResponse(common.WithMsg(common.MsgNoRoute)))
		c.Abort()
	}
}

// NoMethod 方法不存在
func NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg(common.MsgNoRoute)))
		c.Abort()
	}
}

func AllowPathSkipper(paths ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		urlPath := c.Request.URL.Path
		for _, p := range paths {
			if urlPath == p {
				return true
			}
		}
		return false
	}
}

// AllowPathPrefixSkipper 请求路径 白名单
func AllowPathPrefixSkipper(prefixes ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		urlPath := c.Request.URL.Path
		pathLen := len(urlPath)
		for _, p := range prefixes {
			if pl := len(p); pathLen >= pl && urlPath[:pl] == p {
				return true
			}
		}
		return false
	}
}

// AllowPathPrefixNoSkipper 请求路径 黑名单
func AllowPathPrefixNoSkipper(prefixes ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		uPath := c.Request.URL.Path
		pathLen := len(uPath)
		for _, prefix := range prefixes {
			if prefixLen := len(prefix); pathLen >= prefixLen && uPath[:prefixLen] == prefix {
				return false
			}
		}
		return true
	}
}

// AllowMethodAndPathPrefixSkipper 请求方法+请求路径 白名单
func AllowMethodAndPathPrefixSkipper(prefixes ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		urlPath := JoinRouter(c.Request.Method, c.Request.URL.Path)
		pathLen := len(urlPath)

		for _, prefix := range prefixes {
			if prefixLen := len(prefix); pathLen >= prefixLen && urlPath[:prefixLen] == prefix {
				return true
			}
		}
		return false
	}
}

// AllowRemoteAddressSkipper IP白名单
func AllowRemoteAddressSkipper(address ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		remoteAddr := c.ClientIP()
		for _, addr := range address {
			if addr == remoteAddr {
				return true
			}
		}
		return false
	}
}

// AllowRemoteAddressNoSkipper IP黑名单
func AllowRemoteAddressNoSkipper(address ...string) SkipperFunc {
	return func(c *gin.Context) bool {
		remoteAddr := c.ClientIP()
		for _, addr := range address {
			if addr == remoteAddr {
				return false
			}
		}
		return true
	}
}

// JoinRouter 方法与路径相连
func JoinRouter(m, p string) string {
	if len(p) > 0 && p[0] != '/' {
		p = "/" + p
	}
	return fmt.Sprintf("%s%s", strings.ToUpper(m), path.Clean(p))
}

// SkipHandler 统一处理跳过函数
func SkipHandler(c *gin.Context, skippers ...SkipperFunc) bool {
	for _, skipper := range skippers {
		if skipper(c) {
			return true
		}
	}

	return false
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		log.Printf("Received Authorization header: %s", authHeader) // 添加日志
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("未提供认证信息")))
			return
		}

		// 简单的token验证
		if !strings.HasPrefix(authHeader, "Bearer generated-token-") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("无效的token")))
			return
		}

		// 可以从token中提取用户名等简单信息
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("非法的token格式")))
			return
		}

		// 把用户名存入上下文供后续使用
		username := strings.TrimPrefix(parts[1], "generated-token-")
		c.Set("username", username)

		c.Next()
	}
}
