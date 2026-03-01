// Package classification 用户管理API
//
// 这是一个基于Gin框架的用户管理系统API，提供用户注册、登录、个人信息管理等功能
//
//	Schemes: http
//	BasePath: /
//	Version: 1.0.0
//	Host: localhost:8080
//
//	Consumes:
//	- application/json
//	- application/x-www-form-urlencoded
//
//	Produces:
//	- application/json
//	- text/html
//
//	SecurityDefinitions:
//	sessionAuth:
//	     type: apiKey
//	     name: Cookie
//	     in: header
//	     description: 会话认证，基于cookie
//
// swagger:meta
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// User 表示系统用户
// swagger:model User
type User struct {
	// 用户名，唯一标识
	// Example: "john_doe"
	Username string `json:"username"`
	// 密码（bcrypt加密存储）
	// Example: "$2a$10$xyz..."
	Password string `json:"password"`
	// 用户真实姓名
	// Example: "张三"
	Name string `json:"name"`
}

// RegisterRequest 注册请求
// swagger:parameters registerUser
type RegisterRequest struct {
	// in:body
	Body struct {
		// 用户名，必须唯一
		// Required: true
		// Example: "john_doe"
		Username string `json:"username" form:"username"`
		// 密码，至少6位
		// Required: true
		// Example: "password123"
		Password string `json:"password" form:"password"`
		// 用户真实姓名
		// Required: true
		// Example: "张三"
		Name string `json:"name" form:"name"`
	} `json:"body"`
}

// LoginRequest 登录请求
// swagger:parameters loginUser
type LoginRequest struct {
	// in:body
	Body struct {
		// 用户名
		// Required: true
		// Example: "john_doe"
		Username string `json:"username" form:"username"`
		// 密码
		// Required: true
		// Example: "password123"
		Password string `json:"password" form:"password"`
	} `json:"body"`
}

// ChangePasswordRequest 修改密码请求
// swagger:parameters changePassword
type ChangePasswordRequest struct {
	// in:body
	Body struct {
		// 原密码
		// Required: true
		// Example: "oldpassword123"
		OldPassword string `json:"oldpassword" form:"oldpassword"`
		// 新密码
		// Required: true
		// Example: "newpassword123"
		Password string `json:"password" form:"password"`
		// 确认新密码
		// Required: true
		// Example: "newpassword123"
		Password1 string `json:"password1" form:"password1"`
	} `json:"body"`
}

// ChangeProfileRequest 修改资料请求
// swagger:parameters changeProfile
type ChangeProfileRequest struct {
	// in:body
	Body struct {
		// 新用户名，可选
		// Example: "new_john_doe"
		NewUsername string `json:"newusername" form:"newusername"`
		// 新姓名，可选
		// Example: "李四"
		NewName string `json:"newname" form:"newname"`
	} `json:"body"`
}

// UserListResponse 用户列表响应
// swagger:response userListResponse
type UserListResponse struct {
	// in:body
	Body struct {
		// 用户数据映射
		Users map[string]gin.H `json:"users"`
		// 用户数量
		Count int `json:"count"`
	} `json:"body"`
}

// SuccessResponse 通用成功响应
// swagger:response successResponse
type SuccessResponse struct {
	// in:body
	Body struct {
		// 响应消息
		// Example: "操作成功"
		Message string `json:"message"`
	} `json:"body"`
}

// ErrorResponse 通用错误响应
// swagger:response errorResponse
type ErrorResponse struct {
	// in:body
	Body struct {
		// 错误信息
		// Example: "操作失败"
		Error string `json:"error"`
	} `json:"body"`
}

// 全局变量定义
var (
	// userStore 内存中的用户存储
	userStore = make(map[string]*User)
	// userFile 用户数据文件路径
	userFile = "users.json"
	// mutex 读写锁，保证并发安全
	mutex sync.RWMutex
	// sessionSecret 会话加密密钥
	sessionSecret = make([]byte, 32)
)

// main 应用程序入口点
// @title 用户管理API
// @version 1.0.0
// @description 基于Gin框架的用户管理系统，提供完整的用户认证和个人信息管理功能
// @host localhost:8080
// @BasePath /
// @securityDefinitions.sessionAuth sessionAuth
func main() {
	// 初始化：加载用户数据
	if err := loadFiles(); err != nil {
		fmt.Printf("用户数据加载失败: %v\n", err)
		fmt.Println("将以空用户数据库启动...")
	} else {
		fmt.Printf("成功加载 %d 个用户数据\n", len(userStore))
	}

	// 生成会话加密密钥
	if _, err := rand.Read(sessionSecret); err != nil {
		fmt.Printf("生成会话密钥失败: %v", err)
		return
	}

	// 初始化Gin引擎
	r := gin.Default()

	// 配置会话中间件
	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("authSession", store))

	// 加载HTML模板文件
	r.LoadHTMLGlob("./template/*.html")

	// 注册API路由
	registerRoutes(r)

	// 启动HTTP服务器
	fmt.Println("服务器启动在 :8080")
	fmt.Println("API文档地址: http://localhost:8080/docs")
	if err := r.Run(":8080"); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}

