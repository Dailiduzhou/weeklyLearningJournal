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

	mutex.RLock()
	userStore[username] = newUser
	mutex.RUnlock()

	if err := saveUserstoFiles(); err != nil {
		mutex.Lock()
		delete(userStore, name)
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

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理密码时发生内部错误"})
		return
	}

	storedpassword := userStore[username].Password
	if string(hashedPasswordBytes) != storedpassword {
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
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或已过期的认证令牌"})
		c.Abort()
		return
	}
	mutex.RLock()
	c.HTML(http.StatusAccepted, "profiles.html", gin.H{
		"username": userStore[username].Username,
		"password": userStore[username].Password,
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
	password := c.PostForm("password")
	password1 := c.PostForm("password1")

	mutex.RLock()
	_, exists := userStore[username]
	mutex.RUnlock()
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户不存在",
			"msg":   "即将跳转……",
		})
		time.Sleep(1 * time.Second)

	}
	if password != password1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "两次密码不一致",
		})
		time.Sleep(1 * time.Second)
		c.Redirect(http.StatusOK, "http://localhost:8080/changepassword")
	}

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
