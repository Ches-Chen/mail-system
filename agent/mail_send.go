package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

const (
	mailSendUrl               = "/a/m"
	mailHandlerTimeoutSeconds = 5
	defaultSmtpPort           = mail.DefaultPort
)

type Response struct {
	Success bool      `json:"success"`
	Msg     string    `json:"msg"`
	Payload *EmailReq `json:"payload"`
}

// EmailAddress 邮箱信息
type EmailAddress struct {
	Name string `json:"name"` // 昵称
	Addr string `json:"addr"` // 邮箱地址
}

// FileAttachment 文件附件信息
type FileAttachment struct {
	ContentType mail.ContentType `json:"content_type"` // 文件类型
	Encoding    mail.Encoding    `json:"encoding"`     // 文件编码(Bas64, quoted-printable)
	Name        string           `json:"name"`         // 文件名
	Content     []byte           `json:"content"`      // 文件内容(base64)
}

// SMTPServer SMTP服务器
type SMTPServer struct {
	Host string `json:"host"`
	Port *int   `json:"port,omitempty"`
}

type SMTPAuth struct {
	User string `json:"user"` // 用户名
	Pass string `json:"pass"` // 密码
}

// EmailReq 邮件请求
type EmailReq struct {
	ID          string           `json:"id"`                   // 邮件唯一标识
	Server      SMTPServer       `json:"server"`               // 邮件服务器地址
	Auth        *SMTPAuth        `json:"auth,omitempty"`       // 认证
	From        *EmailAddress    `json:"from"`                 // 发件人
	To          []EmailAddress   `json:"to"`                   // 收件人列表
	CC          []EmailAddress   `json:"cc,omitempty"`         // 抄送列表
	BCC         []EmailAddress   `json:"bcc,omitempty"`        // 密送列表
	Priority    *mail.Importance `json:"priority,omitempty"`   // 消息优先级
	UserAgent   *string          `json:"user_agent,omitempty"` // X-Mailer
	ContentType mail.ContentType `json:"content_type"`         //邮件正文类型
	Encoding    *mail.Encoding   `json:"encoding,omitempty"`   // 邮件编码
	Subject     string           `json:"subject"`              // 邮件主题
	Body        string           `json:"body"`                 // 邮件正文
	Attachments []FileAttachment `json:"files,omitempty"`      // 附件列表
}

func parseContentType(contentType string) (mimeType, charset string, err error) {
	parts := strings.SplitN(contentType, ";", 2)
	mimeType = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		params := strings.Split(parts[1], ";")
		for _, param := range params {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(strings.ToLower(kv[0]))
				value := strings.TrimSpace(strings.Trim(kv[1], "\""))
				if key == "charset" {
					charset = value
					if !strings.EqualFold(charset, "utf-8") {
						return "", "", fmt.Errorf("unsupported charset: %s", charset)
					}
					break
				}
			}
		}
	}
	return mimeType, charset, nil
}

