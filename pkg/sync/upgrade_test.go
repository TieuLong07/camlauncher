package sync

import (
	"Aether/pkg/models"
	"archive/zip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeArchive(t *testing.T, dir, name string, files map[string]string) *models.ModpackArchive {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for name, content := range files {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(b)
	return &models.ModpackArchive{URL: "file://" + filepath.ToSlash(path), SHA256: hex.EncodeToString(hash[:]), Size: int64(len(b))}
}
func writeFile(t *testing.T, root, rel, value string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(value), 0644); err != nil {
		t.Fatal(err)
	}
}
func assertFile(t *testing.T, root, rel, expected string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || string(b) != expected {
		t.Fatalf("%s = %q, %v; want %q", rel, b, err, expected)
	}
}
func assertAbsent(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatalf("%s should be absent: %v", rel, err)
	}
}

func TestUpgradeOfficialArchivePreservesPlayerData(t *testing.T) {
	for _, id := range []string{"chilltown", "kamaron", "future-server"} {
		t.Run(id, func(t *testing.T) {
			root := t.TempDir()
			instance := filepath.Join(root, "instance")
			a1 := writeArchive(t, root, "v1.zip", map[string]string{"mods/old.jar": "old", "mods/same.jar": "v1", "config/server.toml": "v1", "kubejs/server_scripts/a.js": "v1", "options.txt": "default", "saves/world/level.dat": "server"})
			m := &models.ServerManifest{ID: id, Name: id, Version: "1", Archive: a1, ExpectedMods: []string{"old.jar", "same.jar"}}
			if err := SyncServerModpack(instance, m, nil); err != nil {
				t.Fatal(err)
			}
			writeFile(t, instance, "mods/personal.jar", "personal")
			writeFile(t, instance, "options.txt", "player")
			writeFile(t, instance, "saves/world/level.dat", "world")
			writeFile(t, instance, "resourcepacks/mine.zip", "mine")
			writeFile(t, instance, "shaderpacks/mine.zip", "shader")
			a2 := writeArchive(t, root, "v2.zip", map[string]string{"mods/new.jar": "new", "mods/same.jar": "v2", "config/server.toml": "v2", "kubejs/server_scripts/a.js": "v2", "options.txt": "override", "saves/world/level.dat": "override", "resourcepacks/mine.zip": "override"})
			m.Version = "2"
			m.Archive = a2
			m.ExpectedMods = []string{"new.jar", "same.jar"}
			manifestPath := filepath.Join(root, "manifest.json")
			b, _ := json.Marshal(m)
			os.WriteFile(manifestPath, b, 0644)
			check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
			if err != nil || !check.HasUpdate {
				t.Fatalf("update undetected: %+v %v", check, err)
			}
			if err = SyncServerModpack(instance, m, nil); err != nil {
				t.Fatal(err)
			}
			for rel, want := range map[string]string{"mods/new.jar": "new", "mods/same.jar": "v2", "mods/personal.jar": "personal", "config/server.toml": "v2", "kubejs/server_scripts/a.js": "v2", "options.txt": "player", "saves/world/level.dat": "world", "resourcepacks/mine.zip": "mine", "shaderpacks/mine.zip": "shader"} {
				assertFile(t, instance, rel, want)
			}
			assertAbsent(t, instance, "mods/old.jar")
			check, err = CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
			if err != nil || check.HasUpdate || len(check.ExtraMods) != 1 || check.ExtraMods[0] != "personal.jar" {
				t.Fatalf("unexpected repeat update: %+v %v", check, err)
			}
		})
	}
}

func TestIncrementalUpdateRetainsArchiveFiles(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "base.zip", map[string]string{"mods/a.jar": "a", "config/a.toml": "base"})
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	writeFile(t, instance, "config/a.toml", "user-edit")
	content := []byte("new")
	h := sha1.Sum(content)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(content) }))
	defer server.Close()
	m.Version = "2"
	m.Files = []models.ModpackFile{{Path: "mods/b.jar", URL: server.URL, SHA1: hex.EncodeToString(h[:]), Size: int64(len(content))}}
	m.ExpectedMods = []string{"a.jar", "b.jar"}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "config/a.toml", "user-edit")
	assertFile(t, instance, "mods/b.jar", "new")
}

