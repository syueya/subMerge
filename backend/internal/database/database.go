package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 打开 SQLite 并自动迁移（纯 Go 驱动，无需 CGO）
func Open(dbPath string) (*gorm.DB, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dataDir, 0o700); err != nil {
			return nil, fmt.Errorf("restrict data dir permissions: %w", err)
		}
	}

	// Warn 级日志，但忽略 RecordNotFound（如「尚无 published 版本」属正常）
	gormLogger := logger.New(
		log.New(os.Stderr, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dbPath, 0o600); err != nil {
			return nil, fmt.Errorf("restrict db permissions: %w", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&Admin{},
		&Session{},
		&Source{},
		&Proxy{},
		&Rule{},
		&ProxyGroup{},
		&ShareToken{},
		&APIKey{},
		&Release{},
		&NetCheckSetting{},
		&SystemSetting{},
	); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	// SQLite 的既有行在新增列后可能仍为空；统一回填为远程订阅源，保持历史行为。
	if err := db.Model(&Source{}).Where("kind IS NULL OR kind = ?", "").Update("kind", "remote").Error; err != nil {
		return nil, fmt.Errorf("migrate source kind: %w", err)
	}
	return db, nil
}