func SendEmail(ctx context.Context, client *resty.Client) error {
	ticker := time.NewTicker(time.Duration(mailHandlerTimeoutSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var reply Response
			// 获取邮件请求
			resp, err := client.R().
				SetContext(ctx).
				SetContentLength(true).
				SetResult(&reply).
				Post(mailSendUrl)
			if err != nil {
				log.Warnf("[SendEmail] 发送邮件请求错误: %v", err)
				continue
			}

			if resp.StatusCode() != http.StatusOK || !reply.Success || reply.Payload == nil {
				continue
			}

			emailReq := reply.Payload
			log.Debugf("[SendEmail] 收到邮件请求: %+v", emailReq)

			// 创建邮件消息
			msg := mail.NewMsg()

			// 发件人
			if err := msg.FromFormat(emailReq.From.Name, emailReq.From.Addr); err != nil {
				log.Warnf("[SendEmail] 发件人格式错误: %v", err)
				continue
			}

			// 收件人
			for _, to := range emailReq.To {
				if err := msg.AddToFormat(to.Name, to.Addr); err != nil {
					log.Warnf("[SendEmail] 收件人格式错误: %v", err)
				}
			}

			// 抄送
			for _, cc := range emailReq.CC {
				if err := msg.AddCcFormat(cc.Name, cc.Addr); err != nil {
					log.Warnf("[SendEmail] 抄送人格式错误: %v", err)
				}
			}

			// 密送
			for _, bcc := range emailReq.BCC {
				if err := msg.AddBccFormat(bcc.Name, bcc.Addr); err != nil {
					log.Warnf("[SendEmail] 密送人格式错误: %v", err)
				}
			}

			// 邮件内容处理
			mimeType, _, err := parseContentType(string(emailReq.ContentType))
			if err != nil {
				log.Warnf("[SendEmail] 内容类型解析错误: %v", err)
				continue
			}

			switch mimeType {
			case "text/plain":
				msg.SetBodyString(mail.TypeTextPlain, emailReq.Body)
			case "text/html":
				msg.SetBodyString(mail.TypeTextHTML, emailReq.Body)
			default:
				log.Warnf("[SendEmail] 不支持的内容类型: %s", mimeType)
				continue
			}

			// 附件
			if len(emailReq.Attachments) > 0 {
				files := make([]*mail.File, len(emailReq.Attachments))
				for i, file := range emailReq.Attachments {
					files[i] = &mail.File{
						ContentType: file.ContentType,
						Enc:         file.Encoding,
						Header:      make(textproto.MIMEHeader),
						Name:        file.Name,
						Writer: func(w io.Writer) (int64, error) {
							n, err := w.Write(file.Content)
							return int64(n), err
						},
					}
				}
				msg.SetAttachments(files)
			}

			// 创建SMTP客户端
			port := defaultSmtpPort
			if emailReq.Server.Port != nil {
				port = *emailReq.Server.Port
			}

			mailOpts := []mail.Option{
				mail.WithPort(port),
				mail.WithSSL(),                        // 强制SSL
				mail.WithTLSPolicy(mail.TLSMandatory), // 强制TLS
				mail.WithSMTPAuth(mail.SMTPAuthLogin), // 使用LOGIN认证
				mail.WithTimeout(10 * time.Second),    // 设置超时
			}

			// 根据端口设置SSL
			if port == 465 {
				mailOpts = append(mailOpts, mail.WithSSL())
			}

			mailClient, err := mail.NewClient(emailReq.Server.Host, mailOpts...)
			if err != nil {
				log.Warnf("[SendEmail] 创建邮件客户端失败: %v", err)
				continue
			}

			// 设置认证信息
			if emailReq.Auth != nil {
				mailClient.SetSMTPAuth(mail.SMTPAuthPlain)
				mailClient.SetUsername(emailReq.Auth.User)
				mailClient.SetPassword(emailReq.Auth.Pass)
			}

			// 发送邮件
			var sendSuccess bool
			if err := mailClient.DialAndSendWithContext(ctx, msg); err != nil {
				if isAuthError(err) {
					log.Errorf("[SendEmail] SMTP认证失败：%v", err)
					sendSuccess = false
				} else if isPostSubmissionError(err) {
					log.Debugf("[SendEmail] 邮件已成功提交（服务器连接已关闭）")
					sendSuccess = true // 标记为成功
				} else {
					log.Warnf("[SendEmail] 发送失败：%v", err)
					sendSuccess = false
				}
			} else {
				sendSuccess = true
			}

			// 确认邮件发送状态（带重试）
			const maxRetries = 3
			for i := 0; i < maxRetries; i++ {
				if err := VerifyEmail(ctx, client, &EmailVerifyReq{
					ID:      emailReq.ID,
					Success: sendSuccess,
				}); err != nil {
					log.Warnf("[VerifyEmail] 确认请求失败(尝试 %d/%d): %v", i+1, maxRetries, err)
					time.Sleep(1 * time.Second)
					continue
				}
				break
			}
		}
	}
}

// 增强错误判断逻辑
func isPostSubmissionError(err error) bool {
	errorPatterns := []string{
		"reset", "eof", "closed",
		"not connected", "already closed",
	}
	errMsg := strings.ToLower(err.Error())
	for _, pattern := range errorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "535") ||
		strings.Contains(err.Error(), "authentication failed")
}
