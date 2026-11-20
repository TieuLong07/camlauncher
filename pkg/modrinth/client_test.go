package modrinth

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPack(t *testing.T) {
	for _, tc := range []struct{ kind, folder string }{
		{"shader", "shaderpacks"}, {"resourcepack", "resourcepacks"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			payload := []byte("test zip content")
			sum := sha512.Sum512(payload)
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/project/example":
					fmt.Fprintf(w, `{"project_type":%q}`, tc.kind)
				case "/project/example/version":
					if r.URL.Query().Get("game_versions") != `["1.20.1"]` || r.URL.Query().Has("loaders") {
						t.Errorf("unexpected version filters: %s", r.URL.RawQuery)
					}
					json.NewEncoder(w).Encode([]ProjectVersion{{Files: []VersionFile{{
						URL: server.URL + "/file", Filename: "example.zip", Primary: true,
						Size: int64(len(payload)), Hashes: map[string]string{"sha512": hex.EncodeToString(sum[:])},
					}}}})
				case "/file":
					w.Write(payload)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			name, err := installPack(server.Client(), server.URL, "example", "1.20.1", tc.kind, root)
			if err != nil || name != "example.zip" {
				t.Fatalf("install = %q, %v", name, err)
			}
			got, err := os.ReadFile(filepath.Join(root, tc.folder, name))
			if err != nil || string(got) != string(payload) {
				t.Fatalf("installed file = %q, %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(root, "mods", name)); !os.IsNotExist(err) {
				t.Fatalf("pack appeared in mods: %v", err)
			}
			if _, err := installPack(server.Client(), server.URL, "example", "1.20.1", tc.kind, root); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
		})
	}
}

func TestInstallPackRejectsInvalidDownload(t *testing.T) {
	for _, tc := range []struct {
		name, filename, digest string
		status                 int
	}{
		{"bad hash", "pack.zip", "invalid", http.StatusOK},
		{"bad status", "pack.zip", "", http.StatusBadGateway},
		{"path traversal", "../pack.zip", "", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/project/example":
					fmt.Fprint(w, `{"project_type":"shader"}`)
				case "/project/example/version":
					json.NewEncoder(w).Encode([]ProjectVersion{{Files: []VersionFile{{
						URL: server.URL + "/file", Filename: tc.filename, Primary: true,
						Hashes: map[string]string{"sha512": tc.digest},
					}}}})
				case "/file":
					w.WriteHeader(tc.status)
					fmt.Fprint(w, "content")
				}
			}))
			defer server.Close()
			root := t.TempDir()
			_, err := installPack(server.Client(), server.URL, "example", "1.20.1", "shader", root)
			if err == nil {
				t.Fatal("expected failure")
			}
			entries, readErr := os.ReadDir(filepath.Join(root, "shaderpacks"))
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("partial files remain: %v", entries)
			}
		})
	}
}
