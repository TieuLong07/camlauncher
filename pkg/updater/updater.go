package updater

import (
	"Aether/pkg/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

const (
	CurrentLauncherVersion    = "0.2.0-beta"
	DefaultUpdateManifestURL  = "https://raw.githubusercontent.com/TieuLong07/camlauncher/main/manifests/launcher.json"
	FallbackUpdateManifestURL = "https://apps.owlys.io.vn/CamLauncher/releases/latest.json"
)

// GetAppVersion returns the hardcoded current application version
func GetAppVersion() string {
	return CurrentLauncherVersion
}

// CleanupOldVersions removes any .old binary left over from a previous update
func CleanupOldVersions() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	oldPath := exePath + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		_ = os.Remove(oldPath)
	}
}

// CheckLauncherUpdate checks if a newer version of CamLauncher is available
func CheckLauncherUpdate(manifestURL string) (*models.LauncherUpdateInfo, error) {
	if manifestURL == "" {
		manifestURL = DefaultUpdateManifestURL
	}

	// Support local file url for testing (e.g. file:///d:/...)
	if strings.HasPrefix(manifestURL, "file://") {
		localPath := strings.TrimPrefix(manifestURL, "file://")
		data, err := os.ReadFile(localPath)
		if err != nil {
			return nil, err
		}
		var info models.LauncherUpdateInfo
		if err := json.Unmarshal(data, &info); err != nil {
			return nil, err
		}
		info.CurrentVersion = CurrentLauncherVersion
		info.HasUpdate = isNewerVersion(info.LatestVersion, CurrentLauncherVersion)
		return &info, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(manifestURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		// Try fallback endpoint
		if manifestURL != FallbackUpdateManifestURL {
			resp, err = client.Get(FallbackUpdateManifestURL)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối kiểm tra cập nhật launcher: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("máy chủ cập nhật trả về mã HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info models.LauncherUpdateInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("file thông tin cập nhật không hợp lệ: %w", err)
	}

	info.CurrentVersion = CurrentLauncherVersion
	info.HasUpdate = isNewerVersion(info.LatestVersion, CurrentLauncherVersion)

	return &info, nil
}

// ApplyLauncherUpdate downloads the new .exe, hot-replaces it, and relaunches the app
func ApplyLauncherUpdate(
	downloadURL string,
	expectedSHA256 string,
	progressFn func(models.ProgressEvent),
) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("không thể xác định vị trí file exe hiện tại: %w", err)
	}

	exeDir := filepath.Dir(currentExe)
	tmpExe := filepath.Join(exeDir, "CamLauncher_update.tmp")
	defer os.Remove(tmpExe)

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "downloading_update",
			Message:    "Đang tải bản cập nhật CamLauncher mới...",
			Percentage: 0,
		})
	}

	// 1. Download new executable
	client := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("lỗi tải bản cập nhật: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("máy chủ trả về mã HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tmpExe)
	if err != nil {
		return fmt.Errorf("không thể ghi file tạm: %w", err)
	}

	totalSize := resp.ContentLength
	var downloaded int64
	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	buf := make([]byte, 64*1024)
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				out.Close()
				return writeErr
			}
			downloaded += int64(n)

			if time.Since(lastUpdate) > 200*time.Millisecond {
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
						Stage:          "downloading_update",
						Message:        fmt.Sprintf("Đang tải bản cập nhật (%.1f MB / %.1f MB)", float64(downloaded)/(1024*1024), float64(totalSize)/(1024*1024)),
						Percentage:     pct,
						CurrentItem:    "CamLauncher.exe",
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
			out.Close()
			return readErr
		}
	}
	out.Close()

	// 2. Verify SHA256 if provided
	if expectedSHA256 != "" {
		computedHash := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(computedHash, expectedSHA256) {
			return fmt.Errorf("sai chữ ký SHA256 (nhận: %s, kỳ vọng: %s)", computedHash, expectedSHA256)
		}
	}

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "ready",
			Message:    "Tải hoàn tất! Đang khởi động lại ứng dụng...",
			Percentage: 100,
		})
	}

	// 3. Hot-replacement on Windows
	oldExe := currentExe + ".old"
	_ = os.Remove(oldExe)

	// Try in-process NTFS rename trick
	renameErr := os.Rename(currentExe, oldExe)
	if renameErr == nil {
		moveErr := os.Rename(tmpExe, currentExe)
		if moveErr == nil {
			// Spawns the new process
			cmd := exec.Command(currentExe)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
			_ = cmd.Start()
			// Exit current process cleanly
			os.Exit(0)
			return nil
		}
	}

	// Fallback: PowerShell supervisor helper script that waits for our PID to die, replaces exe, and launches it
	pid := os.Getpid()
	psScript := fmt.Sprintf(`
		Start-Sleep -Milliseconds 500;
		Wait-Process -Id %d -ErrorAction SilentlyContinue;
		Move-Item -Force '%s' '%s';
		Start-Process '%s';
	`, pid, tmpExe, currentExe, currentExe)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()

	// Exit current process
	os.Exit(0)
	return nil
}

func isNewerVersion(latest, current string) bool {
	lParts := parseVersion(latest)
	cParts := parseVersion(current)

	for i := 0; i < len(lParts) && i < len(cParts); i++ {
		if lParts[i] > cParts[i] {
			return true
		}
		if lParts[i] < cParts[i] {
			return false
		}
	}
	return len(lParts) > len(cParts)
}

func parseVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	var nums []int
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		nums = append(nums, n)
	}
	return nums
}
