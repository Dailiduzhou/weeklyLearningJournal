package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"-"` // 不在JSON中序列化
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name"`
}

type RegisterRequest struct {
	Body struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
		Name     string `json:"name" form:"name"`
	} `json:"body"`
}

type LoginRequest struct {
	Body struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
	} `json:"body"`
}

type ChangePasswordRequest struct {
	Body struct {
		OldPassword string `json:"oldpassword" form:"oldpassword"`
		Password    string `json:"password" form:"password"`
		Password1   string `json:"password1" form:"password1"`
	} `json:"body"`
}

type UpdateProfileRequest struct {
	Body struct {
		Username string `json:"username,omitempty" form:"username"`
		Name     string `json:"name,omitempty" form:"name"`
	} `json:"body"`
}

type UserListResponse struct {
	Body struct {
		Users []User `json:"users"`
		Count int    `json:"count"`
	} `json:"body"`
}

type UserResponse struct {
	Body User `json:"body"`
}

type SuccessResponse struct {
	Body struct {
		Message string `json:"message"`
	} `json:"body"`
}

type ErrorResponse struct {
	Body struct {
		Error string `json:"error"`
	} `json:"body"`
}

var (
	userStore     = make(map[string]*User)
	userFile      = "users.json"
	mutex         sync.RWMutex
	sessionSecret = make([]byte, 32)
	db            *sql.DB
	ctx           = context.Background()
)

func main() {
	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号,正在保存数据...")
		if err := persistUsers(); err != nil {
			fmt.Printf("保存数据失败: %v\n", err)
		}
		fmt.Println("数据已保存,程序退出")
		os.Exit(0)
	}()

	initMySQL()

	if err := loadData(); err != nil {
		fmt.Printf("用户数据加载失败: %v\n", err)
		fmt.Println("将以空用户数据库启动...")
	} else {
		fmt.Printf("成功加载 %d 个用户数据\n", len(userStore))
	}

	if _, err := rand.Read(sessionSecret); err != nil {
		fmt.Printf("生成会话密钥失败: %v", err)
		return
	}

	r := gin.Default()

	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("authSession", store))

	r.LoadHTMLGlob("./template/*.html")

	registerRoutes(r)

	fmt.Println("服务器启动在 :8080")
	fmt.Println("API文档地址: http://localhost:8080/docs")
	if err := r.Run(":8080"); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}

