package router

import(
	"github.com/gin-gonic/gin"
	"chat-app/controller"
)

func InitRouter() *gin.Engine{
	r := gin.Default()
	// 注册路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, world!",
		})
	})
	r.POST("/Register", controller.Register)
	r.POST("/Login", controller.Login)
	return r
}