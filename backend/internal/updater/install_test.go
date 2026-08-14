package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileInstallerInstallsAndRollsBackExecutableAndData(t *testing.T) {
	enableTestUpdates(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "bin", "submerge")
	staged := filepath.Join(dir, "staged")
	database := filepath.Join(dir, "data", "submerge.db")
	writeTestFile(t, target, "old-binary", 0o755)
	writeTestFile(t, staged, "new-binary", 0o600)
	writeTestFile(t, database, "old-data", 0o600)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	installer := NewFileInstaller(target, filepath.Join(dir, "rollback"))
	installer.Now = func() time.Time { return now }

	state, err := installer.Install(context.Background(), InstallRequest{
		StagedPath: staged, CurrentVersion: "1.1.9", NewVersion: "1.2.0",
		ExpectedSHA256: testHash("new-binary"), DataFiles: SQLiteDataFiles(database),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, target); got != "new-binary" {
		t.Fatalf("installed binary = %q", got)
	}
	if !state.TargetExisted || state.PreviousVersion != "1.1.9" || len(state.DataBackups) != 3 || !state.DataBackups[0].Existed || state.DataBackups[1].Existed || state.DataBackups[2].Existed {
		t.Fatalf("install state = %+v", state)
	}
	writeTestFile(t, database, "new-data", 0o600)
	writeTestFile(t, database+"-wal", "new-wal", 0o600)
	rolledBack, err := installer.Rollback(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, target); got != "old-binary" {
		t.Fatalf("rolled back binary = %q", got)
	}
	if got := readTestFile(t, database); got != "old-data" {
		t.Fatalf("rolled back data = %q", got)
	}
	if _, err := os.Stat(database + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("new WAL remains after rollback: %v", err)
	}
	if rolledBack.RolledBackAt == nil || !rolledBack.RolledBackAt.Equal(now) {
		t.Fatalf("rollback state = %+v", rolledBack)
	}
	if _, err := installer.Rollback(context.Background()); !errors.Is(err, ErrRollbackConsumed) {
		t.Fatalf("second rollback error = %v", err)
	}
}

func TestFileInstallerSupportsDockerOverrideWithoutExistingTarget(t *testing.T) {
	enableTestUpdates(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "runtime", "submerge")
	staged := filepath.Join(dir, "download")
	writeTestFile(t, staged, "new", 0o600)
	installer := NewFileInstaller(target, "")
	state, err := installer.Install(context.Background(), InstallRequest{
		StagedPath: staged, CurrentVersion: "1.1.9", NewVersion: "1.2.0", ExpectedSHA256: testHash("new"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.TargetExisted || readTestFile(t, target) != "new" {
		t.Fatalf("Docker install state = %+v", state)
	}
	if _, err := installer.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("new target remains after rollback: %v", err)
	}
}

func TestFileInstallerHashFailurePreservesCurrentExecutable(t *testing.T) {
	enableTestUpdates(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "submerge")
	staged := filepath.Join(dir, "staged")
	writeTestFile(t, target, "old", 0o755)
	writeTestFile(t, staged, "new", 0o600)
	installer := NewFileInstaller(target, filepath.Join(dir, "rollback"))
	_, err := installer.Install(context.Background(), InstallRequest{
		StagedPath: staged, CurrentVersion: "1.1.9", NewVersion: "1.2.0", ExpectedSHA256: testHash("wrong"),
	})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("hash error = %v", err)
	}
	if got := readTestFile(t, target); got != "old" {
		t.Fatalf("current executable changed to %q", got)
	}
	if _, err := installer.State(); !errors.Is(err, ErrNoRollback) {
		t.Fatalf("state error = %v", err)
	}
}

func TestFileInstallerRejectsDowngradeAndCancellation(t *testing.T) {
	enableTestUpdates(t)
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged")
	writeTestFile(t, staged, "new", 0o600)
	installer := NewFileInstaller(filepath.Join(dir, "target"), filepath.Join(dir, "rollback"))
	request := InstallRequest{StagedPath: staged, CurrentVersion: "1.2.0", NewVersion: "1.1.0", ExpectedSHA256: testHash("new")}
	if _, err := installer.Install(context.Background(), request); err == nil {
		t.Fatal("downgrade was accepted")
	}
	request.NewVersion = "1.3.0"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := installer.Install(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
}

func TestRuntimeOptions(t *testing.T) {
	updateDir := filepath.Join(t.TempDir(), "runtime")
	options, err := runtimeOptions(filepath.Join(t.TempDir(), "submerge"), updateDir, "exit")
	if err != nil {
		t.Fatal(err)
	}
	if options.TargetPath != filepath.Join(updateDir, "submerge") || options.RestartMode != RestartModeExit {
		t.Fatalf("runtime options = %+v", options)
	}
	if _, err := runtimeOptions(filepath.Join(t.TempDir(), "submerge"), "relative", "exit"); err == nil {
		t.Fatal("relative update directory was accepted")
	}
	if _, err := runtimeOptions(filepath.Join(t.TempDir(), "submerge"), "", "unknown"); err == nil {
		t.Fatal("unknown restart mode was accepted")
	}
}

func writeTestFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func testHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func enableTestUpdates(t *testing.T) {
	t.Helper()
	old := PublicKeyBase64
	PublicKeyBase64 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	t.Cleanup(func() { PublicKeyBase64 = old })
}
