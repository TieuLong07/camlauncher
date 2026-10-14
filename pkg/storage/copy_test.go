package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirectoryCopiesNestedFiles(t *testing.T) {
	src := filepath.Join(t.TempDir(), "Cooking TQ")
	dst := filepath.Join(t.TempDir(), "Chill Town")
	if err := os.MkdirAll(filepath.Join(src, "mods"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "mods", "pack.jar"), []byte("modpack"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirectory(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "mods", "pack.jar"))
	if err != nil || string(data) != "modpack" {
		t.Fatalf("copy missing or corrupt: %q, %v", data, err)
	}
}
