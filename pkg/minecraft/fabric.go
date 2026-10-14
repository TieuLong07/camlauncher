package minecraft

import (
	"Aether/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FabricLoaderInfo struct {
	Loader struct {
		Separator string `json:"separator"`
		Build     int    `json:"build"`
		Maven     string `json:"maven"`
		Version   string `json:"version"`
		Stable    bool   `json:"stable"`
	} `json:"loader"`
	Intermediary struct {
		Maven   string `json:"maven"`
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"intermediary"`
}

// GetLatestFabricLoader retrieves the latest compatible Fabric loader for a Minecraft version
func GetLatestFabricLoader(gameVersion string) (string, error) {
	url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s", gameVersion)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("không có Fabric cho phiên bản %s", gameVersion)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var loaders []FabricLoaderInfo
	if err := json.Unmarshal(body, &loaders); err != nil {
		return "", err
	}

	if len(loaders) == 0 {
		return "", fmt.Errorf("không tìm thấy phiên bản Fabric loader phù hợp")
	}

	// First one is usually latest
	return loaders[0].Loader.Version, nil
}

// FetchFabricVersionProfile gets the full Fabric version detail (custom mainClass and libraries)
func FetchFabricVersionProfile(gameVersion, loaderVersion string) (*models.VersionDetail, error) {
	url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/%s/profile/json", gameVersion, loaderVersion)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
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

	return &detail, nil
}
