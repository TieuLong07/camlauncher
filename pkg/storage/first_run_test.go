package storage

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreCreatesFirstRunFolders(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	store := NewStore()
	for _, dir := range []string{"instances", "libraries", "assets", "versions", "runtimes"} {
		if info, err := os.Stat(filepath.Join(store.LauncherDir, dir)); err != nil || !info.IsDir() {
			t.Fatalf("missing directory %s: %v", dir, err)
		}
	}
	if len(store.GetInstances()) != 2 {
		t.Fatalf("expected two default instances, got %d", len(store.GetInstances()))
	}
	for _, instance := range store.GetInstances() {
		for _, dir := range []string{"mods", "resourcepacks", "shaderpacks", "saves", "config"} {
			if info, err := os.Stat(filepath.Join(instance.Path, dir)); err != nil || !info.IsDir() {
				t.Fatalf("missing %s/%s: %v", instance.Name, dir, err)
			}
		}
	}
	reopened := NewStore()
	if len(reopened.GetInstances()) != 2 {
		t.Fatalf("instances lost after restart: %d", len(reopened.GetInstances()))
	}
}
func TestFindOfficialChillTownSkipsOtherServer(t *testing.T) {
	instances := []models.Instance{{ID: "other", Name: "Chill Town", ServerID: "different-server"}, {ID: "legacy", Name: "Cooking TQ"}}
	got := FindOfficialChillTown(instances)
	if got == nil || got.ID != "legacy" {
		t.Fatalf("wrong instance selected: %+v", got)
	}
}
