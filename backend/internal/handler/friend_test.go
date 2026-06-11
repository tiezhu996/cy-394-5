package handler

import (
	"bytes"
	"encoding/json"
	"fitnessapi/internal/model"
	"fitnessapi/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockUserSvc struct {
	getUserFn    func(id uint) (model.User, error)
	addFriendFn  func(userID uint, friendKey string) (model.FriendInfo, error)
	listFriendsFn func(userID uint) ([]model.FriendInfo, error)
	getFriendIDsFn func(userID uint) ([]uint, error)
}

func (m *mockUserSvc) GetUser(id uint) (model.User, error) {
	return m.getUserFn(id)
}
func (m *mockUserSvc) AddFriendByKey(userID uint, key string) (model.FriendInfo, error) {
	return m.addFriendFn(userID, key)
}
func (m *mockUserSvc) ListFriends(userID uint) ([]model.FriendInfo, error) {
	return m.listFriendsFn(userID)
}
func (m *mockUserSvc) GetFriendIDs(userID uint) ([]uint, error) {
	return m.getFriendIDsFn(userID)
}

type mockRecordSvc struct {
	listFn        func(userID uint, start, end *time.Time) ([]model.WorkoutRecord, error)
	listByIDsFn   func(userIDs []uint, start, end *time.Time) ([]model.WorkoutRecord, error)
}

