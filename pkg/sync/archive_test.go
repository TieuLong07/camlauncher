package sync

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchiveRejectsTraversalAndWritesFiles(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "pack.zip")
	out := filepath.Join(dir, "out")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("mods/example.jar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("mod")); err != nil {
		t.Fatal(err)
	}
	traversal, err := w.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := traversal.Write([]byte("bad")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ExtractArchive(archivePath, out); err == nil {
		t.Fatal("expected traversal archive to be rejected")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.txt")); !os.IsNotExist(err) {
		t.Fatal("archive escaped destination")
	}
}
