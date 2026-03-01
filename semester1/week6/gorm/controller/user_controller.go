package controller

import (
	"gorm-mvc-demo/model"
	"gorm-mvc-demo/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserController 用户控制器
type UserController struct {
	userService *service.UserService
}

// NewUserController 创建用户控制器实例
func NewUserController(db *gorm.DB) *UserController {
	return &UserController{
		userService: service.NewUserService(db),
	}
}

// CreateUser 创建用户
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user model.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.userService.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "用户创建成功",
		"user":    user,
	})
}

// GetAllUsers 获取所有用户
func (c *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := c.userService.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// GetUserByID 根据ID获取用户
func (c *UserController) GetUserByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	user, err := c.userService.GetUserByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUser 更新用户
func (c *UserController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var updates map[string]interface{}
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.userService.UpdateUser(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "用户更新成功"})
}

// DeleteUser 删除用户
func (c *UserController) DeleteUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	if err := c.userService.DeleteUser(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}
