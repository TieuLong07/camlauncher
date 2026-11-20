package sync

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncPreservesUnmanagedPlayerMods(t *testing.T) {
	dir := t.TempDir()
	mods := filepath.Join(dir, "mods")
	if err := os.MkdirAll(mods, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"official.jar", "personal.jar"} {
		if err := os.WriteFile(filepath.Join(mods, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	m := &models.ServerManifest{ID: "x", Version: "1", ExpectedMods: []string{"official.jar"}}
	if err := SyncServerModpack(dir, m, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(mods, "personal.jar"))
	if err != nil || string(b) != "personal.jar" {
		t.Fatalf("user mod removed or changed: %v", err)
	}
}

func TestSyncRejectsMissingExpectedMod(t *testing.T) {
	dir := t.TempDir()
	m := &models.ServerManifest{ID: "x", Version: "1", ExpectedMods: []string{"missing.jar"}}
	if err := SyncServerModpack(dir, m, nil); err == nil {
		t.Fatal("expected error when required mod is absent and not downloadable")
	}
	if _, err := os.Stat(filepath.Join(dir, "sync_state.json")); !os.IsNotExist(err) {
		t.Fatalf("must not write successful state: %v", err)
	}
}