func initMySQL() {
	var err error
	db, err = sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/ginserver?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		fmt.Printf("MySQL连接失败: %v\n", err)
		fmt.Println("将继续使用文件存储作为主存储")
		return
	}

	if err := db.Ping(); err != nil {
		fmt.Printf("MySQL连接失败: %v\n", err)
		fmt.Println("将继续使用文件存储作为主存储")
	} else {
		fmt.Println("成功连接到MySQL服务器")
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL
		)`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			fmt.Printf("创建用户表失败: %v\n", err)
			fmt.Println("将继续使用文件存储作为主存储")
		} else {
			fmt.Println("成功创建/确认用户表")
		}
	}
}

func loadData() error {
	// 1. 从文件加载
	if err := loadFiles(); err != nil {
		return err
	}

	// 2. 如果MySQL已连接,同步文件数据到MySQL
	if db != nil {
		fmt.Println("检测到MySQL连接,同步文件数据到数据库...")
		if err := persistUsers(); err != nil {
			fmt.Printf("同步文件数据到MySQL失败: %v\n", err)
		}
	}
	return nil
}

func loadFiles() error {
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		emptyData := []byte("[]")
		if err := os.WriteFile(userFile, emptyData, 0644); err != nil {
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
		// 重置ID为0,因为从文件加载不需要ID
		user.ID = 0
		userStore[user.Username] = &user
	}

	return nil
}

func saveToFiles() error {
	mutex.RLock()
	defer mutex.RUnlock()

	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
		u := *user
		u.ID = 0
		users = append(users, u)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户数据失败: %v", err)
	}

	tempFile := userFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	if err := os.Rename(tempFile, userFile); err != nil {
		return fmt.Errorf("文件重命名失败: %v", err)
	}

	return nil
}

func persistUsers() error {
	// 优先保存到文件
	if err := saveToFiles(); err != nil {
		return err
	}

	// 如果MySQL已连接,尝试同步到MySQL
	if db != nil {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			fmt.Printf("开始事务失败: %v\n", err)
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec("TRUNCATE TABLE users")
		if err != nil {
			fmt.Printf("清空MySQL用户表失败: %v\n", err)
			return err
		}

		stmt, err := tx.Prepare("INSERT INTO users (username, password, name) VALUES (?, ?, ?)")
		if err != nil {
			fmt.Printf("准备插入语句失败: %v\n", err)
			return err
		}
		defer stmt.Close()

		mutex.RLock()
		defer mutex.RUnlock()

		for _, user := range userStore {
			result, err := stmt.Exec(user.Username, user.Password, user.Name)
			if err != nil {
				fmt.Printf("保存用户到MySQL失败: %v\n", err)
				return err
			}

			id, err := result.LastInsertId()
			if err != nil {
				fmt.Printf("获取最后插入ID失败: %v\n", err)
				return err
			}

			user.ID = int(id)
		}

		if err := tx.Commit(); err != nil {
			fmt.Printf("提交事务失败: %v\n", err)
			return err
		}
	}

	return nil
}

func registerRoutes(r *gin.Engine) {
	r.GET("/swagger.json", swaggerJSONHandler)
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/*any", swaggerUIHandler)

	r.GET("/login", loginHandler)
	r.POST("/api/auth/login", loginAPIHandler)
	r.POST("/api/auth/logout", logoutAPIHandler)

	r.GET("/register", registerHandler)
	r.POST("/api/users", registerAPIHandler)

	r.GET("/profiles", profilePageHandler)
	r.GET("/api/users/me", getCurrentUserHandler)
	r.PUT("/api/users/me", updateUserProfileHandler)
	r.PUT("/api/users/me/password", changePasswordHandler)

	r.GET("/changeprofiles", changeProfilePageHandler)
	r.GET("/changepassword", changePasswordPageHandler)

	r.GET("/admin/users", adminUserListPageHandler)
	r.GET("/api/users", listUsersHandler)

	r.GET("/userdata", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/users")
	})
}

func swaggerJSONHandler(c *gin.Context) {
	c.File("./swagger.json")
}

func swaggerUIHandler(c *gin.Context) {
	swaggerUIHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>用户管理API文档</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3/swagger-ui.css  ">
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
    <script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js  "></script>
    <script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-standalone-preset.js  "></script>
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

func registerHandler(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{
		"title": "用户注册",
	})
}

func registerAPIHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	name := c.PostForm("name")

	if username == "" || password == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名、密码和姓名不能为空",
		})
		return
	}

	mutex.RLock()
	_, exists := userStore[username]
	mutex.RUnlock()

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名已存在",
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码加密失败",
		})
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

	if err := persistUsers(); err != nil {
		mutex.Lock()
		delete(userStore, username)
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "用户数据保存失败",
		})
		return
	}

	mutex.RLock()
	createdUser := userStore[username]
	mutex.RUnlock()

	c.JSON(http.StatusCreated, gin.H{
		"username": createdUser.Username,
		"name":     createdUser.Name,
	})
}

func loginHandler(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"title": "用户登录",
	})
}

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

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "密码错误",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("username", username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "会话创建失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

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

func getCurrentUserHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
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

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.Name,
	})
}

func changePasswordHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}

	oldpassword := c.PostForm("oldpassword")
	password := c.PostForm("password")
	password1 := c.PostForm("password1")

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

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "原密码错误",
		})
		return
	}

	if password != password1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "两次输入的新密码不一致",
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密码加密失败",
		})
		return
	}

	// 更新内存中的密码
	mutex.Lock()
	oldHash := userStore[username.(string)].Password
	userStore[username.(string)].Password = string(hashedPasswordBytes)
	mutex.Unlock()

	if err := persistUsers(); err != nil {
		// 回滚内存中的更改
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

func updateUserProfileHandler(c *gin.Context) {
	session := sessions.Default(c)
	sessUsername := session.Get("username")

	if sessUsername == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}
	currentUsername := sessUsername.(string)

	newUsername := c.PostForm("username")
	newName := c.PostForm("name")

	mutex.RLock()
	user, exists := userStore[currentUsername]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

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

	oldUsername := user.Username
	oldName := user.Name

	if newUsername != "" {
		mutex.Lock()
		if _, exists := userStore[newUsername]; exists {
			mutex.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "新用户名已存在",
			})
			return
		}

		// 更新用户名
		userStore[newUsername] = userStore[oldUsername]
		userStore[newUsername].Username = newUsername
		if newName != "" {
			userStore[newUsername].Name = newName
		}
		delete(userStore, oldUsername)
		mutex.Unlock()

		session.Set("username", newUsername)
		session.Save()
	} else if newName != "" {
		mutex.Lock()
		userStore[currentUsername].Name = newName
		mutex.Unlock()
	}

	if err := persistUsers(); err != nil {
		// 回滚操作
		mutex.Lock()
		if newUsername != "" {
			userStore[oldUsername] = userStore[newUsername]
			userStore[oldUsername].Username = oldUsername
			userStore[oldUsername].Name = oldName
			delete(userStore, newUsername)

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

	c.JSON(http.StatusOK, gin.H{
		"username": updatedUser.Username,
		"name":     updatedUser.Name,
	})
}

func adminUserListPageHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if username.(string) != "admin" {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"Message":      "权限不足,需要管理员权限",
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

func listUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证,请先登录",
		})
		return
	}

	if username.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足,需要管理员权限",
		})
		return
	}

	mutex.RLock()
	users := make([]User, 0, len(userStore))
	for _, user := range userStore {
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

func logoutAPIHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

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
