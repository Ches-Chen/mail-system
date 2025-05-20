package domain

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Password  string    `gorm:"not null" json:"password"`
	Phone     string    `gorm:"unique;not null;size:15" json:"phone"`
	Role      string    `gorm:"type:enum('user', 'admin');default:'user'" json:"role"`
	Status    string    `gorm:"type:enum('active', 'disabled');default:'active'" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
	LastLogin time.Time `gorm:"default:null" json:"last_login"`
}

type UserMailAccount struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"not null" json:"user_id"`
	Email       string    `gorm:"unique;not null" json:"email"`
	AuthCode    string    `gorm:"not null" json:"auth_code"`
	DisplayName string    `gorm:"default:null" json:"display_name"`
	Status      string    `gorm:"type:enum('active', 'disabled');default:'active'" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

type EmailRecord struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FromUserID    int64     `gorm:"not null" json:"from_user_id"`
	FromEmail     string    `gorm:"type:varchar(100);not null;index" json:"from_email"`
	ToUserID      int64     `gorm:"default:null" json:"to_user_id"`
	ToEmail       string    `gorm:"type:varchar(100);not null;index" json:"to_email"`
	RecipientType string    `gorm:"type:enum('to','cc','bcc');default:'to'" json:"recipient_type"`
	Status        string    `gorm:"type:enum('pending', 'success', 'fail');default:'pending'" json:"status"`
	SentAt        time.Time `gorm:"default:null" json:"sent_at"`
	EmailReqID    string    `gorm:"type:varchar(36);index" json:"email_req_id"`
	RetryCount    int       `gorm:"default:0" json:"retry_count"`
	LastCheckedAt time.Time `gorm:"default:null" json:"last_checked_at"`
}

type Blacklist struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Reason    string    `gorm:"default:null" json:"reason"`
	CreatedBy int64     `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &UserMailAccount{}, &EmailRecord{}, &Blacklist{})
}
