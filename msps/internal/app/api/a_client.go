package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
	"gorm.io/gorm"
	"io"
	"msps/internal/app/model/common"
	"msps/internal/app/model/domain"
	"net/http"
	"sync"
	"time"
)

const (
	defaultQueueCapacity              = 1 << 10
	StatusUnknown        VerifyStatus = 0 // 未返回邮件发送状态
	StatusSuccess        VerifyStatus = 1 // 发送成功
	StatusFailed         VerifyStatus = 2 // 发送失败
)

// EmailCheckRequest 定义检查邮箱请求结构
type EmailCheckRequest struct {
	Sender     string   `json:"sender" binding:"required,email"`
	Recipients []string `json:"recipients" binding:"required,min=1"`
}

type VerifyStatus int

type MailQueue struct {
	Queue          chan domain.EmailReq
	QueueCapacity  int
	remainingCount int
	mu             sync.Mutex
}

func NewMailQueue(capacity int) *MailQueue {
	return &MailQueue{
		Queue:          make(chan domain.EmailReq, capacity),
		QueueCapacity:  capacity,
		remainingCount: capacity,
	}
}

func (q *MailQueue) Enqueue(req domain.EmailReq) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.remainingCount == 0 {
		return errQueueFull
	}

	q.Queue <- req
	q.remainingCount--
	return nil
}

func (m *MailVerifyMap) CheckMap(id string) VerifyStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	if res, ok := m.Map[id]; ok {
		if res.Success {
			return StatusSuccess
		}
		return StatusFailed
	}
	return StatusUnknown
}

func (m *MailProbeMap) CheckMap(id string) (domain.EmailProbeReq, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if res, ok := m.Map[id]; ok {
		delete(m.Map, id)
		return res, true
	}
	return domain.EmailProbeReq{}, false
}

var (
	EmailQueue   = NewMailQueue(defaultQueueCapacity)
	errQueueFull = errors.New("queue is full")
)

type Client struct {
	DB *gorm.DB
}

// HandleSentEmail
// @Summary 邮件发送处理
// @Description 处理来自Client的邮件发送请求
// @tags Client
// @Accept multipart/form-data
// @Produce json
// @Param data formData string true "邮件请求参数，作为formData的'data'字段传递"
// @Param attachments formData file false "邮件附件，作为formData的'attachments'字段传递（可选）"
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"访问受限","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"路径不存在","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /c/email/send [post]
func (a *Client) HandleSentEmail(c *gin.Context) {
	// 验证用户登录状态
	_, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("表单解析错误")))
		return
	}

	// 处理请求时从 data 字段读取完整 JSON
	dataFormValue := c.Request.FormValue("data")
	if dataFormValue == "" {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("缺少data字段")))
		return
	}

	// 反序列化到 EmailReq 结构体
	var req domain.EmailReq
	if err := json.Unmarshal([]byte(dataFormValue), &req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("data字段格式错误")))
		return
	}

	// 1. 检查发件人邮箱是否在黑名单中
	if a.isEmailBlacklisted(req.From.Addr) {
		c.JSON(http.StatusForbidden, common.NewResponse(
			common.WithMsg(fmt.Sprintf("发件邮箱 %s 已被封禁", req.From.Addr)),
		))
		return
	}

	// 2. 检查收件人邮箱是否在黑名单中
	var blacklistedRecipients []string
	allRecipients := append(append(req.To, req.CC...), req.BCC...)
	for _, recipient := range allRecipients {
		if a.isEmailBlacklisted(recipient.Addr) {
			blacklistedRecipients = append(blacklistedRecipients, recipient.Addr)
		}
	}

	if len(blacklistedRecipients) > 0 {
		errorMsg := "以下邮箱已被封禁:\n"
		for _, email := range blacklistedRecipients {
			errorMsg += fmt.Sprintf("%s\n", email)
		}
		c.JSON(http.StatusForbidden, common.NewResponse(
			common.WithMsg(errorMsg),
		))
		return
	}

	// 附件处理
	files, ok := c.Request.MultipartForm.File["attachments"]
	if ok && len(files) > 0 {
		for _, fileHeader := range files {
			var fileAttachment domain.FileAttachment
			fileAttachment.Name = fileHeader.Filename

			fileHandle, err := fileHeader.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("附件头信息错误")))
				return
			}

			if req.Encoding != nil {
				fileAttachment.Encoding = *req.Encoding
			}

			fileAttachment.ContentType = mail.ContentType(fileHeader.Header.Get("Content-Type"))

			fileContent, err := io.ReadAll(fileHandle)
			if err != nil {
				c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("附件读取错误")))
				return
			}

			if err := fileHandle.Close(); err != nil {
				log.Printf("关闭附件文件失败: %v", err)
			}

			fileAttachment.Content = fileContent
			req.Attachments = append(req.Attachments, fileAttachment)
		}
	}

	// 将请求加入队列
	if err := EmailQueue.Enqueue(req); err != nil {
		if errors.Is(err, errQueueFull) {
			c.JSON(http.StatusTooManyRequests, common.NewResponse(common.WithMsg("队列已满")))
			return
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg(common.MsgInternalServerError)))
			return
		}
	}

	// 保存邮件记录到数据库
	if err := a.saveEmailRecords(req); err != nil {
		log.Printf("保存邮件记录失败: %v", err)
		// 这里不返回错误，因为邮件已经成功加入队列
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true)))
}

