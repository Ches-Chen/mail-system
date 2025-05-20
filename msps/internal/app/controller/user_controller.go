package controller

import (
	"fmt"
	"gorm.io/gorm"
	"msps/internal/app/model/common"
	"msps/internal/app/model/domain"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (uc *UserController) Register(c *gin.Context) {
	var user domain.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error()),
		))
		return
	}

	// 手机号格式校验
	if !isValidPhone(user.Phone) {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("手机号格式不正确"),
		))
		return
	}

	if err := uc.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error()),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("用户注册成功"),
	))
}

func (uc *UserController) Login(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"msg":     "服务器内部错误",
			})
		}
	}()

	if uc.DB == nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("数据库连接未初始化"),
		))
		return
	}

	var input struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("请求参数错误"),
		))
		return
	}

	var user domain.User
	if err := uc.DB.Where("username = ? OR phone = ?", input.Identifier, input.Identifier).First(&user).Error; err != nil {
		response := common.NewResponse(
			common.WithSuccess(false),
		)

		if err == gorm.ErrRecordNotFound {
			response.Msg = "用户不存在"
			c.JSON(http.StatusUnauthorized, response)
		} else {
			response.Msg = "数据库查询错误"
			c.JSON(http.StatusInternalServerError, response)
		}
		return
	}

	// 检查用户状态
	if user.Status == "disabled" {
		c.JSON(http.StatusForbidden, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("账号已被封禁"),
		))
		return
	}

	if user.Password != input.Password {
		c.JSON(http.StatusUnauthorized, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("密码错误"),
		))
		return
	}

	user.LastLogin = time.Now()
	if err := uc.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("更新登录时间失败"),
		))
		return
	}

	token := "generated-token-" + user.Username

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("登录成功"),
		common.WithPayload(gin.H{
			"user":  user,
			"token": token,
		}),
	))
}

func (uc *UserController) UpdateUserInfo(c *gin.Context) {
	currentUserID, err := uc.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("用户未登录或会话已过期")))
		return
	}

	var input struct {
		Username string `json:"username"`
		Phone    string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	var user domain.User
	if err := uc.DB.First(&user, currentUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("用户不存在")))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg(err.Error())))
		}
		return
	}

	// 更新用户名
	if input.Username != "" {
		// 检查用户名唯一性
		var count int64
		if err := uc.DB.Model(&domain.User{}).Where("username = ? AND id != ?", input.Username, currentUserID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("用户名唯一性检查失败"),
			))
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("用户名已存在"),
			))
			return
		}
		user.Username = input.Username
	}

	// 更新手机号
	if input.Phone != "" {
		if !isValidPhone(input.Phone) {
			c.JSON(http.StatusBadRequest, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("手机号格式不正确"),
			))
			return
		}
		// 检查手机号唯一性
		var count int64
		if err := uc.DB.Model(&domain.User{}).Where("phone = ? AND id != ?", input.Phone, currentUserID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("手机号唯一性检查失败"),
			))
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("手机号已存在"),
			))
			return
		}
		user.Phone = input.Phone
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("用户信息更新失败"),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("用户信息更新成功"),
		common.WithPayload(user),
	))
}

// GetCurrentUserID 获取当前用户ID的辅助函数
func (uc *UserController) GetCurrentUserID(c *gin.Context) (int64, error) {
	username, exists := c.Get("username")
	if !exists {
		return 0, fmt.Errorf("用户未登录")
	}

	var user domain.User
	if err := uc.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("用户不存在")
		}
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}
	return user.ID, nil
}

func (uc *UserController) GetUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Invalid page number")))
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Invalid limit number")))
		return
	}

	offset := (page - 1) * limit

	var users []domain.User
	var total int64

	// 获取总记录数
	if err := uc.DB.Model(&domain.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	// 获取分页数据
	if err := uc.DB.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	// 创建分页响应结构体
	paginatedData := common.PaginatedResponse{
		Users: users,
		Total: total,
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(paginatedData),
	))
}

