package minecraft

import (
	"Aether/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const MojangManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

// FetchVersionManifest retrieves all official Minecraft versions (releases & snapshots)
func FetchVersionManifest(cacheDir string) (*models.VersionManifest, error) {
	cacheFile := filepath.Join(cacheDir, "version_manifest_v2.json")

	// Try cached manifest if fresh (< 2 hours)
	if info, err := os.Stat(cacheFile); err == nil && time.Since(info.ModTime()) < 2*time.Hour {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var manifest models.VersionManifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				return &manifest, nil
			}
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(MojangManifestURL)
	if err != nil {
		// Fallback to cache if available even if older
		if data, errRead := os.ReadFile(cacheFile); errRead == nil {
			var manifest models.VersionManifest
			if errUnmarshal := json.Unmarshal(data, &manifest); errUnmarshal == nil {
				return &manifest, nil
			}
		}
		return nil, fmt.Errorf("không thể kết nối máy chủ Mojang: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest models.VersionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}

	// Cache to disk
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.WriteFile(cacheFile, body, 0644)

	return &manifest, nil
}

// FetchVersionDetail retrieves the full version specification (libraries, assets, client.jar)
func FetchVersionDetail(versionURL string, cacheFile string) (*models.VersionDetail, error) {
	// Check cache
	if data, err := os.ReadFile(cacheFile); err == nil {
		var detail models.VersionDetail
		if err := json.Unmarshal(data, &detail); err == nil {
			return &detail, nil
		}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(versionURL)
	if err != nil {
		return nil, fmt.Errorf("không thể tải thông tin phiên bản: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var detail models.VersionDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, err
	}

	// Save to cache
	_ = os.MkdirAll(filepath.Dir(cacheFile), 0755)
	_ = os.WriteFile(cacheFile, body, 0644)

	return &detail, nil
}
