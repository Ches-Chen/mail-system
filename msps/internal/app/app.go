package app

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"msps/internal/app/config"
	"msps/internal/app/injector"
)

func Init(ctx context.Context) (func(), error) {
	// 使用wire构建依赖
	inj, cleanFunc, err := injector.BuildInjector(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build injector")
	}

	// 初始化HTTP服务器
	cleanHttp := initHttpServer(ctx, inj.Engine, config.GlobalConfig().HttpPort)

	return func() {
		cleanFunc()
		cleanHttp()
	}, nil
}

func Run(ctx context.Context) error {
	state := 1
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	cleanFunc, err := Init(ctx)
	if err != nil {
		return err
	}

EXIT:
	for {
		s := <-sig
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, os.Interrupt:
			state = 0
			break EXIT
		case syscall.SIGHUP:
		default:
			break EXIT
		}
	}

	cleanFunc()
	time.Sleep(time.Second)
	os.Exit(state)
	return nil
}

func initHttpServer(ctx context.Context, engine *gin.Engine, port uint) func() {
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        engine,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("HTTP server error: %v", err)
		}
	}()

	return func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Errorf("HTTP server shutdown error: %v", err)
		}
	}
}
