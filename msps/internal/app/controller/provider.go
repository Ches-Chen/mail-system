package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"
	"msps/internal/app/api"
)

var ProviderSet = wire.NewSet(
	NewUserController,
	NewClient,
	NewAgent,
	NewEmailController,
	wire.Bind(new(UserControllerInterface), new(*UserController)),
	wire.Bind(new(EmailControllerInterface), new(*EmailController)),
)

type UserControllerInterface interface {
	Login(c *gin.Context)
	Register(c *gin.Context)
}

type EmailControllerInterface interface {
	HandleSentEmail(c *gin.Context)
	HandleVerifyEmail(c *gin.Context)
}

type UserController struct {
	DB *gorm.DB
}

type EmailController struct {
	DB             *gorm.DB
	Client         *api.Client
	Agent          *api.Agent
	UserController UserControllerInterface
	StatusChecker  *EmailStatusChecker
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

func NewEmailController(
	db *gorm.DB,
	userCtrl UserControllerInterface,
	client *api.Client,
	agent *api.Agent,
) *EmailController {
	return &EmailController{
		DB:             db,
		Client:         client,
		Agent:          agent,
		UserController: userCtrl,
	}
}

func NewClient(db *gorm.DB, userCtrl UserControllerInterface) *api.Client {
	client := &api.Client{
		DB: db,
	}

	// 初始化并启动状态检查器
	statusChecker := NewEmailStatusChecker(db)
	go statusChecker.Start()

	return client
}

func NewAgent() *api.Agent {
	return &api.Agent{}
}

// HandleSentEmail 实现 EmailControllerInterface 接口方法
func (e *EmailController) HandleSentEmail(c *gin.Context) {
	e.Client.HandleSentEmail(c)
}

// HandleVerifyEmail 实现 EmailControllerInterface 接口方法
func (e *EmailController) HandleVerifyEmail(c *gin.Context) {
	e.Client.HandleVerifyEmail(c)
}

func (e *EmailController) StopStatusChecker() {
	if e.StatusChecker != nil {
		e.StatusChecker.Stop()
	}
}
