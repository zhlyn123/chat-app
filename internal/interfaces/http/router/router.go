package router

import (
	"chat-app/internal/interfaces/http/controller"
	"chat-app/internal/interfaces/http/middleware"
	"chat-app/internal/interfaces/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitRouter(hub *websocket.Hub) *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, world!",
		})
	})

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.POST("/register", controller.Register)
	r.POST("/login", controller.Login)

	r.GET("/ws", func(c *gin.Context) {
		websocket.ServeWs(hub, c.Writer, c.Request)
	})

	auth := r.Group("/api")
	auth.Use(middleware.JWTAuth())
	{
		auth.GET("/userinfo", controller.GetUserInfo)
		auth.POST("/logout", controller.Logout)
		auth.GET("/users", controller.GetAllUsersWithStatus)

		auth.GET("/messages/history", controller.GetHistory)
		auth.GET("/messages/unread", controller.GetUnreadCounts)
		auth.GET("/messages/unread/summary", controller.GetUnreadSummary)
		auth.POST("/messages/read", controller.MarkMessagesRead)
		auth.POST("/messages/revoke", func(c *gin.Context) {
			controller.RevokeMessage(c, hub)
		})

		auth.POST("/groups", controller.CreateGroup)
		auth.GET("/groups", controller.GetMyGroups)
		auth.POST("/groups/join", controller.JoinGroup)
		auth.POST("/groups/leave", controller.LeaveGroup)
		auth.GET("/groups/:id/members", controller.GetGroupMembers)
	}

	return r
}
