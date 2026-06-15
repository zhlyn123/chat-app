package controller

import (
	"chat-app/internal/application/service"
	"chat-app/internal/interfaces/websocket"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	targetIDStr := c.Query("target_id")
	isGroupStr := c.Query("is_group")
	beforeIDStr := c.DefaultQuery("before_id", "0")
	limitStr := c.DefaultQuery("limit", "20")

	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil || targetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_id"})
		return
	}

	isGroup, err := strconv.ParseBool(isGroupStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is_group"})
		return
	}

	beforeID, err := strconv.ParseUint(beforeIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before_id"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := service.GetMessageHistory(userID, uint(targetID), isGroup, uint(beforeID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, messages)
}

func GetUnreadCounts(c *gin.Context) {
	userID := c.GetUint("user_id")
	unread, err := service.CountUnreadMessages(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, unread)
}

func GetUnreadSummary(c *gin.Context) {
	userID := c.GetUint("user_id")
	unread, err := service.CountUnreadSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, unread)
}

func MarkMessagesRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		TargetID uint `json:"target_id"`
		IsGroup  bool `json:"is_group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := service.MarkConversationAsRead(userID, req.TargetID, req.IsGroup); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "marked as read"})
}

func RevokeMessage(c *gin.Context, hub *websocket.Hub) {
	userID := c.GetUint("user_id")
	messageIDStr := c.Query("message_id")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message_id"})
		return
	}

	msg, err := service.RecoverMessage(uint(messageID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hub.BroadcastRevoke(*msg)

	c.JSON(http.StatusOK, gin.H{"msg": "message revoked"})
}
