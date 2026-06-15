package service

import (
	"chat-app/internal/domain/model"
	"chat-app/internal/infrastructure/config"
	"errors"

	"gorm.io/gorm"
)

type GroupListItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	CreatedBy   uint   `json:"created_by"`
	MemberCount int64  `json:"member_count"`
}

type GroupMemberInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

func CreateGroup(name string, createdBy uint) (*model.Group, error) {
	if name == "" {
		return nil, errors.New("group name is required")
	}

	var group model.Group
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		group = model.Group{
			Name:      name,
			CreatedBy: createdBy,
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		return tx.Create(&model.GroupMember{
			GroupID: group.ID,
			UserID:  createdBy,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func GetMyGroups(userID uint) ([]GroupListItem, error) {
	var memberships []model.GroupMember
	if err := config.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []GroupListItem{}, nil
	}

	groupIDs := make([]uint, 0, len(memberships))
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.GroupID)
	}

	var groups []model.Group
	if err := config.DB.Where("id IN ?", groupIDs).Order("id DESC").Find(&groups).Error; err != nil {
		return nil, err
	}

	var counts []struct {
		GroupID uint
		Count   int64
	}
	if err := config.DB.Model(&model.GroupMember{}).
		Select("group_id, COUNT(*) as count").
		Where("group_id IN ?", groupIDs).
		Group("group_id").
		Scan(&counts).Error; err != nil {
		return nil, err
	}

	countByGroupID := make(map[uint]int64, len(counts))
	for _, count := range counts {
		countByGroupID[count.GroupID] = count.Count
	}

	items := make([]GroupListItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, GroupListItem{
			ID:          group.ID,
			Name:        group.Name,
			CreatedBy:   group.CreatedBy,
			MemberCount: countByGroupID[group.ID],
		})
	}
	return items, nil
}

func JoinGroup(groupID, userID uint) (*model.Group, error) {
	var group model.Group
	if err := config.DB.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("group not found")
		}
		return nil, err
	}

	member := model.GroupMember{
		GroupID: groupID,
		UserID:  userID,
	}
	err := config.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		FirstOrCreate(&member).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func LeaveGroup(groupID, userID uint) error {
	result := config.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.GroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("group membership not found")
	}
	return nil
}

func GetGroupMembers(groupID uint) []uint {
	var members []model.GroupMember
	config.DB.Where("group_id = ?", groupID).Find(&members)

	userIDs := make([]uint, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	return userIDs
}

func GetGroupMemberInfos(groupID uint) ([]GroupMemberInfo, error) {
	var users []model.User
	err := config.DB.Model(&model.User{}).
		Select("users.id, users.username").
		Joins("JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Order("users.id ASC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	online := GetOnlineUsers()
	members := make([]GroupMemberInfo, 0, len(users))
	for _, user := range users {
		members = append(members, GroupMemberInfo{
			ID:       user.ID,
			Username: user.Username,
			Online:   online[user.ID] != "",
		})
	}
	return members, nil
}
