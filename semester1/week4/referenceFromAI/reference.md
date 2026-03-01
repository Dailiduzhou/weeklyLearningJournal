``` go
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
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// User 表示系统用户
// swagger:model User
type User struct {
	// 用户ID，唯一标识
	// Example: "1"
	ID string `json:"id,omitempty"`
	// 用户名，唯一标识
	// Example: "john_doe"
	Username string `json:"username"`
	// 密码（bcrypt加密存储）
	// Example: "$2a$10$xyz..."
	Password string `json:"password,omitempty"`
	// 用户真实姓名
	// Example: "张三"
	Name string `json:"name"`
}

// RegisterRequest 注册请求
// swagger:parameters createUser
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
// swagger:parameters changeUserPassword
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

// UpdateProfileRequest 更新资料请求
// swagger:parameters updateUserProfile
type UpdateProfileRequest struct {
	// in:body
	Body struct {
		// 新用户名，可选
		// Example: "new_john_doe"
		Username string `json:"username,omitempty" form:"username"`
		// 新姓名，可选
		// Example: "李四"
		Name string `json:"name,omitempty" form:"name"`
	} `json:"body"`
}

// UserListResponse 用户列表响应
// swagger:response userListResponse
type UserListResponse struct {
	// in:body
	Body struct {
		// 用户数据列表
		Users []User `json:"users"`
		// 用户数量
		Count int `json:"count"`
	} `json:"body"`
}

// UserResponse 单个用户响应
// swagger:response userResponse
type UserResponse struct {
	// in:body
	Body User `json:"body"`
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
	// Redis 客户端
	rdb *redis.Client
	// context
	ctx = context.Background()
)

// main 应用程序入口点
// @title 用户管理API
// @version 1.0.0
// @description 基于Gin框架的用户管理系统，提供完整的用户认证和个人信息管理功能
// @host localhost:8080
// @BasePath /
// @securityDefinitions.sessionAuth sessionAuth
func main() {
	// 初始化Redis连接
	initRedis()

	// 初始化：从文件加载用户数据（文件为主存储）
	if err := loadData(); err != nil {
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

// initRedis 初始化Redis连接
func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis默认地址
		Password: "",               // 无密码
		DB:       0,                // 默认DB
		PoolSize: 10,               // 连接池大小
	})

	// 检查Redis连接
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		fmt.Printf("Redis连接失败: %v\n", err)
		fmt.Println("将继续使用文件存储作为主存储")
	} else {
		fmt.Println("成功连接到Redis服务器")
	}
}

// loadData 加载用户数据，文件为主存储，Redis为副本
func loadData() error {
	// 从文件加载到内存
	if err := loadFiles(); err != nil {
		return err
	}

	// 如果 Redis 可用，将当前内存数据同步到 Redis
	if _, err := rdb.Ping(ctx).Result(); err == nil {
		fmt.Println("将文件数据同步到Redis")
		if err := saveToRedis(); err != nil {
			fmt.Printf("启动时同步到Redis失败: %v\n", err)
		}
	} else {
		fmt.Println("Redis不可用，跳过启动时同步")
	}

	return nil
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

// saveToRedis 保存用户数据到Redis（仅负责 Redis）
func saveToRedis() error {
	mutex.RLock()
	defer mutex.RUnlock()

	// 检查Redis连接
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis不可用: %v", err)
	}

	pipe := rdb.Pipeline()

	for username, user := range userStore {
		// 将用户序列化为JSON
		userData, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("序列化用户 %s 失败: %v", username, err)
		}

		// 设置Redis键 (格式: user:username)
		key := "user:" + username
		pipe.Set(ctx, key, userData, 0) // 0表示永不过期
	}

	// 执行管道命令
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("保存到Redis失败: %v", err)
	}

	return nil
}

// saveToFiles 保存用户数据到文件（主存储）
func saveToFiles() error {
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

// persistUsers 将内存数据同步到文件（必需）以及Redis（尽力而为）
func persistUsers() error {
	// 1. 必须先写文件，保证主存储成功
	if err := saveToFiles(); err != nil {
		return err
	}

	// 2. 再同步到 Redis，失败只记录日志，不阻塞主流程
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		fmt.Printf("Redis不可用，跳过同步: %v\n", err)
		return nil
	}

	if err := saveToRedis(); err != nil {
		fmt.Printf("同步到Redis失败: %v\n", err)
	}

	return nil
}

// registerRoutes 注册所有路由
func registerRoutes(r *gin.Engine) {
	// 文档路由
	r.GET("/swagger.json", swaggerJSONHandler)
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/*any", swaggerUIHandler)

	// 认证路由
	r.GET("/login", loginHandler)
	r.POST("/api/auth/login", loginAPIHandler)
	r.POST("/api/auth/logout", logoutAPIHandler)

	// 用户注册页面和API
	r.GET("/register", registerHandler)
	r.POST("/api/users", registerAPIHandler)

	// 当前用户相关路由
	r.GET("/profiles", profilePageHandler)
	r.GET("/api/users/me", getCurrentUserHandler)
	r.PUT("/api/users/me", updateUserProfileHandler)
	r.PUT("/api/users/me/password", changePasswordHandler)

	r.GET("/changeprofiles", changeProfilePageHandler)  // 返回修改资料页面
	r.GET("/changepassword", changePasswordPageHandler) // 返回修改密码页面

	// 管理路由
	r.GET("/admin/users", adminUserListPageHandler)
	r.GET("/api/users", listUsersHandler)

	// 兼容旧路由（可选）
	r.GET("/userdata", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/users")
	})
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

// registerAPIHandler 处理用户注册请求 (RESTful API)
// @Summary 创建新用户
// @Description 创建新用户账户
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "用户名" minLength(3) maxLength(20)
// @Param password formData string true "密码" minLength(6)
// @Param name formData string true "姓名" minLength(1) maxLength(50)
// @Success 201 {object} userResponse "用户创建成功"
// @Failure 400 {object} errorResponse "用户名、密码和姓名不能为空或用户名已存在"
// @Failure 500 {object} errorResponse "用户数据保存失败或密码加密失败"
// @Router /api/users [post]
func registerAPIHandler(c *gin.Context) {
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

	// 持久化（文件为主，Redis为辅）
	if err := persistUsers(); err != nil {
		// 文件失败时，回滚内存
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "用户数据保存失败",
		})
		return
	}

	// 返回创建的用户信息（不包含密码）
	c.JSON(http.StatusCreated, gin.H{
		"username": newUser.Username,
		"name":     newUser.Name,
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

// loginAPIHandler 处理用户登录请求 (RESTful API)
// @Summary 用户登录
// @Description 验证用户凭证并创建会话
// @Tags 认证
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "用户名"
// @Param password formData string true "密码"
// @Success 200 {object} userResponse "登录成功，返回用户信息"
// @Failure 400 {object} errorResponse "用户名和密码不能为空"
// @Failure 403 {object} errorResponse "密码错误"
// @Router /api/auth/login [post]
func loginAPIHandler(c *gin.Context) {
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

	// 返回用户信息（不包含密码）
	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

// profilePageHandler 返回用户个人资料页面
// @Summary 个人资料页面
// @Description 显示当前登录用户的个人资料
// @Tags 页面
// @Produce html
// @Success 200 {string} string "个人资料页面HTML"
// @Failure 302 {string} string "未登录，重定向到登录页"
// @Router /profiles [get]
func profilePageHandler(c *gin.Context) {
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

// getCurrentUserHandler 获取当前登录用户信息 (RESTful API)
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 用户
// @Produce json
// @Security sessionAuth
// @Success 200 {object} userResponse "返回当前用户信息"
// @Failure 401 {object} errorResponse "未认证"
// @Router /api/users/me [get]
func getCurrentUserHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证，请先登录",
		})
		return
	}

	mutex.RLock()
	user, exists := userStore[username.(string)]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户不存在",
		})
		return
	}

	// 返回用户信息（不包含密码）
	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

// changePasswordHandler 修改当前用户密码 (RESTful API)
// @Summary 修改当前用户密码
// @Description 修改当前登录用户的密码
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param oldpassword formData string true "原密码"
// @Param password formData string true "新密码"
// @Param password1 formData string true "确认新密码"
// @Security sessionAuth
// @Success 200 {object} successResponse "密码修改成功"
// @Failure 400 {object} errorResponse "所有密码字段不能为空或原密码错误或两次输入的新密码不一致或用户不存在"
// @Failure 401 {object} errorResponse "未认证"
// @Failure 500 {object} errorResponse "密码加密失败或密码更新失败"
// @Router /api/users/me/password [put]
func changePasswordHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证，请先登录",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

	// 验证原密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "原密码错误",
		})
		return
	}

	// 验证新密码一致性
	if password != password1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "两次输入的新密码不一致",
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

	// 更新密码（记录旧密码用于失败回滚）
	mutex.Lock()
	oldHash := userStore[username.(string)].Password
	userStore[username.(string)].Password = string(hashedPasswordBytes)
	mutex.Unlock()

	// 持久化保存
	if err := persistUsers(); err != nil {
		// 文件失败时，将内存回滚
		mutex.Lock()
		userStore[username.(string)].Password = oldHash
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// updateUserProfileHandler 更新当前用户资料 (RESTful API)
// @Summary 更新当前用户资料
// @Description 更新当前登录用户的用户名和姓名
// @Tags 用户
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string false "新用户名"
// @Param name formData string false "新姓名"
// @Security sessionAuth
// @Success 200 {object} userResponse "资料修改成功，返回更新后的用户信息"
// @Failure 400 {object} errorResponse "未检测到有效更改或新用户名与原用户名相同或新姓名与原姓名相同"
// @Failure 401 {object} errorResponse "未认证"
// @Failure 500 {object} errorResponse "资料更新失败"
// @Router /api/users/me [put]
func updateUserProfileHandler(c *gin.Context) {
	session := sessions.Default(c)
	sessUsername := session.Get("username")

	if sessUsername == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证，请先登录",
		})
		return
	}
	currentUsername := sessUsername.(string)

	newUsername := c.PostForm("username")
	newName := c.PostForm("name")

	mutex.RLock()
	user := userStore[currentUsername]
	mutex.RUnlock()

	// 检查是否有有效更改
	if newName == "" && newUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "未检测到有效更改",
		})
		return
	}

	if newUsername != "" && newUsername == user.Username {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "新用户名与原用户名相同",
		})
		return
	}

	if newName != "" && newName == user.Name {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "新姓名与原姓名相同",
		})
		return
	}

	// 记录旧值用于失败时回滚
	oldUsername := user.Username
	oldName := user.Name

	// 在内存中更新
	if newUsername != "" {
		mutex.Lock()
		// 检查新用户名是否已存在
		if _, exists := userStore[newUsername]; exists {
			mutex.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "新用户名已存在",
			})
			return
		}
		userStore[newUsername] = userStore[oldUsername]
		userStore[newUsername].Username = newUsername
		if newName != "" {
			userStore[newUsername].Name = newName
		}
		delete(userStore, oldUsername)
		mutex.Unlock()

		// 更新会话
		session.Set("username", newUsername)
		session.Save()
	} else if newName != "" {
		// 只更新姓名
		mutex.Lock()
		userStore[currentUsername].Name = newName
		mutex.Unlock()
	}

	// 持久化保存
	if err := persistUsers(); err != nil {
		// 文件失败时，回滚内存和会话
		mutex.Lock()
		if newUsername != "" {
			// 将用户还原到旧用户名和旧姓名
			userStore[oldUsername] = userStore[newUsername]
			userStore[oldUsername].Username = oldUsername
			userStore[oldUsername].Name = oldName
			delete(userStore, newUsername)

			// 会话还原
			session.Set("username", oldUsername)
			session.Save()
		} else if newName != "" {
			userStore[currentUsername].Name = oldName
		}
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "资料更新失败",
		})
		return
	}

	// 获取更新后的用户
	var updatedUser *User
	if newUsername != "" {
		mutex.RLock()
		updatedUser = userStore[newUsername]
		mutex.RUnlock()
	} else {
		mutex.RLock()
		updatedUser = userStore[currentUsername]
		mutex.RUnlock()
	}

	// 返回更新后的用户信息
	c.JSON(http.StatusOK, gin.H{
		"username": updatedUser.Username,
		"name":     updatedUser.Name,
	})
}

// adminUserListPageHandler 返回管理员用户列表页面
// @Summary 管理员用户列表页面
// @Description 返回管理员查看用户列表的页面
// @Tags 管理
// @Produce html
// @Success 200 {string} string "管理员用户列表页面HTML"
// @Failure 302 {string} string "未登录或非管理员，重定向到登录页"
// @Router /admin/users [get]
func adminUserListPageHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if username.(string) != "admin" {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"Message":      "权限不足，需要管理员权限",
			"RedirectURL":  "/profiles",
			"RedirectName": "个人资料页",
			"Delay":        3000,
		})
		return
	}

	mutex.RLock()
	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
		users = append(users, *user)
	}
	mutex.RUnlock()

	c.HTML(http.StatusOK, "admin_users.html", gin.H{
		"users": users,
	})
}

// listUsersHandler 获取所有用户列表 (RESTful API)
// @Summary 获取所有用户列表
// @Description 管理员获取所有注册用户的数据
// @Tags 管理
// @Produce json
// @Security sessionAuth
// @Success 200 {object} userListResponse "返回所有用户数据"
// @Failure 401 {object} errorResponse "未认证"
// @Failure 403 {object} errorResponse "权限不足，需要管理员权限"
// @Router /api/users [get]
func listUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证，请先登录",
		})
		return
	}

	// 权限检查
	if username.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足，需要管理员权限",
		})
		return
	}

	mutex.RLock()
	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
		// 创建不包含密码的用户副本
		users = append(users, User{
			Username: user.Username,
			Name:     user.Name,
		})
	}
	mutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// logoutAPIHandler 处理用户登出 (RESTful API)
// @Summary 用户登出
// @Description 清除用户会话
// @Tags 认证
// @Success 200 {object} successResponse "登出成功"
// @Router /api/auth/logout [post]
func logoutAPIHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

// changeProfilePageHandler 返回修改资料页面
// @Summary 修改资料页面
// @Description 返回修改用户资料表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "修改资料页面HTML"
// @Failure 302 {string} string "未登录，重定向到登录页"
// @Router /changeprofiles [get]
func changeProfilePageHandler(c *gin.Context) {
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

// changePasswordPageHandler 返回修改密码页面
// @Summary 修改密码页面
// @Description 返回修改密码表单页面
// @Tags 页面
// @Produce html
// @Success 200 {string} string "修改密码页面HTML"
// @Failure 302 {string} string "未登录，重定向到登录页"
// @Router /changepassword [get]
func changePasswordPageHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "修改密码",
	})
}
```