package repositories

import (
	"context"

	"pg_forum/models"
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
	FindByAuthor(ctx context.Context, authorID string, page, limit int) ([]models.Post, error)
	Update(ctx context.Context, id string, update map[string]interface{}) error
	Delete(ctx context.Context, id string) error

	AddComment(ctx context.Context, postID string, comment *models.Comment) error
	FindCommentByID(ctx context.Context, commentID string) (*models.Comment, error)
	DeleteComment(ctx context.Context, postID, commentID string) error

	HasUserVotedPost(ctx context.Context, postID, userID string) (bool, error)
	HasUserVotedComment(ctx context.Context, commentID, userID string) (bool, error)
	VotePost(ctx context.Context, postID, userID, action string) error
	VoteComment(ctx context.Context, commentID, userID, action string) error
}
