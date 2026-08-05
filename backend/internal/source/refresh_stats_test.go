package source

import (
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
)

func TestDiffProxySnapshots(t *testing.T) {
	old := []database.Proxy{
		{Fingerprint: "same", Name: "US-node", Region: "US", Type: "ss", Server: "same.example", Port: 443, Enabled: false, RawJSON: `{"name":"US-node"}`},
		{Fingerprint: "modified", Name: "JP-old", Region: "JP", Type: "ss", Server: "old.example", Port: 443, RawJSON: `{"name":"JP-old"}`},
		{Fingerprint: "removed", Name: "HK-old", Region: "HK", Type: "ss", Server: "removed.example", Port: 443, RawJSON: `{"name":"HK-old"}`},
	}
	next := []preparedProxy{
		// enabled 变化不计 modified
		{fingerprint: "same", name: "US-node", region: "US", typ: "ss", server: "same.example", port: 443, rawJSON: `{"name":"US-node"}`},
		{fingerprint: "modified", name: "JP-new", region: "JP", typ: "ss", server: "new.example", port: 443, rawJSON: `{"name":"JP-new"}`},
		{fingerprint: "added", name: "SG-new", region: "SG", typ: "ss", server: "added.example", port: 443, rawJSON: `{"name":"SG-new"}`},
	}

	stats := diffProxySnapshots(old, next)
	if stats.previous != 3 || stats.kept != 1 || stats.added != 1 || stats.removed != 1 || stats.modified != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}
