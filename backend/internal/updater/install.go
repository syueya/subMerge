package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoRollback       = errors.New("no installed update is available to roll back")
	ErrRollbackConsumed = errors.New("the installed update was already rolled back")
)

// Installer is the lifecycle-facing update installation contract.
type Installer interface {
	Install(context.Context, InstallRequest) (InstallState, error)
	Rollback(context.Context) (InstallState, error)
	State() (InstallState, error)
}

type InstallRequest struct {
	StagedPath     string
	CurrentVersion string
	NewVersion     string
	ExpectedSHA256 string
	DataFiles      []string
}

type BackupArtifact struct {
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath,omitempty"`
	Existed      bool   `json:"existed"`
}

type InstallState struct {
	InstalledAt      time.Time        `json:"installedAt"`
	RolledBackAt     *time.Time       `json:"rolledBackAt,omitempty"`
	PreviousVersion  string           `json:"previousVersion"`
	InstalledVersion string           `json:"installedVersion"`
	InstalledSHA256  string           `json:"installedSha256"`
	TargetPath       string           `json:"targetPath"`
	TargetExisted    bool             `json:"targetExisted"`
	BinaryBackupPath string           `json:"binaryBackupPath,omitempty"`
	DataBackups      []BackupArtifact `json:"dataBackups,omitempty"`
}

// FileInstaller atomically switches a Linux executable and keeps one durable
// rollback transaction. DataFiles must be quiesced before Install is called;
// the lifecycle owner is responsible for closing SQLite first.
type FileInstaller struct {
	TargetPath  string
	RollbackDir string
	StatePath   string
	Now         func() time.Time
	mu          sync.Mutex
}

func NewFileInstaller(targetPath, rollbackDir string) *FileInstaller {
	if rollbackDir == "" {
		rollbackDir = filepath.Join(filepath.Dir(targetPath), ".submerge-update")
	}
	return &FileInstaller{
		TargetPath:  filepath.Clean(targetPath),
		RollbackDir: filepath.Clean(rollbackDir),
		StatePath:   filepath.Join(filepath.Clean(rollbackDir), "state.json"),
		Now:         time.Now,
	}
}

