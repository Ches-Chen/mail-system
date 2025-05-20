package router

import (
	"github.com/gin-gonic/gin"

	"msps/internal/app/middleware"
)

func (r *Router) registerApi(engine *gin.Engine) {
	engine.Use(middleware.Cors())
	engine.Use(gin.Recovery())

	r.registerAgentApi(engine)
	r.RegisterClientApi(engine)
	r.RegisterSwagger(engine)
}
