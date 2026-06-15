package config

import (
	"chat-app/internal/infrastructure/logger"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis() {
	addr := GetEnv("REDIS_ADDR", "localhost:6379")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: GetEnv("REDIS_PASSWORD", ""),
		DB:       GetEnvInt("REDIS_DB", 0),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Logger.Warn("redis connection failed, fallback to local memory", "error", err)
		Redis = nil
		return
	}

	Redis = client
	logger.Logger.Info("redis connected", "addr", addr)
}
