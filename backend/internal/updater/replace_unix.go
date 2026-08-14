//go:build !windows

package updater

import "os"

func replaceTarget(prepared, target string) error {
	return os.Rename(prepared, target)
}