func (uc *UserController) AssignRole(c *gin.Context) {
	// 获取当前用户ID
	currentUserID, err := uc.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var input struct {
		ID   uint   `json:"id"`
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	// 检查是否尝试修改自己
	if input.ID == uint(currentUserID) {
		c.JSON(http.StatusForbidden, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Cannot modify your own role")))
		return
	}

	// 验证角色值
	if input.Role != "admin" && input.Role != "user" {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Invalid role value")))
		return
	}

	// 直接更新role字段，不查询用户
	result := uc.DB.Model(&domain.User{}).
		Where("id = ?", input.ID).
		Update("role", input.Role)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(result.Error.Error())))
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("User not found")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("用户角色更新成功"),
	))
}

func (uc *UserController) ManageUserStatus(c *gin.Context) {

	// 获取当前用户ID
	currentUserID, err := uc.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(common.WithMsg("用户未登录")))
		return
	}

	var input struct {
		ID     uint   `json:"id"`
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	// 检查是否尝试修改自己
	if input.ID == uint(currentUserID) {
		c.JSON(http.StatusForbidden, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Cannot modify your own status")))
		return
	}

	result := uc.DB.Model(&domain.User{}).
		Where("id = ?", input.ID).
		Update("status", input.Status)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(result.Error.Error())))
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("User not found")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("用户状态更新成功")))
}

func (uc *UserController) SearchUsers(c *gin.Context) {
	type SearchRequest struct {
		Keyword string `json:"keyword"`
		Page    int    `json:"page"`
		Limit   int    `json:"limit"`
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("Invalid request body"),
		))
		return
	}

	offset := (req.Page - 1) * req.Limit

	var users []domain.User
	if err := uc.DB.Where("username LIKE ? OR phone LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		Offset(offset).
		Limit(req.Limit).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithPayload(users),
	))
}

func (uc *UserController) UpdateUserProfile(c *gin.Context) {

	var input struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Phone    string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	var user domain.User
	if err := uc.DB.First(&user, input.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("User not found")))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg(err.Error())))
		}
		return
	}

	// 更新用户信息
	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Phone != "" {
		user.Phone = input.Phone
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("用户信息更新成功"),
	))
}

// 中国手机号格式验证（11位，以1开头）
func isValidPhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	match, _ := regexp.MatchString(pattern, phone)
	return match
}

func (uc *UserController) ForgetPwd(c *gin.Context) {
	var input struct {
		Identifier  string `json:"identifier" binding:"required"` // 可以是用户名或手机号
		Email       string `json:"email" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=4"`
	}

	// 绑定输入参数
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("请求参数错误: "+err.Error()),
		))
		return
	}

	// 验证新密码长度
	if len(input.NewPassword) < 4 {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("密码长度不能少于4位"),
		))
		return
	}

	// 开启事务
	tx := uc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 验证用户名或手机号是否存在
	var user domain.User
	query := tx.Where("username = ? OR phone = ?", input.Identifier, input.Identifier)
	if err := query.First(&user).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("用户名或手机号不存在"),
			))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("数据库查询错误"),
			))
		}
		return
	}

	// 2. 验证邮箱是否属于该用户
	var mailAccount domain.UserMailAccount
	if err := tx.Where("user_id = ? AND email = ?", user.ID, input.Email).First(&mailAccount).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("该邮箱未绑定到此用户"),
			))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("邮箱验证失败"),
			))
		}
		return
	}

	// 3. 更新密码
	user.Password = input.NewPassword
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("密码更新失败"),
		))
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("密码重置失败"),
		))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("密码重置成功"),
	))
}

// VerifyPassword 验证用户密码
func (uc *UserController) VerifyPassword(c *gin.Context) {
	currentUserID, err := uc.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("用户未登录或会话已过期")))
		return
	}
	log.Printf("Received Authorization header: %s", currentUserID)

	var input struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg(err.Error())))
		return
	}

	var user domain.User
	if err := uc.DB.First(&user, currentUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg("用户不存在")))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(
				common.WithSuccess(false),
				common.WithMsg(err.Error())))
		}
		return
	}

	if user.Password != input.Password {
		c.JSON(http.StatusUnauthorized, common.NewResponse(
			common.WithSuccess(false),
			common.WithMsg("密码不正确")))
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(
		common.WithSuccess(true),
		common.WithMsg("密码验证成功")))
}
