package handler

import (
	"fitnessapi/internal/config"
	"fitnessapi/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Login(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "1", "exp": time.Now().Add(24 * time.Hour).Unix()})
		signed, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": signed})
	}
}

type UserHandler struct{ userSvc service.UserServiceAPI }

func NewUserHandler(userSvc service.UserServiceAPI) UserHandler {
	return UserHandler{userSvc: userSvc}
}

type AddFriendRequest struct {
	FriendKey string `json:"friend_key" binding:"required"`
}

func (h UserHandler) AddFriend(c *gin.Context) {
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "1"))
	var req AddFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "缺少好友码"})
		return
	}
	friend, err := h.userSvc.AddFriendByKey(uint(userID), req.FriendKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friend": friend})
}

func (h UserHandler) ListFriends(c *gin.Context) {
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "1"))
	friends, err := h.userSvc.ListFriends(uint(userID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": friends, "count": len(friends)})
}

func (h UserHandler) Profile(c *gin.Context) {
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "1"))
	user, err := h.userSvc.GetUser(uint(userID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}
