package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Dailiduzhou/library_manage_sys/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func InitSession(r *gin.Engine) error {
	useRedis := os.Getenv("USE_REDIS")
	if useRedis == "" {
		useRedis = "true"
	}

	sessionSecret := make([]byte, 32)

	sessionSecret = []byte(os.Getenv("SESSION_SECRET"))
	if len(sessionSecret) == 0 {
		sessionSecret = []byte("aebd2a80a82c5067554dc481e1dc7615d8d30c075e1424d6843d934479e4786e")
	}

	var store sessions.Store
	var err error

	store, err = initRedisStore(sessionSecret)
	if err != nil {
		log.Printf("Redis 存储初始化失败，回退到 Cookie 存储: %v", err)
		store = initCookieStore(sessionSecret)
	}

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   false,
		SameSite: 0,
	})

	r.Use(sessions.Sessions("mysession", store))

	log.Println("Session 中间件初始化完成")
	return nil
}

func initRedisStore(sessionSecret []byte) (sessions.Store, error) {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	store, err := redis.NewStore(
		10,
		"tcp",
		redisAddr,
		"",
		redisPassword,
		sessionSecret,
	)

	if err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	log.Println("Redis 存储初始化成功")
	return store, nil
}

func initCookieStore(sessionSecret []byte) sessions.Store {
	log.Println("使用 Cookie 存储（开发环境）")
	store := cookie.NewStore(sessionSecret)
	return store
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, models.Response{
				Code: 401,
				Msg:  "请先登录",
			})
			c.Abort()
			return
		}

		var finalID uint

		switch v := userID.(type) {
		case uint:
			finalID = v
		case int:
			finalID = uint(v)
		case float64:
			finalID = uint(v)
		case uint64:
			finalID = uint(v)
		default:

			c.JSON(http.StatusUnauthorized, models.Response{
				Code: 401,
				Msg:  "认证数据异常，请重新登录",
			})
			c.Abort()
			return
		}

		c.Set("user_id", finalID)

		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role")

		if roleStr, ok := role.(string); !ok {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "处理用户身份错误",
			})
			c.Abort()
			return
		} else if roleStr != "admin" {
			c.JSON(http.StatusUnauthorized, models.Response{
				Code: 401,
				Msg:  "权限不足",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
