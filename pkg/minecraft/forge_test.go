package minecraft

import (
	"Aether/pkg/models"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestForgeInstallationRequiresSelectedRelease(t *testing.T) {
	root := t.TempDir()
	writeLibrary := func(coordinate string) {
		t.Helper()
		path := filepath.Join(root, "libraries", filepath.FromSlash(mavenCoordinateToPath(coordinate)))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// A previous Forge install supplies the shared SRG jar, but not the new release.
	writeLibrary("net.minecraft:client:1.20.1-20230612.114412:srg")
	writeLibrary("net.minecraftforge:forge:1.20.1-47.4.10:client")
	if err := verifyForgeClientInstalled(root, "1.20.1", "47.4.28"); err == nil {
		t.Fatal("accepted an older Forge installation")
	}
	writeLibrary("net.minecraftforge:forge:1.20.1-47.4.28:client")
	if err := verifyForgeClientInstalled(root, "1.20.1", "47.4.28"); err == nil {
		t.Fatal("accepted an installation missing language providers")
	}
	for _, artifact := range []string{"fmlcore", "fmlloader", "javafmllanguage", "lowcodelanguage", "mclanguage"} {
		writeLibrary("net.minecraftforge:" + artifact + ":1.20.1-47.4.28")
	}
	writeLibrary("net.minecraftforge:forge:1.20.1-47.4.28:universal")
	if err := verifyForgeClientInstalled(root, "1.20.1", "47.4.28"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureForgeClientInstalled(root, "not-a-java-executable", "1.20.1", "47.4.28", nil); err != nil {
		t.Fatalf("complete install should not run the installer: %v", err)
	}
}

func TestForgeArgumentsPreserveMinecraftArguments(t *testing.T) {
	base := &models.VersionArguments{
		Game: []interface{}{"--username", "${auth_player_name}", "--gameDir", "${game_directory}", "--accessToken", "${auth_access_token}"},
		JVM:  []interface{}{"-cp", "${classpath}"},
	}
	loader := &models.VersionArguments{
		Game: []interface{}{"--launchTarget", "forgeclient"},
		JVM:  []interface{}{"--add-modules", "ALL-MODULE-PATH"},
	}
	merged := mergeVersionArguments(base, loader)
	actual := evaluateArgumentList(merged.Game, map[string]string{
		"${auth_player_name}": "Player", "${game_directory}": "C:/instances/My Pack", "${auth_access_token}": "offline",
	})
	want := []string{"--username", "Player", "--gameDir", "C:/instances/My Pack", "--accessToken", "offline", "--launchTarget", "forgeclient"}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("game arguments = %v, want %v", actual, want)
	}
	if !reflect.DeepEqual(merged.JVM, []interface{}{"-cp", "${classpath}", "--add-modules", "ALL-MODULE-PATH"}) {
		t.Fatalf("JVM arguments lost: %v", merged.JVM)
	}
	if len(base.Game) != 6 || len(loader.Game) != 2 {
		t.Fatal("merge changed source profiles")
	}
}

func TestForgeInstallerFailureIsReturned(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(cache, 0755); err != nil {
		t.Fatal(err)
	}
	// A cached installer avoids any network access; the nonexistent Java must fail.
	path := filepath.Join(cache, "forge-1.20.1-47.4.28-installer.jar")
	if err := os.WriteFile(path, make([]byte, 1000001), 0644); err != nil {
		t.Fatal(err)
	}
	err := EnsureForgeClientInstalled(root, filepath.Join(root, "missing-java.exe"), "1.20.1", "47.4.28", nil)
	if err == nil || !strings.Contains(err.Error(), "cài Forge 47.4.28 thất bại") {
		t.Fatalf("installer failure was not reported: %v", err)
	}
}

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
