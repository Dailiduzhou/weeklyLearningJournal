package middleware

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func InitSession(r *gin.Engine) error {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	sessionSecret := make([]byte, 32)
	if _, err := rand.Read(sessionSecret); err != nil {
		log.Printf("生成会话密钥失败: %v", err)
		sessionSecret = []byte("default-session-secret-for-development-only")
	}

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	log.Printf("尝试连接 Redis: %s (DB: %d)", redisAddr, redisDB)

	store, err := redis.NewStoreWithDB(
		10,
		"tcp",
		redisAddr,
		redisPassword,
		strconv.Itoa(redisDB),
		string(sessionSecret),
	)

	// 如果 Redis 连接失败，使用 Cookie 存储作为备选
	if err != nil {
		log.Printf("Redis 连接失败 (%v)，改用 Cookie 存储作为备选方案", err)
		cookieStore := cookie.NewStore(sessionSecret)
		store = cookieStore
	} else {
		log.Printf("Redis 连接成功")
	}

	r.Use(sessions.Sessions("mysession", store))
	log.Println("Session 中间件初始化成功")

	return nil
}
