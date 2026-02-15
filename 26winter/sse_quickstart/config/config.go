// Package config handles application configuration
package config

import (
	"log"
	"os"
)

type Config struct {
	Port         string
	JWTSecret    string
	CORSOrigin   string
	TokenIssuer  string
	TokenSubject string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", ":8080"),
		JWTSecret:    getEnv("JWT_SECRET", "18e9ba09a3566f97de630c21331ebc11"),
		CORSOrigin:   getEnv("CORS_ORIGIN", "http://localhost:3000"),
		TokenIssuer:  getEnv("TOKEN_ISSUER", "dailiduzhou"),
		TokenSubject: getEnv("TOKEN_SUBJECT", "Billboard"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Printf("[WARN] Failed to get %s, using default value: %s\n", key, fallback)
		return fallback
	}
	return value
}
