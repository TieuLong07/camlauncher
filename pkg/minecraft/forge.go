package minecraft

import (
	"Aether/pkg/models"
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// GetRecommendedForgeVersion gets recommended Forge version for a Minecraft version
func GetRecommendedForgeVersion(gameVersion string) (string, error) {
	url := "https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("không thể lấy thông tin phiên bản Forge: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type PromotionsResponse struct {
		Promos map[string]string `json:"promos"`
	}
	var promos PromotionsResponse
	if err := json.Unmarshal(body, &promos); err != nil {
		return "", err
	}

	// Try recommended, then latest
	if v, ok := promos.Promos[gameVersion+"-recommended"]; ok && v != "" {
		return v, nil
	}
	if v, ok := promos.Promos[gameVersion+"-latest"]; ok && v != "" {
		return v, nil
	}

	return "", fmt.Errorf("không tìm thấy phiên bản Forge cho Minecraft %s", gameVersion)
}

// GetForgeVersionProfile gets or downloads the Forge version profile JSON
func GetForgeVersionProfile(sharedDir, gameVersion, forgeVersion string, progressFn func(models.ProgressEvent)) (*models.VersionDetail, error) {
	cacheKey := fmt.Sprintf("%s-forge-%s", gameVersion, forgeVersion)
	cacheDir := filepath.Join(sharedDir, "versions", cacheKey)
	cacheFile := filepath.Join(cacheDir, cacheKey+".json")

	// 1. Check local cache
	if data, err := os.ReadFile(cacheFile); err == nil {
		var detail models.VersionDetail
		if err := json.Unmarshal(data, &detail); err == nil && detail.MainClass != "" {
			return &detail, nil
		}
	}

	_ = os.MkdirAll(cacheDir, 0755)

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "checking_forge",
			Message:    fmt.Sprintf("Đang chuẩn bị Forge %s cho Minecraft %s...", forgeVersion, gameVersion),
			Percentage: 10,
		})
	}

	// 2. Download Installer jar to extract version.json
	installerURLs := []string{
		fmt.Sprintf("https://maven.minecraftforge.net/net/minecraftforge/forge/%s-%s/forge-%s-%s-installer.jar", gameVersion, forgeVersion, gameVersion, forgeVersion),
		fmt.Sprintf("https://bmclapi2.bangbang93.com/forge/download?mcversion=%s&version=%s&category=installer", gameVersion, forgeVersion),
	}

	var installerData []byte
	client := &http.Client{Timeout: 60 * time.Second}

	for _, u := range installerURLs {
		resp, err := client.Get(u)
		if err == nil && resp.StatusCode == http.StatusOK {
			b, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil && len(b) > 100000 {
				installerData = b
				break
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if len(installerData) == 0 {
		return nil, fmt.Errorf("không thể tải bộ cài Forge %s-%s", gameVersion, forgeVersion)
	}

	// 3. Extract version.json from installer zip
	zipReader, err := zip.NewReader(strings.NewReader(string(installerData)), int64(len(installerData)))
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc file nén bộ cài Forge: %w", err)
	}

	var versionJSONData []byte
	for _, f := range zipReader.File {
		if f.Name == "version.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("không thể mở version.json trong Forge installer: %w", err)
			}
			versionJSONData, _ = io.ReadAll(rc)
			rc.Close()
			break
		}
	}

	if len(versionJSONData) == 0 {
		return nil, fmt.Errorf("không tìm thấy version.json trong bộ cài Forge")
	}

	// Cache to disk
	_ = os.WriteFile(cacheFile, versionJSONData, 0644)

	var detail models.VersionDetail
	if err := json.Unmarshal(versionJSONData, &detail); err != nil {
		return nil, fmt.Errorf("lỗi phân tích version.json của Forge: %w", err)
	}

	return &detail, nil
}

