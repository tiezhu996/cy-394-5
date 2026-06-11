package config

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func MustOpenDB(cfg Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	return db
}

func EnsureSchema(db *gorm.DB) error {
	stmts := []string{
		`ALTER TABLE users ALTER COLUMN friend_key TYPE VARCHAR(80)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_friend_key ON users(friend_key)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_user_id ON friendships(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_friend_id ON friendships(friend_id)`,
		`ALTER TABLE friendships DROP CONSTRAINT IF EXISTS friendships_user_id_friend_id_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_user_friend ON friendships(user_id, friend_id)`,
		`INSERT INTO users (name, friend_key) VALUES ('演示用户', 'friends-demo') ON CONFLICT DO NOTHING`,
		`INSERT INTO users (name, friend_key) VALUES ('好友小明', 'friends-xiaoming') ON CONFLICT DO NOTHING`,
		`INSERT INTO users (name, friend_key) VALUES ('好友小红', 'friends-xiaohong') ON CONFLICT DO NOTHING`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			log.Printf("schema ensure stmt warn: %v (sql=%s)", err, s)
		}
	}
	return nil
}
