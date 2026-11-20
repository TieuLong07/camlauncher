package sync

import (
	"Aether/pkg/models"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestSyncDownloadsExpectedManifestFileOnce(t *testing.T) {
	data := []byte("mod data")
	hash := sha1.Sum(data)
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		_, _ = w.Write(data)
	}))
	defer server.Close()
	manifest := &models.ServerManifest{
		ID: "test", Name: "test", Version: "1",
		Files:        []models.ModpackFile{{Path: "mods/example.jar", URL: server.URL, Size: int64(len(data)), SHA1: hex.EncodeToString(hash[:])}},
		ExpectedMods: []string{"example.jar"},
	}
	if err := SyncServerModpack(filepath.Join(t.TempDir(), "instance"), manifest, nil); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("download requests = %d, want 1", got)
	}
}
