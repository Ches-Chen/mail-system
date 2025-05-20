package main

import (
	"context"
	"github.com/denisbrodbeck/machineid"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

const (
	healthCheckUrl            = "/a/h"
	healthCheckTimeoutSeconds = 5
)

type HeartbeatReq struct {
	ID       string `json:"id"`       // agent唯一标识
	Hostname string `json:"hostname"` // agent所在主机名
}

// HealthCheck 健康检查
func HealthCheck(ctx context.Context, client *resty.Client) error {
	ticker := time.NewTicker(healthCheckTimeoutSeconds * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			hostname, err := os.Hostname()
			if err != nil {
				log.Warnf("Failed to get hostname: %v", err)
				break
			}

			agentId, err := machineid.ID()
			if err != nil {
				log.Warnf("Failed to get machine id: %v", err)
				break
			}

			_, _ = client.R().SetContext(ctx).
				SetContentLength(true).
				SetBody(HeartbeatReq{
					ID:       agentId,
					Hostname: hostname,
				}).
				SetDoNotParseResponse(true).
				Post(healthCheckUrl)
		}
	}
}