func (i *FileInstaller) Install(ctx context.Context, request InstallRequest) (InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, err := EmbeddedPublicKey(); err != nil {
		return InstallState{}, err
	}
	if err := i.validateInstall(request); err != nil {
		return InstallState{}, err
	}
	if err := context.Cause(ctx); err != nil {
		return InstallState{}, err
	}
	if err := os.MkdirAll(i.RollbackDir, 0o700); err != nil {
		return InstallState{}, fmt.Errorf("create rollback directory: %w", err)
	}
	transactionDir, err := os.MkdirTemp(i.RollbackDir, "rollback-*")
	if err != nil {
		return InstallState{}, fmt.Errorf("create rollback transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(transactionDir)
		}
	}()

	state := InstallState{
		InstalledAt:      i.now().UTC(),
		PreviousVersion:  strings.TrimSpace(request.CurrentVersion),
		InstalledVersion: strings.TrimSpace(request.NewVersion),
		InstalledSHA256:  strings.ToLower(strings.TrimSpace(request.ExpectedSHA256)),
		TargetPath:       i.TargetPath,
	}
	if _, err := os.Stat(i.TargetPath); err == nil {
		state.TargetExisted = true
		state.BinaryBackupPath = filepath.Join(transactionDir, "submerge.previous")
		if err := copyFile(i.TargetPath, state.BinaryBackupPath, 0o700); err != nil {
			return InstallState{}, fmt.Errorf("back up current executable: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return InstallState{}, fmt.Errorf("inspect update target: %w", err)
	}

	dataDir := filepath.Join(transactionDir, "data")
	for index, source := range request.DataFiles {
		if err := context.Cause(ctx); err != nil {
			return InstallState{}, err
		}
		clean := filepath.Clean(source)
		artifact := BackupArtifact{OriginalPath: clean}
		if _, err := os.Stat(clean); os.IsNotExist(err) {
			state.DataBackups = append(state.DataBackups, artifact)
			continue
		} else if err != nil {
			return InstallState{}, fmt.Errorf("inspect data backup %q: %w", clean, err)
		}
		backup := filepath.Join(dataDir, fmt.Sprintf("%03d-%s", index, filepath.Base(clean)))
		if err := copyFile(clean, backup, 0o600); err != nil {
			return InstallState{}, fmt.Errorf("back up data file %q: %w", clean, err)
		}
		artifact.Existed = true
		artifact.BackupPath = backup
		state.DataBackups = append(state.DataBackups, artifact)
	}

	prepared, err := prepareExecutable(request.StagedPath, i.TargetPath, state.InstalledSHA256)
	if err != nil {
		return InstallState{}, err
	}
	defer os.Remove(prepared)
	if err := replaceTarget(prepared, i.TargetPath); err != nil {
		return InstallState{}, fmt.Errorf("install update executable: %w", err)
	}
	if err := writeJSONAtomic(i.StatePath, state, 0o600); err != nil {
		_ = i.restoreBinary(state)
		return InstallState{}, fmt.Errorf("persist update rollback state: %w", err)
	}
	committed = true
	return state, nil
}

func (i *FileInstaller) Rollback(ctx context.Context) (InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	state, err := i.state()
	if err != nil {
		return InstallState{}, err
	}
	if state.RolledBackAt != nil {
		return InstallState{}, ErrRollbackConsumed
	}
	if err := context.Cause(ctx); err != nil {
		return InstallState{}, err
	}
	if err := i.restoreBinary(state); err != nil {
		return InstallState{}, err
	}
	for _, artifact := range state.DataBackups {
		if err := context.Cause(ctx); err != nil {
			return InstallState{}, err
		}
		if !artifact.Existed {
			if err := os.Remove(artifact.OriginalPath); err != nil && !os.IsNotExist(err) {
				return InstallState{}, fmt.Errorf("remove new data file %q: %w", artifact.OriginalPath, err)
			}
			continue
		}
		if err := restoreFile(artifact.BackupPath, artifact.OriginalPath, 0o600); err != nil {
			return InstallState{}, fmt.Errorf("restore data file %q: %w", artifact.OriginalPath, err)
		}
	}
	now := i.now().UTC()
	state.RolledBackAt = &now
	if err := writeJSONAtomic(i.StatePath, state, 0o600); err != nil {
		return InstallState{}, fmt.Errorf("persist rollback state: %w", err)
	}
	return state, nil
}

// SQLiteDataFiles returns the database and its two journal sidecars. Closed
// databases normally only have the first file; recording absent sidecars lets
// rollback remove WAL/SHM files created by the newer version.
func SQLiteDataFiles(databasePath string) []string {
	databasePath = filepath.Clean(databasePath)
	return []string{databasePath, databasePath + "-wal", databasePath + "-shm"}
}

func (i *FileInstaller) State() (InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.state()
}

func (i *FileInstaller) state() (InstallState, error) {
	body, err := os.ReadFile(i.StatePath)
	if os.IsNotExist(err) {
		return InstallState{}, ErrNoRollback
	}
	if err != nil {
		return InstallState{}, fmt.Errorf("read update state: %w", err)
	}
	var state InstallState
	if err := json.Unmarshal(body, &state); err != nil {
		return InstallState{}, fmt.Errorf("decode update state: %w", err)
	}
	if state.TargetPath != i.TargetPath || state.InstalledVersion == "" {
		return InstallState{}, errors.New("update state does not match installer target")
	}
	return state, nil
}

func (i *FileInstaller) restoreBinary(state InstallState) error {
	if !state.TargetExisted {
		if err := os.Remove(i.TargetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove installed executable: %w", err)
		}
		return nil
	}
	if state.BinaryBackupPath == "" {
		return errors.New("rollback state is missing the executable backup")
	}
	if err := restoreFile(state.BinaryBackupPath, i.TargetPath, 0o700); err != nil {
		return fmt.Errorf("restore previous executable: %w", err)
	}
	return nil
}

func (i *FileInstaller) validateInstall(request InstallRequest) error {
	if i.TargetPath == "" || i.RollbackDir == "" || i.StatePath == "" {
		return errors.New("update installer paths are not configured")
	}
	if strings.TrimSpace(request.StagedPath) == "" {
		return errors.New("staged update path is required")
	}
	current, err := ParseVersion(request.CurrentVersion)
	if err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	next, err := ParseVersion(request.NewVersion)
	if err != nil {
		return fmt.Errorf("new version: %w", err)
	}
	if next.Compare(current) <= 0 {
		return errors.New("installed update version must be newer than current version")
	}
	hash, err := hex.DecodeString(strings.TrimSpace(request.ExpectedSHA256))
	if err != nil || len(hash) != sha256.Size {
		return errors.New("expected update SHA-256 is invalid")
	}
	return nil
}

func (i *FileInstaller) now() time.Time {
	if i.Now != nil {
		return i.Now()
	}
	return time.Now()
}

func prepareExecutable(source, target, expectedHash string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("create update target directory: %w", err)
	}
	in, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("open staged update: %w", err)
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("staged update must be a regular file")
	}
	out, err := os.CreateTemp(filepath.Dir(target), ".submerge-install-*")
	if err != nil {
		return "", fmt.Errorf("create prepared update: %w", err)
	}
	path := out.Name()
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), in); err != nil {
		return "", fmt.Errorf("copy staged update: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return "", fmt.Errorf("%w: got %s", ErrChecksumMismatch, actual)
	}
	if err := out.Chmod(0o755); err != nil {
		return "", fmt.Errorf("make update executable: %w", err)
	}
	if err := out.Sync(); err != nil {
		return "", fmt.Errorf("sync prepared update: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close prepared update: %w", err)
	}
	ok = true
	return path, nil
}

func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func restoreFile(source, target string, mode os.FileMode) error {
	prepared, err := prepareCopy(source, target, mode)
	if err != nil {
		return err
	}
	defer os.Remove(prepared)
	return replaceTarget(prepared, target)
}

func prepareCopy(source, target string, mode os.FileMode) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".submerge-restore-*")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(path)
	if err := copyFile(source, path, mode); err != nil {
		return "", err
	}
	return path, nil
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(body); err == nil {
		err = tmp.Chmod(mode)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return replaceTarget(tmpPath, path)
}
