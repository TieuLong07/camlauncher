package main

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func TestSuggestedModsAreOptionalAndScopedToKamaron(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	mods, err := app.GetSuggestedMods("kamaron")
	if err != nil || len(mods) != 1 || mods[0].ID != "flashback" || mods[0].Installed {
		t.Fatalf("catalog: %+v, %v", mods, err)
	}
	other, err := app.GetSuggestedMods("chilltown")
	if err != nil || len(other) != 0 {
		t.Fatalf("wrong server catalog: %+v %v", other, err)
	}
	if err := app.SetSuggestedModInstalled("kamaron", "flashback", true); err == nil {
		t.Fatal("must install base modpack first")
	}
	if app.store.FindInstanceByServerID("kamaron") != nil {
		t.Fatal("optional catalog must not create an instance")
	}
	root := t.TempDir()
	app.store.AddInstance(models.Instance{ID: "kamaron", ServerID: "kamaron", Path: root, GameVersion: "1.20.1", Loader: "forge"})
	if err := app.SetSuggestedModInstalled("kamaron", "unknown", true); err == nil {
		t.Fatal("unknown suggestion must fail")
	}
	app.isStarting = true
	if err := app.SetSuggestedModInstalled("kamaron", "flashback", true); err == nil {
		t.Fatal("must not mutate mods during launch")
	}
	app.isStarting = false
	app.isRunning = true
	if err := app.SetSuggestedModInstalled("kamaron", "flashback", false); err == nil {
		t.Fatal("must not mutate mods while playing")
	}
	if _, err := os.Stat(filepath.Join(root, "mods")); !os.IsNotExist(err) {
		t.Fatalf("unexpected file mutation: %v", err)
	}
}