func (m *mockRecordSvc) Create(r *model.WorkoutRecord) error { return nil }
func (m *mockRecordSvc) List(userID uint, s, e *time.Time) ([]model.WorkoutRecord, error) {
	return m.listFn(userID, s, e)
}
func (m *mockRecordSvc) ListByUserIDs(ids []uint, s, e *time.Time) ([]model.WorkoutRecord, error) {
	return m.listByIDsFn(ids, s, e)
}
func (m *mockRecordSvc) Update(r *model.WorkoutRecord) error  { return nil }
func (m *mockRecordSvc) Delete(id uint) error                  { return nil }
func (m *mockRecordSvc) Find(id uint) (model.WorkoutRecord, error) { return model.WorkoutRecord{}, nil }

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAddFriendHandler_OK(t *testing.T) {
	app := setupGin()
	usvc := &mockUserSvc{
		addFriendFn: func(userID uint, key string) (model.FriendInfo, error) {
			if userID != 1 || key != "friends-xiaoming" {
				t.Fatalf("unexpected args: userID=%d key=%s", userID, key)
			}
			return model.FriendInfo{ID: 2, Name: "好友小明", FriendKey: "friends-xiaoming"}, nil
		},
	}
	h := NewUserHandler(usvc)
	app.POST("/friends/add", h.AddFriend)

	body := `{"friend_key":"friends-xiaoming"}`
	req := httptest.NewRequest("POST", "/friends/add?user_id=1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	f, _ := resp["friend"].(map[string]interface{})
	if f["name"] != "好友小明" {
		t.Fatalf("wrong friend name: %+v", f)
	}
}

func TestAddFriendHandler_MissingKey(t *testing.T) {
	app := setupGin()
	h := NewUserHandler(&mockUserSvc{
		addFriendFn: func(uint, string) (model.FriendInfo, error) { t.Fatal("should not be called"); return model.FriendInfo{}, nil },
	})
	app.POST("/friends/add", h.AddFriend)

	req := httptest.NewRequest("POST", "/friends/add", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status want 400 got %d", w.Code)
	}
}

func TestListFriendsHandler_OK(t *testing.T) {
	app := setupGin()
	now := time.Now()
	usvc := &mockUserSvc{
		listFriendsFn: func(userID uint) ([]model.FriendInfo, error) {
			return []model.FriendInfo{
				{ID: 2, Name: "小明", FriendKey: "k1", CreatedAt: now},
				{ID: 3, Name: "小红", FriendKey: "k2", CreatedAt: now},
			}, nil
		},
	}
	h := NewUserHandler(usvc)
	app.GET("/friends", h.ListFriends)

	req := httptest.NewRequest("GET", "/friends?user_id=1", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if int(resp["count"].(float64)) != 2 {
		t.Fatalf("want count 2 got %+v", resp)
	}
}

func TestRankingsHandler_PersonalScope(t *testing.T) {
	app := setupGin()
	usvc := &mockUserSvc{
		getUserFn: func(id uint) (model.User, error) {
			return model.User{ID: 1, Name: "我", FriendKey: "me"}, nil
		},
	}
	rsvc := &mockRecordSvc{
		listFn: func(userID uint, start, end *time.Time) ([]model.WorkoutRecord, error) {
			if userID != 1 {
				t.Fatalf("want userID 1 got %d", userID)
			}
			return []model.WorkoutRecord{
				{UserID: 1, DurationMin: 60, DistanceKm: 10, Calories: 500},
				{UserID: 1, DurationMin: 30, DistanceKm: 5, Calories: 250},
			}, nil
		},
	}
	h := NewRankHandler(rsvc, usvc)
	app.GET("/rankings", h.Rankings)

	req := httptest.NewRequest("GET", "/rankings?user_id=1&scope=week", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["scope"] != "week" {
		t.Fatalf("scope wrong: %+v", resp)
	}
	ranking, _ := resp["ranking"].([]interface{})
	if len(ranking) != 1 {
		t.Fatalf("personal ranking should have 1 entry: %+v", resp)
	}
	summary, _ := resp["summary"].(map[string]interface{})
	if int(summary["total_duration"].(float64)) != 90 {
		t.Fatalf("summary duration wrong: %+v", summary)
	}
}

func TestRankingsHandler_FriendCircleScope(t *testing.T) {
	app := setupGin()
	usvc := &mockUserSvc{
		getUserFn: func(id uint) (model.User, error) {
			if id == 1 {
				return model.User{ID: 1, Name: "我"}, nil
			}
			return model.User{}, nil
		},
		listFriendsFn: func(userID uint) ([]model.FriendInfo, error) {
			return []model.FriendInfo{
				{ID: 2, Name: "小明"},
				{ID: 3, Name: "小红"},
			}, nil
		},
		getFriendIDsFn: func(userID uint) ([]uint, error) {
			return []uint{2, 3}, nil
		},
	}
	rsvc := &mockRecordSvc{
		listByIDsFn: func(ids []uint, start, end *time.Time) ([]model.WorkoutRecord, error) {
			if len(ids) != 3 || ids[0] != 1 {
				t.Fatalf("ids wrong: %+v", ids)
			}
			return []model.WorkoutRecord{
				{UserID: 1, DurationMin: 30, DistanceKm: 5, Calories: 250},
				{UserID: 2, DurationMin: 90, DistanceKm: 15, Calories: 800},
				{UserID: 3, DurationMin: 60, DistanceKm: 10, Calories: 500},
			}, nil
		},
	}
	h := NewRankHandler(rsvc, usvc)
	app.GET("/rankings", h.Rankings)

	req := httptest.NewRequest("GET", "/rankings?user_id=1&scope=month&friend_circle=1", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	ranking, _ := resp["ranking"].([]interface{})
	if len(ranking) != 3 {
		t.Fatalf("friend circle ranking want 3 entries: %+v", resp)
	}
	first, _ := ranking[0].(map[string]interface{})
	if first["name"] != "小明" || int(first["duration"].(float64)) != 90 {
		t.Fatalf("first place wrong: %+v", first)
	}
	summary, _ := resp["summary"].(map[string]interface{})
	if int(summary["count"].(float64)) != 3 {
		t.Fatalf("summary count wrong: %+v", summary)
	}
	if int(resp["user_count"].(float64)) != 3 {
		t.Fatalf("user_count wrong: %+v", resp)
	}
}

var _ = service.UserService{}
