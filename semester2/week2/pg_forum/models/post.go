package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Title   string `gorm:"type:varchar(255);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`
	Author  string `gorm:"type:varchar(100);not null;index:idx_post_author" json:"author"`

	Custom map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'" json:"custom"`

	Upvotes   int `gorm:"default:0" json:"upvotes"`
	Downvotes int `gorm:"default:0" json:"downvotes"`

	CreatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Comment struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	// 关联 Post 表的 ID，加上索引加速查询
	PostID string `gorm:"type:uuid;not null;index:idx_comment_post" json:"post_id"`

	Author  string `gorm:"type:varchar(100);not null" json:"author"`
	Content string `gorm:"type:text;not null" json:"content"`

	ReplyToID *string        `gorm:"type:uuid;index:idx_comment_reply" json:"reply_to_id"`
	Path      string         `gorm:"type:ltree;index:idx_comment_path,type:gist" json:"path"`
	Custom    map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'" json:"custom"`

	CreatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
