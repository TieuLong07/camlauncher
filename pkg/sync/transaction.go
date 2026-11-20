package sync

import (
	"Aether/pkg/minecraft"
	"Aether/pkg/models"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const transactionDir = ".cam-sync-transaction"

type journalEntry struct {
	Path        string `json:"path"`
	HadOriginal bool   `json:"hadOriginal"`
	Install     bool   `json:"install"`
}

type syncJournal struct {
	Entries   []journalEntry `json:"entries"`
	Committed bool           `json:"committed"`
}

func isPlayerPack(rel string) bool {
	clean := strings.ToLower(filepath.ToSlash(rel))
	return strings.HasPrefix(clean, "resourcepacks/") || strings.HasPrefix(clean, "shaderpacks/")
}

func lockInstance(instancePath string) (func(), error) {
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(instancePath, ".cam-sync.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	unlock, err := lockFile(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return func() { unlock(); file.Close() }, nil
}

func validRelativePath(path string) (string, error) {
	if strings.ContainsAny(path, "\\:") {
		return "", fmt.Errorf("đường dẫn không hợp lệ: %s", path)
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("đường dẫn không hợp lệ: %s", path)
	}
	return clean, nil
}

func transactionPath(root, rel string) (string, error) {
	clean, err := validRelativePath(filepath.ToSlash(rel))
	if err != nil {
		return "", err
	}
	path := root
	parts := strings.Split(clean, string(os.PathSeparator))
	for _, part := range parts[:len(parts)-1] {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return "", fmt.Errorf("thư mục không hợp lệ: %s", path)
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return filepath.Join(root, clean), nil
}

func saveJournal(path string, journal syncJournal) error {
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func recoverTransaction(instancePath string) error {
	root := filepath.Join(instancePath, transactionDir)
	data, err := os.ReadFile(filepath.Join(root, "journal.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var journal syncJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return fmt.Errorf("không thể phục hồi cập nhật: %w", err)
	}
	if !journal.Committed {
		for i := len(journal.Entries) - 1; i >= 0; i-- {
			entry := journal.Entries[i]
			target, err := transactionPath(instancePath, entry.Path)
			if err != nil {
				return err
			}
			backup, err := transactionPath(filepath.Join(root, "backup"), entry.Path)
			if err != nil {
				return err
			}
			staged, err := transactionPath(filepath.Join(root, "stage"), entry.Path)
			if err != nil {
				return err
			}
			if _, err := os.Lstat(backup); err == nil {
				if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				if err := os.Rename(backup, target); err != nil {
					return err
				}
			} else if !entry.HadOriginal && entry.Install {
				// A missing staged file means it was moved into place.
				if _, err := os.Lstat(staged); os.IsNotExist(err) {
					if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
						return err
					}
				}
			}
		}
	}
	return os.RemoveAll(root)
}

func archiveFile(archive *models.ModpackArchive, progressFn func(models.ProgressEvent)) (string, error) {
	if strings.HasPrefix(archive.URL, "file://") {
		path := strings.TrimPrefix(archive.URL, "file://")
		if archive.SHA256 != "" && !verifySHA256(path, archive.SHA256) {
			return "", fmt.Errorf("SHA-256 archive không khớp")
		}
		return path, nil
	}
	path := getOrDownloadArchive(archive, progressFn)
	if path == "" {
		return "", fmt.Errorf("không thể tải archive modpack")
	}
	return path, nil
}

// stageArchive extracts selected archive files into a private directory.
func stageArchive(path, stage string, selected map[string]bool) ([]string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	var paths []string
	seen := map[string]bool{}
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		rel, err := validRelativePath(entry.Name)
		if err != nil {
			return nil, err
		}
		if !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("tệp archive không hợp lệ: %s", entry.Name)
		}
		key := strings.ToLower(rel)
		if selected != nil && !selected[key] {
			continue
		}
		if seen[key] {
			return nil, fmt.Errorf("archive chứa tệp trùng: %s", rel)
		}
		seen[key] = true
		if rel == "sync_state.json" || strings.HasPrefix(key, ".cam-sync") {
			continue
		}
		dest, err := transactionPath(stage, rel)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return nil, err
		}
		source, err := entry.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			source.Close()
			return nil, err
		}
		_, copyErr := io.Copy(out, source)
		closeErr := out.Close()
		source.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		paths = append(paths, rel)
	}
	return paths, nil
}

