package minecraft

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindInstanceVersionProfile(t *testing.T) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		t.Skip("APPDATA not set")
	}

	cookingPath := filepath.Join(appData, "TiCaLauncher", "instances", "Cooking TQ")
	if _, err := os.Stat(cookingPath); os.IsNotExist(err) {
		t.Skip("Cooking TQ instance not found on this machine")
	}

	detail, path, err := FindInstanceVersionProfile(cookingPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if detail == nil {
		t.Fatalf("Expected to find version profile in Cooking TQ, got nil")
	}
	t.Logf("Found profile at: %s", path)
	t.Logf("MainClass: %s", detail.MainClass)
	t.Logf("Libraries count: %d", len(detail.Libraries))

	if detail.MainClass != "cpw.mods.bootstraplauncher.BootstrapLauncher" {
		t.Errorf("Expected mainClass 'cpw.mods.bootstraplauncher.BootstrapLauncher', got %q", detail.MainClass)
	}
	if len(detail.Libraries) == 0 {
		t.Errorf("Expected libraries > 0")
	}
	if detail.Arguments == nil || len(detail.Arguments.JVM) == 0 {
		t.Errorf("Expected JVM arguments to be populated")
	}
}

func TestLaunchCookingTQOutput(t *testing.T) {
	t.Skip("Diagnostic test for manual execution only")
	appData := os.Getenv("APPDATA")
	if appData == "" {
		t.Skip("APPDATA not set")
	}

	cookingPath := filepath.Join(appData, "TiCaLauncher", "instances", "Cooking TQ")
	if _, err := os.Stat(cookingPath); os.IsNotExist(err) {
		t.Skip("Cooking TQ not found")
	}

	inst := &models.Instance{
		ID:          "65d1fc75-6acb-4e25-888a-4dd4df770975",
		Name:        "Cooking TQ",
		GameVersion: "1.20.1",
		Loader:      "forge",
		Path:        cookingPath,
		MinMemoryMB: 2048,
		MaxMemoryMB: 8384,
	}

	acc := &models.Account{
		Username:    "TiCaPlayer",
		UUID:        "e96fbb52-f542-3e54-9442-7a4cb8c1e285",
		AccessToken: "offline-token",
	}

	var logs []string
	cmd, err := PrepareAndLaunch(LaunchOptions{
		Instance:    inst,
		Account:     acc,
		LauncherDir: filepath.Join(appData, "TiCaLauncher"),
		LogFn: func(evt models.LogEvent) {
			logs = append(logs, evt.Line)
		},
	})

	if err != nil {
		t.Fatalf("PrepareAndLaunch error: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		t.Logf("Process exited with: %v", err)
	case <-time.After(15 * time.Second):
		t.Logf("Process still running successfully after 15s!")
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	for _, l := range logs {
		t.Logf("LOG: %s", l)
	}
}

func TestEvaluateArgumentList(t *testing.T) {
	args := []interface{}{
		"-DlibraryDirectory=${library_directory}",
		"-cp",
		"${classpath}",
		map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"action": "allow",
					"os": map[string]interface{}{
						"name": "windows",
					},
				},
			},
			"value": "-XX:HeapDumpPath=test",
		},
		map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"action": "allow",
					"os": map[string]interface{}{
						"name": "osx",
					},
				},
			},
			"value": "-XstartOnFirstThread",
		},
	}

	replacements := map[string]string{
		"${library_directory}": "C:/libs",
		"${classpath}":         "A.jar;B.jar",
	}

	evaluated := evaluateArgumentList(args, replacements)
	t.Logf("Evaluated args: %v", evaluated)

	expected := []string{
		"-DlibraryDirectory=C:/libs",
		"-cp",
		"A.jar;B.jar",
		"-XX:HeapDumpPath=test",
	}

	if len(evaluated) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(evaluated), evaluated)
	}
	for i, exp := range expected {
		if evaluated[i] != exp {
			t.Errorf("At %d: expected %q, got %q", i, exp, evaluated[i])
		}
	}
}