// HandleVerifyEmail
// @Summary 邮件发送结果确认
// @Description 处理来自Client的邮件确认请求
// @tags Client
// @Accept json
// @Produce json
// @Param id path string true "邮件唯一标识"
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"访问受限","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"路径不存在","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /c/email/{id}/verify [post]
func (a *Client) HandleVerifyEmail(c *gin.Context) {
	id := c.Param("id")

	// 首先检查数据库中的状态
	var record domain.EmailRecord
	if err := a.DB.Where("email_req_id = ?", id).First(&record).Error; err == nil {
		if record.Status != "pending" {
			c.JSON(http.StatusOK, common.NewResponse(
				common.WithSuccess(true),
				common.WithPayload(map[string]interface{}{
					"status": record.Status,
					"msg":    "邮件状态已更新",
				})))
			return
		}
	}

	// 如果数据库中没有或仍为pending，检查VerifyMap
	status := VerifyMap.CheckMap(id)

	type Payload struct {
		Status string `json:"status"`
		Msg    string `json:"msg,omitempty"`
	}

	switch status {
	case StatusUnknown:
		c.JSON(http.StatusOK, common.NewResponse(
			common.WithSuccess(true),
			common.WithPayload(Payload{"pending", "邮件仍在处理中"})))
	case StatusSuccess:
		// 更新数据库
		a.updateEmailStatus(id, "success")
		c.JSON(http.StatusOK, common.NewResponse(
			common.WithSuccess(true),
			common.WithPayload(Payload{"success", "发送成功"})))
	case StatusFailed:
		// 更新数据库
		a.updateEmailStatus(id, "fail")
		c.JSON(http.StatusOK, common.NewResponse(
			common.WithSuccess(true),
			common.WithPayload(Payload{"fail", "发送失败"})))
	}
}

// CheckEmailBlacklist
// @Summary 检查邮箱黑名单状态
// @Description 检查一个或多个邮箱是否在黑名单中
// @tags Client
// @Accept json
// @Produce json
// @Param data body EmailCheckRequest true "要检查的邮箱列表"
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":{"blacklisted_emails":["test@example.com"],"valid_emails":["valid@example.com"]}}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Router /c/email/check-blacklist [post]
func (a *Client) CheckEmailBlacklist(c *gin.Context) {
	var req EmailCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("请求参数错误")))
		return
	}

	// 检查发件人是否在黑名单中
	var blacklistedSender string
	if a.isEmailBlacklisted(req.Sender) {
		blacklistedSender = req.Sender
	}

	// 去重收件人邮箱
	uniqueRecipients := make(map[string]bool)
	for _, email := range req.Recipients {
		uniqueRecipients[email] = true
	}

	// 查询收件人黑名单
	var blacklistedRecipients []string
	for email := range uniqueRecipients {
		if a.isEmailBlacklisted(email) {
			blacklistedRecipients = append(blacklistedRecipients, email)
		}
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(map[string]interface{}{
			"blacklisted_sender":     blacklistedSender,
			"blacklisted_recipients": blacklistedRecipients,
		}),
	))
}

