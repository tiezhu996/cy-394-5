package service

import (
	"fitnessapi/internal/constants"
	"fitnessapi/internal/model"
	"fitnessapi/internal/repository"
	"time"
)

type RecordServiceAPI interface {
	Create(record *model.WorkoutRecord) error
	List(userID uint, start, end *time.Time) ([]model.WorkoutRecord, error)
	ListByUserIDs(userIDs []uint, start, end *time.Time) ([]model.WorkoutRecord, error)
	Update(record *model.WorkoutRecord) error
	Delete(id uint) error
	Find(id uint) (model.WorkoutRecord, error)
}

type RecordService struct{ repo repository.RecordRepository }

func NewRecordService(repo repository.RecordRepository) RecordService {
	return RecordService{repo: repo}
}

func (s RecordService) Create(record *model.WorkoutRecord) error {
	if record.Calories <= 0 {
		record.Calories = float64(record.DurationMin) / 60 * constants.METValues[constants.SportType(record.SportType)] * 70
	}
	return s.repo.Create(record)
}
func (s RecordService) List(userID uint, start, end *time.Time) ([]model.WorkoutRecord, error) {
	return s.repo.List(userID, start, end)
}
func (s RecordService) ListByUserIDs(userIDs []uint, start, end *time.Time) ([]model.WorkoutRecord, error) {
	return s.repo.ListByUserIDs(userIDs, start, end)
}
func (s RecordService) Update(record *model.WorkoutRecord) error  { return s.repo.Update(record) }
func (s RecordService) Delete(id uint) error                      { return s.repo.Delete(id) }
func (s RecordService) Find(id uint) (model.WorkoutRecord, error) { return s.repo.Find(id) }

type UserRankItem struct {
	UserID   uint    `json:"user_id"`
	Name     string  `json:"name,omitempty"`
	Duration int     `json:"duration"`
	Distance float64 `json:"distance"`
	Calories float64 `json:"calories"`
}

func BuildRanking(records []model.WorkoutRecord, userNames map[uint]string) []UserRankItem {
	agg := map[uint]*UserRankItem{}
	for _, r := range records {
		item, ok := agg[r.UserID]
		if !ok {
			item = &UserRankItem{UserID: r.UserID}
			agg[r.UserID] = item
		}
		item.Duration += r.DurationMin
		item.Distance += r.DistanceKm
		item.Calories += r.Calories
	}
	result := make([]UserRankItem, 0, len(agg))
	for _, item := range agg {
		if name, ok := userNames[item.UserID]; ok {
			item.Name = name
		}
		result = append(result, *item)
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Duration > result[i].Duration {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}
