package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveLocalWorldSaves(t *testing.T) {
	root := t.TempDir()
	world := filepath.Join(root, "saves", "world", "level.dat")
	if err := os.MkdirAll(filepath.Dir(world), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(world, []byte("server-owned world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveLocalWorldSaves(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "saves")); !os.IsNotExist(err) {
		t.Fatal("local saves directory still exists")
	}
}
