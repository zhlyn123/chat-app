package router

import (
	"chat-app/controller"
	"chat-app/middleware"
	"chat-app/websocket"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	// 注册路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, world!",
		})
	})

	//注册
	r.POST("/register", controller.Register)

	//登录
	r.POST("/login", controller.Login)

	//WebSocket 服务
	r.GET("/ws", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})
	
	//受保护的路由
	auth := r.Group("/api")
	auth.Use(middleware.JWTAuth())
	{
		auth.GET("/userinfo", controller.UserInfo)
	}

	return r
}
