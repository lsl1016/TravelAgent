// Package repository 封装 MySQL 持久化访问。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"travelagent/backend/internal/model"
)

var ErrUnavailable = errors.New("mysql unavailable")

// Store 封装所有 MySQL repository 方法共享的 GORM 连接。
type Store struct {
	db *gorm.DB
}

// Open 创建 GORM MySQL 连接；未配置 DSN 时返回不可用的空 Store。
func Open(dsn string) (*Store, error) {
	if dsn == "" {
		return &Store{}, nil
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// DB 返回底层 GORM 连接。
func (s *Store) DB() *gorm.DB {
	if s == nil {
		return nil
	}
	return s.db
}

// SQLDB 返回 database/sql 连接池。
func (s *Store) SQLDB() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	return s.db.DB()
}

// Ping 检查 MySQL 连接是否可用。
func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.SQLDB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// AutoMigrate 使用 GORM 自动迁移模型，主要用于本地开发。
func (s *Store) AutoMigrate() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	return s.db.AutoMigrate(
		&model.User{},
		&model.Session{},
		&model.ChatMessage{},
		&model.Preference{},
		&model.Trip{},
		&model.AgentRun{},
		&model.UserStatistics{},
	)
}

// now 返回统一使用的 UTC 时间。
func now() time.Time {
	return time.Now().UTC()
}
