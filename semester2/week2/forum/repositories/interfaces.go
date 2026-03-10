package repositories

import (
	"context"

	"forum/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
}

type PostRepository interface {
	Create(ctx context.Context, post *models.Post) error
	FindByID(ctx context.Context, id string) (*models.Post, error)
	FindAll(ctx context.Context, page, limit int) ([]models.Post, error)
	Update(ctx context.Context, id string, update map[string]interface{}) error
	Delete(ctx context.Context, id string) error

	AddComment(ctx context.Context, postID string, comment *models.Comment) error
	DeleteComment(ctx context.Context, postID, commentID string) error

	VotePost(ctx context.Context, postID, userID, action string) error
	VoteComment(ctx context.Context, postID, commentID, userID, action string) error
}
