package router

import "github.com/gin-gonic/gin"

// registerAgentApi 注册有关agent的API
func (r *Router) registerAgentApi(engine *gin.Engine) {
	g := engine.Group("/a")
	// 健康检查
	g.POST("/h", r.AgentApi.HealthCheck)
	// 邮件发送处理
	g.POST("/m", r.AgentApi.HandleSentEmail)
	// 邮件确认处理
	g.POST("/v", r.AgentApi.HandleVerifyEmail)
}
