package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"

	"msps/internal/app/config"
	"msps/internal/app/controller"
	"msps/internal/app/router"
)

// @title msps
// @version 1.0
// @description 实现一个管理系统的后端API服务
// @host localhost:8080
// @BasePath /
// @schemes http https
func main() {
	// 1. 初始化配置
	if err := config.InitConfig(); err != nil {
		logrus.Fatalf("初始化配置失败: %v", err)
	}

	// 2. 创建CLI应用
	mApp := cli.NewApp()
	mApp.Name = "msps"
	mApp.Flags = []cli.Flag{
		&cli.UintFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			Value:       8080,
			DefaultText: "listening port, default 8080",
		},
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Value:       false,
			DefaultText: "debug mode, default false",
		},
		&cli.StringFlag{
			Name:   "shost",
			Value:  "192.168.3.92",
			Hidden: true,
		},
	}

	mApp.Action = func(c *cli.Context) error {
		// 3. 设置配置
		config.SetDebug(c.Bool("debug"))
		config.SetHttpPort(c.Uint("port"))
		config.SetSwagHost(c.String("shost"))

		// 4. 初始化数据库
		db, err := initDatabase()
		if err != nil {
			logrus.Fatalf("数据库初始化失败: %v", err)
		}

		// 5. 初始化应用依赖
		routerInstance, cleanup, err := initAppDependencies(db)
		if err != nil {
			logrus.Fatalf("应用初始化失败: %v", err)
		}
		defer cleanup()

		// 6. 配置并启动服务
		server := configureServer(routerInstance)
		go gracefulStartServer(server)

		// 7. 处理优雅关闭
		waitForShutdown(server)

		return nil
	}

	// 8. 运行CLI应用
	if err := mApp.Run(os.Args); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

// initDatabase 初始化数据库连接
func initDatabase() (*gorm.DB, error) {
	db, err := router.InitDatabase()
	if err != nil {
		return nil, err
	}
	logrus.Info("数据库连接已建立")
	return db, nil
}

// initAppDependencies 初始化应用依赖
func initAppDependencies(db *gorm.DB) (*router.Router, func(), error) {
	// 初始化控制器
	userCtrl := controller.NewUserController(db)

	// 初始化服务
	client := controller.NewClient(db, userCtrl)
	agent := controller.NewAgent()
	emailCtrl := controller.NewEmailController(db, userCtrl, client, agent)

	// 初始化路由
	routerInstance := router.NewRouter(agent, client, userCtrl, emailCtrl)

	// 返回清理函数
	cleanup := func() {
		emailCtrl.StopStatusChecker()
		logrus.Info("已停止所有后台服务")
	}

	return routerInstance, cleanup, nil
}

// configureServer 配置并返回HTTP服务器
func configureServer(routerInstance *router.Router) *http.Server {
	engine := gin.Default()

	// 注册业务路由
	routerInstance.Register(engine)

	// 配置服务器
	return &http.Server{
		Addr:    ":" + strconv.Itoa(int(config.GlobalConfig().HttpPort)),
		Handler: engine,
	}
}

// gracefulStartServer 优雅启动服务器
func gracefulStartServer(server *http.Server) {
	logrus.Infof("服务器启动于 %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.Fatalf("服务器启动失败: %v", err)
	}
}

// waitForShutdown 等待并处理关闭信号
func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("服务器关闭失败: %v", err)
	} else {
		logrus.Info("服务器已正常关闭")
	}
}
