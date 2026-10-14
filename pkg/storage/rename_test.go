package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameDirectoryMovesEntireInstance(t *testing.T) {
	parent := t.TempDir()
	src := filepath.Join(parent, "Cooking TQ")
	dst := filepath.Join(parent, "Chill Town")
	file := filepath.Join(src, "saves", "world", "level.dat")
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("custom world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RenameDirectory(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "saves", "world", "level.dat")); err != nil {
		t.Fatalf("renamed instance lost content: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("legacy instance directory still exists")
	}
}

func TestMigrateLegacyInstancePreservesPartialBackupAndMovesFullSource(t *testing.T) {
	parent := t.TempDir()
	legacy := filepath.Join(parent, "Cooking TQ")
	target := filepath.Join(parent, "Chill Town")
	// The production paths are on the same C: volume but under different roots.
	// This fixture exercises the verified copy fallback used by Windows.
	if err := os.MkdirAll(filepath.Join(legacy, "saves", "world"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "saves", "world", "level.dat"), []byte("full-custom-instance"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "partial.txt"), []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}

	backup, err := MigrateLegacyInstance(legacy, target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "saves", "world", "level.dat")); err != nil {
		t.Fatalf("full legacy instance did not replace target: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backup, "partial.txt")); err != nil {
		t.Fatalf("partial target was not preserved as backup: %v", err)
	}
}
