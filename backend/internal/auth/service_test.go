package auth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/submerge/submerge/backend/internal/database"
)

func TestBootstrapPersistsAvatar(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	svc := NewService(db, time.Hour)
	const avatar = "data:image/jpeg;base64,ZmFrZQ=="

	_, user, err := svc.Bootstrap("admin", "a-secure-password", "管理员", avatar)
	if err != nil {
		t.Fatal(err)
	}
	if user.Avatar != avatar {
		t.Fatalf("avatar = %q, want %q", user.Avatar, avatar)
	}

	stored, err := svc.GetAdmin(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Avatar != avatar {
		t.Fatalf("stored avatar = %q, want %q", stored.Avatar, avatar)
	}
}
