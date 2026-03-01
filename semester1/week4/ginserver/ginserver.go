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

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

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

	r.Run(":8080")
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
			"error": "保存用户数据失败",
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

func profilesHandler(c *gin.Context) {
	tokenString := c.Query("token")
	username := c.Query("username")
	if !jwtValidator(c, tokenString) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
		return
	}

	mutex.RLock()
	c.HTML(http.StatusOK, "profiles.html", gin.H{
		"username": userStore[username].Username,
		"name":     userStore[username].Name,
	})
	mutex.RUnlock()
}

func changePasswordHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "changepassword.html", gin.H{
		"title": "changePasswordhandler",
	})
}

func changepasswordhandler1(c *gin.Context) {
	username := c.PostForm("username")
	oldpassword := c.PostForm("oldpassword")
	password := c.PostForm("password")
	password1 := c.PostForm("password1")

	mutex.RLock()
	user, exists := userStore[username]
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

	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.Write([]byte(`
        <script>
            alert("密码修改成功！");
            setTimeout(function() {
                window.location.href = "/login";
            }, 1000);
        </script>
        <p>密码修改成功，1秒后跳转到登录页面...</p>
    `))
}

func changeprofileHandler(c *gin.Context) {
	tokenString := c.Query("token")
	username := c.Query("username")

	if !jwtValidator(c, tokenString) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
		return
	}

	c.HTML(http.StatusOK, "changeprofiles.html", gin.H{
		"username": username,
	})
}

func changeprofileHandler1(c *gin.Context) {
	newusername := c.PostForm("newusername")
	newname := c.PostForm("newname")
	username := c.PostForm("username")

	mutex.RLock()
	user := userStore[username]
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

func viewUserdataHandler(c *gin.Context) {
	username := c.Query("username")
	tokenString := c.Query("token")

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
