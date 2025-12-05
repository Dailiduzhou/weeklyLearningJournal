package middleware

import (
	"crypto/rand"
	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func InitSession(r *gin.Engine) {
	var sessionSecret = make([]byte, 32)
	if _, err := rand.Read(sessionSecret); err != nil {
		log.Fatalf("生成会话密钥失败: %q", err)
	}

	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("AuthSession", store))
}
