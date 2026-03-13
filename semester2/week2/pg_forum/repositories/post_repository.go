package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pg_forum/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *models.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) FindByID(ctx context.Context, id string) (*models.Post, error) {
	var post models.Post
	err := r.db.WithContext(ctx).
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("path asc")
		}).
		First(&post, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("post not found")
		}
		return nil, err
	}

	return &post, nil
}

func (r *postRepository) FindAll(ctx context.Context, page, limit int) ([]models.Post, error) {
	skip := (page - 1) * limit

	var posts []models.Post
	err := r.db.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(skip).
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("path asc")
		}).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postRepository) FindByAuthor(ctx context.Context, authorID string, page, limit int) ([]models.Post, error) {
	skip := (page - 1) * limit

	var posts []models.Post
	err := r.db.WithContext(ctx).
		Where("author = ?", authorID).
		Order("created_at desc").
		Limit(limit).
		Offset(skip).
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("path asc")
		}).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postRepository) Update(ctx context.Context, id string, update map[string]any) error {
	update["updated_at"] = time.Now()

	result := r.db.WithContext(ctx).
		Model(&models.Post{}).
		Where("id = ?", id).
		Updates(update)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("post not found")
	}
	return nil
}

func (r *postRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Post{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("post not found")
	}
	return nil
}

func (r *postRepository) AddComment(ctx context.Context, postID string, comment *models.Comment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post models.Post
		if err := tx.First(&post, "id = ?", postID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("post not found")
			}
			return err
		}

		comment.PostID = postID

		var parentPath string
		if comment.ReplyToID != nil && *comment.ReplyToID != "" {
			var parent models.Comment
			if err := tx.Select("path").First(&parent, "id = ? AND post_id = ?", *comment.ReplyToID, postID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("comment not found")
				}
				return err
			}
			parentPath = parent.Path
		}

		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		segment := strings.ReplaceAll(comment.ID, "-", "_")
		path := segment
		if parentPath != "" {
			path = fmt.Sprintf("%s.%s", parentPath, segment)
		}

		return tx.Model(comment).Update("path", path).Error
	})
}

func (r *postRepository) FindCommentByID(ctx context.Context, commentID string) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.WithContext(ctx).First(&comment, "id = ?", commentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment not found")
		}
		return nil, err
	}

	return &comment, nil
}

func (r *postRepository) DeleteComment(ctx context.Context, postID, commentID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND post_id = ?", commentID, postID).Delete(&models.Comment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("comment not found")
	}
	return nil
}

func (r *postRepository) HasUserVotedPost(ctx context.Context, postID, userID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PostVote{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *postRepository) HasUserVotedComment(ctx context.Context, commentID, userID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CommentVote{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *postRepository) VotePost(ctx context.Context, postID, userID, action string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post models.Post
		if err := tx.First(&post, "id = ?", postID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("post not found")
			}
			return err
		}

		vote := &models.PostVote{PostID: postID, UserID: userID, Action: action}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(vote)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("already voted")
		}

		field := "down_votes"
		if action == "up" {
			field = "up_votes"
		}

		return tx.Model(&models.Post{}).
			Where("id = ?", postID).
			UpdateColumn(field, gorm.Expr(field+" + ?", 1)).Error
	})
}

func (r *postRepository) VoteComment(ctx context.Context, commentID, userID, action string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment models.Comment
		if err := tx.First(&comment, "id = ?", commentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("comment not found")
			}
			return err
		}

		vote := &models.CommentVote{CommentID: commentID, UserID: userID, Action: action}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(vote)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("already voted")
		}

		field := "down_votes"
		if action == "up" {
			field = "up_votes"
		}

		return tx.Model(&models.Comment{}).
			Where("id = ?", commentID).
			UpdateColumn(field, gorm.Expr(field+" + ?", 1)).Error
	})
}
