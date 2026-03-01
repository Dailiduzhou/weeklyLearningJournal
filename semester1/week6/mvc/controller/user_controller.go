package controller

import (
	"log"
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

// PageCreateUser godoc
// @Summary      用户注册页面
// @Description  返回用户注册的HTML表单页面
// @Tags         pages
// @Produce      html
// @Success      200  {string}  string  "返回 register.html 页面"
// @Router       /users [get]
func (c *UserController) PageCreateUser(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "register.html", gin.H{
		"title": "用户注册",
	})
}

// CreateUser godoc
// @Summary      创建新用户 (注册)
// @Description  通过表单提交用户名、密码和姓名来创建新用户
// @Tags         users
// @Accept       mpfd
// @Produce      json
// @Param        username formData string true "用户名" example:"zhangsan")
// @Param        password formData string true "密码" example:"myPassword123")
// @Param        name     formData string true "真实姓名" example:"张三")
// @Success      201  {object}  map[string]map[string]string  "创建成功，返回用户信息"
// @Failure      400  {object}  map[string]string  "请求参数错误（如字段为空）"
// @Failure      500  {object}  map[string]string  "服务器内部错误（如数据库操作失败）"
// @Router       /users [post]
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

// PageLogin godoc
// @Summary      用户登录页面
// @Description  返回用户登录的HTML表单页面
// @Tags         pages
// @Produce      html
// @Success      200  {string}  string  "返回 login.html 页面"
// @Router       /users/login [get]
func (c *UserController) PageLogin(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "login.html", gin.H{
		"title": "用户登录",
	})
}

// Login godoc
// @Summary      用户登录
// @Description  使用用户名和密码登录，成功后在Cookie中设置Session
// @Tags         auth
// @Accept       mpfd
// @Produce      json
// @Param        username formData string true "用户名" example:"zhangsan")
// @Param        password formData string true "密码" example:"myPassword123")
// @Success      200  {object}  map[string]map[string]string  "登录成功，返回用户基本信息"
// @Failure      400  {object}  map[string]string  "请求参数错误"
// @Failure      403  {object}  map[string]string  "密码错误"
// @Failure      404  {object}  map[string]string  "用户不存在"
// @Failure      500  {object}  map[string]string  "服务器内部错误"
// @Router       /users/login [post]
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

// PageProfiles godoc
// @Summary      获取当前登录用户信息页面 (需登录)
// @Description  验证Session后返回用户个人信息页面。若未登录则重定向到登录页。
// @Tags         users
// @Produce      html
// @Success      200  {string}  string  "返回 profiles.html 页面"
// @Failure      302  {string}  string  "重定向到 /api/users/login (未登录时)"
// @Router       /users/me [get]
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

