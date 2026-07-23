package applog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDailyWriterAndRetention(t *testing.T) {
	if err := InitTimezone(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	zone := Location()

	// 造几天假日志
	writeFake := func(day string) {
		p := filepath.Join(dir, "submerge-"+day+".log")
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	today := time.Now().In(zone)
	writeFake(today.Format("2006-01-02"))
	writeFake(today.AddDate(0, 0, -3).Format("2006-01-02"))
	writeFake(today.AddDate(0, 0, -10).Format("2006-01-02"))
	// 无关文件不应删
	_ = os.WriteFile(filepath.Join(dir, "other.log"), []byte("y\n"), 0o644)

	// 保留 7 天 → 删掉 10 天前的
	n, err := cleanOldLogs(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("deleted=%d want 1", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "submerge-"+today.AddDate(0, 0, -10).Format("2006-01-02")+".log")); !os.IsNotExist(err) {
		t.Fatal("old log should be gone")
	}
	if _, err := os.Stat(filepath.Join(dir, "other.log")); err != nil {
		t.Fatal("unrelated file should remain")
	}

	// Setup both 模式，写一条日志
	if err := Setup("file", dir, 7); err != nil {
		t.Fatal(err)
	}
	defer Close()
	// 触发 Write
	if _, err := writer.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	todayFile := filepath.Join(dir, "submerge-"+today.Format("2006-01-02")+".log")
	b, err := os.ReadFile(todayFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected log content")
	}
}

func TestCleanRetentionZero(t *testing.T) {
	n, err := cleanOldLogs(t.TempDir(), 0)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestInitTimezoneDefaultShanghai(t *testing.T) {
	t.Setenv("TZ", "")
	if err := InitTimezone(); err != nil {
		t.Fatal(err)
	}
	name, offset := time.Now().In(Location()).Zone()
	// Asia/Shanghai 或 FixedZone("CST") 均为 UTC+8
	if offset != 8*3600 {
		t.Fatalf("offset=%d name=%s want UTC+8", offset, name)
	}
	// 标准库 log 使用 time.Local
	if time.Local.String() != Location().String() {
		t.Fatalf("time.Local=%s Location=%s", time.Local, Location())
	}
}

func TestInitTimezoneOverride(t *testing.T) {
	t.Setenv("TZ", "UTC")
	if err := InitTimezone(); err != nil {
		t.Fatal(err)
	}
	_, offset := time.Now().In(Location()).Zone()
	if offset != 0 {
		t.Fatalf("offset=%d want 0 for UTC", offset)
	}
	// 测完恢复默认上海，避免影响同包其它测试
	t.Setenv("TZ", "")
	if err := InitTimezone(); err != nil {
		t.Fatal(err)
	}
}
