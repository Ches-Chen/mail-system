package router

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"msps/docs"
	"msps/internal/app/api"
	"msps/internal/app/config"
	"msps/internal/app/controller"
	"msps/internal/app/model/domain"
	"time"
)

var ProviderSet = wire.NewSet(
	wire.Bind(new(IRouter), new(*Router)),
	NewRouter,
	InitDatabase, // 改为导出的函数名
)

type IRouter interface {
	Register(*gin.Engine)
}

type Router struct {
	AgentApi  *api.Agent
	ClientApi *api.Client
	UserCtrl  *controller.UserController
	EmailCtrl *controller.EmailController
	db        *gorm.DB
}

func NewRouter(
	agent *api.Agent,
	client *api.Client,
	userCtrl *controller.UserController,
	emailCtrl *controller.EmailController,
) *Router {
	return &Router{
		AgentApi:  agent,
		ClientApi: client,
		UserCtrl:  userCtrl,
		EmailCtrl: emailCtrl,
	}
}

// InitDatabase 改为导出的InitDatabase
func InitDatabase() (*gorm.DB, error) {
	dsn := config.GlobalConfig().DatabaseDSN
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := domain.Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func (r *Router) Register(engine *gin.Engine) {
	docs.SwaggerInfo.Host = config.GlobalConfig().SwagHost
	r.registerApi(engine)
}
