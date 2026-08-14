package updater

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvUpdateDir   = "SUBMERGE_UPDATE_DIR"
	EnvRestartMode = "SUBMERGE_RESTART_MODE"

	RestartModeSystemd = "systemd"
	RestartModeExit    = "exit"
)

type RuntimeOptions struct {
	TargetPath  string
	RollbackDir string
	RestartMode string
}

// RuntimeOptionsFromEnv resolves the host installation target. Docker sets
// SUBMERGE_UPDATE_DIR=/app/runtime; the entrypoint executes runtime/submerge
// on the next restart. Native installs update the currently running path.
func RuntimeOptionsFromEnv() (RuntimeOptions, error) {
	executable, err := os.Executable()
	if err != nil {
		return RuntimeOptions{}, err
	}
	return runtimeOptions(executable, os.Getenv(EnvUpdateDir), os.Getenv(EnvRestartMode))
}

func runtimeOptions(executable, updateDir, restartMode string) (RuntimeOptions, error) {
	target := filepath.Clean(executable)
	updateDir = strings.TrimSpace(updateDir)
	if updateDir != "" {
		if !filepath.IsAbs(updateDir) {
			return RuntimeOptions{}, errors.New("SUBMERGE_UPDATE_DIR must be an absolute path")
		}
		target = filepath.Join(filepath.Clean(updateDir), "submerge")
	}
	restartMode = strings.ToLower(strings.TrimSpace(restartMode))
	if restartMode == "" {
		restartMode = RestartModeSystemd
	}
	if restartMode != RestartModeSystemd && restartMode != RestartModeExit {
		return RuntimeOptions{}, errors.New("SUBMERGE_RESTART_MODE must be systemd or exit")
	}
	return RuntimeOptions{
		TargetPath:  target,
		RollbackDir: filepath.Join(filepath.Dir(target), ".submerge-update"),
		RestartMode: restartMode,
	}, nil
}
