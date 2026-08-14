//go:build windows

package updater

import (
	"os"
	"path/filepath"
)

func replaceTarget(prepared, target string) error {
	backup := filepath.Join(filepath.Dir(target), ".submerge-replace-previous")
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(prepared, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	_ = os.Remove(backup)
	return nil
}
