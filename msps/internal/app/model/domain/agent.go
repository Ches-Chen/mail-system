package domain

type EmailVerifyReq struct {
	ID      string `json:"id"`      // 邮件唯一标识
	Success bool   `json:"success"` // 邮件发送是否成功
}

type EmailProbeReq struct {
	Host    string `json:"host"`     // 探针触发的主机和端口
	Refer   string `json:"refer"`    // 来源页面地址
	Index   string `json:"index"`    // 唯一标识邮件ID
	UA      string `json:"ua"`       // 用户浏览器标识
	ProxyIP string `json:"proxy-ip"` // 用户代理IP
	RealIP  string `json:"real-ip"`  // 用户真实IP
}
