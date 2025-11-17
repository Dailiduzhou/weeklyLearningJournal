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

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

var (
	userStore     = make(map[string]*User)
	userFile      = "users.json"
	mutex         sync.RWMutex
	sessionSecret = make([]byte, 32)
)

func main() {
	if err := loadFiles(); err != nil {
		fmt.Printf("加载失败: %v\n", err)
		fmt.Println("将以空用户数据库启动……")
	} else {
		fmt.Printf("成功加载%d个用户数据\n", len(userStore))
	}

	if _, err := rand.Read(sessionSecret); err != nil {
		fmt.Printf("error generating session secret: %v", err)
		return
	}

	r := gin.Default()

	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("authSession", store))

	r.LoadHTMLGlob("./template/*.html")

	r.GET("/swagger.json", swaggerJSONHandler)
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/*any", swaggerUIHandler)

	r.GET("/register", registerHandler)
	r.POST("/register", registerHandler1)
	r.GET("/login", loginHandler)
	r.POST("/login", loginHandler1)
	r.GET("/profiles", profilesHandler)
	r.GET("/changepassword", changePasswordHandler)
	r.POST("/changepassword", changepasswordhandler1)
	r.GET("/changeprofiles", changeprofileHandler)
	r.POST("/changeprofiles", changeprofileHandler1)
	r.GET("/userdata", viewUserdataHandler)
	r.POST("/logout", logoutHandler)

	fmt.Println("服务器启动在 :8080")
	fmt.Println("API文档地址: http://localhost:8080/docs")
	r.Run(":8080")
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

func registerHandler(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{
		"title": "register",
	})
}

func registerHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	name := c.PostForm("name")

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
			"error极简主义": "保存用户数据失败",
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "注册成功！请登录",
	})
}

func loginHandler(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"title": "login",
	})
}

func loginHandler1(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	mutex.RLock()
	user, exists := userStore[username]
	mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "密码错误",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("username", username)
	session.Save()

	c.Redirect(http.StatusFound, "/profiles")
}

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

func changePasswordHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "changePasswordhandler",
	})
}

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

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Message":      "原密码错误",
			"RedirectURL":  "/changepassword",
			"RedirectName": "修改密码页面",
			"Delay":        1000,
		})
		return
	}

	if password != password1 {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Message":      "两次密码输入不一致",
			"RedirectURL":  "/changepassword",
			"RedirectName": "修改密码页面",
			"Delay":        1000,
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理密码时发生内部错误"})
		return
	}

	mutex.Lock()
	userStore[username.(string)].Password = string(hashedPasswordBytes)
	mutex.Unlock()

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, username.(string))
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户数据失败",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

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
		userStore[username.(string)].Username = newusername
		mutex.Unlock()

		session.Set("username", newusername)
		session.Save()
	}

	if newname != "" {
		mutex.Lock()
		userStore[username.(string)].Name = newname
		mutex.Unlock()
	}

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, username.(string))
		mutex.Unlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户数据极简主义失败",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "信息修改成功",
	})
}

func viewUserdataHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil || username.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足，禁止访问",
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

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

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
