package sync

import (
	"Aether/pkg/minecraft"
	"Aether/pkg/models"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FetchServerManifest retrieves a modpack manifest from a remote URL or local file
func FetchServerManifest(manifestURL string) (*models.ServerManifest, error) {
	if manifestURL == "" {
		return nil, fmt.Errorf("URL manifest không được để trống")
	}

	// Support local file for testing
	if strings.HasPrefix(manifestURL, "file://") {
		filePath := strings.TrimPrefix(manifestURL, "file://")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		var m models.ServerManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return &m, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha (ModpackSync)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối máy chủ manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("máy chủ trả về mã HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest models.ServerManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("file manifest không hợp lệ: %w", err)
	}

	return &manifest, nil
}

// CheckServerUpdate compares local instance files with remote server manifest
func CheckServerUpdate(instancePath string, manifestURL string) (*models.UpdateCheckResult, error) {
	manifest, err := FetchServerManifest(manifestURL)
	if err != nil {
		return nil, err
	}

	syncStateFile := filepath.Join(instancePath, "sync_state.json")
	var localState models.LocalSyncState
	if data, err := os.ReadFile(syncStateFile); err == nil {
		_ = json.Unmarshal(data, &localState)
	}

	result := &models.UpdateCheckResult{
		ServerID:       manifest.ID,
		ServerName:     manifest.Name,
		CurrentVersion: localState.SyncedVersion,
		LatestVersion:  manifest.Version,
		Changelog:      manifest.Changelog,
		Manifest:       manifest,
	}

	// 1. Check which remote files are missing or modified
	remotePathSet := make(map[string]bool)
	for _, file := range manifest.Files {
		normalizedRelPath := filepath.FromSlash(file.Path)
		remotePathSet[strings.ToLower(normalizedRelPath)] = true

		localFilePath := filepath.Join(instancePath, normalizedRelPath)
		needsDownload := false

		info, err := os.Stat(localFilePath)
		if os.IsNotExist(err) {
			needsDownload = true
		} else if file.SHA1 != "" && !verifySHA1(localFilePath, file.SHA1) {
			needsDownload = true
		} else if file.Size > 0 && info != nil && info.Size() != file.Size {
			needsDownload = true
		}

		if needsDownload {
			result.FilesToDownload = append(result.FilesToDownload, file)
			result.TotalDownloadSize += file.Size
		}
	}

	// 2. Identify obsolete files (files managed previously by this modpack that admin removed)
	for _, managed := range localState.ManagedFiles {
		normManaged := filepath.FromSlash(managed)
		// Only check in mods or config dirs
		if !remotePathSet[strings.ToLower(normManaged)] {
			fullPath := filepath.Join(instancePath, normManaged)
			if _, err := os.Stat(fullPath); err == nil {
				result.FilesToDelete = append(result.FilesToDelete, normManaged)
			}
		}
	}

	// HasUpdate is true if version changed or files need syncing
	if result.CurrentVersion != manifest.Version || len(result.FilesToDownload) > 0 || len(result.FilesToDelete) > 0 {
		result.HasUpdate = true
	}

	return result, nil
}

// SyncServerModpack applies the sync: downloads new files, removes obsolete files, preserves player settings
func SyncServerModpack(
	instancePath string,
	manifest *models.ServerManifest,
	progressFn func(models.ProgressEvent),
) error {
	if manifest == nil {
		return fmt.Errorf("manifest rỗng")
	}

	_ = os.MkdirAll(instancePath, 0755)

	// Step 1: Read the previous sync state before deciding whether the official archive changed.
	syncStateFile := filepath.Join(instancePath, "sync_state.json")
	var localState models.LocalSyncState
	if data, err := os.ReadFile(syncStateFile); err == nil {
		_ = json.Unmarshal(data, &localState)
	}

	// A full official archive is used for first installs and version upgrades.
	// It replaces only server-owned folders, never saves, screenshots, options, or player files.
	if manifest.Archive != nil && (manifest.Archive.URL != "" || len(manifest.Archive.URLs) > 0) && !localState.LegacyMigrated && localState.SyncedVersion != manifest.Version {
		if localState.SyncedVersion != "" {
			if err := clearArchiveManagedPaths(instancePath); err != nil {
				return err
			}
		}
		if err := installArchive(instancePath, manifest.Archive); err != nil {
			return err
		}
	}

	// Step 2: Apply file-level updates when the manifest includes them.

	var tasks []minecraft.DownloadTask
	var newManagedFiles []string

	for _, f := range manifest.Files {
		relPath := filepath.FromSlash(f.Path)
		newManagedFiles = append(newManagedFiles, relPath)
		destPath := filepath.Join(instancePath, relPath)

		if f.SHA1 == "" || !verifySHA1(destPath, f.SHA1) {
			tasks = append(tasks, minecraft.DownloadTask{
				URL:  f.URL,
				Path: destPath,
				SHA1: f.SHA1,
				Size: f.Size,
			})
		}
	}

	// Step 2: Remove obsolete managed files (protecting player files)
	remoteMap := make(map[string]bool)
	for _, f := range newManagedFiles {
		remoteMap[strings.ToLower(f)] = true
	}

	for _, oldManaged := range localState.ManagedFiles {
		normOld := filepath.FromSlash(oldManaged)
		// Only delete if it was managed and is now removed from manifest
		if !remoteMap[strings.ToLower(normOld)] {
			// SAFETY CHECK: Never delete personal player files
			if isProtectedPlayerFile(normOld) {
				continue
			}
			fullPath := filepath.Join(instancePath, normOld)
			_ = os.Remove(fullPath)
		}
	}

	// Step 3: Download missing/updated files
	if len(tasks) > 0 {
		if progressFn != nil {
			progressFn(models.ProgressEvent{
				InstanceID: manifest.ID,
				Stage:      "syncing_modpack",
				Message:    fmt.Sprintf("Đang đồng bộ %d files modpack máy chủ...", len(tasks)),
				Percentage: 10,
			})
		}

		err := minecraft.DownloadBatch(tasks, 12, manifest.ID, "syncing_modpack", progressFn)
		if err != nil {
			return fmt.Errorf("lỗi đồng bộ modpack: %w", err)
		}
	}

	// Step 4: Save updated sync_state.json
	newState := models.LocalSyncState{
		ServerID:       manifest.ID,
		SyncedVersion:  manifest.Version,
		SyncedAt:       time.Now(),
		ManagedFiles:   newManagedFiles,
		LegacyMigrated: localState.LegacyMigrated,
	}

	stateBytes, _ := json.MarshalIndent(newState, "", "  ")
	_ = os.WriteFile(syncStateFile, stateBytes, 0644)

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			InstanceID: manifest.ID,
			Stage:      "ready",
			Message:    fmt.Sprintf("Đồng bộ máy chủ %s thành công!", manifest.Name),
			Percentage: 100,
		})
	}

	return nil
}