// FindInstanceVersionProfile searches an instance directory for an existing version profile JSON
func FindInstanceVersionProfile(instanceDir string) (*models.VersionDetail, string, error) {
	entries, err := os.ReadDir(instanceDir)
	if err != nil {
		return nil, "", err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		nameLower := strings.ToLower(entry.Name())
		if nameLower == "ticalauncher_instance.json" ||
			nameLower == "minecraftinstance.json" ||
			nameLower == "options.txt" ||
			nameLower == "servers.dat" ||
			nameLower == "usercache.json" ||
			nameLower == "usernamecache.json" ||
			strings.HasPrefix(nameLower, "patchouli_") ||
			strings.HasPrefix(nameLower, "imgui") {
			continue
		}

		filePath := filepath.Join(instanceDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var detail models.VersionDetail
		if err := json.Unmarshal(data, &detail); err == nil {
			// A valid version profile must have mainClass and (libraries or arguments or downloads)
			if detail.MainClass != "" && (len(detail.Libraries) > 0 || detail.Arguments != nil || len(detail.Downloads) > 0) {
				return &detail, filePath, nil
			}
		}
	}

	// Also check versions/ subfolder if present (e.g. .minecraft/versions/<name>/<name>.json)
	versionsDir := filepath.Join(instanceDir, "versions")
	if vEntries, err := os.ReadDir(versionsDir); err == nil {
		for _, vEntry := range vEntries {
			if !vEntry.IsDir() {
				continue
			}
			subDir := filepath.Join(versionsDir, vEntry.Name())
			jsonFiles, _ := filepath.Glob(filepath.Join(subDir, "*.json"))
			for _, jf := range jsonFiles {
				data, err := os.ReadFile(jf)
				if err != nil {
					continue
				}
				var detail models.VersionDetail
				if err := json.Unmarshal(data, &detail); err == nil {
					if detail.MainClass != "" && (len(detail.Libraries) > 0 || detail.Arguments != nil) {
						return &detail, jf, nil
					}
				}
			}
		}
	}

	return nil, "", nil
}

// ExtractForgeVersion extracts the Forge version string from an instance or its VersionDetail
func ExtractForgeVersion(inst *models.Instance, detail *models.VersionDetail) string {
	if inst != nil && inst.LoaderVersion != "" {
		return inst.LoaderVersion
	}

	if detail != nil {
		if detail.Arguments != nil {
			for i, arg := range detail.Arguments.Game {
				if s, ok := arg.(string); ok && s == "--fml.forgeVersion" && i+1 < len(detail.Arguments.Game) {
					if v, ok := detail.Arguments.Game[i+1].(string); ok && v != "" {
						return v
					}
				}
			}
		}

		for _, lib := range detail.Libraries {
			if strings.HasPrefix(lib.Name, "net.minecraftforge:forge:") || strings.HasPrefix(lib.Name, "net.minecraftforge:fmlloader:") {
				parts := strings.Split(lib.Name, ":")
				if len(parts) >= 3 {
					verPart := parts[2]
					subParts := strings.Split(verPart, "-")
					if len(subParts) >= 2 {
						return subParts[1]
					}
					return verPart
				}
			}
		}
	}

	return ""
}

// EnsureForgeClientInstalled ensures that Forge client-extra / forge-client libraries are generated
func EnsureForgeClientInstalled(sharedDir, javaExe, gameVersion, forgeVersion string, progressFn func(models.ProgressEvent)) error {
	if forgeVersion == "" {
		var err error
		forgeVersion, err = GetRecommendedForgeVersion(gameVersion)
		if err != nil || forgeVersion == "" {
			return nil
		}
	}

	targetJar := filepath.Join(sharedDir, "libraries", "net", "minecraftforge", "forge",
		fmt.Sprintf("%s-%s", gameVersion, forgeVersion),
		fmt.Sprintf("forge-%s-%s-client.jar", gameVersion, forgeVersion))

	if _, err := os.Stat(targetJar); err == nil {
		return nil // Already installed and present
	}

	// Also check if any client-*-srg.jar exists in libraries/net/minecraft/client
	srgPattern := filepath.Join(sharedDir, "libraries", "net", "minecraft", "client", fmt.Sprintf("%s-*", gameVersion), "*-srg.jar")
	matches, _ := filepath.Glob(srgPattern)
	if len(matches) > 0 {
		return nil
	}

	if progressFn != nil {
		progressFn(models.ProgressEvent{
			Stage:      "installing_forge",
			Message:    fmt.Sprintf("Đang hoàn tất cài đặt Forge %s (chỉ chạy 1 lần)...", forgeVersion),
			Percentage: 25,
		})
	}

	// 1. Ensure launcher_profiles.json exists (Forge installer requires this file)
	lpPath := filepath.Join(sharedDir, "launcher_profiles.json")
	if _, err := os.Stat(lpPath); os.IsNotExist(err) {
		_ = os.WriteFile(lpPath, []byte(`{"profiles":{}}`), 0644)
	}

	// 2. Download Installer jar to cache
	cacheDir := filepath.Join(sharedDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	installerPath := filepath.Join(cacheDir, fmt.Sprintf("forge-%s-%s-installer.jar", gameVersion, forgeVersion))

	fi, err := os.Stat(installerPath)
	if err != nil || fi.Size() < 1000000 {
		urls := []string{
			fmt.Sprintf("https://maven.minecraftforge.net/net/minecraftforge/forge/%s-%s/forge-%s-%s-installer.jar", gameVersion, forgeVersion, gameVersion, forgeVersion),
			fmt.Sprintf("https://bmclapi2.bangbang93.com/forge/download?mcversion=%s&version=%s&category=installer", gameVersion, forgeVersion),
		}

		var downloaded bool
		client := &http.Client{Timeout: 90 * time.Second}
		for _, u := range urls {
			resp, err := client.Get(u)
			if err == nil && resp.StatusCode == http.StatusOK {
				f, createErr := os.Create(installerPath)
				if createErr == nil {
					_, copyErr := io.Copy(f, resp.Body)
					_ = f.Close()
					resp.Body.Close()
					if copyErr == nil {
						downloaded = true
						break
					}
				}
			}
			if resp != nil {
				resp.Body.Close()
			}
		}

		if !downloaded {
			return fmt.Errorf("không thể tải bộ cài Forge %s-%s để giải nén thư viện", gameVersion, forgeVersion)
		}
	}

	// 3. Run Forge installer headless
	installerCmd := exec.Command(javaExe, "-jar", installerPath, "--installClient", sharedDir)
	if runtime.GOOS == "windows" {
		installerCmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}

	_ = installerCmd.Run()

	// Verify if the client jar or srg jar was generated
	if _, err := os.Stat(targetJar); err == nil {
		return nil
	}
	matches, _ = filepath.Glob(srgPattern)
	if len(matches) > 0 {
		return nil
	}

	return nil
}
