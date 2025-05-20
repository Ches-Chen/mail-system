package controller

import (
	"fmt"
	"gorm.io/gorm"
	"msps/internal/app/model/common"
	"msps/internal/app/model/domain"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type EmailRecordsResponse struct {
	EmailRecords []RecordResponse `json:"email_records"`
	Pagination   Pagination       `json:"pagination"`
}

type Pagination struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
}

type RecordResponse struct {
	FromUsername string    `json:"from_username"`
	FromEmail    string    `json:"from_email"`
	ToUsername   string    `json:"to_username"`
	ToEmail      string    `json:"to_email"`
	Status       string    `json:"status"`
	SentAt       time.Time `json:"sent_at"`
}

// UserMailAccountResponse 用户邮箱账户响应结构
type UserMailAccountResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (ec *EmailController) GetEmailRecords(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := ec.DB.Table("email_records").
		Select("users.username AS from_username, email_records.from_email, to_users.username AS to_username, email_records.to_email, email_records.status, email_records.sent_at").
		Joins("JOIN users ON users.id = email_records.from_user_id").
		Joins("LEFT JOIN users AS to_users ON to_users.id = email_records.to_user_id")

	// 根据状态筛选
	if status != "" {
		query = query.Where("email_records.status = ?", status)
	}

	// 根据时间范围筛选
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			query = query.Where("email_records.sent_at >= ?", startDate)
		}
	}
	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = endDate.AddDate(0, 0, 1) // 包含结束日期当天的所有记录
			query = query.Where("email_records.sent_at < ?", endDate)
		}
	}

	// 查询符合条件的记录总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("获取记录总数失败"),
		))
		return
	}

	var records []RecordResponse
	if err := query.Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("获取记录失败"),
		))
		return
	}

	pagination := Pagination{
		CurrentPage: page,
		PerPage:     limit,
		Total:       int(total),
	}

	response := EmailRecordsResponse{
		EmailRecords: records,
		Pagination:   pagination,
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(response),
	))
}

func (ec *EmailController) ManageBlacklist(c *gin.Context) {
	var input struct {
		Email  string `json:"email"`
		Reason string `json:"reason"`
		Action string `json:"action"` // add or remove
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error()),
		))
		return
	}

	switch input.Action {
	case "add":
		var blacklist domain.Blacklist
		blacklist.Email = input.Email
		blacklist.Reason = input.Reason
		if err := ec.DB.Create(&blacklist).Error; err != nil {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg(err.Error()),
			))
			return
		}
		c.JSON(http.StatusOK, common.NewResponse(
			common.WithSuccess(true),
			common.WithMsg("邮箱已加入黑名单"),
		))
	case "remove":
		if err := ec.DB.Where("email = ?", input.Email).Delete(&domain.Blacklist{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg(err.Error()),
			))
			return
		}
		c.JSON(http.StatusOK, common.NewResponse(
			common.WithSuccess(true),
			common.WithMsg("邮箱已从黑名单移除"),
		))
	default:
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("无效操作，允许的操作：add, remove"),
		))
	}
}

// GetUserMailAccounts 获取用户邮箱账户列表
func (ec *EmailController) GetUserMailAccounts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := ec.DB.Model(&domain.UserMailAccount{}).
		Select("user_mail_accounts.*, users.username").
		Joins("left join users on users.id = user_mail_accounts.user_id")

	// 搜索条件
	if search != "" {
		query = query.Where("users.username LIKE ? OR user_mail_accounts.email LIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("获取记录总数失败"),
		))
		return
	}

	var accounts []UserMailAccountResponse
	if err := query.Offset(offset).Limit(limit).Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("获取邮箱账户失败"),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(map[string]interface{}{
			"accounts": accounts,
			"pagination": map[string]interface{}{
				"current_page": page,
				"per_page":     limit,
				"total":        total,
			},
		}),
	))
}

// UpdateMailAccountStatus 更新邮箱账户状态
func (ec *EmailController) UpdateMailAccountStatus(c *gin.Context) {
	var input struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Reason string `json:"reason"` // 新增reason字段，用于禁用时提供原因
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error()),
		))
		return
	}

	// 验证状态值
	if input.Status != "active" && input.Status != "disabled" {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("无效的状态值"),
		))
		return
	}

	// 获取当前用户ID
	userID, err := ec.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error()),
		))
		return
	}

	// 开启事务
	tx := ec.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新状态
	if err := tx.Model(&domain.UserMailAccount{}).
		Where("id = ?", input.ID).
		Update("status", input.Status).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("更新状态失败"),
		))
		return
	}

	// 如果是禁用操作，需要将邮箱加入黑名单
	if input.Status == "disabled" {
		// 先查询邮箱地址
		var mailAccount domain.UserMailAccount
		if err := tx.Where("id = ?", input.ID).First(&mailAccount).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("获取邮箱信息失败"),
			))
			return
		}

		// 检查是否已在黑名单中
		var existingBlacklist domain.Blacklist
		if err := tx.Table("blacklist").Where("email = ?", mailAccount.Email).First(&existingBlacklist).Error; err == nil {
			// 已存在黑名单中，更新原因
			if err := tx.Table("blacklist").
				Where("email = ?", mailAccount.Email).
				Updates(map[string]interface{}{
					"reason":     input.Reason,
					"created_by": userID,
				}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, common.NewResponse(
					common.WithSuccess(false),
					common.WithMsg("更新黑名单失败"),
				))
				return
			}
		} else {
			// 不存在黑名单中，创建新记录
			blacklist := domain.Blacklist{
				Email:     mailAccount.Email,
				Reason:    input.Reason,
				CreatedBy: userID,
				CreatedAt: time.Now(),
			}
			if err := tx.Table("blacklist").Create(&blacklist).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, common.NewResponse(
					common.WithSuccess(false),
					common.WithMsg("添加到黑名单失败"),
				))
				return
			}
		}
	} else if input.Status == "active" {
		// 如果是激活操作，从黑名单中移除
		var mailAccount domain.UserMailAccount
		if err := tx.Where("id = ?", input.ID).First(&mailAccount).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("获取邮箱信息失败"),
			))
			return
		}

		if err := tx.Table("blacklist").Where("email = ?", mailAccount.Email).Delete(&domain.Blacklist{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("从黑名单移除失败"),
			))
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("提交事务失败"),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("状态更新成功"),
	))
}

// GetCurrentUserID 获取当前用户ID
func (ec *EmailController) GetCurrentUserID(c *gin.Context) (int64, error) {
	username, exists := c.Get("username")
	if !exists {
		return 0, fmt.Errorf("用户未登录")
	}

	var user domain.User
	if err := ec.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("用户不存在")
		}
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}
	return user.ID, nil
}

// GetBlacklist 获取黑名单列表
func (ec *EmailController) GetBlacklist(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// 使用Table明确指定表名
	query := ec.DB.Table("blacklist").
		Select("blacklist.*, users.username as created_by_name").
		Joins("LEFT JOIN users ON users.id = blacklist.created_by")

	// 搜索条件
	if search != "" {
		query = query.Where("blacklist.email LIKE ?", "%"+search+"%")
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(fmt.Sprintf("获取记录总数失败: %v", err)),
		))
		return
	}

	var blacklist []struct {
		domain.Blacklist
		CreatedByName string `json:"created_by_name"`
	}

	if err := query.Offset(offset).Limit(limit).Find(&blacklist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(fmt.Sprintf("获取黑名单失败: %v", err)),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(map[string]interface{}{
			"blacklist": blacklist,
			"pagination": map[string]interface{}{
				"current_page": page,
				"per_page":     limit,
				"total":        total,
			},
		}),
	))
}
