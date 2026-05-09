package controller

import (
	"chat-app/config"
	"chat-app/model"
	"chat-app/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 注册
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "请求参数错误",
		})
	}
	if err := service.Register(req.Username, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "注册成功",
	})
}

// 登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "请求参数错误",
		})
		return
	}

	token, err := service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// UserInfo 返回当前登录用户信息
func UserInfo(c *gin.Context) {
	// 从 JWT 中间件 context 取 user_id
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到用户信息"})
		return
	}

	userID := userIDInterface.(uint)

	// 查询数据库
	var user model.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	// 返回安全信息，不返回密码
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"created":  user.CreatedAt,
	})
}

func GetUserInfo(c *gin.Context){
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到用户信息"})
		return
	}
	
	userID := userIDInterface.(uint)
	user, err := service.GetUserInfo(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"created":  user.CreatedAt,
	})
}
