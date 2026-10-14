package storage

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Name", "Normal Name"},
		{"Name: With / Bad * Chars? < > |", "Name_ With _ Bad _ Chars_ _ _ _"},
		{"Trailing spaces and dots...   ", "Trailing spaces and dots"},
		{"", "Instance"},
		{"CON", "CON_inst"},
		{"aux", "aux_inst"},
		{"com1", "com1_inst"},
	}

	for _, tc := range tests {
		got := SanitizeFolderName(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeFolderName(%q) = %q; expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestStore_UpdateInstanceRename(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ticalauncher_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &Store{
		LauncherDir: tempDir,
		Instances:   []models.Instance{},
	}

	// Create an instance with original name
	inst := models.Instance{
		ID:   "test-uuid-1",
		Name: "Original World",
		Path: filepath.Join(tempDir, "instances", "Original World"),
	}
	store.AddInstance(inst)

	// Ensure the original directory exists on disk with a dummy file inside
	if err := os.MkdirAll(inst.Path, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	testFile := filepath.Join(inst.Path, "dummy_mod.jar")
	if err := os.WriteFile(testFile, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	// Rename the instance to "Renamed World"
	updatedInst := inst
	updatedInst.Name = "Renamed World"
	result, err := store.UpdateInstance(updatedInst)
	if err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}

	expectedNewPath := filepath.Join(tempDir, "instances", "Renamed World")
	if result.Path != expectedNewPath {
		t.Errorf("Expected path %q, got %q", expectedNewPath, result.Path)
	}

	// Verify old directory no longer exists
	if _, err := os.Stat(inst.Path); !os.IsNotExist(err) {
		t.Errorf("Expected old directory %q to be gone", inst.Path)
	}

	// Verify new directory exists and has the dummy file intact
	newFile := filepath.Join(expectedNewPath, "dummy_mod.jar")
	data, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read file from renamed dir: %v", err)
	}
	if string(data) != "dummy content" {
		t.Errorf("Content mismatch: got %q, expected 'dummy content'", string(data))
	}
}

func TestStore_MigrateInstanceFolderNames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ticalauncher_migrate_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	uuidFolder := "2aa6ad77-e7b3-40fc-b6c0-4cfc297d87f5"
	oldPath := filepath.Join(tempDir, "instances", uuidFolder)
	if err := os.MkdirAll(oldPath, 0755); err != nil {
		t.Fatalf("Failed to create old UUID folder: %v", err)
	}
	_ = os.WriteFile(filepath.Join(oldPath, "options.txt"), []byte("gamma:1.0"), 0644)

	store := &Store{
		LauncherDir: tempDir,
		Instances: []models.Instance{
			{
				ID:   uuidFolder,
				Name: "TiCa Boosted 1.21.1",
				Path: oldPath,
			},
		},
	}

	store.MigrateInstanceFolderNames()

	expectedPath := filepath.Join(tempDir, "instances", "TiCa Boosted 1.21.1")
	if store.Instances[0].Path != expectedPath {
		t.Errorf("Expected migrated path %q, got %q", expectedPath, store.Instances[0].Path)
	}

	// Verify file is still present
	data, err := os.ReadFile(filepath.Join(expectedPath, "options.txt"))
	if err != nil {
		t.Fatalf("Failed to read migrated options.txt: %v", err)
	}
	if string(data) != "gamma:1.0" {
		t.Errorf("Expected content 'gamma:1.0', got %q", string(data))
	}
}

func TestStore_ScanUnregisteredInstances(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ticalauncher_scan_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	instancesDir := filepath.Join(tempDir, "instances")
	_ = os.MkdirAll(instancesDir, 0755)

	// Simulate user copying Cooking TQ into instances/
	cookingDir := filepath.Join(instancesDir, "Cooking TQ")
	_ = os.MkdirAll(filepath.Join(cookingDir, "mods"), 0755)

	// Simulate CurseForge minecraftinstance.json
	curseJSON := `{
		"name": "Cooking TQ",
		"gameVersion": "1.20.1",
		"baseModLoader": {
			"name": "forge-47.4.10",
			"type": 1,
			"forgeVersion": "47.4.10"
		}
	}`
	_ = os.WriteFile(filepath.Join(cookingDir, "minecraftinstance.json"), []byte(curseJSON), 0644)

	// Also simulate an ignored file (e.g. Cooking TQ.zip)
	_ = os.WriteFile(filepath.Join(instancesDir, "Cooking TQ.zip"), []byte("zip content"), 0644)

	store := &Store{
		LauncherDir: tempDir,
		Instances:   []models.Instance{},
	}

	newlyAdded := store.ScanUnregisteredInstances()
	if len(newlyAdded) != 1 {
		t.Fatalf("Expected 1 newly added instance, got %d", len(newlyAdded))
	}

	inst := newlyAdded[0]
	if inst.Name != "Cooking TQ" {
		t.Errorf("Expected name 'Cooking TQ', got %q", inst.Name)
	}
	if inst.GameVersion != "1.20.1" {
		t.Errorf("Expected version '1.20.1', got %q", inst.GameVersion)
	}
	if inst.Loader != "forge" {
		t.Errorf("Expected loader 'forge', got %q", inst.Loader)
	}
	if inst.LoaderVersion != "47.4.10" {
		t.Errorf("Expected loader version '47.4.10', got %q", inst.LoaderVersion)
	}

	// Verify standard folders were created
	for _, sub := range []string{"resourcepacks", "shaderpacks", "saves", "config"} {
		subPath := filepath.Join(cookingDir, sub)
		if info, err := os.Stat(subPath); err != nil || !info.IsDir() {
			t.Errorf("Expected standard folder %q to exist", sub)
		}
	}

	// Verify ticalauncher_instance.json was created
	metaPath := filepath.Join(cookingDir, "ticalauncher_instance.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("Expected ticalauncher_instance.json to exist in %q", cookingDir)
	}

	// Calling ScanUnregisteredInstances again should not duplicate
	secondScan := store.ScanUnregisteredInstances()
	if len(secondScan) != 0 {
		t.Errorf("Expected 0 newly added instances on second scan, got %d", len(secondScan))
	}
}
