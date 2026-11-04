package dbbuild

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Builder interface {
	Postgres(ctx context.Context) (*gorm.DB, error)
	MySQL(ctx context.Context) (*gorm.DB, error)
}

type microSvcDB struct {
	cfg *config.Postgres
}

func NewBuilder(dbConfig *config.Postgres) Builder {
	return &microSvcDB{
		cfg: dbConfig,
	}
}

func resolveMySQLHost(configuredHost string, port int) string {
	if envHost := os.Getenv("MYSQL_HOST"); envHost != "" {
		return envHost
	}
	if configuredHost == "" || configuredHost == "postgres" {
		configuredHost = "host.minikube.internal"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(configuredHost, strconv.Itoa(port)), 1500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return configuredHost
	}
	return "127.0.0.1"
}

func (m *microSvcDB) MySQL(ctx context.Context) (*gorm.DB, error) {
	port := m.cfg.Port
	if port == 0 || port == 5432 {
		port = 3306
	}
	if envPort := os.Getenv("MYSQL_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
		}
	}

	host := resolveMySQLHost(m.cfg.Host, port)

	user := m.cfg.Username
	if user == "" || user == "postgres" {
		user = "root"
	}
	if envUser := os.Getenv("MYSQL_USER"); envUser != "" {
		user = envUser
	}

	password := m.cfg.Password
	if envPass := os.Getenv("MYSQL_PASSWORD"); envPass != "" {
		password = envPass
	}
	if password == "" {
		password = "422624"
	}

	dbname := m.cfg.DBName
	if envDB := os.Getenv("MYSQL_DB"); envDB != "" {
		dbname = envDB
	} else if envDB := os.Getenv("POSTGRES_DB"); envDB != "" {
		dbname = envDB
	}
	if dbname == "" {
		dbname = "papercompare"
	}

	// Ensure database exists on local MySQL
	rootDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		user, password, host, port,
	)
	if sqlDB, err := sql.Open("mysql", rootDSN); err == nil {
		_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbname))
		_ = sqlDB.Close()
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=15s&writeTimeout=15s",
		user, password, host, port, dbname,
	)

	fmt.Printf("[MySQL] connecting => host=%q port=%d user=%q db=%q\n", host, port, user, dbname)

	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 15; attempt++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil && sqlDB.PingContext(ctx) == nil {
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetConnMaxIdleTime(5 * time.Minute)
				sqlDB.SetConnMaxLifetime(30 * time.Minute)
				return db, nil
			}
		}
		fmt.Printf("[MySQL] waiting for mysql (%s:%d/%s) attempt %d/15: %v\n", host, port, dbname, attempt, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect mysql after retries: %w", err)
}

func (m *microSvcDB) Postgres(ctx context.Context) (*gorm.DB, error) {
	return m.MySQL(ctx)
}
