package dbbuild

import (
	"context"
	"os"
	"testing"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
)

func TestLocalMySQLMigrate(t *testing.T) {
	pass := os.Getenv("MYSQL_PASSWORD")
	if pass == "" {
		pass = "422624"
	}
	b := NewBuilder(&config.Postgres{
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: pass,
		DBName:   "papercompare",
	})
	db, err := b.MySQL(context.Background())
	if err != nil {
		t.Fatalf("failed to connect local mysql: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("failed to AutoMigrate tables on local mysql: %v", err)
	}
	t.Log("Successfully connected to local MySQL (root@127.0.0.1:3306/papercompare) and migrated all 26 tables!")
}