// SyncServerModpack stages the complete update before modifying the instance.
func SyncServerModpack(instancePath string, manifest *models.ServerManifest, progressFn func(models.ProgressEvent)) (err error) {
	if manifest == nil || manifest.ID == "" || manifest.Version == "" {
		return fmt.Errorf("manifest modpack không hợp lệ")
	}
	unlock, err := lockInstance(instancePath)
	if err != nil {
		return err
	}
	defer unlock()
	if err = recoverTransaction(instancePath); err != nil {
		return fmt.Errorf("không thể phục hồi cập nhật trước: %w", err)
	}

	root := filepath.Join(instancePath, transactionDir)
	if err = os.RemoveAll(root); err != nil {
		return err
	}
	stage := filepath.Join(root, "stage")
	if err = os.MkdirAll(stage, 0755); err != nil {
		return err
	}
	journalPath := filepath.Join(root, "journal.json")
	defer func() {
		if _, statErr := os.Stat(journalPath); statErr == nil {
			if recoveryErr := recoverTransaction(instancePath); recoveryErr != nil {
				err = fmt.Errorf("cập nhật: %v; phục hồi: %w", err, recoveryErr)
			}
		} else {
			_ = os.RemoveAll(root)
		}
	}()
	report := func(message string, percentage float64) {
		if progressFn != nil {
			progressFn(models.ProgressEvent{InstanceID: manifest.ID, Stage: "syncing_modpack", Message: message, Percentage: percentage})
		}
	}
	archiveProgress := func(event models.ProgressEvent) {
		event.InstanceID = manifest.ID
		event.Percentage = 5 + event.Percentage*0.65
		if progressFn != nil {
			progressFn(event)
		}
	}
	report("Đang chuẩn bị bản cập nhật...", 5)

	var previous models.LocalSyncState
	data, readErr := os.ReadFile(filepath.Join(instancePath, "sync_state.json"))
	if readErr == nil {
		if err = json.Unmarshal(data, &previous); err != nil {
			return fmt.Errorf("trạng thái đồng bộ không hợp lệ: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	if previous.ServerID != "" && previous.ServerID != manifest.ID {
		return fmt.Errorf("bản cài thuộc máy chủ khác")
	}

	archiveChanged := manifest.Archive != nil &&
		((previous.ArchiveSHA256 == "" && (previous.SyncedVersion != manifest.Version || len(previous.ManagedModHashes) == 0)) ||
			(previous.ArchiveSHA256 != "" && manifest.Archive.SHA256 != "" && !strings.EqualFold(previous.ArchiveSHA256, manifest.Archive.SHA256)) ||
			(manifest.Archive.SHA256 == "" && previous.SyncedVersion != manifest.Version))
	allowedMods := make(map[string]bool)
	for _, name := range manifest.ExpectedMods {
		if name = strings.TrimSpace(name); name != "" {
			allowedMods[strings.ToLower(name)] = true
		}
	}
	for _, file := range manifest.Files {
		rel, pathErr := validRelativePath(file.Path)
		if pathErr != nil {
			return pathErr
		}
		if strings.EqualFold(filepath.Dir(rel), "mods") {
			allowedMods[strings.ToLower(filepath.Base(rel))] = true
		}
	}
	managedArchivePath := func(rel string) bool {
		return len(manifest.ExpectedMods) == 0 || !strings.EqualFold(filepath.Dir(rel), "mods") ||
			!strings.EqualFold(filepath.Ext(rel), ".jar") || allowedMods[strings.ToLower(filepath.Base(rel))]
	}
	archiveManaged := append([]string{}, previous.ArchiveManagedFiles...)
	if manifest.Archive == nil {
		archiveManaged = nil
	}
	staged := map[string]string{}
	var archivePath string
	if archiveChanged {
		report("Đang tải và chuẩn bị modpack...", 10)
		archivePath, err = archiveFile(manifest.Archive, archiveProgress)
		if err != nil {
			return err
		}
		var paths []string
		paths, err = stageArchive(archivePath, stage, nil)
		if err != nil {
			return err
		}
		archiveManaged = nil
		for _, rel := range paths {
			key := strings.ToLower(rel)
			if !managedArchivePath(rel) {
				if err := os.Remove(filepath.Join(stage, rel)); err != nil {
					return err
				}
				continue
			}
			target, pathErr := transactionPath(instancePath, rel)
			if pathErr != nil {
				return pathErr
			}
			if isProtectedPlayerFile(rel) {
				if !isPlayerPack(rel) {
					_ = os.Remove(filepath.Join(stage, rel))
					continue
				}
				if _, statErr := os.Lstat(target); statErr == nil {
					expected := previous.ManagedPackHashes[key]
					if expected == "" {
						stagedFile, openErr := os.Open(filepath.Join(stage, rel))
						if openErr != nil {
							return openErr
						}
						hash := sha256.New()
						_, hashErr := io.Copy(hash, stagedFile)
						closeErr := stagedFile.Close()
						if hashErr != nil {
							return hashErr
						}
						if closeErr != nil {
							return closeErr
						}
						expected = hex.EncodeToString(hash.Sum(nil))
					}
					if !verifySHA256(target, expected) {
						_ = os.Remove(filepath.Join(stage, rel))
						continue
					}
				} else if !os.IsNotExist(statErr) {
					return statErr
				}
			}
			archiveManaged = append(archiveManaged, rel)
			staged[key] = rel
		}
	}

	managed := map[string]string{}
	for _, rel := range archiveManaged {
		clean, pathErr := validRelativePath(filepath.ToSlash(rel))
		if pathErr != nil {
			return pathErr
		}
		if (!isProtectedPlayerFile(clean) || isPlayerPack(clean)) && managedArchivePath(clean) {
			managed[strings.ToLower(clean)] = clean
		}
	}
	filteredArchive := archiveManaged[:0]
	for _, rel := range archiveManaged {
		if managedArchivePath(rel) {
			filteredArchive = append(filteredArchive, rel)
		}
	}
	archiveManaged = filteredArchive
	scheduled := map[string]bool{}
	var tasks []minecraft.DownloadTask
	for _, file := range manifest.Files {
		rel, pathErr := validRelativePath(file.Path)
		if pathErr != nil {
			return pathErr
		}
		if isProtectedPlayerFile(rel) && !isPlayerPack(rel) {
			return fmt.Errorf("manifest không được thay tệp người chơi: %s", rel)
		}
		key := strings.ToLower(rel)
		if scheduled[key] {
			return fmt.Errorf("manifest có tệp trùng: %s", rel)
		}
		scheduled[key] = true
		managed[key] = rel
		stagePath := filepath.Join(stage, rel)
		target, pathErr := transactionPath(instancePath, rel)
		if pathErr != nil {
			return pathErr
		}
		if isPlayerPack(rel) {
			if _, statErr := os.Lstat(target); statErr == nil && staged[key] == "" {
				expected := previous.ManagedPackHashes[key]
				if (expected == "" || !verifySHA256(target, expected)) && (file.SHA1 == "" || !verifySHA1(target, file.SHA1)) {
					return fmt.Errorf("gói cá nhân trùng tên với tệp chính thức: %s", rel)
				}
			} else if statErr != nil && !os.IsNotExist(statErr) {
				return statErr
			}
		}
		if file.SHA1 != "" && (verifySHA1(stagePath, file.SHA1) || (staged[key] == "" && verifySHA1(target, file.SHA1))) {
			continue
		}
		_ = os.Remove(stagePath)
		tasks = append(tasks, minecraft.DownloadTask{URL: file.URL, Path: stagePath, SHA1: file.SHA1, Size: file.Size})
		staged[key] = rel
	}
	if len(tasks) > 0 {
		report(fmt.Sprintf("Đang tải %d tệp cập nhật...", len(tasks)), 30)
		fileProgress := func(event models.ProgressEvent) {
			event.Percentage = 70 + event.Percentage*0.15
			if progressFn != nil {
				progressFn(event)
			}
		}
		if err = minecraft.DownloadBatch(tasks, 8, manifest.ID, "syncing_modpack", fileProgress); err != nil {
			return fmt.Errorf("lỗi tải tệp: %w", err)
		}
	}
	if !archiveChanged && manifest.Archive != nil && len(previous.ManifestFiles) > 0 {
		archivePaths := make(map[string]bool, len(archiveManaged))
		for _, rel := range archiveManaged {
			archivePaths[strings.ToLower(rel)] = true
		}
		restore := make(map[string]bool)
		for _, rel := range previous.ManifestFiles {
			clean, pathErr := validRelativePath(filepath.ToSlash(rel))
			if pathErr != nil {
				return pathErr
			}
			key := strings.ToLower(clean)
			if !scheduled[key] && archivePaths[key] {
				restore[key] = true
			}
		}
		if len(restore) > 0 {
			if archivePath == "" {
				archivePath, err = archiveFile(manifest.Archive, archiveProgress)
				if err != nil {
					return err
				}
			}
			paths, extractErr := stageArchive(archivePath, stage, restore)
			if extractErr != nil {
				return extractErr
			}
			if len(paths) != len(restore) {
				return fmt.Errorf("archive thiếu tệp cần khôi phục")
			}
			for _, rel := range paths {
				staged[strings.ToLower(rel)] = rel
			}
		}
	}

	missing := map[string]string{}
	for _, name := range manifest.ExpectedMods {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if filepath.Base(name) != name || strings.ContainsAny(name, "\\/:") {
			return fmt.Errorf("tên mod không hợp lệ: %s", name)
		}
		rel := filepath.Join("mods", name)
		key := strings.ToLower(rel)
		managed[key] = rel
		if _, statErr := os.Stat(filepath.Join(stage, rel)); statErr == nil {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(instancePath, rel)); statErr == nil {
			if archiveChanged && !scheduled[key] {
				missing[key] = rel
				continue
			}
			if scheduled[key] {
				continue
			}
			if expected := previous.ManagedModHashes[key]; expected == "" || verifySHA256(filepath.Join(instancePath, rel), expected) {
				continue
			}
		}
		missing[key] = rel
	}
	if len(missing) > 0 && manifest.Archive != nil {
		if archivePath == "" {
			archivePath, err = archiveFile(manifest.Archive, archiveProgress)
			if err != nil {
				return err
			}
		}
		selected := make(map[string]bool, len(missing))
		for key := range missing {
			selected[key] = true
		}
		paths, extractErr := stageArchive(archivePath, stage, selected)
		if extractErr != nil {
			return extractErr
		}
		for _, rel := range paths {
			staged[strings.ToLower(rel)] = rel
		}
	}
	for _, rel := range missing {
		if _, statErr := os.Stat(filepath.Join(stage, rel)); statErr != nil {
			return fmt.Errorf("đồng bộ chưa hoàn tất, thiếu mod %s", rel)
		}
	}

	state := models.LocalSyncState{ServerID: manifest.ID, SyncedVersion: manifest.Version, SyncedAt: time.Now(), LegacyMigrated: previous.LegacyMigrated, ArchiveManagedFiles: archiveManaged, ManagedModHashes: map[string]string{}, ManagedPackHashes: map[string]string{}}
	for _, file := range manifest.Files {
		state.ManifestFiles = append(state.ManifestFiles, filepath.Clean(filepath.FromSlash(file.Path)))
	}
	for _, name := range manifest.ExpectedMods {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		rel := filepath.Join("mods", name)
		path := filepath.Join(instancePath, rel)
		if staged[strings.ToLower(rel)] != "" {
			path = filepath.Join(stage, rel)
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		hash := sha256.New()
		_, hashErr := io.Copy(hash, file)
		closeErr := file.Close()
		if hashErr != nil {
			return hashErr
		}
		if closeErr != nil {
			return closeErr
		}
		state.ManagedModHashes[strings.ToLower(rel)] = hex.EncodeToString(hash.Sum(nil))
	}
	for _, rel := range archiveManaged {
		if !isPlayerPack(rel) {
			continue
		}
		key := strings.ToLower(rel)
		if staged[key] == "" && previous.ManagedPackHashes[key] != "" {
			state.ManagedPackHashes[key] = previous.ManagedPackHashes[key]
			continue
		}
		path := filepath.Join(instancePath, rel)
		if staged[key] != "" {
			path = filepath.Join(stage, rel)
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		hash := sha256.New()
		_, hashErr := io.Copy(hash, file)
		closeErr := file.Close()
		if hashErr != nil {
			return hashErr
		}
		if closeErr != nil {
			return closeErr
		}
		state.ManagedPackHashes[key] = hex.EncodeToString(hash.Sum(nil))
	}
	for _, file := range manifest.Files {
		rel := filepath.Clean(filepath.FromSlash(file.Path))
		if !isPlayerPack(rel) {
			continue
		}
		key := strings.ToLower(rel)
		path := filepath.Join(instancePath, rel)
		if staged[key] != "" {
			path = filepath.Join(stage, rel)
		}
		content, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		hash := sha256.New()
		_, hashErr := io.Copy(hash, content)
		closeErr := content.Close()
		if hashErr != nil {
			return hashErr
		}
		if closeErr != nil {
			return closeErr
		}
		state.ManagedPackHashes[key] = hex.EncodeToString(hash.Sum(nil))
	}
	if manifest.Archive != nil {
		state.ArchiveSHA256 = manifest.Archive.SHA256
	}
	for _, rel := range managed {
		state.ManagedFiles = append(state.ManagedFiles, rel)
	}
	sort.Strings(state.ManagedFiles)
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(stage, "sync_state.json"), stateData, 0644); err != nil {
		return err
	}
	staged["sync_state.json"] = "sync_state.json"

	var removals []string
	for _, old := range previous.ManagedFiles {
		rel, pathErr := validRelativePath(filepath.ToSlash(old))
		if pathErr != nil {
			return pathErr
		}
		key := strings.ToLower(rel)
		canRemove := !isProtectedPlayerFile(rel)
		if strings.EqualFold(filepath.Dir(rel), "mods") && strings.EqualFold(filepath.Ext(rel), ".jar") &&
			previous.ManagedModHashes[key] != "" && !verifySHA256(filepath.Join(instancePath, rel), previous.ManagedModHashes[key]) {
			canRemove = false
		}
		if isPlayerPack(rel) && previous.ManagedPackHashes[key] != "" {
			canRemove = verifySHA256(filepath.Join(instancePath, rel), previous.ManagedPackHashes[key])
		}
		if canRemove && managed[key] == "" && staged[key] == "" {
			removals = append(removals, rel)
		}
	}
	sort.Strings(removals)
	var installs []string
	for key, rel := range staged {
		if key != "sync_state.json" {
			installs = append(installs, rel)
		}
	}
	sort.Strings(installs)
	installs = append(installs, "sync_state.json")
	journal := syncJournal{}
	prepare := func(rel string, install bool) error {
		target, err := transactionPath(instancePath, rel)
		if err != nil {
			return err
		}
		info, statErr := os.Lstat(target)
		if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		if statErr == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("tệp đích không hợp lệ: %s", rel)
		}
		if !install && os.IsNotExist(statErr) {
			return nil
		}
		journal.Entries = append(journal.Entries, journalEntry{Path: rel, HadOriginal: statErr == nil, Install: install})
		return nil
	}
	for _, rel := range removals {
		if err = prepare(rel, false); err != nil {
			return err
		}
	}
	for _, rel := range installs {
		if err = prepare(rel, true); err != nil {
			return err
		}
	}
	if err = saveJournal(journalPath, journal); err != nil {
		return err
	}
	report("Đang áp dụng bản cập nhật...", 90)
	apply := func(entry journalEntry) error {
		rel, install := entry.Path, entry.Install
		target, err := transactionPath(instancePath, rel)
		if err != nil {
			return err
		}
		if entry.HadOriginal {
			backup := filepath.Join(root, "backup", rel)
			if err := os.MkdirAll(filepath.Dir(backup), 0755); err != nil {
				return err
			}
			if err := os.Rename(target, backup); err != nil {
				return err
			}
		}
		if install {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Rename(filepath.Join(stage, rel), target); err != nil {
				return err
			}
		}
		return nil
	}
	for _, entry := range journal.Entries {
		if err = apply(entry); err != nil {
			return err
		}
	}
	journal.Committed = true
	if err = saveJournal(journalPath, journal); err != nil {
		return err
	}
	if progressFn != nil {
		progressFn(models.ProgressEvent{InstanceID: manifest.ID, Stage: "ready", Message: fmt.Sprintf("Đồng bộ máy chủ %s thành công!", manifest.Name), Percentage: 100})
	}
	return nil
}
