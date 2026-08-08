package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/submerge/submerge/backend/internal/applog"
)

func TestIsAllowedLogName(t *testing.T) {
	if !isAllowedLogName("submerge-2026-08-04.log") {
		t.Fatal("expected valid name")
	}
	for _, bad := range []string{
		"other.log",
		"submerge-2026-08-04.txt",
		"../submerge-2026-08-04.log",
		"submerge-20260804.log",
		"submerge-xx.log",
		"",
	} {
		if isAllowedLogName(bad) {
			t.Fatalf("expected reject %q", bad)
		}
	}
}

func TestListAndDetails(t *testing.T) {
	if err := applog.InitTimezone(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	zone := applog.Location()
	today := time.Now().In(zone).Format("2006-01-02")
	older := time.Now().In(zone).AddDate(0, 0, -1).Format("2006-01-02")

	content := strings.Join([]string{
		"2026/08/04 10:12:51.903722 main.go:95: [INFO] purged 1 expired sessions",
		"",
		"2026/08/04 10:12:51.904724 main.go:120: [WARN] something odd",
		"",
		"2026/08/04 19:33:03.238461 refresh.go:68: [ERROR] boom",
		"",
		"2026/08/04 19:33:03.239596 refresh.go:272: [DEBUG] sample line",
		"",
	}, "\n")
	path := filepath.Join(dir, "submerge-"+today+".log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "submerge-"+older+".log"), []byte("x\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "other.log"), []byte("nope\n"), 0o644)

	svc := NewService(dir)
	list, err := svc.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Files) != 2 {
		t.Fatalf("files=%d want 2 (other.log ignored)", len(list.Files))
	}
	if list.Files[0].Name != "submerge-"+today+".log" {
		t.Fatalf("expected newest first, got %s", list.Files[0].Name)
	}

	filtered, err := svc.List(today)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Files) != 1 {
		t.Fatalf("filter files=%d", len(filtered.Files))
	}

	details, err := svc.Details("submerge-"+today+".log", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Items) != 4 {
		t.Fatalf("items=%d want 4", len(details.Items))
	}
	if details.Items[0].Level != "debug" || details.Items[1].Level != "error" {
		t.Fatalf("levels=%q %q", details.Items[0].Level, details.Items[1].Level)
	}
	if details.Items[0].Caller != "refresh.go:272" {
		t.Fatalf("caller=%q", details.Items[0].Caller)
	}
	if details.Items[0].Timestamp <= details.Items[1].Timestamp {
		t.Fatalf("expected newest first: %d <= %d", details.Items[0].Timestamp, details.Items[1].Timestamp)
	}

	// 路径穿越拒绝
	if _, err := svc.Details("../etc/passwd", 10); err == nil {
		t.Fatal("expected path traversal reject")
	}
	if _, err := svc.Details("other.log", 10); err == nil {
		t.Fatal("expected non-prefix reject")
	}
}

func TestDetailsLineLimit(t *testing.T) {
	if err := applog.InitTimezone(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("2026/08/04 10:00:00.000000 main.go:1: [INFO] line\n\n")
	}
	name := "submerge-2026-08-04.log"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir)
	d, err := svc.Details(name, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 5 {
		t.Fatalf("got %d", len(d.Items))
	}
}

func TestParseEntriesContinuation(t *testing.T) {
	text := "2026/08/04 10:00:00.123456 foo.go:1: [INFO] first\ncont line\n\n"
	items := parseEntries(text, time.FixedZone("CST", 8*3600))
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if !strings.Contains(items[0].Content, "cont line") {
		t.Fatalf("content=%q", items[0].Content)
	}
}
