package main

import (
	"Aether/pkg/models"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisteredOfficialServerUsesSharedUpdatePipeline(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	root := t.TempDir()
	manifest := models.ServerManifest{ID: "future-server", Name: "Future Server", Version: "1", GameVersion: "1.20.1", Loader: "vanilla"}
	file := filepath.Join(root, "manifest.json")
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
	officialModpacks["future-server"] = "file://" + filepath.ToSlash(file)
	defer delete(officialModpacks, "future-server")
	first, err := app.CheckOfficialModpack("future-server")
	if err != nil || !first.HasUpdate {
		t.Fatalf("check new: %+v %v", first, err)
	}
	instance, err := app.SyncOfficialModpack("future-server")
	if err != nil {
		t.Fatal(err)
	}
	if instance.ServerID != "future-server" || instance.ServerVersion != "1" {
		t.Fatalf("instance: %+v", instance)
	}
	checked, err := app.CheckOfficialModpack("future-server")
	if err != nil || checked.HasUpdate {
		t.Fatalf("check installed: %+v %v", checked, err)
	}
	manifest.Version = "2"
	manifest.GameVersion = "1.21.1"
	data, _ = json.Marshal(manifest)
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
	checked, err = app.CheckOfficialModpack("future-server")
	if err != nil || !checked.HasUpdate {
		t.Fatalf("check new version: %+v %v", checked, err)
	}
	updated, err := app.SyncOfficialModpack("future-server")
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != instance.ID || updated.GameVersion != "1.21.1" || updated.ServerVersion != "2" {
		t.Fatalf("updated instance: %+v", updated)
	}
	manifest.Loader = "fabric"
	manifest.LoaderVersion = "0.16.5"
	data, _ = json.Marshal(manifest)
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
	checked, err = app.CheckOfficialModpack("future-server")
	if err != nil || !checked.HasUpdate {
		t.Fatalf("loader change was not detected: %+v %v", checked, err)
	}
}
func TestBuiltInServersUseRegisteredManifest(t *testing.T) {
	for _, id := range []string{"chilltown", "kamaron"} {
		t.Run(id, func(t *testing.T) {
			t.Setenv("APPDATA", t.TempDir())
			app := NewApp()
			root := t.TempDir()
			manifest := models.ServerManifest{ID: id, Name: id, Version: "1", GameVersion: "1.20.1", Loader: "forge", LoaderVersion: "47.4.10"}
			path := filepath.Join(root, "manifest.json")
			data, _ := json.Marshal(manifest)
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			original := officialModpacks[id]
			officialModpacks[id] = "file://" + filepath.ToSlash(path)
			defer func() { officialModpacks[id] = original }()
			check, err := app.CheckOfficialModpack(id)
			if err != nil || !check.HasUpdate {
				t.Fatalf("initial check: %+v %v", check, err)
			}
			inst, err := app.SyncOfficialModpack(id)
			if err != nil {
				t.Fatal(err)
			}
			if inst.ServerID != id {
				t.Fatalf("installed ID %q", inst.ServerID)
			}
			check, err = app.CheckOfficialModpack(id)
			if err != nil || check.HasUpdate {
				t.Fatalf("after sync: %+v %v", check, err)
			}
			manifest.Version = "2"
			data, _ = json.Marshal(manifest)
			if err = os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			check, err = app.CheckOfficialModpack(id)
			if err != nil || !check.HasUpdate {
				t.Fatalf("upgrade check: %+v %v", check, err)
			}
			if _, err = app.SyncOfficialModpack(id); err != nil {
				t.Fatal(err)
			}
			check, err = app.CheckOfficialModpack(id)
			if err != nil || check.HasUpdate {
				t.Fatalf("after upgrade: %+v %v", check, err)
			}
		})
	}
}
func TestKamaronDoesNotAdoptAnotherServer(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	dir := filepath.Join(app.store.LauncherDir, "instances", KamaronInstanceFolder)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	app.store.AddInstance(models.Instance{ID: "other", Name: "Other", Path: dir, ServerID: "other-server"})
	if _, err := app.CheckOfficialModpack("kamaron"); err == nil {
		t.Fatal("expected conflict during check")
	}
	if _, err := app.SyncOfficialModpack("kamaron"); err == nil {
		t.Fatal("expected conflict during sync")
	}
	got := app.store.FindInstanceByServerID("other-server")
	if got == nil || got.ID != "other" {
		t.Fatal("other server was changed")
	}
}
