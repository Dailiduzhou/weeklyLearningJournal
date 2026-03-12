package services

import (
	"context"

	domainErrors "pg_forum/errors"
	"pg_forum/models"
	"pg_forum/repositories"
)

type postService struct {
	postRepo repositories.PostRepository
}

func NewPostService(postRepo repositories.PostRepository) PostService {
	return &postService{postRepo: postRepo}
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

func (s *postService) GetPostsByAuthor(authorID string, page, limit int) ([]models.Post, error) {
	ctx := context.Background()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	posts, err := s.postRepo.FindByAuthor(ctx, authorID, page, limit)
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

	var replyToPtr *string
	if replyToID != "" {
		replyToPtr = &replyToID
	}

	comment := &models.Comment{
		Author:    authorID,
		Content:   content,
		Custom:    custom,
		ReplyToID: replyToPtr,
	}

	err := s.postRepo.AddComment(ctx, postID, comment)
	if err != nil {
		if err.Error() == "post not found" {
			return domainErrors.ErrPostNotFound
		}
		if err.Error() == "comment not found" {
			return domainErrors.ErrCommentNotFound
		}
		return err
	}

	return nil
}

func (s *postService) DeleteComment(postID, commentID, authorID string) error {
	ctx := context.Background()

	comment, err := s.postRepo.FindCommentByID(ctx, commentID)
	if err != nil {
		return domainErrors.ErrCommentNotFound
	}

	if comment.PostID != postID {
		return domainErrors.ErrCommentNotFound
	}

	if comment.Author != authorID {
		return domainErrors.ErrUnauthorized
	}

	return s.postRepo.DeleteComment(ctx, postID, commentID)
}

func (s *postService) VotePost(postID, userID, action string) error {
	ctx := context.Background()

	hasVoted, err := s.postRepo.HasUserVotedPost(ctx, postID, userID)
	if err != nil {
		return err
	}
	if hasVoted {
		return domainErrors.ErrAlreadyVoted
	}

	if action != "up" && action != "down" {
		return domainErrors.ErrInvalidVoteAction
	}

	err = s.postRepo.VotePost(ctx, postID, userID, action)
	if err != nil {
		if err.Error() == "post not found" {
			return domainErrors.ErrPostNotFound
		}
		if err.Error() == "already voted" {
			return domainErrors.ErrAlreadyVoted
		}
		return err
	}

	return nil
}

func (s *postService) VoteComment(postID, commentID, userID, action string) error {
	ctx := context.Background()

	comment, err := s.postRepo.FindCommentByID(ctx, commentID)
	if err != nil {
		return domainErrors.ErrCommentNotFound
	}
	if comment.PostID != postID {
		return domainErrors.ErrCommentNotFound
	}

	hasVoted, err := s.postRepo.HasUserVotedComment(ctx, commentID, userID)
	if err != nil {
		return err
	}
	if hasVoted {
		return domainErrors.ErrAlreadyVoted
	}

	if action != "up" && action != "down" {
		return domainErrors.ErrInvalidVoteAction
	}

	err = s.postRepo.VoteComment(ctx, commentID, userID, action)
	if err != nil {
		if err.Error() == "comment not found" {
			return domainErrors.ErrCommentNotFound
		}
		if err.Error() == "already voted" {
			return domainErrors.ErrAlreadyVoted
		}
		return err
	}

	return nil
}
