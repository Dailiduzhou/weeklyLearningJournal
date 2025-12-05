package controller

import (
	"mvc/model"
	"mvc/service"
	"mvc/utils"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{
		userService: service.NewUserService(db),
	}
}

func (c *UserController) PageCreateUser(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "register.html", gin.H{
		"title": "用户注册",
	})
}

func (c *UserController) CreateUser(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	name := ctx.PostForm("name")

	if username == "" || password == "" || name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名、密码和姓名不能为空",
		})
		return
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码加密失败",
		})
		return
	}

	user := &model.User{
		Username: username,
		Password: hashedPassword,
		Name:     name,
	}

	if err := c.userService.CreateUser(user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建用户失败",
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

func (c *UserController) PageLogin(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "login.html", gin.H{
		"title": "用户登录",
	})
}

func (c *UserController) Login(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")

	if username == "" || password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名和密码不能为空",
		})
		return
	}

	user, err := c.userService.GetUser(username)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	if err := utils.ComparePassword(user.Password, password); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "密码错误",
		})
		return
	}

	session := sessions.Default(ctx)
	session.Set("username", username)
	if err := session.Save(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "会话创建失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

func (c *UserController) PageProfiles(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := session.Get("username")

	user, err := c.userService.GetUser(username.(string))
	if err != nil {
		session.Clear()
		session.Save()
		ctx.Redirect(http.StatusFound, "/api/users/login")
		return
	}

	ctx.HTML(http.StatusOK, "profiles.html", gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}
