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
//	JWT:
//	     type: apiKey
//	     name: Authorization
//	     in: header
//	     description: JWT认证令牌，格式为"Bearer {token}"
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
// swagger:model User
type User struct {
	// 用户名
	// required: true
	// example: "john_doe"
	Username string `json:"username"`

	// 密码
	// required: true
	// example: "password123"
	Password string `json:"password"`

	// 姓名
	// required: true
	// example: "John Doe"
	Name string `json:"name"`
}

// LoginRequest 登录请求
// swagger:model LoginRequest
type LoginRequest struct {
	// 用户名
	// required: true
	// example: "john_doe"
	Username string `json:"username"`

	// 密码
	// required: true
	// example: "password123"
	Password string `json:"password"`
}

// RegisterRequest 注册请求
// swagger:model RegisterRequest
type RegisterRequest struct {
	// 用户名
	// required: true
	// example: "john_doe"
	Username string `json:"username"`

	// 密码
	// required: true
	// example: "password123"
	Password string `json:"password"`

	// 姓名
	// required: true
	// example: "John Doe"
	Name string `json:"name"`
}

type ProfilesRequest struct {
	Body struct {
		Username string `json:"username" binding:"required"`
		Token    string `json:"token" binding:"required"`
	} `json:"body"`
}

// ChangePasswordRequest 修改密码请求
// swagger:model ChangePasswordRequest
type ChangePasswordRequest struct {
	// 用户名
	// required: true
	// example: "john_doe"
	Username string `json:"username"`

	// 原密码
	// required: true
	// example: "oldpassword123"
	OldPassword string `json:"oldpassword"`

	// 新密码
	// required: true
	// example: "newpassword123"
	Password string `json:"password"`

	// 确认新密码
	// required: true
	// example: "newpassword123"
	Password1 string `json:"password1"`
}

// ChangeProfileRequest 修改个人信息请求
// swagger:model ChangeProfileRequest
type ChangeProfileRequest struct {
	// 原用户名
	// required: true
	// example: "john_doe"
	Username string `json:"username"`

	// 新用户名
	// example: "john_new"
	NewUsername string `json:"newusername"`

	// 新姓名
	// example: "John New"
	NewName string `json:"newname"`
}

