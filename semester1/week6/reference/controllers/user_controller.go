package controllers

import (
	"gin-mvc-mysql/models"
	"gin-mvc-mysql/repositories"
	"gin-mvc-mysql/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userRepo *repositories.UserRepository
}

func NewUserController() *UserController {
	return &UserController{
		userRepo: &repositories.UserRepository{},
	}
}

// GetUsers 获取所有用户 (GET /users)
func (c *UserController) GetUsers(ctx *gin.Context) {
	users, err := c.userRepo.GetAllUsers()
	if err != nil {
		utils.SendError(ctx, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// 使用JSON响应[citation:7]
	utils.SendSuccess(ctx, "用户列表获取成功", users)

	// 如果使用HTML模板，可以这样渲染：
	// ctx.HTML(http.StatusOK, "users.html", gin.H{
	//     "title": "用户列表",
	//     "users": users,
	// })
}

// GetUser 获取单个用户 (GET /users/:id)
func (c *UserController) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.SendError(ctx, http.StatusBadRequest, "无效的用户ID")
		return
	}

	user, err := c.userRepo.GetUserByID(uint(id))
	if err != nil {
		utils.SendError(ctx, http.StatusNotFound, "用户不存在")
		return
	}

	utils.SendSuccess(ctx, "用户详情获取成功", user)
}

// CreateUser 创建用户 (POST /users)
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user models.User

	// 绑定JSON数据
	if err := ctx.ShouldBindJSON(&user); err != nil {
		utils.SendError(ctx, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 验证必要字段
	if user.Username == "" || user.Email == "" {
		utils.SendError(ctx, http.StatusBadRequest, "用户名和邮箱不能为空")
		return
	}

	// 创建用户
	if err := c.userRepo.CreateUser(&user); err != nil {
		utils.SendError(ctx, http.StatusInternalServerError, "创建用户失败")
		return
	}

	utils.SendSuccess(ctx, "用户创建成功", user)
}

// UpdateUser 更新用户 (PUT /users/:id)
func (c *UserController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.SendError(ctx, http.StatusBadRequest, "无效的用户ID")
		return
	}

	// 获取现有用户
	user, err := c.userRepo.GetUserByID(uint(id))
	if err != nil {
		utils.SendError(ctx, http.StatusNotFound, "用户不存在")
		return
	}

	// 绑定更新数据
	var updateData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		utils.SendError(ctx, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 更新字段
	if updateData.Username != "" {
		user.Username = updateData.Username
	}
	if updateData.Email != "" {
		user.Email = updateData.Email
	}

	// 保存更改
	if err := c.userRepo.UpdateUser(user); err != nil {
		utils.SendError(ctx, http.StatusInternalServerError, "更新用户失败")
		return
	}

	utils.SendSuccess(ctx, "用户更新成功", user)
}

// DeleteUser 删除用户 (DELETE /users/:id)
func (c *UserController) DeleteUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.SendError(ctx, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := c.userRepo.DeleteUser(uint(id)); err != nil {
		utils.SendError(ctx, http.StatusInternalServerError, "删除用户失败")
		return
	}

	utils.SendSuccess(ctx, "用户删除成功", nil)
}
