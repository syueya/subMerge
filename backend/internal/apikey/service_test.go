package apikey

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "apikey.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	box, err := crypto.NewBox("test-encryption-key-32-bytes-long!!")
	if err != nil {
		t.Fatal(err)
	}
	return NewService(db, box)
}

func TestCreateListSecretAndAuth(t *testing.T) {
	svc := testService(t)
	item, err := svc.Create(common.CreateAPIKeyRequest{
		Name:   "ops-agent",
		Scopes: []string{"read", "publish"},
		Note:   "test",
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if item.Key == "" || !strings.HasPrefix(item.Key, "smk_") {
		t.Fatalf("expected smk_ key, got %q", item.Key)
	}
	if item.KeyMasked == "" {
		t.Fatal("expected keyMasked")
	}

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("list len = %d", len(list.Items))
	}
	if list.Items[0].Key != "" {
		t.Fatal("list must not include plaintext key")
	}

	secret, err := svc.Secret(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if secret.Key != item.Key {
		t.Fatalf("secret mismatch")
	}

	row, scopes, err := svc.FindActiveByRaw(item.Key)
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != item.ID {
		t.Fatalf("row id = %d", row.ID)
	}
	if !common.HasAPIKeyScope(scopes, common.APIKeyScopeRead) ||
		!common.HasAPIKeyScope(scopes, common.APIKeyScopePublish) ||
		common.HasAPIKeyScope(scopes, common.APIKeyScopeWrite) {
		t.Fatalf("scopes = %v", scopes)
	}
}

func TestRevokeDisableExpireRegenerate(t *testing.T) {
	svc := testService(t)
	item, err := svc.Create(common.CreateAPIKeyRequest{
		Name:   "bot",
		Scopes: []string{"*"},
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	oldKey := item.Key

	if _, err := svc.Revoke(item.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.FindActiveByRaw(oldKey); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("want revoked, got %v", err)
	}

	regen, err := svc.Regenerate(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if regen.Key == "" || regen.Key == oldKey {
		t.Fatal("expected new key")
	}
	if _, _, err := svc.FindActiveByRaw(oldKey); err == nil {
		t.Fatal("old key should be dead")
	}
	if _, _, err := svc.FindActiveByRaw(regen.Key); err != nil {
		t.Fatal(err)
	}

	st := common.APIKeyStatusDisabled
	if _, err := svc.Update(item.ID, common.UpdateAPIKeyRequest{Status: &st}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.FindActiveByRaw(regen.Key); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("want disabled, got %v", err)
	}

	// re-enable + expire
	st = common.APIKeyStatusActive
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err := svc.Update(item.ID, common.UpdateAPIKeyRequest{Status: &st, ExpiresAt: &past}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.FindActiveByRaw(regen.Key); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("want expired, got %v", err)
	}
}

func TestNormalizeScopesStar(t *testing.T) {
	scopes, err := common.NormalizeAPIKeyScopes([]string{"read", "*", "write"})
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 1 || scopes[0] != common.APIKeyScopeAll {
		t.Fatalf("got %v", scopes)
	}
}
