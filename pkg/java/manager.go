package java

import (
	"Aether/pkg/models"
	"Aether/pkg/system"
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// GetRequiredJavaVersion determines which Java version (8, 17, 21) is required
func GetRequiredJavaVersion(gameVersion string, detail *models.VersionDetail) int {
	if detail != nil && detail.JavaVersion != nil && detail.JavaVersion.MajorVersion > 0 {
		return detail.JavaVersion.MajorVersion
	}

	// Heuristic parsing from Minecraft version string (e.g. "1.21.1", "1.20.4", "1.16.5", "1.8.9")
	cleanVer := strings.TrimSpace(gameVersion)
	parts := strings.Split(cleanVer, ".")
	if len(parts) >= 2 {
		minor, _ := strconv.Atoi(parts[1])
		patch := 0
		if len(parts) >= 3 {
			patch, _ = strconv.Atoi(parts[2])
		}

		// 1.20.5+ requires Java 21
		if minor > 20 || (minor == 20 && patch >= 5) {
			return 21
		}
		// 1.17 to 1.20.4 requires Java 17
		if minor >= 17 {
			return 17
		}
		// 1.16.5 and below requires Java 8
		return 8
	}

	return 21 // Safe modern default
}

// CheckEnvironmentSetup scans all installed Java runtimes and determines suitability for Chill Town & others
func CheckEnvironmentSetup(launcherDir, configuredJava string) *models.EnvironmentSetupStatus {
	runtimes := system.DetectJavaRuntimes()
	sysStats := system.GetSystemStats(launcherDir)

	status := &models.EnvironmentSetupStatus{
		AllRuntimes:    runtimes,
		TotalRAMGB:     int(sysStats.TotalRAMMB / 1024),
		RecommendedRAM: 4096,
		SelectedJava:   configuredJava,
	}

	if status.TotalRAMGB >= 16 {
		status.RecommendedRAM = 8192
	} else if status.TotalRAMGB >= 12 {
		status.RecommendedRAM = 6144
	}

	for _, rt := range runtimes {
		if !rt.Is64Bit {
			continue
		}
		if rt.MajorVersion == 17 && status.Java17Path == "" {
			status.HasJava17 = true
			status.Java17Path = rt.Path
		}
		if rt.MajorVersion == 21 && status.Java21Path == "" {
			status.HasJava21 = true
			status.Java21Path = rt.Path
		}
		if rt.MajorVersion == 8 && status.Java8Path == "" {
			status.HasJava8 = true
			status.Java8Path = rt.Path
		}
	}

	// Chill Town (Forge 1.20.1) requires 64-bit Java 17
	status.ChillTownReady = status.HasJava17

	if status.SelectedJava == "" {
		if status.HasJava17 {
			status.SelectedJava = status.Java17Path
		} else if len(runtimes) > 0 {
			status.SelectedJava = runtimes[0].Path
		}
	}

	return status
}

// EnsureJavaRuntime finds or automatically downloads a portable JRE for any machine
func EnsureJavaRuntime(
	launcherDir string,
	majorVersion int,
	progressFn func(models.ProgressEvent),
) (string, error) {
	if majorVersion <= 0 {
		majorVersion = 21
	}

	runtimesDir := filepath.Join(launcherDir, "runtimes")
	_ = os.MkdirAll(runtimesDir, 0755)

	targetJavaDir := filepath.Join(runtimesDir, fmt.Sprintf("java-%d", majorVersion))

	// 1. Check if we already downloaded and extracted this Java version
	if foundPath := findJavaExecutableInDir(targetJavaDir); foundPath != "" {
		if testJavaExecutable(foundPath) {
			return foundPath, nil
		}
	}

	// 2. Check if user's Windows machine already has matching Java installed locally
	localRuntimes := system.DetectJavaRuntimes()
	for _, rt := range localRuntimes {
		if rt.MajorVersion == majorVersion && rt.Is64Bit {
			if testJavaExecutable(rt.Path) {
				return rt.Path, nil
			}
		}
	}

	// 3. Fallback check for any compatible Java if major version matches or is newer (for <= 1.16 Java 8 is required strictly; 1.17+ needs >= 17)
	for _, rt := range localRuntimes {
		if majorVersion >= 17 && rt.MajorVersion >= majorVersion && rt.Is64Bit {
			if testJavaExecutable(rt.Path) {
				return rt.Path, nil
			}
		}
	}

	// 4. NOT FOUND: Auto-download portable Adoptium Eclipse Temurin JRE!
	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "downloading_java",
			Message:    fmt.Sprintf("Máy chưa có Java %d. Đang tự động tải Java %d (Eclipse Temurin JRE)...", majorVersion, majorVersion),
			Percentage: 0,
		})
	}

	downloadURL := fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%d/ga/windows/x64/jre/hotspot/normal/eclipse",
		majorVersion,
	)

	zipFile := filepath.Join(runtimesDir, fmt.Sprintf("temp_jre_%d.zip", majorVersion))
	defer os.Remove(zipFile)

	err := downloadWithProgress(downloadURL, zipFile, fmt.Sprintf("Java %d JRE", majorVersion), progressFn)
	if err != nil {
		// Fallback to Azul Zulu if Adoptium is blocked
		zuluURL := getZuluFallbackURL(majorVersion)
		if zuluURL != "" {
			if progressFn != nil {
				progressFn(models.ProgressEvent{
					Stage:      "downloading_java",
					Message:    fmt.Sprintf("Chuyển sang nguồn dự phòng Azul Zulu JRE %d...", majorVersion),
					Percentage: 0,
				})
			}
			err = downloadWithProgress(zuluURL, zipFile, fmt.Sprintf("Azul Zulu %d", majorVersion), progressFn)
		}
		if err != nil {
			return "", fmt.Errorf("không thể tự động tải Java %d: %w", majorVersion, err)
		}
	}

	// 5. Extract Zip to targetJavaDir
	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "extracting_java",
			Message:    fmt.Sprintf("Đang giải nén Java %d...", majorVersion),
			Percentage: 90,
		})
	}

	_ = os.RemoveAll(targetJavaDir)
	_ = os.MkdirAll(targetJavaDir, 0755)

	if err := unzip(zipFile, targetJavaDir); err != nil {
		return "", fmt.Errorf("lỗi giải nén Java: %w", err)
	}

	// 6. Locate java.exe in the extracted folder
	foundPath := findJavaExecutableInDir(targetJavaDir)
	if foundPath == "" {
		return "", fmt.Errorf("không tìm thấy java.exe sau khi giải nén Java %d", majorVersion)
	}

	if !testJavaExecutable(foundPath) {
		return "", fmt.Errorf("file java.exe tải về không khởi chạy được trên hệ điều hành này")
	}

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "ready",
			Message:    fmt.Sprintf("Java %d đã sẵn sàng!", majorVersion),
			Percentage: 100,
		})
	}

	return foundPath, nil
}

