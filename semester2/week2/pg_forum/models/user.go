package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Username string `gorm:"type:varchar(255);not null;unique;index" json:"username"`
	Email    string `gorm:"type:varchar(100);not null;unique;index" json:"email"`

	Password string `gorm:"type:varchar(255);not null" json:"-"`

	CreatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type RegisterDTO struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginDTO struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"` // 由于 User.Password 有 json:"-"，这里返回给前端时密码会自动被隐藏，非常安全
}
