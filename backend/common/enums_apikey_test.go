package common

import "testing"

func TestNormalizeAPIKeyScopes(t *testing.T) {
	if _, err := NormalizeAPIKeyScopes(nil); err == nil {
		t.Fatal("expected error on empty")
	}
	if _, err := NormalizeAPIKeyScopes([]string{"admin"}); err == nil {
		t.Fatal("expected invalid scope")
	}
	got, err := NormalizeAPIKeyScopes([]string{" Read ", "publish", "read"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != APIKeyScopeRead || got[1] != APIKeyScopePublish {
		t.Fatalf("got %v", got)
	}
	if !HasAPIKeyScope(got, APIKeyScopeRead) || HasAPIKeyScope(got, APIKeyScopeWrite) {
		t.Fatal("HasAPIKeyScope mismatch")
	}
	if !HasAPIKeyScope([]APIKeyScope{APIKeyScopeAll}, APIKeyScopeWrite) {
		t.Fatal("* should cover write")
	}
}
