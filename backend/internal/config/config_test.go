package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("SESSION_TTL", "0s")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid SESSION_TTL to fail")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	for _, key := range []string{
			"SESSION_TTL", "SOURCE_FETCH_TIMEOUT", "SOURCE_REFRESH_INTERVAL",
			"SOURCE_MAX_BYTES", "RATE_LIMIT_LOGIN", "RATE_LIMIT_SUBSCRIBE",
			"ADMIN_USERNAME", "ADMIN_PASSWORD",
			"LOG_OUTPUT", "LOG_DIR", "LOG_FILE", "LOG_RETENTION_DAYS",
			"DATA_DIR", "DB_PATH", "STATIC_DIR",
		} {
			_ = os.Unsetenv(key)
		}
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SessionTTL != 24*time.Hour || cfg.SourceMaxBytes != 8<<20 {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
		if cfg.LogOutput != "console" {
			t.Fatalf("expected LogOutput=console, got %q", cfg.LogOutput)
		}
		if filepath.Base(filepath.Clean(cfg.LogDir)) != "log" {
			t.Fatalf("expected fixed LogDir base=log, got %q", cfg.LogDir)
		}
		if filepath.Base(filepath.Clean(cfg.DataDir)) != "data" {
			t.Fatalf("expected DataDir base=data, got %q", cfg.DataDir)
		}
		if filepath.Base(cfg.DBPath) != "submerge.db" {
			t.Fatalf("expected DBPath .../submerge.db, got %q", cfg.DBPath)
		}
		if cfg.LogRetentionDays != 7 {
			t.Fatalf("expected LogRetentionDays=7, got %d", cfg.LogRetentionDays)
		}
		if cfg.Version == "" {
			t.Fatal("expected Version from VERSION file")
		}
		// 环境变量不应覆盖版本 / 路径
		t.Setenv("APP_VERSION", "9.9.9")
		t.Setenv("LOG_DIR", "E:/tmp/should-not-use")
		t.Setenv("DATA_DIR", "E:/tmp/data-should-not-use")
		t.Setenv("DB_PATH", "E:/tmp/db-should-not-use.db")
		t.Setenv("STATIC_DIR", "E:/tmp/static-should-not-use")
		cfg2, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg2.Version == "9.9.9" {
			t.Fatal("APP_VERSION env must not override VERSION file")
		}
		if cfg2.Version != cfg.Version {
			t.Fatalf("version changed unexpectedly: %q vs %q", cfg.Version, cfg2.Version)
		}
		if cfg2.LogDir != cfg.LogDir {
			t.Fatalf("LOG_DIR env must not override fixed log dir: %q vs %q", cfg.LogDir, cfg2.LogDir)
		}
		if cfg2.DataDir != cfg.DataDir || cfg2.DBPath != cfg.DBPath || cfg2.StaticDir != cfg.StaticDir {
			t.Fatalf("path envs must not override defaults: data=%q db=%q static=%q",
				cfg2.DataDir, cfg2.DBPath, cfg2.StaticDir)
		}
	}

func TestLoadLogOutputAndRetention(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("LOG_OUTPUT", "both")
	t.Setenv("LOG_RETENTION_DAYS", "14")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogOutput != "both" {
		t.Fatalf("got LogOutput=%q", cfg.LogOutput)
	}
	if cfg.LogRetentionDays != 14 {
		t.Fatalf("got LogRetentionDays=%d", cfg.LogRetentionDays)
	}
}

func TestLoadLogRetentionZero(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("LOG_RETENTION_DAYS", "0")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogRetentionDays != 0 {
		t.Fatalf("got %d", cfg.LogRetentionDays)
	}
}
