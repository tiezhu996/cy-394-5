package handler

import (
	"fitnessapi/internal/model"
	"fitnessapi/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RankHandler struct {
	recordSvc service.RecordService
	userSvc   service.UserService
}

func NewRankHandler(recordSvc service.RecordService, userSvc service.UserService) RankHandler {
	return RankHandler{recordSvc: recordSvc, userSvc: userSvc}
}

func parseScope(scope string) (start, end *time.Time) {
	now := time.Now()
	switch scope {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday - 1))
		s := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
		e := s.AddDate(0, 0, 7)
		return &s, &e
	case "month":
		s := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		e := s.AddDate(0, 1, 0)
		return &s, &e
	default:
		return nil, nil
	}
}

func (h RankHandler) Rankings(c *gin.Context) {
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "1"))
	scope := c.DefaultQuery("scope", "week")
	friendCircle := c.Query("friend_circle")
	start, end := parseScope(scope)

	var records []model.WorkoutRecord
	var err error
	var userIDs []uint
	var userNames map[uint]string

	if friendCircle == "1" || friendCircle == "true" {
		friendIDs, ferr := h.userSvc.GetFriendIDs(uint(userID))
		if ferr != nil {
			c.Error(ferr)
			return
		}
		userIDs = append([]uint{uint(userID)}, friendIDs...)
		records, err = h.recordSvc.ListByUserIDs(userIDs, start, end)
		users, uerr := h.userSvc.ListFriends(uint(userID))
		if uerr != nil {
			c.Error(uerr)
			return
		}
		userNames = map[uint]string{}
		for _, f := range users {
			userNames[f.ID] = f.Name
		}
		self, serr := h.userSvc.GetUser(uint(userID))
		if serr == nil {
			userNames[self.ID] = self.Name
		}
	} else {
		userIDs = []uint{uint(userID)}
		records, err = h.recordSvc.List(uint(userID), start, end)
		self, serr := h.userSvc.GetUser(uint(userID))
		userNames = map[uint]string{}
		if serr == nil {
			userNames[self.ID] = self.Name
		}
	}

	if err != nil {
		c.Error(err)
		return
	}

	ranking := service.BuildRanking(records, userNames)
	summary := service.BuildSummary(records)

	c.JSON(http.StatusOK, gin.H{
		"scope":         scope,
		"friend_circle": friendCircle,
		"ranking":       ranking,
		"summary":       summary,
		"user_count":    len(userIDs),
	})
}
