package service

import (
	"errors"
	"fitnessapi/internal/model"
	"fitnessapi/internal/repository"

	"gorm.io/gorm"
)

type UserService struct {
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
}

func NewUserService(userRepo repository.UserRepository, friendshipRepo repository.FriendshipRepository) UserService {
	return UserService{userRepo: userRepo, friendshipRepo: friendshipRepo}
}

func (s UserService) GetUser(id uint) (model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s UserService) AddFriendByKey(userID uint, friendKey string) (model.FriendInfo, error) {
	friend, err := s.userRepo.FindByFriendKey(friendKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.FriendInfo{}, errors.New("好友码不存在")
		}
		return model.FriendInfo{}, err
	}
	if friend.ID == userID {
		return model.FriendInfo{}, errors.New("不能添加自己为好友")
	}
	exists, err := s.friendshipRepo.Exists(userID, friend.ID)
	if err != nil {
		return model.FriendInfo{}, err
	}
	if exists {
		return model.FriendInfo{}, errors.New("已经是好友了")
	}
	fs := &model.Friendship{UserID: userID, FriendID: friend.ID}
	if err := s.friendshipRepo.Create(fs); err != nil {
		return model.FriendInfo{}, err
	}
	reverse := &model.Friendship{UserID: friend.ID, FriendID: userID}
	_ = s.friendshipRepo.Create(reverse)
	return model.FriendInfo{
		ID:        friend.ID,
		Name:      friend.Name,
		FriendKey: friend.FriendKey,
		CreatedAt: friend.CreatedAt,
	}, nil
}

func (s UserService) ListFriends(userID uint) ([]model.FriendInfo, error) {
	return s.friendshipRepo.ListFriends(userID)
}

func (s UserService) GetFriendIDs(userID uint) ([]uint, error) {
	return s.friendshipRepo.ListFriendIDs(userID)
}
