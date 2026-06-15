package service

import (
	"chat-app/internal/domain/model"
	"chat-app/internal/infrastructure/config"
	"context"
	"errors"
	"fmt"
	"strconv"
)


func SaveMessage(sendID, receiverID uint, content string, isGroup bool) (*model.Message, error) {
	msg := &model.Message{
		SenderID:   sendID,
		ReceiverID: receiverID,
		Content:    content,
		IsGroup:    isGroup,
	}
	if err := config.DB.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

func GetMessageHistory(userID, targetID uint, isGroup bool, beforeID uint, limit int) ([]model.Message, error) {
	var msgs []model.Message
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	db := config.DB.Model(&model.Message{})
	if isGroup {
		db = db.Where("receiver_id = ? AND is_group = ?", targetID, true)
	} else {
		db = db.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID, targetID, targetID, userID)
	}
	if beforeID > 0 {
		db = db.Where("id < ?", beforeID)
	}

	err := db.Order("id DESC").Limit(limit).Find(&msgs).Error
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func MarkMessagesAsRead(senderID, receiverID uint) error {
	if err := config.DB.Model(&model.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND is_read = ?", senderID, receiverID, false).
		Update("is_read", true).Error; err != nil {
		return err
	}

	ClearUnreadMessages(receiverID, senderID)
	return nil
}

func MarkConversationAsRead(userID, targetID uint, isGroup bool) error {
	if isGroup {
		ClearUnreadGroupMessages(userID, targetID)
		return nil
	}
	return MarkMessagesAsRead(targetID, userID)
}

func CountUnreadMessages(userID uint) (map[uint]int64, error) {
	if config.Redis != nil {
		values, err := config.Redis.HGetAll(context.Background(), unreadRedisKey(userID)).Result()
		if err == nil && len(values) > 0 {
			unread := make(map[uint]int64, len(values))
			for senderIDStr, countStr := range values {
				senderID, err := strconv.ParseUint(senderIDStr, 10, 64)
				if err != nil {
					continue
				}
				count, err := strconv.ParseInt(countStr, 10, 64)
				if err != nil {
					continue
				}
				unread[uint(senderID)] = count
			}
			return unread, nil
		}
	}

	var results []struct {
		SenderID uint  `json:"sender_id"`
		Count    int64 `json:"count"`
	}
	if err := config.DB.Model(&model.Message{}).
		Select("sender_id, COUNT(*) as count").
		Where("receiver_id = ? AND is_read = ?", userID, false).
		Group("sender_id").
		Find(&results).Error; err != nil {
		return nil, err
	}

	unread := make(map[uint]int64)
	for _, res := range results {
		unread[res.SenderID] = res.Count
	}
	return unread, nil
}

type UnreadSummary struct {
	Users  map[uint]int64 `json:"users"`
	Groups map[uint]int64 `json:"groups"`
}

func CountUnreadSummary(userID uint) (*UnreadSummary, error) {
	users, err := CountUnreadMessages(userID)
	if err != nil {
		return nil, err
	}

	groups := make(map[uint]int64)
	if config.Redis != nil {
		values, err := config.Redis.HGetAll(context.Background(), unreadGroupRedisKey(userID)).Result()
		if err == nil {
			for groupIDStr, countStr := range values {
				groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
				if err != nil {
					continue
				}
				count, err := strconv.ParseInt(countStr, 10, 64)
				if err != nil {
					continue
				}
				groups[uint(groupID)] = count
			}
		}
	}

	return &UnreadSummary{
		Users:  users,
		Groups: groups,
	}, nil
}

func IncrementUnreadMessage(receiverID, senderID uint) {
	if config.Redis == nil {
		return
	}
	config.Redis.HIncrBy(
		context.Background(),
		unreadRedisKey(receiverID),
		strconv.FormatUint(uint64(senderID), 10),
		1,
	)
}

func IncrementGroupUnreadMessage(userID, groupID uint) {
	if config.Redis == nil {
		return
	}
	config.Redis.HIncrBy(
		context.Background(),
		unreadGroupRedisKey(userID),
		strconv.FormatUint(uint64(groupID), 10),
		1,
	)
}

func ClearUnreadMessages(userID, senderID uint) {
	if config.Redis == nil {
		return
	}
	config.Redis.HDel(
		context.Background(),
		unreadRedisKey(userID),
		strconv.FormatUint(uint64(senderID), 10),
	)
}

func ClearUnreadGroupMessages(userID, groupID uint) {
	if config.Redis == nil {
		return
	}
	config.Redis.HDel(
		context.Background(),
		unreadGroupRedisKey(userID),
		strconv.FormatUint(uint64(groupID), 10),
	)
}

func unreadRedisKey(userID uint) string {
	return fmt.Sprintf("unread:%d", userID)
}

func unreadGroupRedisKey(userID uint) string {
	return fmt.Sprintf("unread_groups:%d", userID)
}

func GetUnreadMessages(userID uint) ([]model.Message, error) {
	var msgs []model.Message
	if err := config.DB.Where("receiver_id = ? AND is_read = ?", userID, false).
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func RecoverMessage(messageID, userID uint) (*model.Message, error) {
	var msg model.Message
	if err := config.DB.First(&msg, messageID).Error; err != nil {
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, errors.New("only sender can revoke message")
	}
	if err := config.DB.Delete(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}
