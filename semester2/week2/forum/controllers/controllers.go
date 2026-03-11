package controllers

import (
	"errors"
	"net/http"
	"strconv"

	domainErrors "forum/errors"
	"forum/models"
	"forum/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService services.UserService
}

func NewAuthController(userService services.UserService) *AuthController {
	return &AuthController{
		userService: userService,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new account and return an auth token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body models.RegisterDTO true "Registration payload"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var dto models.RegisterDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.userService.Register(dto.Username, dto.Email, dto.Password)
	if err != nil {
		if errors.Is(err, domainErrors.ErrUserAlreadyExists) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// Login godoc
// @Summary Login
// @Description Authenticate with email and password, return an auth token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body models.LoginDTO true "Login payload"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var dto models.LoginDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.userService.Login(dto.Email, dto.Password)
	if err != nil {
		if errors.Is(err, domainErrors.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

type PostController struct {
	postService services.PostService
}

func NewPostController(postService services.PostService) *PostController {
	return &PostController{
		postService: postService,
	}
}

// CreatePost godoc
// @Summary Create a post
// @Description Create a new post.
// @Tags Posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.CreatePostDTO true "Post payload"
// @Success 201 {object} models.Post
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts [post]
func (c *PostController) CreatePost(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	var dto models.CreatePostDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := c.postService.CreatePost(userID, dto.Title, dto.Content, dto.Custom)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, post)
}

// GetPost godoc
// @Summary Get a post
// @Description Get a single post by ID.
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} models.Post
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id} [get]
func (c *PostController) GetPost(ctx *gin.Context) {
	postID := ctx.Param("id")

	post, err := c.postService.GetPost(postID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, post)
}

// GetPosts godoc
// @Summary List posts
// @Description Get a paginated list of posts.
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Success 200 {object} models.PostListResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts [get]
func (c *PostController) GetPosts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	posts, err := c.postService.GetPosts(page, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"page":  page,
		"limit": limit,
	})
}

// UpdatePost godoc
// @Summary Update a post
// @Description Update an existing post by ID.
// @Tags Posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Param payload body models.UpdatePostDTO true "Update payload"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id} [put]
func (c *PostController) UpdatePost(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")

	var dto models.UpdatePostDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.postService.UpdatePost(postID, userID, dto)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrUnauthorized) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "post updated successfully"})
}

// DeletePost godoc
// @Summary Delete a post
// @Description Delete a post by ID.
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} models.MessageResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id} [delete]
func (c *PostController) DeletePost(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")

	err := c.postService.DeletePost(postID, userID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrUnauthorized) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "post deleted successfully"})
}

// AddComment godoc
// @Summary Add a comment
// @Description Add a comment to a post.
// @Tags Comments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Param payload body models.AddCommandDTO true "Comment payload"
// @Success 201 {object} models.MessageResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id}/comments [post]
func (c *PostController) AddComment(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")

	var dto models.AddCommandDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.postService.AddComment(postID, userID, dto.Content, dto.ReplyToID, dto.Custom)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "comment added successfully"})
}

// DeleteComment godoc
// @Summary Delete a comment
// @Description Delete a comment by ID.
// @Tags Comments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Post ID"
// @Param commentId path string true "Comment ID"
// @Success 200 {object} models.MessageResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id}/comments/{commentId} [delete]
func (c *PostController) DeleteComment(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")
	commentID := ctx.Param("commentId")

	err := c.postService.DeleteComment(postID, commentID, userID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) || errors.Is(err, domainErrors.ErrCommentNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrUnauthorized) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}

// VotePost godoc
// @Summary Vote on a post
// @Description Upvote or downvote a post.
// @Tags Votes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Param payload body models.VoteDTO true "Vote payload"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id}/vote [post]
func (c *PostController) VotePost(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")

	var dto models.VoteDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.postService.VotePost(postID, userID, dto.Action)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrAlreadyVoted) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrInvalidVoteAction) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "vote recorded successfully"})
}

// VoteComment godoc
// @Summary Vote on a comment
// @Description Upvote or downvote a comment.
// @Tags Votes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Param commentId path string true "Comment ID"
// @Param payload body models.VoteDTO true "Vote payload"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /posts/{id}/comments/{commentId}/vote [post]
func (c *PostController) VoteComment(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	postID := ctx.Param("id")
	commentID := ctx.Param("commentId")

	var dto models.VoteDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.postService.VoteComment(postID, commentID, userID, dto.Action)
	if err != nil {
		if errors.Is(err, domainErrors.ErrPostNotFound) || errors.Is(err, domainErrors.ErrCommentNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrAlreadyVoted) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domainErrors.ErrInvalidVoteAction) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "vote recorded successfully"})
}