func TestUpdateFailurePreservesPreviousFilesAndVersion(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "v1.zip", map[string]string{"mods/a.jar": "old"})
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	m.Version = "2"
	m.Archive = writeArchive(t, root, "v2.zip", map[string]string{"mods/a.jar": "new"})
	m.Files = []models.ModpackFile{{Path: "config/fail.cfg", URL: "http://127.0.0.1:1/no"}}
	if err := SyncServerModpack(instance, m, nil); err == nil {
		t.Fatal("expected download failure")
	}
	assertFile(t, instance, "mods/a.jar", "old")
	b, err := os.ReadFile(filepath.Join(instance, "sync_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state models.LocalSyncState
	if err = json.Unmarshal(b, &state); err != nil || state.SyncedVersion != "1" {
		t.Fatalf("state %s %v", b, err)
	}
}

func TestRecoverInterruptedUpdate(t *testing.T) {
	instance := t.TempDir()
	root := filepath.Join(instance, transactionDir)
	writeFile(t, root, "backup/mods/a.jar", "old")
	writeFile(t, instance, "mods/a.jar", "new")
	writeFile(t, instance, "mods/new.jar", "new")
	j := syncJournal{Entries: []journalEntry{{Path: filepath.Join("mods", "a.jar"), HadOriginal: true, Install: true}, {Path: filepath.Join("mods", "new.jar"), Install: true}}}
	if err := saveJournal(filepath.Join(root, "journal.json"), j); err != nil {
		t.Fatal(err)
	}
	if err := recoverTransaction(instance); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "mods/a.jar", "old")
	assertAbsent(t, instance, "mods/new.jar")
}

func TestOfficialChecksUseRemoteManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"id":"chilltown","version":"remote"}`) }))
	defer server.Close()
	m, err := FetchServerManifest(server.URL + "/chilltown.json")
	if err != nil || m.Version != "remote" {
		t.Fatalf("manifest = %+v, %v", m, err)
	}
}

func TestConcurrentSyncsSerialize(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "a.zip", map[string]string{"mods/a.jar": "a"})
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- SyncServerModpack(instance, m, nil) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertFile(t, instance, "mods/a.jar", "a")
}
func TestArchiveRejectsUnsafePath(t *testing.T) {
	root := t.TempDir()
	archive := writeArchive(t, root, "bad.zip", map[string]string{"../escape": "bad"})
	err := SyncServerModpack(filepath.Join(root, "instance"), &models.ServerManifest{ID: "test", Version: "1", Archive: archive}, nil)
	if err == nil || !strings.Contains(err.Error(), "đường dẫn") {
		t.Fatalf("expected path error, got %v", err)
	}
	assertAbsent(t, root, "escape")
}
func TestDetectAndRepairModifiedMod(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "base.zip", map[string]string{"mods/a.jar": "official"})
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	writeFile(t, instance, "mods/a.jar", "modified")
	manifestPath := filepath.Join(root, "manifest.json")
	b, _ := json.Marshal(m)
	os.WriteFile(manifestPath, b, 0644)
	check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || !check.HasUpdate || len(check.ModifiedMods) != 1 || check.ModifiedMods[0] != "a.jar" {
		t.Fatalf("modified not detected: %+v %v", check, err)
	}
	if err = SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "mods/a.jar", "official")
	check, err = CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || check.HasUpdate {
		t.Fatalf("repair not clean: %+v %v", check, err)
	}
}
func TestArchiveChangedWithoutVersionBump(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: writeArchive(t, root, "a.zip", map[string]string{"mods/a.jar": "one"}), ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	m.Archive = writeArchive(t, root, "b.zip", map[string]string{"mods/a.jar": "two"})
	manifestPath := filepath.Join(root, "manifest.json")
	b, _ := json.Marshal(m)
	os.WriteFile(manifestPath, b, 0644)
	check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || !check.HasUpdate {
		t.Fatalf("archive update missing: %+v %v", check, err)
	}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "mods/a.jar", "two")
}
func TestRemovedExpectedModDetectedWithoutVersionBump(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: writeArchive(t, root, "a.zip", map[string]string{"mods/a.jar": "a", "mods/b.jar": "b"}), ExpectedMods: []string{"a.jar", "b.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	m.ExpectedMods = []string{"a.jar"}
	manifestPath := filepath.Join(root, "manifest.json")
	b, _ := json.Marshal(m)
	os.WriteFile(manifestPath, b, 0644)
	check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || !check.HasUpdate {
		t.Fatalf("removal not detected: %+v %v", check, err)
	}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertAbsent(t, instance, "mods/b.jar")
}
func TestArchiveUpdatesManagedPacksAndPreservesPersonalChanges(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	a1 := writeArchive(t, root, "a.zip", map[string]string{"mods/a.jar": "a", "resourcepacks/official.zip": "old", "shaderpacks/official.zip": "old", "resourcepacks/edited.zip": "old"})
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: a1, ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	writeFile(t, instance, "resourcepacks/personal.zip", "personal")
	writeFile(t, instance, "resourcepacks/edited.zip", "user-edit")
	m.Version = "2"
	m.Archive = writeArchive(t, root, "b.zip", map[string]string{"mods/a.jar": "a", "resourcepacks/official.zip": "new", "shaderpacks/official.zip": "new", "resourcepacks/edited.zip": "new"})
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "resourcepacks/official.zip", "new")
	assertFile(t, instance, "shaderpacks/official.zip", "new")
	assertFile(t, instance, "resourcepacks/personal.zip", "personal")
	assertFile(t, instance, "resourcepacks/edited.zip", "user-edit")
}
func TestRemovedManifestOverrideRestoresArchive(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "base.zip", map[string]string{"mods/a.jar": "a", "config/server.toml": "base"})
	data := []byte("override")
	h := sha1.Sum(data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"},
		Files: []models.ModpackFile{{Path: "config/server.toml", URL: server.URL, SHA1: hex.EncodeToString(h[:]), Size: int64(len(data))}}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "config/server.toml", "override")
	m.Files = nil
	path := filepath.Join(root, "manifest.json")
	b, _ := json.Marshal(m)
	os.WriteFile(path, b, 0644)
	check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(path))
	if err != nil || !check.HasUpdate {
		t.Fatalf("override removal undetected: %+v %v", check, err)
	}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "config/server.toml", "base")
	check, err = CheckServerUpdate(instance, "file://"+filepath.ToSlash(path))
	if err != nil || check.HasUpdate {
		t.Fatalf("unexpected update: %+v %v", check, err)
	}
}
func TestLegacyStateGetsVerifiedBaseline(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "base.zip", map[string]string{"mods/a.jar": "official"})
	writeFile(t, instance, "mods/a.jar", "modified")
	writeFile(t, instance, "mods/personal.jar", "personal")
	old := models.LocalSyncState{ServerID: "test", SyncedVersion: "1", ManagedFiles: []string{"mods/a.jar"}}
	data, _ := json.Marshal(old)
	writeFile(t, instance, "sync_state.json", string(data))
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	manifestPath := filepath.Join(root, "manifest.json")
	b, _ := json.Marshal(m)
	os.WriteFile(manifestPath, b, 0644)
	result, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || !result.HasUpdate {
		t.Fatalf("legacy baseline not requested: %+v %v", result, err)
	}
	if err = SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "mods/a.jar", "official")
	assertFile(t, instance, "mods/personal.jar", "personal")
	result, err = CheckServerUpdate(instance, "file://"+filepath.ToSlash(manifestPath))
	if err != nil || result.HasUpdate {
		t.Fatalf("baseline remains pending: %+v %v", result, err)
	}
}
func TestBadArchiveChecksumLeavesInstalledVersion(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: writeArchive(t, root, "v1.zip", map[string]string{"mods/a.jar": "old"}), ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	m.Version = "2"
	m.Archive = writeArchive(t, root, "v2.zip", map[string]string{"mods/a.jar": "new"})
	m.Archive.SHA256 = strings.Repeat("0", 64)
	if err := SyncServerModpack(instance, m, nil); err == nil {
		t.Fatal("expected checksum failure")
	}
	assertFile(t, instance, "mods/a.jar", "old")
	data, err := os.ReadFile(filepath.Join(instance, "sync_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state models.LocalSyncState
	if err = json.Unmarshal(data, &state); err != nil || state.SyncedVersion != "1" {
		t.Fatalf("state = %s, %v", data, err)
	}
}
func TestLegacyOfficialResourcePackBecomesManagedWhenUnchanged(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	archive := writeArchive(t, root, "v1.zip", map[string]string{"mods/a.jar": "a", "resourcepacks/official.zip": "old"})
	writeFile(t, instance, "mods/a.jar", "a")
	writeFile(t, instance, "resourcepacks/official.zip", "old")
	old := models.LocalSyncState{ServerID: "test", SyncedVersion: "1", ManagedFiles: []string{"mods/a.jar"}}
	data, _ := json.Marshal(old)
	writeFile(t, instance, "sync_state.json", string(data))
	m := &models.ServerManifest{ID: "test", Version: "1", Archive: archive, ExpectedMods: []string{"a.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	m.Version = "2"
	m.Archive = writeArchive(t, root, "v2.zip", map[string]string{"mods/a.jar": "a", "resourcepacks/official.zip": "new"})
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "resourcepacks/official.zip", "new")
}
func TestRemovedModifiedOfficialModBecomesPersonal(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	m := &models.ServerManifest{ID: "test", Name: "test", Version: "1", Archive: writeArchive(t, root, "one.zip", map[string]string{"mods/a.jar": "official", "mods/b.jar": "official"}), ExpectedMods: []string{"a.jar", "b.jar"}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	writeFile(t, instance, "mods/b.jar", "personal edit")
	m.Version = "2"
	m.Archive = writeArchive(t, root, "two.zip", map[string]string{"mods/a.jar": "official"})
	m.ExpectedMods = []string{"a.jar"}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "mods/b.jar", "personal edit")
	file := filepath.Join(root, "manifest.json")
	data, _ := json.Marshal(m)
	os.WriteFile(file, data, 0644)
	check, err := CheckServerUpdate(instance, "file://"+filepath.ToSlash(file))
	if err != nil || check.HasUpdate || len(check.ExtraMods) != 1 {
		t.Fatalf("unexpected check: %+v %v", check, err)
	}
}

func TestIncrementalPlayerPacksRespectOwnership(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instance")
	content := map[string]string{"/resource": "resource-v1", "/shader": "shader-v1"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, content[r.URL.Path]) }))
	defer server.Close()
	entry := func(path, endpoint string) models.ModpackFile {
		data := []byte(content[endpoint])
		h := sha1.Sum(data)
		return models.ModpackFile{Path: path, URL: server.URL + endpoint, SHA1: hex.EncodeToString(h[:]), Size: int64(len(data))}
	}
	m := &models.ServerManifest{ID: "test", Version: "1", Files: []models.ModpackFile{entry("resourcepacks/official.zip", "/resource"), entry("shaderpacks/official.zip", "/shader")}}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "resourcepacks/official.zip", "resource-v1")
	assertFile(t, instance, "shaderpacks/official.zip", "shader-v1")
	content["/resource"] = "resource-v2"
	content["/shader"] = "shader-v2"
	m.Version = "2"
	m.Files = []models.ModpackFile{entry("resourcepacks/official.zip", "/resource"), entry("shaderpacks/official.zip", "/shader")}
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, instance, "resourcepacks/official.zip", "resource-v2")
	assertFile(t, instance, "shaderpacks/official.zip", "shader-v2")
	writeFile(t, instance, "shaderpacks/official.zip", "personal edit")
	m.Version = "3"
	m.Files = nil
	if err := SyncServerModpack(instance, m, nil); err != nil {
		t.Fatal(err)
	}
	assertAbsent(t, instance, "resourcepacks/official.zip")
	assertFile(t, instance, "shaderpacks/official.zip", "personal edit")
}

func TestIncrementalPlayerPackCollisionDoesNotOverwrite(t *testing.T) {
	instance := t.TempDir()
	writeFile(t, instance, "resourcepacks/official.zip", "personal")
	data := []byte("official")
	h := sha1.Sum(data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	m := &models.ServerManifest{ID: "test", Version: "1", Files: []models.ModpackFile{{Path: "resourcepacks/official.zip", URL: server.URL, SHA1: hex.EncodeToString(h[:]), Size: int64(len(data))}}}
	if err := SyncServerModpack(instance, m, nil); err == nil {
		t.Fatal("expected collision error")
	}
	assertFile(t, instance, "resourcepacks/official.zip", "personal")
	assertAbsent(t, instance, "sync_state.json")
}
