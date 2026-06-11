package service

import (
	"errors"
	"fitnessapi/internal/model"
	"fitnessapi/internal/repository"
	"testing"
	"time"

	"gorm.io/gorm"
)

type mockUserRepo struct {
	users map[uint]model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: map[uint]model.User{
		1: {ID: 1, Name: "演示用户", FriendKey: "friends-demo", CreatedAt: time.Now()},
		2: {ID: 2, Name: "好友小明", FriendKey: "friends-xiaoming", CreatedAt: time.Now()},
		3: {ID: 3, Name: "好友小红", FriendKey: "friends-xiaohong", CreatedAt: time.Now()},
	}}
}

func (m *mockUserRepo) FindByID(id uint) (model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return u, nil
}
func (m *mockUserRepo) FindByFriendKey(friendKey string) (model.User, error) {
	for _, u := range m.users {
		if u.FriendKey == friendKey {
			return u, nil
		}
	}
	return model.User{}, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) ListByIDs(ids []uint) ([]model.User, error) {
	var out []model.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

var _ repository.UserRepo = (*mockUserRepo)(nil)

type mockFriendshipRepo struct {
	relations map[uint]map[uint]bool
}

func newMockFriendshipRepo() *mockFriendshipRepo {
	return &mockFriendshipRepo{relations: map[uint]map[uint]bool{}}
}

func (m *mockFriendshipRepo) Create(fs *model.Friendship) error {
	if _, ok := m.relations[fs.UserID]; !ok {
		m.relations[fs.UserID] = map[uint]bool{}
	}
	m.relations[fs.UserID][fs.FriendID] = true
	return nil
}
func (m *mockFriendshipRepo) ListFriendIDs(userID uint) ([]uint, error) {
	var ids []uint
	for fid := range m.relations[userID] {
		ids = append(ids, fid)
	}
	return ids, nil
}
func (m *mockFriendshipRepo) Exists(userID, friendID uint) (bool, error) {
	if friends, ok := m.relations[userID]; ok {
		return friends[friendID], nil
	}
	return false, nil
}
func (m *mockFriendshipRepo) ListFriends(userID uint) ([]model.FriendInfo, error) {
	var out []model.FriendInfo
	return out, nil
}

var _ repository.FriendshipRepo = (*mockFriendshipRepo)(nil)

func buildUserService() (UserService, *mockUserRepo, *mockFriendshipRepo) {
	ur := newMockUserRepo()
	fr := newMockFriendshipRepo()
	return NewUserService(ur, fr), ur, fr
}

func TestAddFriendByKey_Success(t *testing.T) {
	svc, _, fr := buildUserService()
	friend, err := svc.AddFriendByKey(1, "friends-xiaoming")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if friend.ID != 2 || friend.Name != "好友小明" {
		t.Fatalf("wrong friend returned: %+v", friend)
	}
	if !fr.relations[1][2] {
		t.Fatalf("forward relation missing")
	}
	if !fr.relations[2][1] {
		t.Fatalf("reverse relation missing")
	}
}

func TestAddFriendByKey_NotFound(t *testing.T) {
	svc, _, _ := buildUserService()
	_, err := svc.AddFriendByKey(1, "friends-not-exist")
	if err == nil {
		t.Fatalf("expected error for missing friend key")
	}
	if !errors.Is(err, nil) && err.Error() != "好友码不存在" {
		t.Fatalf("wrong error msg: %v", err)
	}
}

func TestAddFriendByKey_Self(t *testing.T) {
	svc, _, _ := buildUserService()
	_, err := svc.AddFriendByKey(1, "friends-demo")
	if err == nil || err.Error() != "不能添加自己为好友" {
		t.Fatalf("expected self error, got: %v", err)
	}
}

func TestAddFriendByKey_AlreadyFriends(t *testing.T) {
	svc, _, _ := buildUserService()
	if _, err := svc.AddFriendByKey(1, "friends-xiaoming"); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	_, err := svc.AddFriendByKey(1, "friends-xiaoming")
	if err == nil || err.Error() != "已经是好友了" {
		t.Fatalf("expected already friends error, got: %v", err)
	}
}

func TestListFriends(t *testing.T) {
	svc, _, fr := buildUserService()
	fr.relations[1] = map[uint]bool{2: true, 3: true}
	ids, err := svc.GetFriendIDs(1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("want 2 friends got %d", len(ids))
	}
}

func TestGetUser(t *testing.T) {
	svc, _, _ := buildUserService()
	u, err := svc.GetUser(1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.FriendKey != "friends-demo" {
		t.Fatalf("wrong user returned: %+v", u)
	}
}
