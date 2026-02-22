package models

// @externalDocs description="GORM Documentation" url="https://gorm.io/docs/"
import (
	"time"

	"gorm.io/gorm"
)

const (
	DefaultCoverPath = "uploads/default.png"
	DefaultSummary   = "暂无简介"
)

// @Description 通用响应结构
// @param code int "状态码"
// @param msg string "消息内容"
// @param data object "业务数据，类型根据接口变化"
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// @Summary 图书模型
// @Description 图书详细信息
type Book struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Summary   string `json:"summary"`
	CoverPath string `json:"cover_path"`

	InitialStock int `json:"initial_stock" gorm:"default:0" binding:"gte=0"`
	Stock        int `json:"stock" gorm:"default:0" binding:"gte=0"`
	TotalStock   int `json:"total_stock" gorm:"defualt:0" binding:"gte=0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// @Summary 用户信息
// @Description 用户结构体
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"default:'user'" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// @Description 借阅记录
// @property id uint "记录ID"
// @property created_at string "创建时间 (RFC3339)"
// @property updated_at string "更新时间 (RFC3339)"
// @property deleted_at string "删除时间 (RFC3339) 可为空"
// @property user_id uint "用户ID"
// @property book_id uint "图书ID"
// @property borrow_date string "借出时间 (RFC3339)"
// @property return_date string "归还时间 (RFC3339) 未归还时为空"
// @property status string "状态: borrowed/returned"
type BorrowRecord struct {
	// 显式定义 gorm.Model 的字段
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	UserID     uint       `json:"user_id"`
	BookID     uint       `json:"book_id"`
	BorrowDate time.Time  `json:"borrow_date"`
	ReturnDate *time.Time `json:"return_date"`
	Status     string     `json:"status"` // borrowed/returned

	// 关联关系
	User *User `json:"user,omitempty" swaggerignore:"true"`
	Book *Book `json:"book,omitempty" swaggerignore:"true"`
}
