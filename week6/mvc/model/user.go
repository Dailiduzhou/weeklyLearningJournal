package model

import (
	"time"
)

// User 用户数据模型
// @Description 系统用户的核心信息
type User struct {
	ID        int       `gorm:"primaryKey" json:"id" example:"1"`                                 // 用户唯一ID
	Username  string    `gorm:"size:100;not null" json:"username" example:"zhangsan"`             // 用户名，用于登录
	Password  string    `gorm:"size:100;not null" json:"password,omitempty" swaggerignore:"true"` // 密码哈希，文档中隐藏
	Name      string    `gorm:"size:100;not null" json:"name" example:"张三"`                       // 用户真实姓名
	CreatedAt time.Time `json:"created_at"`                                                       // 创建时间
	UpdatedAt time.Time `json:"updated_at"`                                                       // 最后更新时间
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
