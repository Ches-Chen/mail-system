package controller

import (
	"gorm.io/gorm"
	"log"
	"msps/internal/app/api"
	"msps/internal/app/model/domain"
	"time"
)

type EmailStatusChecker struct {
	db       *gorm.DB
	stopChan chan struct{}
}

func NewEmailStatusChecker(db *gorm.DB) *EmailStatusChecker {
	return &EmailStatusChecker{
		db:       db,
		stopChan: make(chan struct{}),
	}
}

func (esc *EmailStatusChecker) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			esc.checkPendingEmails()
		case <-esc.stopChan:
			return
		}
	}
}

func (esc *EmailStatusChecker) Stop() {
	if esc.stopChan != nil {
		close(esc.stopChan)
	}
}

func (esc *EmailStatusChecker) checkPendingEmails() {
	var records []domain.EmailRecord

	// 获取所有pending状态的记录
	if err := esc.db.Where("status = ?", "pending").Find(&records).Error; err != nil {
		log.Printf("Failed to fetch pending emails: %v", err)
		return
	}

	for _, record := range records {
		// 检查邮件状态
		status := api.VerifyMap.CheckMap(record.EmailReqID)

		var updateFields = make(map[string]interface{})
		updateFields["last_checked_at"] = time.Now()

		switch status {
		case api.StatusSuccess:
			updateFields["status"] = "success"
			updateFields["sent_at"] = time.Now()
		case api.StatusFailed:
			updateFields["status"] = "fail"
			updateFields["sent_at"] = time.Now()
		case api.StatusUnknown:
			// 更新重试次数
			updateFields["retry_count"] = record.RetryCount + 1

			// 如果重试超过一定次数，标记为失败
			if record.RetryCount >= 10 { // 假设最大重试10次
				updateFields["status"] = "fail"
				updateFields["sent_at"] = time.Now()
			}
		}

		// 更新数据库
		if err := esc.db.Model(&domain.EmailRecord{}).
			Where("id = ?", record.ID).
			Updates(updateFields).Error; err != nil {
			log.Printf("Failed to update email record %d: %v", record.ID, err)
		}
	}
}
