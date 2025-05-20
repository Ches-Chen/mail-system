package router

import (
	"github.com/gin-gonic/gin"
	"msps/internal/app/middleware"
)

func (r *Router) RegisterClientApi(engine *gin.Engine) {

	p := engine.Group("/c")

	p.POST("/register", r.UserCtrl.Register)
	p.POST("/login", r.UserCtrl.Login)
	p.POST("/forget_pwd", r.UserCtrl.ForgetPwd)

	g := engine.Group("/c")
	g.Use(middleware.AuthMiddleware())
	{
		// 邮件相关
		g.POST("/email/send", r.ClientApi.HandleSentEmail)
		g.POST("/check_blacklist", r.ClientApi.CheckEmailBlacklist)
		g.POST("/email/:id/verify", r.ClientApi.HandleVerifyEmail)

		// 邮件管理
		e := g.Group("/email")
		{
			e.GET("/get_all_email_records", r.EmailCtrl.GetEmailRecords)
			e.POST("/blacklist", r.EmailCtrl.ManageBlacklist)
			e.GET("/get_mail", r.EmailCtrl.GetUserMailAccounts)
			e.POST("/update_mail_status", r.EmailCtrl.UpdateMailAccountStatus)
			e.GET("/get_black", r.EmailCtrl.GetBlacklist)

			e.GET("/accounts", r.ClientApi.HandleListEmailAccounts)
			e.POST("/add/accounts", r.ClientApi.HandleCreateEmailAccount)
			e.PUT("/accounts/:id", r.ClientApi.HandleUpdateEmailAccount)
			e.DELETE("/delete/accounts/:id", r.ClientApi.HandleDeleteEmailAccount)
		}

		// 用户管理
		u := g.Group("/users")
		{
			u.GET("/get_userinfo", r.ClientApi.HandleGetUserInfo)
			u.POST("/update_userinfo", r.UserCtrl.UpdateUserInfo)
			u.POST("/verify_password", r.UserCtrl.VerifyPassword)

			u.GET("/get_all_users", r.UserCtrl.GetUsers)
			u.POST("/assign_role", r.UserCtrl.AssignRole)
			u.POST("/manage_status", r.UserCtrl.ManageUserStatus)
			u.POST("/search_user", r.UserCtrl.SearchUsers)
			u.POST("/update_userprofile", r.UserCtrl.UpdateUserProfile)
		}
	}
}
