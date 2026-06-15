package config

import (
	"chat-app/internal/infrastructure/logger"
	"time"

	driver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	cfg := driver.NewConfig()
	cfg.User = GetEnv("MYSQL_USER", "root")
	cfg.Passwd = GetEnv("MYSQL_PASSWORD", "")
	cfg.Net = "tcp"
	cfg.Addr = GetEnv("MYSQL_HOST", "127.0.0.1") + ":" + GetEnv("MYSQL_PORT", "3306")
	cfg.DBName = GetEnv("MYSQL_DB", "chatdb")
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Params = map[string]string{
		"charset": "utf8mb4",
	}

	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		logger.Logger.Error("mysql connection failed", "error", err)
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Logger.Error("mysql db handle failed", "error", err)
		panic(err)
	}

	sqlDB.SetMaxOpenConns(GetEnvInt("MYSQL_MAX_OPEN_CONNS", 100))
	sqlDB.SetMaxIdleConns(GetEnvInt("MYSQL_MAX_IDLE_CONNS", 10))
	sqlDB.SetConnMaxLifetime(GetEnvDuration("MYSQL_CONN_MAX_LIFETIME", 30*time.Minute))
	sqlDB.SetConnMaxIdleTime(GetEnvDuration("MYSQL_CONN_MAX_IDLE_TIME", 10*time.Minute))

	DB = db
	logger.Logger.Info("mysql connected", "addr", cfg.Addr, "db", cfg.DBName)
}
