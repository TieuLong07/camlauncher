package sync

import (
	"Aether/pkg/models"
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncServerModpackExtractsOfficialArchive(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "chill-town.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("mods/chill.jar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("chill")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	instance := filepath.Join(dir, "instance")
	manifest := &models.ServerManifest{
		ID: "chilltown", Name: "Chill Town", Version: "0.1.0",
		Archive: &models.ModpackArchive{URL: "file://" + filepath.ToSlash(archive)},
	}
	if err := SyncServerModpack(instance, manifest, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(instance, "mods", "chill.jar")); err != nil {
		t.Fatalf("archive content missing: %v", err)
	}
}
