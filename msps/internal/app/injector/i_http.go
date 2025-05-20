package injector

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"msps/internal/app/config"
	"msps/internal/app/middleware"
	"msps/internal/app/router"
)

func initHttpServer(router router.IRouter) *gin.Engine {
	if config.GlobalConfig().IsDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(
		middleware.Recovery(nil),
		middleware.Cors(),
	)

	if config.GlobalConfig().IsDebug {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.DocExpansion("none"),
		))
	}

	router.Register(engine)
	return engine
}
