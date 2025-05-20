package domain

import "github.com/wneessen/go-mail"

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
	ID          string           `json:"id"`                 // 邮件唯一标识
	Server      SMTPServer       `json:"server"`             // 邮件服务器地址
	Auth        *SMTPAuth        `json:"auth,omitempty"`     // 认证
	From        *EmailAddress    `json:"from"`               // 发件人
	To          []EmailAddress   `json:"to"`                 // 收件人列表
	CC          []EmailAddress   `json:"cc,omitempty"`       // 抄送列表
	BCC         []EmailAddress   `json:"bcc,omitempty"`      // 密送列表
	ContentType mail.ContentType `json:"content_type"`       // 邮件正文类型
	Encoding    *mail.Encoding   `json:"encoding,omitempty"` // 邮件编码
	Subject     string           `json:"subject"`            // 邮件主题
	Body        string           `json:"body"`               // 邮件正文
	Attachments []FileAttachment `json:"files,omitempty"`    // 附件列表
}
