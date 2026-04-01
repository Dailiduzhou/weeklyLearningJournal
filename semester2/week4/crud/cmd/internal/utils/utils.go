package utils

import (
	"time"

	"crud/cmd/internal/config"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateAccessToken(c *config.Config, ID int64, role string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"ID":   ID,
		"Role": role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(c.JwtAuth.AccessExpire) * time.Second),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.JwtAuth.AccessSecret))
}

func GenerateRefreshToken(c *config.Config, ID int64) (string, error) {
}
