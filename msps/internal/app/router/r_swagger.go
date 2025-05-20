package router

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"msps/docs"
)

func (r *Router) RegisterSwagger(engine *gin.Engine) {
	g := engine.Group("/s")

	docs.SwaggerInfo.BasePath = ""
	docs.SwaggerInfo.Title = "管理后台接口"
	docs.SwaggerInfo.Description = "实现一个管理系统的后端API服务"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