// Logout godoc
// @Summary      用户登出
// @Description  清除当前用户的Session会话信息
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]string  "登出成功"
// @Failure      500  {object}  map[string]string  "服务器内部错误"
// @Security     SessionAuth
// @Router       /users/logout [post]
func (c *UserController) Logout(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Clear()
	session.Save()
	ctx.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

// PageChangeProfile godoc
// @Summary      修改用户资料页面 (需登录)
// @Description  返回修改个人资料的HTML表单页面。若未登录则重定向。
// @Tags         pages
// @Produce      html
// @Success      200  {string}  string  "返回 changeprofiles.html 页面"
// @Failure      302  {string}  string  "重定向到 /api/users/login (未登录时)"
// @Security     SessionAuth
// @Router       /users/profiles [get]
func (c *UserController) PageChangeProfile(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := session.Get("username")

	if username == "" {
		ctx.Redirect(http.StatusFound, "/api/users/login")
		return
	}

	ctx.HTML(http.StatusOK, "changeprofiles.html", gin.H{
		"username": username,
	})
}

// ChangeProfile godoc
// @Summary      修改用户资料 (需登录)
// @Description  提交表单以修改当前登录用户的用户名或姓名
// @Tags         users
// @Accept       mpfd
// @Produce      json
// @Param        newusername formData string false "新用户名" example:"newzhangsan"
// @Param        newname     formData string false "新姓名" example:"李四"
// @Success      200  {object}  map[string]string  "用户信息更新成功"
// @Failure      400  {object}  map[string]string  "请求参数错误（如未更改或与原信息相同）"
// @Failure      401  {object}  map[string]string  "未认证，请先登录"
// @Failure      403  {object}  map[string]string  "用户不存在"
// @Failure      500  {object}  map[string]string  "服务器内部错误（如数据库或会话更新失败）"
// @Security     SessionAuth
// @Router       /users/profiles [put]
func (c *UserController) ChangeProfile(ctx *gin.Context) {
	session := sessions.Default(ctx)
	sessionUsername := session.Get("username")

	if sessionUsername == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}

	user, err := c.userService.GetUser(sessionUsername.(string))
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "用户不存在",
		})
	}

	newUsername := ctx.PostForm("newusername")
	newName := ctx.PostForm("newname")

	if newUsername == "" && newName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "未检测到有效更改",
		})
		return
	}

	if newUsername != "" && newUsername == user.Username {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "新用户名与原用户名相同",
		})
		return
	}

	if newName != "" && newName == user.Name {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "新姓名与原姓名相同",
		})
		return
	}

	var updates map[string]any
	switch {
	case newUsername != "" && newName == "":
		updates = map[string]any{
			"Username": newUsername,
		}
	case newName != "" && newUsername == "":
		updates = map[string]any{
			"Name": newName,
		}
	case newUsername != "" && newName != "":
		updates = map[string]any{
			"Username": newUsername,
			"Name":     newName,
		}
	}

	if err := c.userService.UpdateUser(user.Username, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新用户信息失败",
		})
		return
	}

	if newUsername != "" {
		session.Set("username", newUsername)
		if err = session.Save(); err != nil {
			log.Println("会话创建失败")
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "会话创建失败,请重试",
			})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "用户信息更新成功",
	})
}

// PageChangePassword godoc
// @Summary      修改密码页面 (需登录)
// @Description  返回修改密码的HTML表单页面。若未登录则重定向。
// @Tags         pages
// @Produce      html
// @Success      200  {string}  string  "返回 changepassword.html 页面"
// @Failure      302  {string}  string  "重定向到 /login (未登录时)"
// @Security     SessionAuth
// @Router       /users/password [get]
func (c *UserController) PageChangePassword(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := session.Get("username")

	if username == "" {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	ctx.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "修改密码",
	})
}

func (c *UserController) ChangePassword(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := session.Get("username")

	if username == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}

	oldpassword := ctx.PostForm("oldpassword")
	password := ctx.PostForm("password")
	password1 := ctx.PostForm("password1")

	if oldpassword == "" || password == "" || password1 == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "所有密码字段不能为空",
		})
		return
	}

	user, err := c.userService.GetUser(username.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	if err := utils.ComparePassword(user.Password, oldpassword); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "原密码错误",
		})
		return
	}

	if password != password1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "两次输入的新密码不一致",
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

	updates := map[string]any{
		"Password": hashedPassword,
	}

	if err := c.userService.UpdateUser(user.Username, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新用户信息失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// ChangePassword godoc
// @Summary      修改用户密码 (需登录)
// @Description  提交表单以修改当前登录用户的密码
// @Tags         users
// @Accept       mpfd
// @Produce      json
// @Param        oldpassword formData string true "原密码" example:"oldPassword123"
// @Param        password    formData string true "新密码" example:"newPassword456"
// @Param        password1   formData string true "确认新密码" example:"newPassword456"
// @Success      200  {object}  map[string]string  "密码修改成功"
// @Failure      400  {object}  map[string]string  "请求参数错误（如字段为空或两次输入不一致）"
// @Failure      401  {object}  map[string]string  "未认证，请先登录"
// @Failure      403  {object}  map[string]string  "原密码错误"
// @Failure      404  {object}  map[string]string  "用户不存在"
// @Failure      500  {object}  map[string]string  "服务器内部错误"
// @Security     SessionAuth
// @Router       /users/password [post]
func (c *UserController) UsersData(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := session.Get("username")

	if username == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}

	if username.(string) != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足,需要管理员权限",
		})
		return
	}

	var users []model.User
	if err := c.userService.GetAllUsers(&users); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取用户信息失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}
