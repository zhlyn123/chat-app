package model

import "time"

// Message stores chat messages.
type Message struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   uint      `gorm:"not null;index:idx_single_chat,priority:1;index:idx_unread_sender,priority:3" json:"sender_id"`
	ReceiverID uint      `gorm:"not null;index:idx_single_chat,priority:2;index:idx_group_chat,priority:1;index:idx_unread_sender,priority:1" json:"receiver_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	IsGroup    bool      `gorm:"default:false;index:idx_group_chat,priority:2" json:"is_group"`
	IsRead     bool      `gorm:"default:false;index:idx_unread_sender,priority:2" json:"is_read"`
	CreatedAt  time.Time `gorm:"index:idx_single_chat,priority:3;index:idx_group_chat,priority:3" json:"created_at"`
}

type Group struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CreatedBy uint      `gorm:"not null;index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupMember struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	GroupID uint `gorm:"not null;uniqueIndex:idx_group_member" json:"group_id"`
	UserID  uint `gorm:"not null;uniqueIndex:idx_group_member;index" json:"user_id"`
}
