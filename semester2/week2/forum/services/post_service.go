package services

import (
	"context"

	domainErrors "forum/errors"
	"forum/models"
	"forum/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type postService struct {
	postRepo repositories.PostRepository
}

func NewPostService(postRepo repositories.PostRepository) PostService {
	return &postService{
		postRepo: postRepo,
	}
}

func (s *postService) CreatePost(authorID, title, content string, custom map[string]interface{}) (*models.Post, error) {
	ctx := context.Background()

	post := &models.Post{
		Title:   title,
		Content: content,
		Author:  authorID,
		Custom:  custom,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *postService) GetPost(id string) (*models.Post, error) {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, domainErrors.ErrPostNotFound
	}

	return post, nil
}

func (s *postService) GetPosts(page, limit int) ([]models.Post, error) {
	ctx := context.Background()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	posts, err := s.postRepo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *postService) UpdatePost(id, authorID string, update models.UpdatePostDTO) error {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return domainErrors.ErrPostNotFound
	}

	if post.Author != authorID {
		return domainErrors.ErrUnauthorized
	}

	updateMap := make(map[string]interface{})
	if update.Title != "" {
		updateMap["title"] = update.Title
	}
	if update.Content != "" {
		updateMap["content"] = update.Content
	}
	if update.Custom != nil {
		updateMap["custom"] = update.Custom
	}

	return s.postRepo.Update(ctx, id, updateMap)
}

func (s *postService) DeletePost(id, authorID string) error {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return domainErrors.ErrPostNotFound
	}

	if post.Author != authorID {
		return domainErrors.ErrUnauthorized
	}

	return s.postRepo.Delete(ctx, id)
}

func (s *postService) AddComment(postID, authorID, content, replyToID string, custom map[string]interface{}) error {
	ctx := context.Background()

	comment := &models.Comment{
		Author:  authorID,
		Content: content,
		Custom:  custom,
	}

	if replyToID != "" {
		replyToObjectID, err := primitive.ObjectIDFromHex(replyToID)
		if err != nil {
			return err
		}
		comment.ReplyToID = &replyToObjectID
	}

	return s.postRepo.AddComment(ctx, postID, comment)
}

func (s *postService) DeleteComment(postID, commentID, authorID string) error {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return domainErrors.ErrPostNotFound
	}

	var targetComment *models.Comment
	for i := range post.Comments {
		if post.Comments[i].ID.Hex() == commentID {
			targetComment = &post.Comments[i]
			break
		}
	}

	if targetComment == nil {
		return domainErrors.ErrCommentNotFound
	}

	if targetComment.Author != authorID {
		return domainErrors.ErrUnauthorized
	}

	return s.postRepo.DeleteComment(ctx, postID, commentID)
}

func (s *postService) VotePost(postID, userID, action string) error {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return domainErrors.ErrPostNotFound
	}

	for _, voterID := range post.UpVoterIDs {
		if voterID == userID {
			return domainErrors.ErrAlreadyVoted
		}
	}
	for _, voterID := range post.DownVoterIDs {
		if voterID == userID {
			return domainErrors.ErrAlreadyVoted
		}
	}

	if action != "up" && action != "down" {
		return domainErrors.ErrInvalidVoteAction
	}

	return s.postRepo.VotePost(ctx, postID, userID, action)
}

func (s *postService) VoteComment(postID, commentID, userID, action string) error {
	ctx := context.Background()

	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return domainErrors.ErrPostNotFound
	}

	var targetComment *models.Comment
	for i := range post.Comments {
		if post.Comments[i].ID.Hex() == commentID {
			targetComment = &post.Comments[i]
			break
		}
	}

	if targetComment == nil {
		return domainErrors.ErrCommentNotFound
	}

	for _, voterID := range targetComment.UpVoterIDs {
		if voterID == userID {
			return domainErrors.ErrAlreadyVoted
		}
	}
	for _, voterID := range targetComment.DownVoterIDs {
		if voterID == userID {
			return domainErrors.ErrAlreadyVoted
		}
	}

	if action != "up" && action != "down" {
		return domainErrors.ErrInvalidVoteAction
	}

	return s.postRepo.VoteComment(ctx, postID, commentID, userID, action)
}
