package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Email     string    `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Age       int       `gorm:"default:0" json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
