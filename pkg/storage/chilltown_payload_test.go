package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFullInstanceFixtureIncludesCustomPlayerFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Cooking TQ")
	for path, content := range map[string]string{
		"mods/official.jar":    "mod",
		"config/server.toml":   "config",
		"options.txt":          "player-settings",
		"optionsof.txt":        "video-settings",
		"keybindings/keys.txt": "custom-keys",
		"shaderpacks/look.zip": "shader",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"mods/official.jar", "config/server.toml", "options.txt", "optionsof.txt", "keybindings/keys.txt", "shaderpacks/look.zip"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("missing fixture content %q: %v", path, err)
		}
	}
}
