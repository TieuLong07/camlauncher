package minecraft

import (
	"Aether/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GetLoaderVersions fetches available loader versions and indicates the recommended one
func GetLoaderVersions(gameVersion, loader string) (*models.LoaderVersionResult, error) {
	cleanLoader := strings.ToLower(strings.TrimSpace(loader))
	res := &models.LoaderVersionResult{
		Loader:      cleanLoader,
		GameVersion: gameVersion,
		Versions:    []string{},
	}

	if cleanLoader == "" || cleanLoader == "vanilla" {
		return res, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	switch cleanLoader {
	case "fabric":
		url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s", gameVersion)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("lỗi kết nối Fabric API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return res, nil // Fabric might not support this version
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var items []struct {
			Loader struct {
				Version string `json:"version"`
				Stable  bool   `json:"stable"`
			} `json:"loader"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, err
		}

		seen := make(map[string]bool)
		for _, it := range items {
			v := it.Loader.Version
			if v != "" && !seen[v] {
				seen[v] = true
				res.Versions = append(res.Versions, v)
				if res.Recommended == "" && it.Loader.Stable {
					res.Recommended = v
				}
			}
			if len(res.Versions) >= 40 {
				break
			}
		}
		if res.Recommended == "" && len(res.Versions) > 0 {
			res.Recommended = res.Versions[0]
		}

	case "forge":
		// 1. Fetch promotions (recommended & latest)
		promoURL := "https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json"
		promoReq, _ := http.NewRequest("GET", promoURL, nil)
		promoReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) CamLauncher/0.1.0-alpha")

		var recVersion string
		var latestVersion string

		if pResp, err := client.Do(promoReq); err == nil && pResp.StatusCode == http.StatusOK {
			if pBody, err := io.ReadAll(pResp.Body); err == nil {
				var promoData struct {
					Promos map[string]string `json:"promos"`
				}
				if json.Unmarshal(pBody, &promoData) == nil {
					recVersion = promoData.Promos[gameVersion+"-recommended"]
					latestVersion = promoData.Promos[gameVersion+"-latest"]
				}
			}
			pResp.Body.Close()
		}

		// 2. Fetch list of all Forge versions from BMCLAPI
		bmclURL := fmt.Sprintf("https://bmclapi2.bangbang93.com/forge/minecraft/%s", gameVersion)
		bmclReq, _ := http.NewRequest("GET", bmclURL, nil)
		bmclReq.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha")

		if bResp, err := client.Do(bmclReq); err == nil && bResp.StatusCode == http.StatusOK {
			if bBody, err := io.ReadAll(bResp.Body); err == nil {
				var bmclItems []struct {
					Version string `json:"version"`
				}
				if json.Unmarshal(bBody, &bmclItems) == nil {
					seen := make(map[string]bool)
					// Add recommended first if present
					if recVersion != "" {
						res.Versions = append(res.Versions, recVersion)
						seen[recVersion] = true
					}
					if latestVersion != "" && !seen[latestVersion] {
						res.Versions = append(res.Versions, latestVersion)
						seen[latestVersion] = true
					}

					for i := len(bmclItems) - 1; i >= 0; i-- {
						v := bmclItems[i].Version
						if v != "" && !seen[v] {
							seen[v] = true
							res.Versions = append(res.Versions, v)
						}
						if len(res.Versions) >= 40 {
							break
						}
					}
				}
			}
			bResp.Body.Close()
		}

		// Fallback if BMCLAPI list was empty
		if len(res.Versions) == 0 {
			if recVersion != "" {
				res.Versions = append(res.Versions, recVersion)
			}
			if latestVersion != "" && latestVersion != recVersion {
				res.Versions = append(res.Versions, latestVersion)
			}
		}

		if recVersion != "" {
			res.Recommended = recVersion
		} else if latestVersion != "" {
			res.Recommended = latestVersion
		} else if len(res.Versions) > 0 {
			res.Recommended = res.Versions[0]
		}

	case "neoforge":
		url := fmt.Sprintf("https://bmclapi2.bangbang93.com/neoforge/list/%s", gameVersion)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha")

		if resp, err := client.Do(req); err == nil && resp.StatusCode == http.StatusOK {
			if body, err := io.ReadAll(resp.Body); err == nil {
				var items []struct {
					Version string `json:"version"`
				}
				if json.Unmarshal(body, &items) == nil {
					seen := make(map[string]bool)
					for i := len(items) - 1; i >= 0; i-- {
						v := items[i].Version
						if v != "" && !seen[v] {
							seen[v] = true
							res.Versions = append(res.Versions, v)
						}
						if len(res.Versions) >= 40 {
							break
						}
					}
				}
			}
			resp.Body.Close()
		}

		if len(res.Versions) > 0 {
			res.Recommended = res.Versions[0]
		}
	}

	return res, nil
}
