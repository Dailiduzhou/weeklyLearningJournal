// Package utils implements basic needs
package utils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"sse/model"

	"github.com/golang-jwt/jwt/v5"
)

var mySecret = []byte("18e9ba09a3566f97de630c21331ebc11")

func getJWTSecret() []byte {
	return []byte(getEnv("JWT_SECRET", string(mySecret)))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Printf("[WARN] Failed to get %s, using default value: %s\n", key, fallback)
		return fallback
	} else {
		return value
	}
}

func GenerateToken(userID uint32) (string, error) {
	secret := getJWTSecret()

	expireTime := time.Now().Add(2 * time.Hour)

	claims := model.MyClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "dailiduzhou",
			ExpiresAt: jwt.NewNumericDate(expireTime),
			Subject:   "Billboard",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string) (uint32, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.MyClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return 0, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*model.MyClaims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, errors.New("invalid token")
}

func ParseAuthHeader(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
