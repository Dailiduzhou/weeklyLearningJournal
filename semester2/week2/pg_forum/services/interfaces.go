package services

import "pg_forum/models"

type UserService interface {
	Register(username, email, password string) (*models.AuthResponse, error)
	Login(email, password string) (*models.AuthResponse, error)
}

type JWTService interface {
	GenerateToken(userID string) (string, error)
	ValidateToken(tokenString string) (string, error)
}

type PostService interface {
	CreatePost(authorID, title, content string, custom map[string]interface{}) (*models.Post, error)
	GetPost(id string) (*models.Post, error)
	GetPosts(page, limit int) ([]models.Post, error)
	GetPostsByAuthor(authorID string, page, limit int) ([]models.Post, error)
	UpdatePost(id, authorID string, update models.UpdatePostDTO) error
	DeletePost(id, authorID string) error

	AddComment(postID, authorID, content, replyToID string, custom map[string]interface{}) error
	DeleteComment(postID, commentID, authorID string) error

	VotePost(postID, userID, action string) error
	VoteComment(postID, commentID, userID, action string) error
}