func clearArchiveManagedPaths(instancePath string) error {
	for _, name := range []string{"mods", "config", "defaultconfigs", "kubejs", "resourcepacks", "shaderpacks", "tlm_custom_pack", "CustomSkinLoader", "customnpcs", "patchouli_books", "patchouli_data.json", "fancymenu_data", "modernfix", "moonlight-global-datapacks", "skyblockbuilder", "xaero", "schematics", "building_blueprints", "PCL", "hotai", "emberstextapi", "fieldguide_cache", "local", "saves"} {
		if err := os.RemoveAll(filepath.Join(instancePath, name)); err != nil {
			return err
		}
	}
	return nil
}

func installArchive(instancePath string, archive *models.ModpackArchive) error {
	if strings.HasPrefix(archive.URL, "file://") {
		path := strings.TrimPrefix(archive.URL, "file://")
		if archive.SHA256 != "" && !verifySHA256(path, archive.SHA256) {
			return fmt.Errorf("gói modpack chính thức có SHA-256 không khớp")
		}
		if err := ExtractArchive(path, instancePath); err != nil {
			return fmt.Errorf("không thể giải nén gói modpack: %w", err)
		}
		return nil
	}

	urls := archive.URLs
	if len(urls) == 0 && archive.URL != "" {
		urls = []string{archive.URL}
	}
	if len(urls) == 0 {
		return fmt.Errorf("gói modpack không có URL")
	}

	tmpFile, err := os.CreateTemp("", "camlauncher-modpack-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 15 * time.Minute}
	out, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	for index, url := range urls {
		req, reqErr := http.NewRequest("GET", url, nil)
		if reqErr != nil {
			_ = out.Close()
			return reqErr
		}
		req.Header.Set("User-Agent", "CamLauncher/0.1.2-alpha (ModpackSync)")
		resp, getErr := client.Do(req)
		if getErr != nil {
			_ = out.Close()
			return fmt.Errorf("không thể tải phần %d của gói modpack: %w", index+1, getErr)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			_ = out.Close()
			return fmt.Errorf("máy chủ gói modpack trả HTTP %d ở phần %d", resp.StatusCode, index+1)
		}
		_, copyErr := io.Copy(out, resp.Body)
		_ = resp.Body.Close()
		if copyErr != nil {
			_ = out.Close()
			return copyErr
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	if archive.SHA256 != "" && !verifySHA256(tmpPath, archive.SHA256) {
		return fmt.Errorf("gói modpack chính thức có SHA-256 không khớp")
	}
	if err := ExtractArchive(tmpPath, instancePath); err != nil {
		return fmt.Errorf("không thể giải nén gói modpack: %w", err)
	}
	return nil
}

func verifySHA256(path, expected string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expected)
}

func isProtectedPlayerFile(relPath string) bool {
	lower := strings.ToLower(relPath)
	protected := []string{
		"options.txt",
		"optionsof.txt",
		"servers.dat",
		"saves",
		"screenshots",
		"resourcepacks",
		"shaderpacks",
	}
	for _, p := range protected {
		if lower == p || strings.HasPrefix(lower, p+"/") || strings.HasPrefix(lower, p+"\\") {
			return true
		}
	}
	return false
}

func verifySHA1(filePath, expectedSHA1 string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false
	}

	return strings.EqualFold(hex.EncodeToString(hasher.Sum(nil)), expectedSHA1)
}