func getZuluFallbackURL(major int) string {
	switch major {
	case 21:
		return "https://cdn.azul.com/zulu/bin/zulu21.38.21-ca-jre21.0.5-win_x64.zip"
	case 17:
		return "https://cdn.azul.com/zulu/bin/zulu17.54.21-ca-jre17.0.13-win_x64.zip"
	case 8:
		return "https://cdn.azul.com/zulu/bin/zulu8.82.0.21-ca-jre8.0.432-win_x64.zip"
	default:
		return ""
	}
}

func findJavaExecutableInDir(dir string) string {
	var found string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "java.exe") {
			found = path
			return io.EOF // Stop walk
		}
		return nil
	})
	return found
}

func testJavaExecutable(javaPath string) bool {
	cmd := exec.Command(javaPath, "-version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

func downloadWithProgress(
	url, destPath, label string,
	progressFn func(models.ProgressEvent),
) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Follow all redirects
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("máy chủ trả về mã HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024) // 64KB buffer
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if time.Since(lastUpdate) > 250*time.Millisecond {
				lastUpdate = time.Now()
				elapsed := time.Since(startTime).Seconds()
				var speedKbps float64
				if elapsed > 0 {
					speedKbps = float64(downloaded) / 1024.0 / elapsed
				}

				pct := 0.0
				if totalSize > 0 {
					pct = (float64(downloaded) / float64(totalSize)) * 100.0
				}

				if progressFn != nil {
					progressFn(models.ProgressEvent{
						Stage:          "downloading_java",
						Message:        fmt.Sprintf("Đang tải %s (%.1f MB / %.1f MB)", label, float64(downloaded)/(1024*1024), float64(totalSize)/(1024*1024)),
						Percentage:     pct,
						CurrentItem:    label,
						CompletedItems: int(downloaded / (1024 * 1024)),
						TotalItems:     int(totalSize / (1024 * 1024)),
						SpeedKbps:      speedKbps,
					})
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
