package models

type CreatePostDTO struct {
	Title   string         `json:"title" binding:"required"`
	Content string         `json:"content" binding:"required"`
	Author  string         `json:"author" binding:"required"`
	Custom  map[string]any `json:"custom"`
}

type AddCommandDTO struct {
	PostID    string         `json:"postId" binding:"required"`
	Author    string         `json:"author" binding:"required"`
	Content   string         `json:"content" binding:"required"`
	ReplyToID string         `json:"replyToId"`
	Custom    map[string]any `json:"custom"`
}

type UpdatePostDTO struct {
	Title   string         `json:"title"`
	Content string         `json:"content"`
	Custom  map[string]any `json:"custom"`
}

// VoteDTO 投票请求载荷
type VoteDTO struct {
	Action string `json:"action" binding:"required,oneof=up down"` // 只能是 up 或 down
}
