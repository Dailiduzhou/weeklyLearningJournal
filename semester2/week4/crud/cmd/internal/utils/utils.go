package utils

import (
	"time"

	"crud/cmd/internal/config"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func GenerateJTI() string {
	return uuid.New().String()
}

func GenerateAccessToken(c *config.Config, ID int64, role, jti string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"ID":   ID,
		"Role": role,
		"jti":  jti,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(c.JwtAuth.AccessExpire) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.JwtAuth.AccessSecret))
}

func GenerateRefreshToken(c *config.Config, ID int64, jti string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"ID":  ID,
		"jti": jti,
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(c.JwtAuth.RefreshExpire) * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.JwtAuth.RefreshSecret))
}

func ParseRefreshToken(c *config.Config, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return []byte(c.JwtAuth.RefreshSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.NewValidationError("invalid token", jwt.ValidationErrorClaimsInvalid)
}
