package main

import (
	"chat-app/config"
	"chat-app/router"
	"chat-app/model"
)

func main() {
	// 初始化数据库
	config.InitDB()

	//自动迁移数据库
	config.DB.AutoMigrate(&model.User{})

	r := router.InitRouter()

	r.Run(":8181")
}