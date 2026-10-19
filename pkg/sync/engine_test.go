package sync

import (
	"Aether/pkg/models"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModDriftDetection(t *testing.T) {
	instanceDir := t.TempDir()
	modsDir := filepath.Join(instanceDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 2 local mods
	_ = os.WriteFile(filepath.Join(modsDir, "mod-a.jar"), []byte("a"), 0644)
	_ = os.WriteFile(filepath.Join(modsDir, "mod-b.jar"), []byte("b"), 0644)

	manifest := models.ServerManifest{
		ID:           "test",
		Name:         "Test Server",
		Version:      "1.0.0",
		ExpectedMods: []string{"mod-a.jar", "mod-b.jar", "mod-c.jar"},
	}

	manifestFile := filepath.Join(t.TempDir(), "manifest.json")
	data, _ := json.Marshal(manifest)
	_ = os.WriteFile(manifestFile, data, 0644)

	res, err := CheckServerUpdate(instanceDir, "file://"+manifestFile)
	if err != nil {
		t.Fatalf("CheckServerUpdate failed: %v", err)
	}

	if !res.IsDrifted {
		t.Errorf("Expected IsDrifted to be true, got false")
	}
	if len(res.MissingMods) != 1 || res.MissingMods[0] != "mod-c.jar" {
		t.Errorf("Expected missing mod-c.jar, got %v", res.MissingMods)
	}
	if len(res.ExtraMods) != 0 {
		t.Errorf("Expected 0 extra mods, got %v", res.ExtraMods)
	}

	// Now add an unauthorized extra mod
	_ = os.WriteFile(filepath.Join(modsDir, "cheats-xray.jar"), []byte("xray"), 0644)

	res2, err := CheckServerUpdate(instanceDir, "file://"+manifestFile)
	if err != nil {
		t.Fatalf("CheckServerUpdate 2 failed: %v", err)
	}
	if len(res2.ExtraMods) != 1 || res2.ExtraMods[0] != "cheats-xray.jar" {
		t.Errorf("Expected extra cheats-xray.jar, got %v", res2.ExtraMods)
	}
}
