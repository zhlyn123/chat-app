package main

import (
	"chat-app/internal/domain/model"
	"chat-app/internal/infrastructure/config"
	"chat-app/internal/infrastructure/logger"
	"chat-app/internal/interfaces/http/router"
	"chat-app/internal/interfaces/websocket"
)

func main() {
	logger.InitLogger()
	config.LoadEnvFile(".env")
	config.InitJWT()
	config.InitDB()
	config.InitRedis()

	hub := websocket.NewHub()
	go hub.Run()
	go websocket.SubscribeMessages(hub)
	config.DB.AutoMigrate(&model.User{}, &model.Message{}, &model.Group{}, &model.GroupMember{})

	r := router.InitRouter(hub)
	r.Run(":" + config.GetEnv("APP_PORT", "8181"))
}
