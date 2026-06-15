package config

import (
	"chat-app/internal/infrastructure/logger"
	"time"
)

var JWTSecret []byte

const TokenExpireTime = time.Hour * 24

func InitJWT() {
	secret := GetEnv("JWT_SECRET", "")
	if secret == "" {
		logger.Logger.Error("JWT_SECRET is required")
		panic("JWT_SECRET is required")
	}

	JWTSecret = []byte(secret)
}
