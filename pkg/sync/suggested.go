package sync

import (
	"Aether/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// supersededFlashbackSHA1 lists client jars we published or shipped earlier and
// will replace in place. 14ee0d… is the upstream jar (crashes the client with a
// channel mismatch); 08af39… is a withdrawn patch whose ABSENT check referenced
// a field of the wrong type and threw NoSuchFieldError at class init.
var supersededFlashbackSHA1 = map[string]bool{
	"14ee0df1f23bfcb77c598de02c8bd29b4477f061": true,
	"08af39898f8c4ad4015c4ee081c9c6ddb8dc1261": true,
}

// isSupersededFlashback reports whether the jar on disk is one of the builds we
// are allowed to overwrite without asking the player to clean up manually.
func isSupersededFlashback(path, modID string) bool {
	if modID != "flashback" {
		return false
	}
	for sha1 := range supersededFlashbackSHA1 {
		if verifySHA1(path, sha1) {
			return true
		}
	}
	return false
}

// SuggestedMods is separate from the official manifest: nothing here is mandatory.
func SuggestedMods(serverID string) []models.SuggestedMod {
	if serverID != "kamaron" {
		return []models.SuggestedMod{}
	}
	return []models.SuggestedMod{{
		ID: "flashback", Name: "Flashback", Version: "1.0.1",
		Description: "Ghi lại phiên chơi, chỉnh góc máy và xuất video Minecraft.",
		FileName:    "flashback-1.0.1.jar",
		URL:         "https://github.com/TieuLong07/camlauncher/releases/download/kamaron-v1.0.0/flashback-1.0.1-forge-clientfix.jar",
		SHA1:        "ee6cef75a2561ec7ace681a241afaf06ba17c498", Size: 204244560,
	}}
}

func suggestedState(root string) (models.LocalSyncState, error) {
	var state models.LocalSyncState
	data, err := os.ReadFile(filepath.Join(root, "sync_state.json"))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func InspectSuggestedMods(root, serverID string) ([]models.SuggestedMod, error) {
	mods := SuggestedMods(serverID)
	if root == "" {
		return mods, nil
	}
	unlock, err := lockInstance(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := recoverTransaction(root); err != nil {
		return nil, err
	}
	state, err := suggestedState(root)
	if err != nil {
		return nil, err
	}
	for i := range mods {
		mod := &mods[i]
		path, err := transactionPath(root, filepath.Join("mods", mod.FileName))
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		mod.Installed = err == nil
		if mod.Installed {
			mod.Modified = !info.Mode().IsRegular() || !verifySHA1(path, mod.SHA1)
			if isSupersededFlashback(path, mod.ID) {
				mod.Modified = false
				mod.Managed = true
			}
		}
		for _, rel := range state.ManagedFiles {
			if strings.EqualFold(filepath.ToSlash(rel), "mods/"+mod.FileName) {
				mod.Managed = true
			}
		}
	}
	return mods, nil
}

func SetSuggestedMod(root, serverID, modID string, install bool) error {
	for _, mod := range SuggestedMods(serverID) {
		if mod.ID == modID {
			return setSuggestedMod(root, mod, install, &http.Client{Timeout: 10 * time.Minute})
		}
	}
	return fmt.Errorf("mod gợi ý không tồn tại")
}

func setSuggestedMod(root string, mod models.SuggestedMod, install bool, client *http.Client) error {
	unlock, err := lockInstance(root)
	if err != nil {
		return err
	}
	defer unlock()
	if err := recoverTransaction(root); err != nil {
		return err
	}
	path, err := transactionPath(root, filepath.Join("mods", mod.FileName))
	if err != nil {
		return err
	}
	info, statErr := os.Lstat(path)
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	staleFlashback := statErr == nil && info.Mode().IsRegular() && isSupersededFlashback(path, mod.ID)
	if statErr == nil && (!info.Mode().IsRegular() || (!verifySHA1(path, mod.SHA1) && !staleFlashback)) {
		return fmt.Errorf("%s trên máy đã thay đổi; hãy xử lý tệp này trong thư mục mods trước", mod.Name)
	}
	if install && (os.IsNotExist(statErr) || staleFlashback) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		resp, err := client.Get(mod.URL)
		if err != nil {
			return fmt.Errorf("không tải được %s: %w", mod.Name, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("không tải được %s (HTTP %d)", mod.Name, resp.StatusCode)
		}
		tmp, err := os.CreateTemp(filepath.Dir(path), ".cam-optional-*.part")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		n, copyErr := io.Copy(tmp, io.LimitReader(resp.Body, mod.Size+1))
		closeErr := tmp.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n != mod.Size || !verifySHA1(tmp.Name(), mod.SHA1) {
			return fmt.Errorf("tệp %s tải về không đúng kích thước hoặc mã băm; vui lòng thử lại", mod.Name)
		}
		if staleFlashback {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
		if err := os.Rename(tmp.Name(), path); err != nil {
			return err
		}
	}
	// Explicit opt-in adopts a former official mod as a personal mod. Otherwise
	// upgrading from Kamaron 1.0.3 would delete Flashback during the next sync.
	if err := releaseSuggestedMod(root, mod.FileName); err != nil {
		return err
	}
	if !install && statErr == nil {
		return os.Remove(path)
	}
	return nil
}

func releaseSuggestedMod(root, filename string) error {
	state, err := suggestedState(root)
	if err != nil {
		return err
	}
	rel := "mods/" + filename
	changed := false
	filter := func(paths []string) []string {
		result := paths[:0]
		for _, path := range paths {
			if strings.EqualFold(filepath.ToSlash(path), rel) {
				changed = true
			} else {
				result = append(result, path)
			}
		}
		return result
	}
	state.ManagedFiles = filter(state.ManagedFiles)
	state.ArchiveManagedFiles = filter(state.ArchiveManagedFiles)
	state.ManifestFiles = filter(state.ManifestFiles)
	for key := range state.ManagedModHashes {
		if strings.EqualFold(filepath.ToSlash(key), rel) {
			delete(state.ManagedModHashes, key)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(root, ".cam-optional-state-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp.Name(), filepath.Join(root, "sync_state.json"))
}
