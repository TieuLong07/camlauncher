package updater

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrereleaseOrdering(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.3.0", "0.3.0-beta", true},
		{"0.3.0-beta.2", "0.3.0-beta.1", true},
		{"0.3.0-beta", "0.3.0", false},
		{"0.3.1-beta", "0.3.0", true},
	}
	for _, c := range cases {
		if got := isNewerVersion(c.latest, c.current); got != c.want {
			t.Errorf("%s > %s = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestFallbackUsesCanonicalOwlysDomain(t *testing.T) {
	if FallbackUpdateManifestURL != "https://owlys.io.vn/CamLauncher/releases/latest.json" {
		t.Fatalf("stale fallback: %s", FallbackUpdateManifestURL)
	}
}

func TestReplacementRestoresOriginalWhenNewFileMissing(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(current, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	err := replaceExecutable(current, filepath.Join(dir, "missing.exe"))
	if err == nil {
		t.Fatal("expected replace failure")
	}
	data, readErr := os.ReadFile(current)
	if readErr != nil || string(data) != "original" {
		t.Fatalf("original not restored: %q %v", data, readErr)
	}
}

func TestRunningExecutableReplacement(t *testing.T) {
	if os.Getenv("CAM_TEST_REPLACEMENT") == "child" {
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		if err := replaceExecutable(exe, exe+".new"); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(exe, "-test.run=^TestRunningExecutableReplacement$")
		cmd.Env = append(os.Environ(), "CAM_TEST_REPLACEMENT=new")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Getenv("CAM_TEST_REPLACEMENT") == "new" {
		if err := os.WriteFile(os.Getenv("CAM_TEST_SENTINEL"), []byte("new started"), 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	copyFile := func(dst string) {
		in, e := os.Open(source)
		if e != nil {
			t.Fatal(e)
		}
		defer in.Close()
		out, e := os.Create(dst)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = io.Copy(out, in); e != nil {
			t.Fatal(e)
		}
		if e = out.Close(); e != nil {
			t.Fatal(e)
		}
	}
	exe := filepath.Join(dir, "app.exe")
	copyFile(exe)
	copyFile(exe + ".new")
	sentinel := filepath.Join(dir, "started")
	cmd := exec.Command(exe, "-test.run=^TestRunningExecutableReplacement$")
	cmd.Env = append(os.Environ(), "CAM_TEST_REPLACEMENT=child", "CAM_TEST_SENTINEL="+sentinel)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v %s", err, output)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sentinel); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("replacement did not start")
}

func TestMalformedPrimaryFallsBack(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{bad")) }))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"latestVersion":"9.1.0","downloadUrl":"https://example.org/app.exe","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`))
	}))
	defer fallback.Close()
	info, err := fetchManifest(primary.Client(), []string{primary.URL, fallback.URL})
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestVersion != "9.1.0" {
		t.Fatalf("wrong fallback result %+v", info)
	}
}

func TestActualUpdaterOnIsolatedExecutable(t *testing.T) {
	if os.Getenv("CAM_APPLY_CHILD") == "1" {
		if err := ApplyLauncherUpdate(os.Getenv("CAM_UPDATE_URL"), os.Getenv("CAM_UPDATE_HASH"), nil); err != nil {
			t.Fatal(err)
		}
		t.Fatal("updater unexpectedly returned")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "new.go")
	marker := filepath.Join(dir, "new-started")
	code := `package main
import "os"
func main(){_ = os.WriteFile(os.Getenv("CAM_MARKER"),[]byte("started"),0600)}`
	if err := os.WriteFile(src, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	newExe := filepath.Join(dir, "new.exe")
	if out, err := exec.Command("go", "build", "-o", newExe, src).CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	payload, err := os.ReadFile(newExe)
	if err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(payload))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
	defer server.Close()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(current, original, 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(current, "-test.run=^TestActualUpdaterOnIsolatedExecutable$")
	cmd.Env = append(os.Environ(), "CAM_APPLY_CHILD=1", "CAM_UPDATE_URL="+server.URL, "CAM_UPDATE_HASH="+hash, "CAM_MARKER="+marker)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update subprocess: %v %s", err, out)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		if _, err := os.Stat(marker); err == nil {
			installed, err := os.ReadFile(current)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(installed)) != hash {
				t.Fatal("installed binary hash mismatch")
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("new executable did not start")
}

func TestApplyRejectsMissingChecksum(t *testing.T) {
	err := ApplyLauncherUpdate("http://127.0.0.1:1/no.exe", "", nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "sha256") {
		t.Fatalf("expected missing SHA256 error before network access, got %v", err)
	}
}