// APIResponse API响应
// swagger:model APIResponse
type APIResponse struct {
	// 消息
	Message string `json:"message,omitempty"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 用户数据
	Data interface{} `json:"data,omitempty"`
}

// CustomClaims JWT声明
// swagger:model CustomClaims
type CustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var (
	userStore = make(map[string]*User)
	userFile  = "users.json"
	mutex     sync.RWMutex
	jwtSecret = make([]byte, 32)
)

func main() {
	if err := loadFiles(); err != nil {
		fmt.Printf("加载失败: %v\n", err)
		fmt.Println("将以空用户数据库启动……")
	} else {
		fmt.Printf("成功加载%d个用户数据\n", len(userStore))
	}

	if _, err := rand.Read(jwtSecret); err != nil {
		fmt.Printf("error generating jwtSecret: %v", err)
		return
	}

	r := gin.Default()

	r.LoadHTMLGlob("./template/*.html")

	// 添加 Swagger 文档路由
	r.GET("/swagger.json", swaggerJSONHandler)
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/*any", swaggerUIHandler)

	// 原有路由
	r.GET("/register", registerHandler)
	r.POST("/register", registerHandler1)
	r.GET("/login", loginHandler)
	r.POST("/login", loginHandler1)
	r.GET("/profiles", profilesHandler)
	r.POST("/profiles", profilesHandler1)
	r.GET("/changepassword", changePasswordHandler)
	r.POST("/changepassword", changepasswordhandler1)
	r.GET("/changeprofiles", changeprofileHandler)
	r.POST("/changeprofiles", changeprofileHandler1)
	r.GET("/userdata", viewUserdataHandler)

	fmt.Println("服务器启动在 :8080")
	fmt.Println("API文档地址: http://localhost:8080/docs")
	r.Run(":8080")
}

// swagger:operation GET /swagger.json 文档 getSwaggerJSON
//
// 返回 Swagger JSON 格式的 API 文档
//
// ---
// produces:
// - application/json
// responses:
//
//	'200':
//	  description: 成功返回 Swagger JSON 文档
//	  schema:
//	    type: object
func swaggerJSONHandler(c *gin.Context) {
	c.File("./swagger.json")
}

// swagger:operation GET /docs 文档 getSwaggerUI
//
// 返回 Swagger UI 界面
//
// ---
// produces:
// - text/html
// responses:
//
//	'200':
//	  description: 成功返回 Swagger UI
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

// swagger:operation GET /register 页面 registerPage
//
// 返回用户注册页面
//
// ---
// produces:
// - text/html
// responses:
//
//	'200':
//	  description: 成功返回注册页面
//	  schema:
//	    type: string
func registerHandler(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{
		"title": "register",
	})
}

// swagger:operation POST /register 用户 registerUser
//
// 注册新用户
//
// ---
// consumes:
// - application/x-www-form-urlencoded
// produces:
// - application/json
// parameters:
//   - name: username
//     in: formData
//     description: 用户名
//     required: true
//     type: string
//   - name: password
//     in: formData
//     description: 密码
//     required: true
//     type: string
//   - name: name
//     in: formData
//     description: 姓名
//     required: true
//     type: string
//
// responses:
//
//	'202':
//	  description: 注册成功
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'400':
//	  description: 用户已存在或信息为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'500':
//	  description: 服务器内部错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func registerHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	name := c.PostForm("name")

	// 检查参数是否为空
	if username == "" || password == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "信息不能为空",
		})
		return
	}

	mutex.RLock()
	_, exists := userStore[username]
	mutex.RUnlock()

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户已存在",
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理密码时发生内部错误"})
		return
	}

	newUser := &User{
		Username: username,
		Password: string(hashedPasswordBytes),
		Name:     name,
	}

	mutex.Lock()
	userStore[username] = newUser
	mutex.Unlock()

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户数据失败",
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "注册成功！请登录",
	})
}

// swagger:operation GET /login 页面 loginPage
//
// 返回用户登录页面
//
// ---
// produces:
// - text/html
// responses:
//
//	'200':
//	  description: 成功返回登录页面
//	  schema:
//	    type: string
func loginHandler(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"title": "login",
	})
}

// swagger:operation POST /login 用户 loginUser
//
// 用户登录并返回JWT令牌
//
// ---
// consumes:
// - application/x-www-form-urlencoded
// produces:
// - application/json
// parameters:
//   - name: username
//     in: formData
//     description: 用户名
//     required: true
//     type: string
//   - name: password
//     in: formData
//     description: 密码
//     required: true
//     type: string
//
// responses:
//
//	'302':
//	  description: 登录成功，重定向到个人资料页面
//	'400':
//	  description: 用户不存在或信息为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'403':
//	  description: 密码错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'500':
//	  description: 服务器内部错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func loginHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 检查参数是否为空
	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "信息不能为空",
		})
		return
	}

	mutex.RLock()
	_, exists := userStore[username]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(userStore[username].Password), []byte(password))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "密码错误",
		})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &CustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ginserver",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	redirectURL := fmt.Sprintf("/profiles?token=%s&username=%s", tokenString, username)
	c.Redirect(http.StatusFound, redirectURL)
}

// swagger:operation GET /profiles 用户 getProfile
//
// 返回用户个人资料页面
//
// ---
// produces:
// - text/html
// parameters:
//   - name: token
//     in: query
//     description: JWT认证令牌
//     required: true
//     type: string
//   - name: username
//     in: query
//     description: 用户名
//     required: true
//     type: string
//
// responses:
//
//	'200':
//	  description: 成功返回个人资料页面
//	  schema:
//	    type: string
//	'400':
//	  description: 认证令牌和用户名不能为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'401':
//	  description: 无效或已过期的认证令牌
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func profilesHandler(c *gin.Context) {
	tokenString := c.Query("token")
	username := c.Query("username")

	// 检查参数是否为空
	if tokenString == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "认证令牌和用户名不能为空",
		})
		return
	}

	if !jwtValidator(c, tokenString) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
		return
	}

	mutex.RLock()
	c.HTML(http.StatusOK, "profiles.html", gin.H{
		"username":    userStore[username].Username,
		"name":        userStore[username].Name,
		"tokenString": tokenString,
	})
	mutex.RUnlock()
}

func profilesHandler1(c *gin.Context) {
	var req ProfilesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bad input",
		})
		return
	}

	if req.Body.Token == "" || req.Body.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "empty info",
		})
		return
	}

	if !jwtValidator(c, req.Body.Token) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "无效或已过期的认证令牌",
		})
		return
	}

	mutex.RLock()
	user, exists := userStore[req.Body.Username]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

	mutex.RLock()
	c.HTML(http.StatusOK, "profiles.html", gin.H{
		"username":    userStore[user.Username].Username,
		"name":        userStore[user.Username].Name,
		"tokenString": req.Body.Token,
	})
	mutex.RUnlock()
}

// swagger:operation GET /changepassword 页面 changePasswordPage
//
// 返回修改密码页面
//
// ---
// produces:
// - text/html
// responses:
//
//	'200':
//	  description: 成功返回修改密码页面
//	  schema:
//	    type: string
func changePasswordHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "changePasswordhandler",
	})
}

// swagger:operation POST /changepassword 用户 changePassword
//
// 修改用户密码
//
// ---
// consumes:
// - application/x-www-form-urlencoded
// produces:
// - text/html
// - application/json
// parameters:
//   - name: username
//     in: formData
//     description: 用户名
//     required: true
//     type: string
//   - name: oldpassword
//     in: formData
//     description: 原密码
//     required: true
//     type: string
//   - name: password
//     in: formData
//     description: 新密码
//     required: true
//     type: string
//   - name: password1
//     in: formData
//     description: 确认新密码
//     required: true
//     type: string
//
// responses:
//
//	'200':
//	  description: 密码修改成功
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'400':
//	  description: 用户不存在或两次密码不一致或信息为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'403':
//	  description: 原密码错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'500':
//	  description: 服务器内部错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func changepasswordhandler1(c *gin.Context) {
	username := c.PostForm("username")
	oldpassword := c.PostForm("oldpassword")
	password := c.PostForm("password")
	password1 := c.PostForm("password1")

	// 检查参数是否为空
	if username == "" || password == "" || oldpassword == "" || password1 == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "信息不能为空",
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

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "原密码错误",
		})
		return
	}

	if password != password1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "两次密码输入不一致",
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理密码时发生内部错误"})
		return
	}

	mutex.Lock()
	userStore[username].Password = string(hashedPasswordBytes)
	mutex.Unlock()

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户数据失败",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// swagger:operation GET /changeprofiles 页面 changeProfilePage
//
// 返回修改个人信息页面
//
// ---
// produces:
// - text/html
// parameters:
//   - name: token
//     in: query
//     description: JWT认证令牌
//     required: true
//     type: string
//   - name: username
//     in: query
//     description: 用户名
//     required: true
//     type: string
//
// responses:
//
//	'200':
//	  description: 成功返回修改个人信息页面
//	  schema:
//	    type: string
//	'400':
//	  description: 认证令牌和用户名不能为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'401':
//	  description: 无效或已过期的认证令牌
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func changeprofileHandler(c *gin.Context) {
	tokenString := c.Query("token")
	username := c.Query("username")

	// 检查参数是否为空
	if tokenString == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "认证令牌和用户名不能为空",
		})
		return
	}

	if !jwtValidator(c, tokenString) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
		return
	}

	c.HTML(http.StatusOK, "changeprofiles.html", gin.H{
		"username": username,
	})
}

// swagger:operation POST /changeprofiles 用户 changeProfile
//
// 修改用户个人信息
//
// ---
// consumes:
// - application/x-www-form-urlencoded
// produces:
// - application/json
// parameters:
//   - name: username
//     in: formData
//     description: 原用户名
//     required: true
//     type: string
//   - name: newusername
//     in: formData
//     description: 新用户名
//     type: string
//   - name: newname
//     in: formData
//     description: 新姓名
//     type: string
//
// responses:
//
//	'200':
//	  description: 信息修改成功
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'400':
//	  description: 未发生更改或用户名/姓名与原值相同或信息为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'500':
//	  description: 服务器内部错误
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func changeprofileHandler1(c *gin.Context) {
	newusername := c.PostForm("newusername")
	newname := c.PostForm("newname")
	username := c.PostForm("username")

	// 检查参数是否为空
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名不能为空",
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

	if newname == "" && newusername == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "未发生更改",
		})
		return
	}
	if newusername == user.Username {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名与原用户名相同",
		})
		return
	}

	if newname == user.Name {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "姓名与原姓名相同",
		})
		return
	}

	if newusername != "" {
		mutex.Lock()
		userStore[username].Username = newusername
		mutex.Unlock()
	}

	if newname != "" {
		mutex.Lock()
		userStore[username].Name = newname
		mutex.Unlock()
	}

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户数据失败",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "信息修改成功",
	})
}

// swagger:operation GET /userdata 用户 getUserData
//
// 管理员查看所有用户数据
//
// ---
// produces:
// - application/json
// parameters:
//   - name: username
//     in: query
//     description: 用户名（必须为admin）
//     required: true
//     type: string
//   - name: token
//     in: query
//     description: JWT认证令牌
//     required: true
//     type: string
//
// responses:
//
//	'200':
//	  description: 成功返回用户数据
//	  schema:
//	    type: object
//	    properties:
//	      users:
//	        type: object
//	        description: 用户数据映射
//	      count:
//	        type: integer
//	        description: 用户数量
//	'400':
//	  description: 用户名和认证令牌不能为空
//	  schema:
//	    $ref: '#/definitions/APIResponse'
//	'403':
//	  description: 权限不足或token失效
//	  schema:
//	    $ref: '#/definitions/APIResponse'
func viewUserdataHandler(c *gin.Context) {
	username := c.Query("username")
	tokenString := c.Query("token")

	// 检查参数是否为空
	if username == "" || tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名和认证令牌不能为空",
		})
		return
	}

	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足，禁止访问",
		})
		return
	}

	if !jwtValidator(c, tokenString) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "token失效",
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

// 辅助函数，不生成swagger文档
func loadFiles() error {
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		emptydata := []byte("[]")
		if err := os.WriteFile(userFile, emptydata, 0644); err != nil {
			return fmt.Errorf("创建用户文件失败: %v", err)
		}
		return nil
	}

	data, err := os.ReadFile(userFile)
	if err != nil {
		return fmt.Errorf("读取用户数据失败: %v", err)
	}

	if len(data) == 0 {
		data = []byte("[]")
	}

	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("解析用户数据失败: %v", err)
	}
	mutex.Lock()
	defer mutex.Unlock()

	for i := range users {
		user := users[i]
		userStore[user.Username] = &user
	}

	return nil
}

// 辅助函数，不生成swagger文档
func saveUserstoFiles() error {
	mutex.RLock()
	defer mutex.RUnlock()

	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
		users = append(users, *user)
	}

	data, err := json.MarshalIndent(users, " ", "    ")
	if err != nil {
		return fmt.Errorf("序列化用户时失败: %v", err)
	}

	tempFile := userFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件时失败: %v", err)
	}

	if err := os.Rename(tempFile, userFile); err != nil {
		return fmt.Errorf("重命名失败: %v", err)
	}

	return nil
}

// 辅助函数，不生成swagger文档
func jwtValidator(c *gin.Context, tokenString string) bool {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		if !c.Writer.Written() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
			c.Abort()
		}
		return false
	}
	return true
}
