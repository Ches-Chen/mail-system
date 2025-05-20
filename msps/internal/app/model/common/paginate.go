package common

import "msps/internal/app/model/domain"

// PaginatedResponse 分页响应结构体
type PaginatedResponse struct {
	Users []domain.User `json:"users"`
	Total int64         `json:"total"`
}
