package controller

import (
	"chat-app/internal/application/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func CreateGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	group, err := service.CreateGroup(req.Name, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

func GetMyGroups(c *gin.Context) {
	userID := c.GetUint("user_id")
	groups, err := service.GetMyGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func JoinGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		GroupID uint `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	group, err := service.JoinGroup(req.GroupID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

func LeaveGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		GroupID uint `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	if err := service.LeaveGroup(req.GroupID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "left group"})
}

func GetGroupMembers(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	members, err := service.GetGroupMemberInfos(uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}
