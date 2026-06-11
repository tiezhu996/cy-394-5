package service

import (
	"fitnessapi/internal/model"
	"testing"
	"time"
)

func TestBuildSummary(t *testing.T) {
	records := []model.WorkoutRecord{
		{UserID: 1, SportType: "run", DurationMin: 30, DistanceKm: 5.0, Calories: 300},
		{UserID: 1, SportType: "ride", DurationMin: 45, DistanceKm: 15.0, Calories: 400},
		{UserID: 1, SportType: "run", DurationMin: 20, DistanceKm: 3.0, Calories: 200},
	}
	s := BuildSummary(records)
	if s.Count != 3 {
		t.Fatalf("count want 3 got %d", s.Count)
	}
	if s.TotalDuration != 95 {
		t.Fatalf("duration want 95 got %d", s.TotalDuration)
	}
	if s.TotalDistance != 23.0 {
		t.Fatalf("distance want 23 got %f", s.TotalDistance)
	}
	if s.TotalCalories != 900 {
		t.Fatalf("calories want 900 got %f", s.TotalCalories)
	}
	if len(s.TypeShares) != 2 {
		t.Fatalf("type_shares want 2 got %d", len(s.TypeShares))
	}
}

func TestBuildSummaryEmpty(t *testing.T) {
	s := BuildSummary(nil)
	if s.Count != 0 || s.TotalDuration != 0 {
		t.Fatalf("empty summary should be zero")
	}
}

func TestBuildRanking(t *testing.T) {
	now := time.Now()
	records := []model.WorkoutRecord{
		{UserID: 1, DurationMin: 30, DistanceKm: 5, Calories: 300, OccurredAt: now},
		{UserID: 2, DurationMin: 60, DistanceKm: 10, Calories: 600, OccurredAt: now},
		{UserID: 1, DurationMin: 20, DistanceKm: 3, Calories: 200, OccurredAt: now},
		{UserID: 3, DurationMin: 40, DistanceKm: 8, Calories: 400, OccurredAt: now},
	}
	names := map[uint]string{1: "A", 2: "B", 3: "C"}
	r := BuildRanking(records, names)
	if len(r) != 3 {
		t.Fatalf("ranking size want 3 got %d", len(r))
	}
	if r[0].UserID != 2 || r[0].Duration != 60 || r[0].Name != "B" {
		t.Fatalf("first place should be user 2 with 60min, got %+v", r[0])
	}
	if r[1].UserID != 1 || r[1].Duration != 50 {
		t.Fatalf("second place should be user 1 with 50min, got %+v", r[1])
	}
	if r[2].UserID != 3 || r[2].Duration != 40 {
		t.Fatalf("third place should be user 3 with 40min, got %+v", r[2])
	}
}

func TestBuildRankingEmpty(t *testing.T) {
	r := BuildRanking(nil, nil)
	if len(r) != 0 {
		t.Fatalf("empty ranking should be zero")
	}
}

func TestPersonalRecords(t *testing.T) {
	records := []model.WorkoutRecord{
		{DistanceKm: 10.5, DurationMin: 60, WeightKg: 75},
		{DistanceKm: 5.0, DurationMin: 25, WeightKg: 70},
	}
	pr := PersonalRecords(records)
	if pr["longest_distance"] != 10.5 {
		t.Fatalf("longest want 10.5 got %f", pr["longest_distance"])
	}
	if pr["max_weight"] != 75 {
		t.Fatalf("max_weight want 75 got %f", pr["max_weight"])
	}
	pace := 25.0 / 5.0
	if pr["fastest_pace"] != pace {
		t.Fatalf("pace want %f got %f", pace, pr["fastest_pace"])
	}
}
