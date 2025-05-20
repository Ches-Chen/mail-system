package main

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
)

const (
	mailVerifyUrl = "/a/v"
)

type EmailVerifyReq struct {
	ID      string `json:"id"`      // 邮件唯一标识
	Success bool   `json:"success"` // 发送结果是否成功
}

func VerifyEmail(ctx context.Context, client *resty.Client, req *EmailVerifyReq) error {
	resp, err := client.R().
		SetContext(ctx).
		SetContentLength(true).
		SetBody(req).
		Post(mailVerifyUrl)
	if err != nil {
		return fmt.Errorf("verify request failed: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("verify request failed with status %d", resp.StatusCode())
	}

	return nil
}
