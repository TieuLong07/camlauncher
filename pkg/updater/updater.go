package updater

import (
	"Aether/pkg/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/mod/semver"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	CurrentLauncherVersion    = "0.5.5-beta"
	DefaultUpdateManifestURL  = "https://raw.githubusercontent.com/TieuLong07/camlauncher/main/manifests/launcher.json"
	FallbackUpdateManifestURL = "https://owlys.io.vn/CamLauncher/releases/latest.json"
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
	discardBackup(exePath + ".old")
}

// discardBackup deletes a leftover .old binary, or renames it to a unique name
// when Windows still has the image mapped by a lingering process. A stale .old
// must never block a later update.
func discardBackup(oldPath string) {
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if err := os.Remove(oldPath); err == nil {
		return
	}
	_ = os.Rename(oldPath, fmt.Sprintf("%s.%d", oldPath, time.Now().Unix()))
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
	urls := []string{manifestURL}
	if manifestURL != FallbackUpdateManifestURL {
		urls = append(urls, FallbackUpdateManifestURL)
	}
	return fetchManifest(client, urls)
}

func fetchManifest(client *http.Client, urls []string) (*models.LauncherUpdateInfo, error) {
	var lastErr error
	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("máy chủ cập nhật trả về mã HTTP %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		var info models.LauncherUpdateInfo
		if err := json.Unmarshal(body, &info); err != nil {
			lastErr = err
			continue
		}
		v := "v" + strings.TrimPrefix(strings.TrimSpace(info.LatestVersion), "v")
		_, hashErr := hex.DecodeString(info.SHA256)
		if !semver.IsValid(v) || !strings.HasPrefix(info.DownloadURL, "https://") || len(info.SHA256) != 64 || hashErr != nil {
			lastErr = fmt.Errorf("thông tin cập nhật không hợp lệ")
			continue
		}
		info.CurrentVersion = CurrentLauncherVersion
		info.HasUpdate = isNewerVersion(info.LatestVersion, CurrentLauncherVersion)
		return &info, nil
	}
	return nil, fmt.Errorf("không thể kiểm tra cập nhật: %v", lastErr)
}

// ApplyLauncherUpdate downloads the new .exe, hot-replaces it, and relaunches the app
func ApplyLauncherUpdate(
	downloadURL string,
	expectedSHA256 string,
	progressFn func(models.ProgressEvent),
) error {
	if len(expectedSHA256) != 64 {
		return fmt.Errorf("SHA256 bản cập nhật không hợp lệ")
	}
	if _, err := hex.DecodeString(expectedSHA256); err != nil {
		return fmt.Errorf("SHA256 bản cập nhật không hợp lệ")
	}
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

	// Replace transactionally. If either placement or launch fails, restore the
	// original and keep this process alive to surface the error to the user.
	if err := replaceExecutable(currentExe, tmpExe); err != nil {
		return err
	}
	cmd := exec.Command(currentExe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	if err := cmd.Start(); err != nil {
		if removeErr := os.Remove(currentExe); removeErr != nil {
			return fmt.Errorf("không thể mở bản mới: %v; không thể gỡ bản mới để khôi phục: %w", err, removeErr)
		}
		if restoreErr := os.Rename(currentExe+".old", currentExe); restoreErr != nil {
			return fmt.Errorf("không thể mở bản mới: %v; bản cũ được giữ tại %s: %w", err, currentExe+".old", restoreErr)
		}
		return fmt.Errorf("không thể mở bản mới, đã khôi phục bản cũ: %w", err)
	}
	os.Exit(0)
	return nil
}

// replaceExecutable leaves the original at .old on success and restores it on
// failure. A pre-existing .old is normally a leftover from an update whose old
// process never exited: Windows refuses to delete an image that is still mapped,
// but it does allow a rename, so quarantine it instead of blocking the update.
func replaceExecutable(current, replacement string) error {
	old := current + ".old"
	if _, err := os.Stat(old); err == nil {
		if removeErr := os.Remove(old); removeErr != nil {
			quarantine := fmt.Sprintf("%s.%d", old, time.Now().Unix())
			if renameErr := os.Rename(old, quarantine); renameErr != nil {
				return fmt.Errorf("bản sao cũ vẫn tồn tại tại %s và đang bị một tiến trình khác giữ; hãy đóng hết cửa sổ CamLauncher rồi thử lại", old)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("không thể kiểm tra bản sao cũ: %w", err)
	}
	if err := os.Rename(current, old); err != nil {
		return fmt.Errorf("không thể lưu bản hiện tại: %w", err)
	}
	if err := os.Rename(replacement, current); err != nil {
		if restoreErr := os.Rename(old, current); restoreErr != nil {
			return fmt.Errorf("không thể cài bản mới: %v; bản cũ được giữ tại %s: %w", err, old, restoreErr)
		}
		return fmt.Errorf("không thể cài bản mới, đã khôi phục bản cũ: %w", err)
	}
	return nil
}

func isNewerVersion(latest, current string) bool {
	normalize := func(v string) string { return "v" + strings.TrimPrefix(strings.TrimSpace(v), "v") }
	l, c := normalize(latest), normalize(current)
	return semver.IsValid(l) && semver.IsValid(c) && semver.Compare(l, c) > 0
}