// HandleListEmailAccounts
// @Summary 获取用户邮箱列表
// @Description 根据用户ID获取关联的邮箱账户
// @tags Email Account
// @Produce json
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":[{"id":1,"user_id":1,"email":"test@example.com","display_name":"Test Name","auth_code":"xxx"}]}
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Router /c/email/accounts [get]
func (a *Client) HandleListEmailAccounts(c *gin.Context) {
	userID, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var accounts []domain.UserMailAccount
	if err := a.DB.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg("数据库查询失败")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true), common.WithPayload(accounts)))
}

// HandleCreateEmailAccount
// @Summary 添加新邮箱账户
// @Description 为当前用户添加新的邮箱账户
// @tags Email Account
// @Accept json
// @Produce json
// @Param data body domain.UserMailAccount true "邮箱账户信息"
// @Success 200 {object} common.Response "{"success":true,"msg":"添加成功","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 409 {object} common.Response "{"success":false,"msg":"邮箱已存在","data":null}"
// @Router /c/email/accounts [post]
func (a *Client) HandleCreateEmailAccount(c *gin.Context) {
	userID, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var account domain.UserMailAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("参数错误")))
		return
	}

	account.UserID = userID

	// 检查邮箱唯一性
	if err := a.DB.Where("email = ?", account.Email).First(&domain.UserMailAccount{}).Error; err == nil {
		c.JSON(http.StatusConflict, common.NewResponse(common.WithMsg("邮箱已存在")))
		return
	}

	if err := a.DB.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg("创建失败")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true), common.WithMsg("添加成功")))
}

// HandleUpdateEmailAccount
// @Summary 更新邮箱账户信息
// @Description 更新指定邮箱账户的信息
// @tags Email Account
// @Accept json
// @Produce json
// @Param id path int true "邮箱账户ID"
// @Param data body domain.UserMailAccount true "更新的邮箱账户信息"
// @Success 200 {object} common.Response "{"success":true,"msg":"更新成功","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"无权限操作","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"记录不存在","data":null}"
// @Failure 409 {object} common.Response "{"success":false,"msg":"邮箱已存在","data":null}"
// @Router /c/email/accounts/{id} [put]
func (a *Client) HandleUpdateEmailAccount(c *gin.Context) {
	userID, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var account domain.UserMailAccount
	if err := a.DB.First(&account, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, common.NewResponse(common.WithMsg("记录不存在")))
		return
	}

	// 检查权限
	if account.UserID != userID {
		c.JSON(http.StatusForbidden, common.NewResponse(common.WithMsg("无权限操作")))
		return
	}

	var updateData domain.UserMailAccount
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg("参数错误")))
		return
	}

	// 检查邮箱唯一性（排除当前记录）
	if updateData.Email != account.Email {
		if err := a.DB.Where("email = ? AND id != ?", updateData.Email, account.ID).First(&domain.UserMailAccount{}).Error; err == nil {
			c.JSON(http.StatusConflict, common.NewResponse(common.WithMsg("邮箱已存在")))
			return
		}
	}

	if err := a.DB.Model(&account).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg("更新失败")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true), common.WithMsg("更新成功")))
}

// HandleDeleteEmailAccount
// @Summary 删除邮箱账户
// @Description 删除指定的邮箱账户
// @tags Email Account
// @Produce json
// @Param id path int true "邮箱账户ID"
// @Success 200 {object} common.Response "{"success":true,"msg":"删除成功","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"无权限操作","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"记录不存在","data":null}"
// @Router /c/email/accounts/{id} [delete]
func (a *Client) HandleDeleteEmailAccount(c *gin.Context) {
	userID, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var account domain.UserMailAccount
	if err := a.DB.First(&account, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, common.NewResponse(common.WithMsg("记录不存在")))
		return
	}

	// 检查权限
	if account.UserID != userID {
		c.JSON(http.StatusForbidden, common.NewResponse(common.WithMsg("无权限操作")))
		return
	}

	if err := a.DB.Delete(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg("删除失败")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true), common.WithMsg("删除成功")))
}

// HandleGetUserInfo
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @tags User
// @Produce json
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":{"id":1,"username":"testuser","email":"test@example.com","created_at":"2023-01-01T00:00:00Z"}}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /c/user/info [get]
func (a *Client) HandleGetUserInfo(c *gin.Context) {
	// 获取当前用户ID
	userID, err := a.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	// 查询用户信息
	var user domain.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, common.NewResponse(common.WithMsg("用户不存在")))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg("数据库查询失败")))
		return
	}

	userInfo := map[string]interface{}{
		"username": user.Username,
		"password": user.Password,
		"phone":    user.Phone,
		"role":     user.Role,
		"status":   user.Status,
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(userInfo),
	))
}

