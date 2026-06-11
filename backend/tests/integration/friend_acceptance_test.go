//go:build integration

package integration

import (
	"context"
	"fitnessapi/internal/config"
	"fitnessapi/internal/model"
	"fitnessapi/internal/repository"
	"fitnessapi/internal/service"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func startPostgres(t *testing.T) (testcontainers.Container, *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	pg, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("fitness"),
		tcpg.WithUsername("fitness"),
		tcpg.WithPassword("fitness_pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("cannot start postgres container (docker may not be available): %v", err)
	}
	host, err := pg.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := pg.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}
	dsn := fmt.Sprintf("host=%s port=%s user=fitness password=fitness_pass dbname=fitness sslmode=disable TimeZone=Asia/Shanghai", host, port.Port())
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	return pg, db
}

func createLegacyUsersTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 模拟旧版 schema：只有 users 表，friend_key 无唯一约束，也没有 friendships 表
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(80) NOT NULL,
			friend_key VARCHAR(80),
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS workout_records (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			sport_type VARCHAR(40) NOT NULL,
			duration_min INTEGER NOT NULL DEFAULT 0,
			distance_km DOUBLE PRECISION NOT NULL DEFAULT 0,
			calories DOUBLE PRECISION NOT NULL DEFAULT 0,
			heart_rate INTEGER NOT NULL DEFAULT 0,
			weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
			occurred_at TIMESTAMPTZ DEFAULT now(),
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		)`,
		`INSERT INTO users (name, friend_key) VALUES
			('Alice', 'alice-key'),
			('Bob', 'bob-key'),
			('Carol', 'carol-key')
		ON CONFLICT DO NOTHING`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("legacy schema init failed: %v (sql=%s)", err, s)
		}
	}
}

func TestFriendAcceptanceWithRealPostgres(t *testing.T) {
	pg, db := startPostgres(t)
	defer func() { _ = pg.Terminate(context.Background()) }()

	// Step 1: 建立旧版 schema（无 friendships、无 friend_key 唯一约束）
	t.Log("==> Step 1: create legacy users/workout_records tables")
	createLegacyUsersTable(t, db)

	// 验证旧库状态
	var legacyUserCount int64
	db.Table("users").Count(&legacyUserCount)
	if legacyUserCount != 3 {
		t.Fatalf("legacy users count want 3 got %d", legacyUserCount)
	}

	// Step 2: 执行生产环境真实迁移链路 AutoMigrate + EnsureSchema
	t.Log("==> Step 2: run AutoMigrate + EnsureSchema")
	if err := db.AutoMigrate(&model.User{}, &model.WorkoutRecord{}, &model.Goal{}, &model.Friendship{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := config.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// 验证迁移结果：friendships 表存在、friend_key 唯一、种子数据已入
	var friendshipCount int64
	db.Table("friendships").Count(&friendshipCount)
	t.Logf("friendships table exists, current rows: %d", friendshipCount)

	var users []model.User
	db.Order("id").Find(&users)
	t.Logf("users after migration: %d", len(users))
	foundDemo := false
	for _, u := range users {
		if u.FriendKey == "friends-demo" {
			foundDemo = true
		}
	}
	if !foundDemo {
		t.Fatalf("seed user '演示用户' (friends-demo) missing after EnsureSchema")
	}

	// Step 3: 通过服务层「好友码添加好友」
	t.Log("==> Step 3: add friends via AddFriendByKey service")
	userRepo := repository.NewUserRepository(db)
	fsRepo := repository.NewFriendshipRepository(db)
	recordRepo := repository.NewRecordRepository(db)
	userSvc := service.NewUserService(userRepo, fsRepo)
	recordSvc := service.NewRecordService(recordRepo)

	// 取 Alice 的 ID
	var alice model.User
	if err := db.Where("friend_key = ?", "alice-key").First(&alice).Error; err != nil {
		t.Fatalf("find alice: %v", err)
	}
	var bob model.User
	db.Where("friend_key = ?", "bob-key").First(&bob)
	var carol model.User
	db.Where("friend_key = ?", "carol-key").First(&carol)

	// Alice 加 Bob 和 Carol
	bf1, err := userSvc.AddFriendByKey(alice.ID, "bob-key")
	if err != nil {
		t.Fatalf("Alice add Bob failed: %v", err)
	}
	if bf1.ID != bob.ID || bf1.Name != "Bob" {
		t.Fatalf("Alice add Bob got wrong friend: %+v", bf1)
	}
	bf2, err := userSvc.AddFriendByKey(alice.ID, "carol-key")
	if err != nil {
		t.Fatalf("Alice add Carol failed: %v", err)
	}
	if bf2.Name != "Carol" {
		t.Fatalf("Alice add Carol wrong name: %+v", bf2)
	}

	// 重复添加应报错
	if _, err := userSvc.AddFriendByKey(alice.ID, "bob-key"); err == nil {
		t.Fatalf("duplicate add should return error")
	}

	// 加自己应报错
	if _, err := userSvc.AddFriendByKey(alice.ID, "alice-key"); err == nil {
		t.Fatalf("add self should return error")
	}

	// 好友列表
	friends, err := userSvc.ListFriends(alice.ID)
	if err != nil {
		t.Fatalf("ListFriends: %v", err)
	}
	if len(friends) != 2 {
		t.Fatalf("Alice should have 2 friends, got %d", len(friends))
	}
	// 反向关系：Bob 的好友列表里应有 Alice
	bobFriends, _ := userSvc.ListFriends(bob.ID)
	if len(bobFriends) < 1 {
		t.Fatalf("reverse friendship missing, Bob has no friends")
	}

	// Step 4: 写入运动记录
	t.Log("==> Step 4: insert workout records for Alice/Bob/Carol")
	now := time.Now()
	mustCreateRecord(t, recordSvc, &model.WorkoutRecord{
		UserID: alice.ID, SportType: "run", DurationMin: 40, DistanceKm: 6, Calories: 400, OccurredAt: now,
	})
	mustCreateRecord(t, recordSvc, &model.WorkoutRecord{
		UserID: alice.ID, SportType: "ride", DurationMin: 60, DistanceKm: 20, Calories: 500, OccurredAt: now,
	})
	mustCreateRecord(t, recordSvc, &model.WorkoutRecord{
		UserID: bob.ID, SportType: "run", DurationMin: 90, DistanceKm: 15, Calories: 900, OccurredAt: now,
	})
	mustCreateRecord(t, recordSvc, &model.WorkoutRecord{
		UserID: carol.ID, SportType: "swim", DurationMin: 45, DistanceKm: 1.5, Calories: 450, OccurredAt: now,
	})

	// Step 5: 构建好友圈排行榜（Alice 视角）
	t.Log("==> Step 5: build friend-circle ranking via real repository data")
	friendIDs, err := userSvc.GetFriendIDs(alice.ID)
	if err != nil {
		t.Fatalf("GetFriendIDs: %v", err)
	}
	allIDs := append([]uint{alice.ID}, friendIDs...)
	records, err := recordSvc.ListByUserIDs(allIDs, nil, nil)
	if err != nil {
		t.Fatalf("ListByUserIDs: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("want 4 records got %d", len(records))
	}
	userNames := map[uint]string{alice.ID: alice.Name, bob.ID: bob.Name, carol.ID: carol.Name}
	for _, f := range friends {
		userNames[f.ID] = f.Name
	}
	ranking := service.BuildRanking(records, userNames)
	t.Logf("ranking result: %+v", ranking)

	// 排名期望：Bob(90) > Alice(100) ... 等等, Alice=40+60=100!
	// 修正期望 Alice=100, Bob=90, Carol=45 → Alice 第一
	if len(ranking) != 3 {
		t.Fatalf("ranking size want 3 got %d", len(ranking))
	}
	if ranking[0].Name != "Alice" || ranking[0].Duration != 100 {
		t.Fatalf("1st place should be Alice 100min, got %+v", ranking[0])
	}
	if ranking[1].Name != "Bob" || ranking[1].Duration != 90 {
		t.Fatalf("2nd place should be Bob 90min, got %+v", ranking[1])
	}
	if ranking[2].Name != "Carol" || ranking[2].Duration != 45 {
		t.Fatalf("3rd place should be Carol 45min, got %+v", ranking[2])
	}

	// 汇总
	summary := service.BuildSummary(records)
	t.Logf("summary: %+v", summary)
	if summary.TotalDuration != 235 || summary.TotalCalories != 2250 || summary.Count != 4 {
		t.Fatalf("summary wrong: total_duration=%d total_calories=%f count=%d (want 235/2250/4)",
			summary.TotalDuration, summary.TotalCalories, summary.Count)
	}

	// 仅个人排行榜（非好友圈）
	t.Log("==> Step 6: personal ranking scope")
	personalRecords, _ := recordSvc.List(alice.ID, nil, nil)
	personalRanking := service.BuildRanking(personalRecords, userNames)
	if len(personalRanking) != 1 || personalRanking[0].Duration != 100 {
		t.Fatalf("personal ranking wrong: %+v", personalRanking)
	}
}

func mustCreateRecord(t *testing.T, svc service.RecordService, r *model.WorkoutRecord) {
	t.Helper()
	if r.OccurredAt.IsZero() {
		r.OccurredAt = time.Now()
	}
	if err := svc.Create(r); err != nil {
		t.Fatalf("create record: %v", err)
	}
}
