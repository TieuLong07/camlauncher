package modrinth

import (
	"Aether/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ModrinthAPIBase = "https://api.modrinth.com/v2"

type SearchResponse struct {
	Hits      []Hit `json:"hits"`
	TotalHits int   `json:"total_hits"`
}

type Hit struct {
	ProjectID   string   `json:"project_id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	IconURL     string   `json:"icon_url"`
	Author      string   `json:"author"`
	Downloads   int      `json:"downloads"`
	Follows     int      `json:"follows"`
	Categories  []string `json:"categories"`
	Versions    []string `json:"versions"`
	ClientSide  string   `json:"client_side"`
	ServerSide  string   `json:"server_side"`
}

type ProjectVersion struct {
	ID           string        `json:"id"`
	ProjectID    string        `json:"project_id"`
	Name         string        `json:"name"`
	VersionNum   string        `json:"version_number"`
	GameVersions []string      `json:"game_versions"`
	Loaders      []string      `json:"loaders"`
	Files        []VersionFile `json:"files"`
}

type VersionFile struct {
	Hashes   map[string]string `json:"hashes"`
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int64             `json:"size"`
}

// SearchMods searches projects on Modrinth
func SearchMods(query string, gameVersion string, loader string, projectType string, limit int) (*models.ModrinthSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if projectType == "" {
		projectType = "mod"
	}

	var facets [][]string
	facets = append(facets, []string{fmt.Sprintf("project_type:%s", projectType)})
	if gameVersion != "" {
		facets = append(facets, []string{fmt.Sprintf("versions:%s", gameVersion)})
	}
	if loader != "" && loader != "vanilla" {
		facets = append(facets, []string{fmt.Sprintf("categories:%s", strings.ToLower(loader))})
	}

	facetsJSON, _ := json.Marshal(facets)

	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("facets", string(facetsJSON))

	reqURL := fmt.Sprintf("%s/search?%s", ModrinthAPIBase, params.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha (contact: hello@owlys.io.vn)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi kết nối Modrinth: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, err
	}

	result := &models.ModrinthSearchResult{
		TotalHits: searchResp.TotalHits,
	}

	for _, h := range searchResp.Hits {
		result.Hits = append(result.Hits, models.ModrinthProject{
			ProjectID:   h.ProjectID,
			Slug:        h.Slug,
			Title:       h.Title,
			Description: h.Description,
			IconURL:     h.IconURL,
			Author:      h.Author,
			Downloads:   h.Downloads,
			Follows:     h.Follows,
			Categories:  h.Categories,
			Versions:    h.Versions,
			ClientSide:  h.ClientSide,
			ServerSide:  h.ServerSide,
		})
	}

	return result, nil
}

// InstallMod installs a specific Modrinth mod into the instance's mods folder
func InstallMod(slugOrID, gameVersion, loader, instanceModsDir string) (string, error) {
	_ = os.MkdirAll(instanceModsDir, 0755)

	// Fetch project versions
	url := fmt.Sprintf("%s/project/%s/version?game_versions=[\"%s\"]&loaders=[\"%s\"]",
		ModrinthAPIBase, slugOrID, gameVersion, strings.ToLower(loader))

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("không tìm thấy phiên bản mod tương thích")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var versions []ProjectVersion
	if err := json.Unmarshal(body, &versions); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("mod %s không có bản cho %s (%s)", slugOrID, gameVersion, loader)
	}

	// Pick first compatible version file
	targetVersion := versions[0]
	var targetFile *VersionFile
	for i := range targetVersion.Files {
		if targetVersion.Files[i].Primary || strings.HasSuffix(targetVersion.Files[i].Filename, ".jar") {
			targetFile = &targetVersion.Files[i]
			break
		}
	}
	if targetFile == nil && len(targetVersion.Files) > 0 {
		targetFile = &targetVersion.Files[0]
	}
	if targetFile == nil {
		return "", fmt.Errorf("không có file jar để tải")
	}

	destPath := filepath.Join(instanceModsDir, targetFile.Filename)

	// Download file
	fileResp, err := client.Get(targetFile.URL)
	if err != nil {
		return "", err
	}
	defer fileResp.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, fileResp.Body)
	if err != nil {
		return "", err
	}

	return targetFile.Filename, nil
}

// TiCaFPSBoostMods defines the elite performance optimization bundle
var TiCaFPSBoostMods = []string{
	"sodium",          // Core modern graphics engine (300%+ FPS boost)
	"iris",            // Shaders loader compatible with Sodium
	"lithium",         // Game physics and chunk tick optimizer
	"ferrite-core",    // Reduces memory usage by up to 50%
	"immediatelyfast", // Optimizes HUD, text rendering, and immediate mode graphics
	"entityculling",   // Skips rendering tiles and entities hidden behind walls
}

// InstallFPSBoostPack installs all 6 performance mods at once
func InstallFPSBoostPack(gameVersion, loader, instanceModsDir string) ([]string, []string) {
	var installed []string
	var failed []string

	for _, slug := range TiCaFPSBoostMods {
		filename, err := InstallMod(slug, gameVersion, loader, instanceModsDir)
		if err == nil {
			installed = append(installed, filename)
		} else {
			failed = append(failed, fmt.Sprintf("%s (%v)", slug, err))
		}
	}

	return installed, failed
}
