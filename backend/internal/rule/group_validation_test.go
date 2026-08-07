package rule

import (
	"testing"

	common "github.com/submerge/submerge/backend/common"
)

func TestNormalizeGroupRequestRejectsEmptyMembers(t *testing.T) {
	_, err := normalizeGroupRequest(common.UpsertProxyGroupRequest{
		Name: "空策略组",
		Type: string(common.ProxyGroupTypeSelect),
	})
	if err == nil {
		t.Fatal("empty members should be rejected")
	}
}
