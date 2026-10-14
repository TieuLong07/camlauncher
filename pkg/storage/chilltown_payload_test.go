package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyChillTownPayloadExcludesPlayerSaves(t *testing.T) {
	src := filepath.Join(t.TempDir(), "Cooking TQ")
	dst := filepath.Join(t.TempDir(), "Chill Town")
	for path, content := range map[string]string{
		"mods/official.jar": "mod",
		"config/server.toml": "config",
		"saves/world/level.dat": "player-world",
		"options.txt": "player-settings",
	} {
		full := filepath.Join(src, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(full, []byte(content), 0644); err != nil { t.Fatal(err) }
	}
	if err := CopyChillTownPayload(src, dst); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(dst, "mods", "official.jar")); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(dst, "config", "server.toml")); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(dst, "saves", "world", "level.dat")); !os.IsNotExist(err) { t.Fatal("player saves must not be migrated") }
	if _, err := os.Stat(filepath.Join(dst, "options.txt")); !os.IsNotExist(err) { t.Fatal("player options must not be migrated") }
}
