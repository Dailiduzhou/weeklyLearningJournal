package middlewares

import (
	"net/http"
	"strings"

	domainErrors "forum/errors"
	"forum/services"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(jwtService services.JWTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			ctx.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			ctx.Abort()
			return
		}

		tokenString := parts[1]
		userID, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": domainErrors.ErrInvalidToken.Error()})
			ctx.Abort()
			return
		}

		ctx.Set("userID", userID)
		ctx.Next()
	}
}
