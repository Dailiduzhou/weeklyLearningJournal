# Restore Users' info
``` go
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

var (
	db        *bbolt.DB
	jwtSecret = []byte("replace-with-secure-secret") // production: load from env/secret manager
)

const usersBucket = "users"

type User struct {
	PasswordHash []byte `json:"password_hash"`
	CreatedAt    int64  `json:"created_at"`
}

// Request DTOs
type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// 可通过环境变量替换 secret
	if s := os.Getenv("AUTH_JWT_SECRET"); s != "" {
		jwtSecret = []byte(s)
	}

	var err error
	db, err = bbolt.Open("users.db", 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Ensure bucket exists
	err = db.Update(func(tx *bbolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte(usersBucket))
		return e
	})
	if err != nil {
		log.Fatalf("create bucket: %v", err)
	}

	r := gin.Default()

	r.POST("/register", handleRegister)
	r.POST("/login", handleLogin)

	auth := r.Group("/").Use(jwtMiddleware())
	{
		auth.GET("/profile", handleProfile)
	}

	log.Println("server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

// Create user with bcrypt hashed password
func createUser(username, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password required")
	}

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(usersBucket))
		if b.Get([]byte(username)) != nil {
			return errors.New("user exists")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u := User{PasswordHash: hash, CreatedAt: time.Now().Unix()}
		data, err := json.Marshal(&u)
		if err != nil {
			return err
		}
		return b.Put([]byte(username), data)
	})
}

// Authenticate checks password correctness
func authenticateUser(username, password string) (bool, error) {
	var u User
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(usersBucket))
		v := b.Get([]byte(username))
		if v == nil {
			return errors.New("user not found")
		}
		return json.Unmarshal(v, &u)
	})
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func handleRegister(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := createUser(req.Username, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "user created"})
}

func handleLogin(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	ok, err := authenticateUser(req.Username, req.Password)
	if err != nil {
		// user not found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// create JWT token
	claims := jwt.MapClaims{
		"sub": req.Username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": s})
}

func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		// expect "Bearer <token>"
		var tokenString string
		_, _ = fmt.Sscanf(auth, "Bearer %s", &tokenString)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// validate signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}
		// set username into context
		c.Set("username", sub)
		c.Next()
	}
}

func handleProfile(c *gin.Context) {
	username, _ := c.Get("username")
	c.JSON(http.StatusOK, gin.H{
		"user": username,
		"msg":  "this is protected profile",
	})
}
```