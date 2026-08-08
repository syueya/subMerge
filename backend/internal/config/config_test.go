package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadEncryptionKeyGeneratesAndReusesKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENCRYPTION_KEY", "")

	key1, err := loadEncryptionKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(key1) != 64 {
		t.Fatalf("generated key len = %d, want 64", len(key1))
	}
	key2, err := loadEncryptionKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	if key1 != key2 {
		t.Fatal("generated key should be stable across loads")
	}
}

func TestLoadEncryptionKeyPrefersEnvironment(t *testing.T) {
	dir := t.TempDir()
	key := "12345678901234567890123456789012"
	t.Setenv("ENCRYPTION_KEY", key)

	got, err := loadEncryptionKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != key {
		t.Fatalf("got key %q, want environment key", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "crypto.key")); !os.IsNotExist(err) {
		t.Fatalf("environment key should not create crypto.key, stat error: %v", err)
	}
}

func TestLoadEncryptionKeyRejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENCRYPTION_KEY", " ")
	if err := os.WriteFile(filepath.Join(dir, "crypto.key"), []byte("bad\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEncryptionKey(dir); err == nil {
		t.Fatal("expected corrupt crypto.key to fail")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("SESSION_TTL", "0s")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid SESSION_TTL to fail")
	}
}

func TestLoadAcceptsPlaceholderLikeKey(t *testing.T) {

	t.Setenv("APP_ENV", "development")
	key := "change-me-to-a-long-random-secret-key"
	t.Setenv("ENCRYPTION_KEY", key)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("placeholder-like key should be accepted when long enough: %v", err)
	}
	if cfg.EncryptionKey != key {
		t.Fatalf("encryption key changed unexpectedly: %q", cfg.EncryptionKey)
	}
}

func TestLoadRejectsShortEncryptionKey(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("expected short ENCRYPTION_KEY to fail")
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
	if cfg.RefreshInterval != 24*time.Hour {
		t.Fatalf("expected RefreshInterval=24h, got %v", cfg.RefreshInterval)
	}
	if cfg.LogOutput != "both" {
		t.Fatalf("expected LogOutput=both, got %q", cfg.LogOutput)
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
		t.Fatal("expected Version from embedded VERSION (synced from frontend/version.ts)")
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
		t.Fatal("APP_VERSION env must not override embedded VERSION")
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
	t.Setenv("LOG_OUTPUT", "none")
	t.Setenv("LOG_RETENTION_DAYS", "14")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogOutput != "both" || cfg.LogRetentionDays != 7 {
		t.Fatalf("web-managed logging env must be ignored: output=%q retention=%d", cfg.LogOutput, cfg.LogRetentionDays)
	}
}

func TestLoadRefreshIntervalEnvIsIgnored(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("SOURCE_REFRESH_INTERVAL", "6")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshInterval != 24*time.Hour {
		t.Fatalf("web-managed refresh env must be ignored: %v", cfg.RefreshInterval)
	}
}

func TestLoadLogRetentionZeroEnvIsIgnored(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("LOG_RETENTION_DAYS", "0")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogRetentionDays != 7 {
		t.Fatalf("web-managed retention env must be ignored: %d", cfg.LogRetentionDays)
	}
}

func TestLoadBoolEnvFlags(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

	// COOKIE_SECURE 仍是部署配置；DEBUG_LOGGING 已移入网页系统设置。
	_ = os.Unsetenv("COOKIE_SECURE")
	_ = os.Unsetenv("DEBUG_LOGGING")
	t.Setenv("APP_ENV", "production")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CookieSecure || cfg.DebugLogging {
		t.Fatalf("production defaults: CookieSecure=%v DebugLogging=%v", cfg.CookieSecure, cfg.DebugLogging)
	}

	t.Setenv("APP_ENV", "development")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DebugLogging {
		t.Fatal("development default should enable debug logging")
	}

	t.Setenv("PUBLIC_BASE_URL", "https://ignored.example")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("DEBUG_LOGGING", "false")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicBaseURL != DefaultPublicBaseURL || len(cfg.TrustedProxies) != 0 || cfg.CookieSecure || !cfg.DebugLogging {
		t.Fatalf("web-managed env must be ignored: %+v", cfg)
	}
}