// registerRoutes 注册所有路由
func registerRoutes(r *gin.Engine) {
	// 文档路由
	r.GET("/swagger.json", swaggerJSONHandler)
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/*any", swaggerUIHandler)

	// 页面路由
	r.GET("/register", registerHandler)
	r.GET("/login", loginHandler)
	r.GET("/profiles", profilesHandler)
	r.GET("/changepassword", changePasswordHandler)
	r.GET("/changeprofiles", changeprofileHandler)

	// 功能路由
	r.POST("/register", registerHandler1)
	r.POST("/login", loginHandler1)
	r.POST("/changepassword", changepasswordhandler1)
	r.POST("/changeprofiles", changeprofileHandler1)
	r.GET("/userdata", viewUserdataHandler)
	r.GET("/logout", logoutHandler)
}

// swaggerJSONHandler 返回Swagger JSON文档
// @Summary 获取Swagger规范文档
// @Description 返回符合OpenAPI规范的Swagger JSON文档
// @Tags 文档
// @Produce json
// @Success 200 {file} string "Swagger JSON文档"
// @Router /swagger.json [get]
func swaggerJSONHandler(c *gin.Context) {
	c.File("./swagger.json")
}

// swaggerUIHandler 返回Swagger UI界面
// @Summary Swagger UI界面
// @Description 返回交互式的API文档界面
// @Tags 文档
// @Produce html
// @Success 200 {string} string "Swagger UI HTML页面"
// @Router /docs [get]
// @Router /docs/{any} [get]
func swaggerUIHandler(c *gin.Context) {
	swaggerUIHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>用户管理API文档</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *,
        *:before,
        *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #2c3e50;
            padding: 10px 0;
        }
        .swagger-ui .topbar .download-url-wrapper {
            display: none;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-standalone-preset.js"></script>
    <script>
    window.onload = function() {
        const ui = SwaggerUIBundle({
            url: '/swagger.json',
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
            validatorUrl: null,
            onComplete: function() {
                console.log('Swagger UI loaded');
            }
        });
        window.ui = ui;
    }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, swaggerUIHTML)
}

// registerHandler 返回用户注册页面
// @Summary 用户注册页面
// @Description 返回用户注册表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "注册页面HTML"
// @Router /register [get]
func registerHandler(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{
		"title": "用户注册",
	})
}

// registerHandler1 处理用户注册请求
// @Summary 用户注册
// @Description 创建新用户账户
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "用户名" minLength(3) maxLength(20)
// @Param password formData string true "密码" minLength(6)
// @Param name formData string true "姓名" minLength(1) maxLength(50)
// @Success 202 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /register [post]
func registerHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	name := c.PostForm("name")

	// 参数验证
	if username == "" || password == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名、密码和姓名不能为空",
		})
		return
	}

	// 检查用户是否存在
	mutex.RLock()
	_, exists := userStore[username]
	mutex.RUnlock()

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名已存在",
		})
		return
	}

	// 密码加密
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码加密失败",
		})
		return
	}

	// 创建用户对象
	newUser := &User{
		Username: username,
		Password: string(hashedPasswordBytes),
		Name:     name,
	}

	// 保存到内存
	mutex.Lock()
	userStore[username] = newUser
	mutex.Unlock()

	// 持久化到文件
	if err := saveUserstoFiles(); err != nil {
		// 回滚操作
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "用户数据保存失败",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "注册成功，请登录",
	})
}

// loginHandler 返回用户登录页面
// @Summary 用户登录页面
// @Description 返回用户登录表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "登录页面HTML"
// @Router /login [get]
func loginHandler(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"title": "用户登录",
	})
}

// loginHandler1 处理用户登录请求
// @Summary 用户登录
// @Description 验证用户凭证并创建会话
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "用户名"
// @Param password formData string true "密码"
// @Success 302 "登录成功，重定向到个人资料页"
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /login [post]
func loginHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名和密码不能为空",
		})
		return
	}

	mutex.RLock()
	user, exists := userStore[username]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "密码错误",
		})
		return
	}

	// 创建会话
	session := sessions.Default(c)
	session.Set("username", username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "会话创建失败",
		})
		return
	}

	c.Redirect(http.StatusFound, "/profiles")
}

// profilesHandler 返回用户个人资料页面
// @Summary 个人资料页面
// @Description 显示当前登录用户的个人资料
// @Tags 页面
// @Produce html
// @Success 200 {string} string "个人资料页面HTML"
// @Failure 302 "未登录，重定向到登录页"
// @Router /profiles [get]
func profilesHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	mutex.RLock()
	user, exists := userStore[username.(string)]
	mutex.RUnlock()

	if !exists {
		session.Clear()
		session.Save()
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.HTML(http.StatusOK, "profiles.html", gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

// changePasswordHandler 返回修改密码页面
// @Summary 修改密码页面
// @Description 返回修改密码表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "修改密码页面HTML"
// @Router /changepassword [get]
func changePasswordHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "修改密码",
	})
}

// changepasswordhandler1 处理修改密码请求
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param oldpassword formData string true "原密码"
// @Param password formData string true "新密码"
// @Param password1 formData string true "确认新密码"
// @Success 200 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /changepassword [post]
func changepasswordhandler1(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	oldpassword := c.PostForm("oldpassword")
	password := c.PostForm("password")
	password1 := c.PostForm("password1")

	// 参数验证
	if oldpassword == "" || password == "" || password1 == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "所有密码字段不能为空",
		})
		return
	}

	mutex.RLock()
	user, exists := userStore[username.(string)]
	mutex.RUnlock()

	if !exists {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Message":      "用户不存在",
			"RedirectURL":  "/changepassword",
			"RedirectName": "修改密码页面",
			"Delay":        1000,
		})
		return
	}

	// 验证原密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword)); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Message":      "原密码错误",
			"RedirectURL":  "/changepassword",
			"RedirectName": "修改密码页面",
			"Delay":        1000,
		})
		return
	}

	// 验证新密码一致性
	if password != password1 {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Message":      "两次输入的新密码不一致",
			"RedirectURL":  "/changepassword",
			"RedirectName": "修改密码页面",
			"Delay":        1000,
		})
		return
	}

	// 加密新密码
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码加密失败",
		})
		return
	}

	// 更新密码
	mutex.Lock()
	userStore[username.(string)].Password = string(hashedPasswordBytes)
	mutex.Unlock()

	// 持久化保存
	if err := saveUserstoFiles(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// changeprofileHandler 返回修改资料页面
// @Summary 修改资料页面
// @Description 返回修改用户资料表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "修改资料页面HTML"
// @Router /changeprofiles [get]
func changeprofileHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.HTML(http.StatusOK, "changeprofiles.html", gin.H{
		"username": username,
	})
}

// changeprofileHandler1 处理修改资料请求
// @Summary 修改用户资料
// @Description 修改当前登录用户的用户名和姓名
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param newusername formData string false "新用户名"
// @Param newname formData string false "新姓名"
// @Success 200 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /changeprofiles [post]
func changeprofileHandler1(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	newusername := c.PostForm("newusername")
	newname := c.PostForm("newname")

	mutex.RLock()
	user := userStore[username.(string)]
	mutex.RUnlock()

	// 检查是否有有效更改
	if newname == "" && newusername == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "未检测到有效更改",
		})
		return
	}

	if newusername == user.Username {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "新用户名与原用户名相同",
		})
		return
	}

	if newname == user.Name {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "新姓名与原姓名相同",
		})
		return
	}

	// 更新用户名
	if newusername != "" {
		mutex.Lock()
		userStore[username.(string)].Username = newusername
		mutex.Unlock()

		// 更新会话
		session.Set("username", newusername)
		session.Save()
	}

	// 更新姓名
	if newname != "" {
		mutex.Lock()
		userStore[username.(string)].Name = newname
		mutex.Unlock()
	}

	// 持久化保存
	if err := saveUserstoFiles(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "资料更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "资料修改成功",
	})
}

// viewUserdataHandler 查看所有用户数据（管理员功能）
// @Summary 查看用户数据
// @Description 管理员查看所有注册用户的数据
// @Tags 管理
// @Produce json
// @Success 200 {object} userListResponse
// @Failure 403 {object} errorResponse
// @Router /userdata [get]
func viewUserdataHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	// 权限检查
	if username == nil || username.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足，需要管理员权限",
		})
		return
	}

	mutex.RLock()
	usersData := make(map[string]gin.H)
	for username, user := range userStore {
		usersData[username] = gin.H{
			"Username": user.Username,
			"Name":     user.Name,
			"Password": user.Password,
		}
	}
	mutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"users": usersData,
		"count": len(usersData),
	})
}

// logoutHandler 处理用户登出
// @Summary 用户登出
// @Description 清除用户会话并重定向到登录页面
// @Tags 用户
// @Success 302 "登出成功，重定向到登录页"
// @Router /logout [get]
func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

// loadFiles 从文件加载用户数据
func loadFiles() error {
	// 检查文件是否存在
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		// 创建空用户文件
		emptyData := []byte("[]")
		if err := os.WriteFile(userFile, emptyData, 0644); err != nil {
			return fmt.Errorf("创建用户文件失败: %v", err)
		}
		return nil
	}

	// 读取文件内容
	data, err := os.ReadFile(userFile)
	if err != nil {
		return fmt.Errorf("读取用户数据失败: %v", err)
	}

	// 处理空文件
	if len(data) == 0 {
		data = []byte("[]")
	}

	// 解析JSON数据
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("解析用户数据失败: %v", err)
	}

	// 加载到内存
	mutex.Lock()
	defer mutex.Unlock()

	for i := range users {
		user := users[i]
		userStore[user.Username] = &user
	}

	return nil
}

// saveUserstoFiles 保存用户数据到文件
func saveUserstoFiles() error {
	mutex.RLock()
	defer mutex.RUnlock()

	// 准备数据
	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
		users = append(users, *user)
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户数据失败: %v", err)
	}

	// 原子写入：先写临时文件，再重命名
	tempFile := userFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	if err := os.Rename(tempFile, userFile); err != nil {
		return fmt.Errorf("文件重命名失败: %v", err)
	}

	return nil
}
