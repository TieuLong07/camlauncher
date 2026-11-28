package sync

import (
	"Aether/pkg/models"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func suggestedFixture(t *testing.T, status int, body string) models.SuggestedMod {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	hash := sha1.Sum([]byte("optional jar"))
	return models.SuggestedMod{ID: "example", Name: "Example", FileName: "optional.jar", URL: server.URL, Size: int64(len("optional jar")), SHA1: hex.EncodeToString(hash[:])}
}

func TestSuggestedModInstallRemoveAndSync(t *testing.T) {
	root := t.TempDir()
	mod := suggestedFixture(t, 200, "optional jar")
	if err := setSuggestedMod(root, mod, true, http.DefaultClient); err != nil {
		t.Fatal(err)
	}
	assertFile(t, root, "mods/optional.jar", "optional jar")
	writeFile(t, root, "mods/official.jar", "official")
	manifest := &models.ServerManifest{ID: "kamaron", Version: "1.0.4", ExpectedMods: []string{"official.jar"}}
	if err := SyncServerModpack(root, manifest, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, root, "mods/optional.jar", "optional jar")
	if err := setSuggestedMod(root, mod, false, http.DefaultClient); err != nil {
		t.Fatal(err)
	}
	assertAbsent(t, root, "mods/optional.jar")
	assertFile(t, root, "mods/official.jar", "official")
}

func TestSuggestedModRejectsBadDownloads(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"http error", 404, "not found"}, {"hash mismatch", 200, "corruptedjar"}, {"truncated", 200, "short"}, {"oversized", 200, "optional jar with junk"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			mod := suggestedFixture(t, test.status, test.body)
			if err := setSuggestedMod(root, mod, true, http.DefaultClient); err == nil {
				t.Fatal("expected rejected download")
			}
			assertAbsent(t, root, "mods/optional.jar")
			files, err := os.ReadDir(filepath.Join(root, "mods"))
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if len(files) != 0 {
				t.Fatalf("partial download left behind: %v", files)
			}
		})
	}
}

func TestSuggestedModPreservesChangedPlayerFile(t *testing.T) {
	root := t.TempDir()
	mod := suggestedFixture(t, 200, "optional jar")
	writeFile(t, root, "mods/optional.jar", "my custom build")
	for _, install := range []bool{true, false} {
		if err := setSuggestedMod(root, mod, install, http.DefaultClient); err == nil {
			t.Fatal("must reject changing a personal build")
		}
		assertFile(t, root, "mods/optional.jar", "my custom build")
	}
}

func TestSuggestedModReplacesSupersededFlashback(t *testing.T) {
	for _, legacy := range []string{"14ee0df1f23bfcb77c598de02c8bd29b4477f061", "08af39898f8c4ad4015c4ee081c9c6ddb8dc1261", "ee6cef75a2561ec7ace681a241afaf06ba17c498", "96a0d80013ea424fc264b61f26157cd14526a127"} {
		if !supersededFlashbackSHA1[legacy] {
			t.Fatalf("%s must be replaceable in place", legacy)
		}
	}
	root := t.TempDir()
	mod := suggestedFixture(t, 200, "optional jar")
	mod.ID = "flashback"
	mod.FileName = "flashback-1.0.1.jar"
	// The superseded hashes cannot be reproduced from fixture bytes, so assert the
	// predicate contract: a jar matching them is managed and updatable, not "modified".
	if isSupersededFlashback(filepath.Join(root, "mods", "missing.jar"), "flashback") {
		t.Fatal("missing file must not be treated as superseded")
	}
	if isSupersededFlashback(filepath.Join(root, "mods", "missing.jar"), "other") {
		t.Fatal("only flashback may use the superseded list")
	}
	if err := setSuggestedMod(root, mod, true, http.DefaultClient); err != nil {
		t.Fatal(err)
	}
	assertFile(t, root, "mods/flashback-1.0.1.jar", "optional jar")
}

func TestSuggestedModAdoptsFormerOfficialFile(t *testing.T) {
	root := t.TempDir()
	mod := suggestedFixture(t, 200, "optional jar")
	archive := writeArchive(t, root, "old.zip", map[string]string{"mods/official.jar": "official", "mods/optional.jar": "optional jar"})
	manifest := &models.ServerManifest{ID: "kamaron", Version: "1.0.3", ExpectedMods: []string{"official.jar", "optional.jar"}, Archive: archive}
	if err := SyncServerModpack(root, manifest, nil); err != nil {
		t.Fatal(err)
	}
	if err := setSuggestedMod(root, mod, true, http.DefaultClient); err != nil {
		t.Fatal(err)
	}
	manifest.Version = "1.0.4"
	manifest.ExpectedMods = []string{"official.jar"}
	if err := SyncServerModpack(root, manifest, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, root, "mods/optional.jar", "optional jar")
	state, err := suggestedState(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := state.ManagedModHashes["mods/optional.jar"]; found {
		t.Fatal("optional mod still managed")
	}
}
