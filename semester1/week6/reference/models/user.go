package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型定义（GORM结构体标签）
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string         `json:"username" gorm:"type:varchar(100);not null;uniqueIndex"`
	Email     string         `json:"email" gorm:"type:varchar(255);not null;uniqueIndex"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 软删除
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
