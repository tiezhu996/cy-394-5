package repository

import (
	"fitnessapi/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return UserRepository{db: db} }

func (r UserRepository) FindByID(id uint) (model.User, error) {
	var user model.User
	return user, r.db.First(&user, id).Error
}

func (r UserRepository) FindByFriendKey(friendKey string) (model.User, error) {
	var user model.User
	return user, r.db.Where("friend_key = ?", friendKey).First(&user).Error
}

func (r UserRepository) ListByIDs(ids []uint) ([]model.User, error) {
	var users []model.User
	if len(ids) == 0 {
		return users, nil
	}
	return users, r.db.Where("id IN ?", ids).Find(&users).Error
}
