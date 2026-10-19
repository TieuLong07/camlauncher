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

	// Check local manifest fallback if present
	for _, localCandidate := range []string{"manifests/chilltown.json", "D:\\2026WORK\\MCLauncher\\manifests\\chilltown.json"} {
		if strings.Contains(manifestURL, "chilltown.json") {
			if data, err := os.ReadFile(localCandidate); err == nil {
				var m models.ServerManifest
				if err := json.Unmarshal(data, &m); err == nil && len(m.ExpectedMods) > 0 {
					return &m, nil
				}
			}
		}
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

	for _, expectedMod := range manifest.ExpectedMods {
		cleanName := strings.TrimSpace(expectedMod)
		if cleanName != "" {
			remotePathSet[strings.ToLower(filepath.FromSlash(filepath.Join("mods", cleanName)))] = true
			remotePathSet[strings.ToLower("mods/"+cleanName)] = true
			remotePathSet[strings.ToLower(cleanName)] = true
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

	// 3. Mod drift check (Missing or Extra mods against ExpectedMods)
	if len(manifest.ExpectedMods) > 0 {
		expectedSet := make(map[string]bool)
		for _, modName := range manifest.ExpectedMods {
			expectedSet[strings.ToLower(strings.TrimSpace(modName))] = true
		}

		modsDir := filepath.Join(instancePath, "mods")
		localModsSet := make(map[string]bool)
		if entries, err := os.ReadDir(modsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
					localModsSet[strings.ToLower(entry.Name())] = true
					if !expectedSet[strings.ToLower(entry.Name())] {
						result.ExtraMods = append(result.ExtraMods, entry.Name())
					}
				}
			}
		}

		for _, expected := range manifest.ExpectedMods {
			trimmed := strings.TrimSpace(expected)
			if trimmed != "" && !localModsSet[strings.ToLower(trimmed)] {
				result.MissingMods = append(result.MissingMods, trimmed)
			}
		}

		if len(result.MissingMods) > 0 || len(result.ExtraMods) > 0 {
			result.IsDrifted = true
			result.HasUpdate = true
		}
	}

	// HasUpdate is true if version changed or files need syncing
	if result.CurrentVersion != manifest.Version || len(result.FilesToDownload) > 0 || len(result.FilesToDelete) > 0 || result.IsDrifted {
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
	for _, expectedMod := range manifest.ExpectedMods {
		cleanName := strings.TrimSpace(expectedMod)
		if cleanName != "" {
			remoteMap[strings.ToLower(filepath.FromSlash(filepath.Join("mods", cleanName)))] = true
			remoteMap[strings.ToLower("mods/"+cleanName)] = true
			remoteMap[strings.ToLower(cleanName)] = true
		}
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

	// Step 2.5: Handle ExpectedMods drift (purge extra mods, restore missing mods)
	if len(manifest.ExpectedMods) > 0 {
		modsDir := filepath.Join(instancePath, "mods")
		_ = os.MkdirAll(modsDir, 0755)

		expectedSet := make(map[string]bool)
		for _, m := range manifest.ExpectedMods {
			cleanName := strings.TrimSpace(m)
			if cleanName != "" {
				expectedSet[strings.ToLower(cleanName)] = true
				newManagedFiles = append(newManagedFiles, filepath.Join("mods", cleanName))
			}
		}

		// Purge extra mods
		if entries, err := os.ReadDir(modsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
					if !expectedSet[strings.ToLower(entry.Name())] {
						_ = os.Remove(filepath.Join(modsDir, entry.Name()))
					}
				}
			}
		}

		// Check which expected mods are missing from disk
		var missingFromArchive []string
		manifestFilesMap := make(map[string]models.ModpackFile)
		for _, f := range manifest.Files {
			manifestFilesMap[strings.ToLower(filepath.Base(f.Path))] = f
		}

		for _, expected := range manifest.ExpectedMods {
			trimmed := strings.TrimSpace(expected)
			if trimmed == "" {
				continue
			}
			targetPath := filepath.Join(modsDir, trimmed)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				if mf, ok := manifestFilesMap[strings.ToLower(trimmed)]; ok {
					tasks = append(tasks, minecraft.DownloadTask{
						URL:  mf.URL,
						Path: targetPath,
						SHA1: mf.SHA1,
						Size: mf.Size,
					})
				} else {
					missingFromArchive = append(missingFromArchive, "mods/"+trimmed)
				}
			}
		}

		// Extract any missing archive files directly from cached or downloaded archive
		if len(missingFromArchive) > 0 && manifest.Archive != nil {
			if progressFn != nil {
				progressFn(models.ProgressEvent{
					InstanceID: manifest.ID,
					Stage:      "syncing_modpack",
					Message:    fmt.Sprintf("Đang khôi phục %d mod thiếu từ bản gốc...", len(missingFromArchive)),
					Percentage: 25,
				})
			}
			archivePath := getOrDownloadArchive(manifest.Archive, progressFn)
			if archivePath != "" {
				_ = ExtractSpecificFiles(archivePath, instancePath, missingFromArchive)
			}
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

	archivePath := getOrDownloadArchive(archive, nil)
	if archivePath == "" {
		return fmt.Errorf("không thể tải hoặc tìm thấy gói modpack chính thức")
	}

	if archive.SHA256 != "" && !verifySHA256(archivePath, archive.SHA256) {
		return fmt.Errorf("gói modpack chính thức có SHA-256 không khớp")
	}
	if err := ExtractArchive(archivePath, instancePath); err != nil {
		return fmt.Errorf("không thể giải nén gói modpack: %w", err)
	}
	return nil
}

func getOrDownloadArchive(archive *models.ModpackArchive, progressFn func(models.ProgressEvent)) string {
	if archive == nil {
		return ""
	}
	cacheDir := filepath.Join(os.Getenv("APPDATA"), "CamLauncher", "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	archiveName := "ChillTown-archive.zip"
	if archive.URL != "" {
		archiveName = filepath.Base(archive.URL)
	}
	cachedPath := filepath.Join(cacheDir, archiveName)

	if info, err := os.Stat(cachedPath); err == nil && info.Size() > 0 {
		if archive.SHA256 == "" || verifySHA256(cachedPath, archive.SHA256) {
			return cachedPath
		}
	}

	// Check local releases directory if available (dev environment)
	relPath := filepath.Join("D:\\2026WORK\\MCLauncher\\releases", archiveName)
	if info, err := os.Stat(relPath); err == nil && info.Size() > 0 {
		if archive.SHA256 == "" || verifySHA256(relPath, archive.SHA256) {
			return relPath
		}
	}

	// Download to cache if not found locally
	urls := archive.URLs
	if len(urls) == 0 && archive.URL != "" {
		urls = []string{archive.URL}
	}
	if len(urls) == 0 {
		return ""
	}

	client := &http.Client{Timeout: 20 * time.Minute}
	out, err := os.OpenFile(cachedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return ""
	}
	defer out.Close()

	for _, url := range urls {
		req, reqErr := http.NewRequest("GET", url, nil)
		if reqErr != nil {
			return ""
		}
		req.Header.Set("User-Agent", "CamLauncher/0.1.5-alpha (ModpackSync)")
		resp, getErr := client.Do(req)
		if getErr != nil || resp.StatusCode != http.StatusOK {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			return ""
		}
		_, _ = io.Copy(out, resp.Body)
		_ = resp.Body.Close()
	}

	if archive.SHA256 != "" && !verifySHA256(cachedPath, archive.SHA256) {
		_ = os.Remove(cachedPath)
		return ""
	}

	return cachedPath
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
