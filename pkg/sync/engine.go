package sync

import (
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
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối máy chủ manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("máy chủ trả về mã HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 8*1024*1024 {
		return nil, fmt.Errorf("manifest quá lớn")
	}

	var manifest models.ServerManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("file manifest không hợp lệ: %w", err)
	}

	return &manifest, nil
}

// CheckServerUpdate compares local instance files with remote server manifest
func CheckServerUpdate(instancePath string, manifestURL string) (*models.UpdateCheckResult, error) {
	unlock, err := lockInstance(instancePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := recoverTransaction(instancePath); err != nil {
		return nil, err
	}
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
		normalizedRelPath, pathErr := validRelativePath(file.Path)
		if pathErr != nil {
			return nil, pathErr
		}
		remotePathSet[strings.ToLower(normalizedRelPath)] = true

		localFilePath := filepath.Join(instancePath, normalizedRelPath)
		needsDownload := false

		info, err := os.Stat(localFilePath)
		if os.IsNotExist(err) {
			needsDownload = true
		} else if err != nil {
			return nil, fmt.Errorf("không thể đọc %s: %w", normalizedRelPath, err)
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
	for _, previous := range localState.ManifestFiles {
		if !remotePathSet[strings.ToLower(filepath.Clean(previous))] {
			result.HasUpdate = true
		}
	}

	for _, expectedMod := range manifest.ExpectedMods {
		cleanName := strings.TrimSpace(expectedMod)
		if cleanName != "" {
			if filepath.Base(cleanName) != cleanName || strings.ContainsAny(cleanName, "\\/:") {
				return nil, fmt.Errorf("tên mod không hợp lệ: %s", cleanName)
			}
			remotePathSet[strings.ToLower(filepath.FromSlash(filepath.Join("mods", cleanName)))] = true
			remotePathSet[strings.ToLower("mods/"+cleanName)] = true
			remotePathSet[strings.ToLower(cleanName)] = true
		}
	}

	// The archive remains part of the remote manifest even when its contents are
	// not repeated in the individual-file list.
	for _, path := range localState.ArchiveManagedFiles {
		clean := filepath.Clean(path)
		key := strings.ToLower(clean)
		if len(manifest.ExpectedMods) > 0 && strings.EqualFold(filepath.Dir(clean), "mods") &&
			strings.EqualFold(filepath.Ext(clean), ".jar") && !remotePathSet[key] {
			continue
		}
		remotePathSet[key] = true
	}

	// 2. Identify obsolete files (files managed previously by this modpack that admin removed)
	previousManaged := make(map[string]bool, len(localState.ManagedFiles))
	for _, managed := range localState.ManagedFiles {
		normManaged, pathErr := validRelativePath(filepath.ToSlash(managed))
		if pathErr != nil {
			return nil, pathErr
		}
		key := strings.ToLower(normManaged)
		preserveModifiedMod := strings.EqualFold(filepath.Dir(normManaged), "mods") &&
			strings.EqualFold(filepath.Ext(normManaged), ".jar") && localState.ManagedModHashes[key] != "" &&
			!verifySHA256(filepath.Join(instancePath, normManaged), localState.ManagedModHashes[key])
		previousManaged[key] = !preserveModifiedMod
		// Only check in mods or config dirs
		if !remotePathSet[strings.ToLower(normManaged)] {
			fullPath := filepath.Join(instancePath, normManaged)
			canRemove := !isProtectedPlayerFile(normManaged)
			if preserveModifiedMod {
				canRemove = false
			}
			if isPlayerPack(normManaged) && localState.ManagedPackHashes[strings.ToLower(normManaged)] != "" {
				canRemove = verifySHA256(fullPath, localState.ManagedPackHashes[strings.ToLower(normManaged)])
			}
			if !canRemove {
				continue
			}
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
					key := strings.ToLower(filepath.Join("mods", entry.Name()))
					if !expectedSet[strings.ToLower(entry.Name())] && !previousManaged[key] {
						result.ExtraMods = append(result.ExtraMods, entry.Name())
					}
				}
			}
		}

		for _, expected := range manifest.ExpectedMods {
			trimmed := strings.TrimSpace(expected)
			if trimmed != "" && !localModsSet[strings.ToLower(trimmed)] {
				result.MissingMods = append(result.MissingMods, trimmed)
			} else if trimmed != "" {
				key := strings.ToLower(filepath.Join("mods", trimmed))
				if expected := localState.ManagedModHashes[key]; expected != "" &&
					!verifySHA256(filepath.Join(modsDir, trimmed), expected) {
					result.ModifiedMods = append(result.ModifiedMods, trimmed)
				}
			}
		}

		if len(result.MissingMods) > 0 || len(result.ModifiedMods) > 0 {
			result.IsDrifted = true
			result.HasUpdate = true
		}
	}

	// HasUpdate is true if version changed or files need syncing
	if result.CurrentVersion != manifest.Version || len(result.FilesToDownload) > 0 || len(result.FilesToDelete) > 0 || result.IsDrifted {
		result.HasUpdate = true
	}
	if manifest.Archive == nil && localState.ArchiveSHA256 != "" {
		result.HasUpdate = true
	}
	if manifest.Archive != nil && localState.ArchiveSHA256 != "" && manifest.Archive.SHA256 != "" &&
		!strings.EqualFold(localState.ArchiveSHA256, manifest.Archive.SHA256) {
		result.HasUpdate = true
	}
	// Older clients did not record archive or mod hashes. A one-time sync
	// establishes a verifiable baseline for installations created by them.
	if manifest.Archive != nil && localState.SyncedVersion != "" && len(localState.ManagedModHashes) == 0 && len(manifest.ExpectedMods) > 0 {
		result.HasUpdate = true
	}

	return result, nil
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
	urls := archive.URLs
	if len(urls) == 0 && archive.URL != "" {
		urls = []string{archive.URL}
	}
	if len(urls) == 0 {
		return ""
	}
	cacheDir := filepath.Join(os.Getenv("APPDATA"), "CamLauncher", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return ""
	}
	cacheKey := sha256.Sum256([]byte(strings.Join(urls, "\n") + "\n" + archive.SHA256))
	cachedPath := filepath.Join(cacheDir, hex.EncodeToString(cacheKey[:])+".zip")
	if archive.SHA256 != "" && verifySHA256(cachedPath, archive.SHA256) {
		return cachedPath
	}

	// The developer's release cache is used only when its contents are verified.
	if archive.SHA256 != "" && archive.URL != "" {
		local := filepath.Join("D:\\2026WORK\\MCLauncher\\releases", filepath.Base(archive.URL))
		if verifySHA256(local, archive.SHA256) {
			return local
		}
	}

	out, err := os.CreateTemp(cacheDir, ".archive-*.part")
	if err != nil {
		return ""
	}
	tempPath := out.Name()
	defer os.Remove(tempPath)
	client := &http.Client{Timeout: 20 * time.Minute}
	var downloaded int64
	lastReport := time.Now()
	for _, source := range urls {
		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			out.Close()
			return ""
		}
		req.Header.Set("User-Agent", "CamLauncher/0.5.1 (ModpackSync)")
		resp, err := client.Do(req)
		if err != nil {
			out.Close()
			return ""
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			out.Close()
			return ""
		}
		writer := &archiveProgressWriter{destination: out, onBytes: func(n int) {
			downloaded += int64(n)
			if progressFn != nil && time.Since(lastReport) >= 200*time.Millisecond {
				lastReport = time.Now()
				percentage := 0.0
				if archive.Size > 0 {
					percentage = float64(downloaded) * 100 / float64(archive.Size)
				}
				progressFn(models.ProgressEvent{Stage: "syncing_modpack", Message: fmt.Sprintf("Đang tải modpack: %.1f / %.1f MB", float64(downloaded)/(1024*1024), float64(archive.Size)/(1024*1024)), Percentage: percentage})
			}
		}}
		_, copyErr := io.Copy(writer, resp.Body)
		closeErr := resp.Body.Close()
		if copyErr != nil || closeErr != nil {
			out.Close()
			return ""
		}
	}
	if err := out.Close(); err != nil {
		return ""
	}
	if archive.Size > 0 {
		info, err := os.Stat(tempPath)
		if err != nil || info.Size() != archive.Size {
			return ""
		}
	}
	if archive.SHA256 != "" && !verifySHA256(tempPath, archive.SHA256) {
		return ""
	}
	if err := os.Rename(tempPath, cachedPath); err != nil {
		return ""
	}
	return cachedPath
}

type archiveProgressWriter struct {
	destination io.Writer
	onBytes     func(int)
}

func (w *archiveProgressWriter) Write(data []byte) (int, error) {
	n, err := w.destination.Write(data)
	w.onBytes(n)
	return n, err
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
		"logs",
		"crash-reports",
		"backups",
		"journeymap",
		"xaero",
		"local",
		"usercache.json",
		"usernamecache.json",
		".cam-sync.lock",
		".cam-sync-transaction",
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
