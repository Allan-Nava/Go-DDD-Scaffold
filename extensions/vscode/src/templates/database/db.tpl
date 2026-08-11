package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB opens the MySQL pool. Replace the config import with your own module
// path, eg. "example.com/myservice/config".
func InitDB(cfg *config.Configuration) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		cfg.DBUsername, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// The DSN carries the password, so it never reaches the error.
		return nil, fmt.Errorf("connect to %s:%s/%s: %w", cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	}

	pool, err := conn.DB()
	if err != nil {
		return nil, fmt.Errorf("open connection pool: %w", err)
	}
	pool.SetMaxIdleConns(cfg.DBIdleConn)
	pool.SetMaxOpenConns(cfg.DBMaxConn)
	pool.SetConnMaxLifetime(time.Hour)

	return conn, nil
}
