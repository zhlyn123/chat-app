package controller

import (
	"chat-app/internal/application/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Register creates a new user.
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "invalid request body",
		})
	}
	if err := service.Register(req.Username, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "register success",
	})
}

// Login authenticates a user.
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "invalid request body",
		})
		return
	}

	loginResp, err := service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// GetUserInfo returns the current authenticated user.
func GetUserInfo(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user info"})
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

// GetAllUsersWithStatus returns users with online status.
func GetAllUsersWithStatus(c *gin.Context) {
	meID := c.GetUint("user_id")

	users, err := service.GetALLUsers(meID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	online := service.GetOnlineUsers()

	var res []gin.H

	for _, u := range users {
		res = append(res, gin.H{
			"id":       u.ID,
			"username": u.Username,
			"online":   online[u.ID] != "",
		})
	}
	c.JSON(http.StatusOK, res)
}

func Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid authorization header"})
		return
	}

	if err := service.BlacklistToken(parts[1]); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "logged out"})
}