// GetCurrentUserID 获取当前用户ID的辅助函数
func (a *Client) GetCurrentUserID(c *gin.Context) (int64, error) {
	username, exists := c.Get("username")

	if !exists {
		return 0, fmt.Errorf("用户未登录")
	}

	var user domain.User
	if err := a.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("用户不存在")
		}
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}
	return user.ID, nil
}

func (a *Client) updateEmailStatus(emailReqID string, status string) {
	if err := a.DB.Model(&domain.EmailRecord{}).
		Where("email_req_id = ?", emailReqID).
		Updates(map[string]interface{}{
			"status":          status,
			"sent_at":         time.Now(),
			"last_checked_at": time.Now(),
		}).Error; err != nil {
		log.Printf("Failed to update email status for %s: %v", emailReqID, err)
	}
}

func (a *Client) saveEmailRecords(req domain.EmailReq) error {
	// 1. 获取发件人用户ID
	fromUserID, err := a.getUserIDByEmail(req.From.Addr)
	if err != nil {
		return fmt.Errorf("failed to get sender user ID for email %s: %v", req.From.Addr, err)
	}

	var records []domain.EmailRecord

	// 2. 处理To收件人
	for _, to := range req.To {
		if err := a.processRecipient(fromUserID, req, to.Addr, "to", &records); err != nil {
			log.Printf("Failed to process TO recipient %s: %v", to.Addr, err)
		}
	}

	// 3. 处理CC收件人
	for _, cc := range req.CC {
		if err := a.processRecipient(fromUserID, req, cc.Addr, "cc", &records); err != nil {
			log.Printf("Failed to process CC recipient %s: %v", cc.Addr, err)
		}
	}

	// 4. 处理BCC收件人
	for _, bcc := range req.BCC {
		if err := a.processRecipient(fromUserID, req, bcc.Addr, "bcc", &records); err != nil {
			log.Printf("Failed to process BCC recipient %s: %v", bcc.Addr, err)
		}
	}

	// 5. 批量插入记录
	if len(records) > 0 {
		if err := a.DB.Create(&records).Error; err != nil {
			return fmt.Errorf("failed to batch insert email records: %v", err)
		}
		log.Printf("Successfully saved %d email records for request ID: %s", len(records), req.ID)
	} else {
		log.Printf("No valid recipients found for email request ID: %s", req.ID)
	}

	return nil
}

// processRecipient 处理单个收件人
func (a *Client) processRecipient(
	fromUserID int64,
	req domain.EmailReq,
	email string,
	recipientType string,
	records *[]domain.EmailRecord,
) error {
	// 1. 获取收件人用户ID
	toUserID, err := a.getUserIDByEmail(email)
	if err != nil {
		return fmt.Errorf("user ID not found for email %s: %v", email, err)
	}

	// 2. 检查黑名单
	if a.isEmailBlacklisted(email) {
		return fmt.Errorf("email %s is blacklisted", email)
	}

	// 3. 添加到记录列表
	*records = append(*records, domain.EmailRecord{
		FromUserID:    fromUserID,
		FromEmail:     req.From.Addr,
		ToUserID:      toUserID,
		ToEmail:       email,
		RecipientType: recipientType,
		Status:        "pending",
		EmailReqID:    req.ID,
		SentAt:        time.Now(),
	})

	return nil
}

// isEmailBlacklisted 检查邮箱是否在黑名单中
func (a *Client) isEmailBlacklisted(email string) bool {
	var count int64

	if err := a.DB.Table("blacklist").
		Where("email = ?", email).
		Count(&count).Error; err != nil {
		log.Printf("Failed to check blacklist for email %s: %v", email, err)
		return false
	}
	return count > 0
}

func (a *Client) getUserIDByEmail(email string) (int64, error) {
	var userID int64

	// 直接查询user_id字段，不扫描整个结构体
	err := a.DB.Model(&domain.UserMailAccount{}).
		Select("user_id").
		Where("email = ?", email).
		Pluck("user_id", &userID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("email %s not found", email)
		}
		return 0, fmt.Errorf("database error: %v", err)
	}

	return userID, nil
}
