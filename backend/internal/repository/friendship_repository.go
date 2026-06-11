package repository

import (
	"fitnessapi/internal/model"

	"gorm.io/gorm"
)

type FriendshipRepository struct{ db *gorm.DB }

func NewFriendshipRepository(db *gorm.DB) FriendshipRepository {
	return FriendshipRepository{db: db}
}

func (r FriendshipRepository) Create(friendship *model.Friendship) error {
	return r.db.Create(friendship).Error
}

func (r FriendshipRepository) ListFriendIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Friendship{}).
		Where("user_id = ?", userID).
		Pluck("friend_id", &ids).Error
	return ids, err
}

func (r FriendshipRepository) Exists(userID, friendID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Count(&count).Error
	return count > 0, err
}

func (r FriendshipRepository) ListFriends(userID uint) ([]model.FriendInfo, error) {
	var friends []model.FriendInfo
	err := r.db.Table("users").
		Select("users.id, users.name, users.friend_key, users.created_at").
		Joins("JOIN friendships ON friendships.friend_id = users.id").
		Where("friendships.user_id = ?", userID).
		Order("users.name ASC").
		Scan(&friends).Error
	return friends, err
}
