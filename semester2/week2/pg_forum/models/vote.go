package models

import (
	"time"

	"gorm.io/gorm"
)

type PostVote struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	PostID string `gorm:"type:uuid;not null;index:idx_post_vote_post_user,unique" json:"postId"`
	UserID string `gorm:"type:varchar(100);not null;index:idx_post_vote_post_user,unique" json:"userId"`
	Action string `gorm:"type:varchar(10);not null" json:"action"`

	CreatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type CommentVote struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	CommentID string `gorm:"type:uuid;not null;index:idx_comment_vote_comment_user,unique" json:"commentId"`
	UserID    string `gorm:"type:varchar(100);not null;index:idx_comment_vote_comment_user,unique" json:"userId"`
	Action    string `gorm:"type:varchar(10);not null" json:"action"`

	CreatedAt time.Time      `gorm:"type:timestamptz;default:now()" json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
