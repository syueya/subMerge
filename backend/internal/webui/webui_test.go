package webui

import (
	"io/fs"
	"testing"
)

func TestFileSystemIsReadableWithoutProductionBuild(t *testing.T) {
	root, _ := FileSystem()
	if root == nil {
		t.Fatal("expected embedded filesystem")
	}
	if _, err := fs.ReadFile(root, "README.txt"); err != nil {
		t.Fatalf("read embedded marker: %v", err)
	}
}
