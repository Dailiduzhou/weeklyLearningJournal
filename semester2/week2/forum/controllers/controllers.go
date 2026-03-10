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
