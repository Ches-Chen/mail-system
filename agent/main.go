package main

import (
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func main() {
	app := cli.NewApp()
	app.HideHelp = true
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "host",
			Aliases:  []string{"h"},
			Required: true,
			Value:    "192.168.3.92",
		},
		&cli.Uint64Flag{
			Name:     "port",
			Aliases:  []string{"p"},
			Required: false,
			Value:    8080,
		},
		&cli.BoolFlag{
			Name:     "debug",
			Aliases:  []string{"d"},
			Required: false,
			Value:    false,
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.ErrorLevel)
			//log.SetOutput(io.Discard)
		}

		client := resty.New()
		// TODO 暂用http
		client.SetBaseURL("http://" + c.String("host") + ":" + strconv.FormatUint(c.Uint64("port"), 10))

		eg, ctx := errgroup.WithContext(c.Context)
		// 健康检查
		eg.Go(func() error {
			if err := HealthCheck(ctx, client); err != nil {
				log.Warnf("health check error: %v", err)
				return err
			}

			return nil
		})

		// 处理邮件请求
		eg.Go(func() error {
			if err := SendEmail(ctx, client); err != nil {
				log.Warnf("mail handler error: %v", err)
				return err
			}

			return nil
		})

		if err := eg.Wait(); err != nil {
			return err
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}
