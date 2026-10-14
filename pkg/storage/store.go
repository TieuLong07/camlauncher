package storage

import (
	"Aether/pkg/models"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store struct {
	mu               sync.RWMutex
	LauncherDir      string
	Accounts         []models.Account              `json:"accounts"`
	Instances        []models.Instance             `json:"instances"`
	CommunityServers []models.CommunityServerEntry `json:"communityServers"`
	Settings         models.AppSettings            `json:"settings"`
}

func NewStore() *Store {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	launcherDir := filepath.Join(appData, "CamLauncher")
	_ = os.MkdirAll(launcherDir, 0755)

	store := &Store{
		LauncherDir:      launcherDir,
		Accounts:         []models.Account{},
		Instances:        []models.Instance{},
		CommunityServers: []models.CommunityServerEntry{},
		Settings: models.AppSettings{
			LauncherDir:       launcherDir,
			DefaultMinRAM:     2048,
			DefaultMaxRAM:     4096,
			DefaultJVMArgs:    "",
			AutoCloseOnLaunch: false,
			Language:          "vi",
			Theme:             "dark",
		},
	}

	store.Load()
	store.ensureDefaults()
	store.MigrateInstanceFolderNames()
	store.ScanUnregisteredInstances()

	return store
}

func (s *Store) getStoreFilePath() string {
	return filepath.Join(s.LauncherDir, "camlauncher_data.json")
}

func (s *Store) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.getStoreFilePath())
	if err != nil {
		return
	}

	type StoredData struct {
		Accounts         []models.Account              `json:"accounts"`
		Instances        []models.Instance             `json:"instances"`
		CommunityServers []models.CommunityServerEntry `json:"communityServers"`
		Settings         models.AppSettings            `json:"settings"`
	}

	var stored StoredData
	if err := json.Unmarshal(data, &stored); err == nil {
		if len(stored.Accounts) > 0 {
			s.Accounts = stored.Accounts
		}
		if len(stored.Instances) > 0 {
			s.Instances = stored.Instances
		}
		if len(stored.CommunityServers) > 0 {
			s.CommunityServers = stored.CommunityServers
		}
		if stored.Settings.LauncherDir != "" {
			s.Settings = stored.Settings
		}
	}
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type StoredData struct {
		Accounts         []models.Account              `json:"accounts"`
		Instances        []models.Instance             `json:"instances"`
		CommunityServers []models.CommunityServerEntry `json:"communityServers"`
		Settings         models.AppSettings            `json:"settings"`
	}

	stored := StoredData{
		Accounts:         s.Accounts,
		Instances:        s.Instances,
		CommunityServers: s.CommunityServers,
		Settings:         s.Settings,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.getStoreFilePath(), data, 0644)
}

func (s *Store) ensureDefaults() {
	// If no accounts, add a default offline account
	if len(s.Accounts) == 0 {
		s.Accounts = append(s.Accounts, models.Account{
			ID:          uuid.New().String(),
			Username:    "CamPlayer",
			Type:        models.AccountOffline,
			UUID:        "e96fbb52-f542-3e54-9442-7a4cb8c1e285",
			AccessToken: "offline-token",
			SkinURL:     "https://minotar.net/skin/CamPlayer",
			IsActive:    true,
			CreatedAt:   time.Now(),
		})
	}

	// If no instances, add recommended presets
	if len(s.Instances) == 0 {
		inst1ID := uuid.New().String()
		inst2ID := uuid.New().String()

		s.Instances = append(s.Instances,
			models.Instance{
				ID:              inst1ID,
				Name:            "TiCa Boosted 1.21.1",
				GameVersion:     "1.21.1",
				VersionType:     "release",
				Loader:          "fabric",
				LoaderVersion:   "0.16.5",
				Icon:            "emerald",
				MinMemoryMB:     2048,
				MaxMemoryMB:     4096,
				Path:            filepath.Join(s.LauncherDir, "instances", "TiCa Boosted 1.21.1"),
				CreatedAt:       time.Now(),
				LastPlayed:      time.Now(),
				PlayTimeMinutes: 0,
			},
			models.Instance{
				ID:              inst2ID,
				Name:            "Vanilla 1.21.1",
				GameVersion:     "1.21.1",
				VersionType:     "release",
				Loader:          "vanilla",
				Icon:            "pickaxe",
				MinMemoryMB:     2048,
				MaxMemoryMB:     4096,
				Path:            filepath.Join(s.LauncherDir, "instances", "Vanilla 1.21.1"),
				CreatedAt:       time.Now(),
				LastPlayed:      time.Now(),
				PlayTimeMinutes: 0,
			},
		)
	}

	// If no community servers configured, create the default Battle Royale server
	if len(s.CommunityServers) == 0 {
		sampleManifestPath := filepath.Join(s.LauncherDir, "sample_battle_royale.json")
		createSampleServerManifest(sampleManifestPath)

		s.CommunityServers = append(s.CommunityServers, models.CommunityServerEntry{
			ID:          "tica-battle-royale",
			Name:        "TiCa Battle Royale",
			ManifestURL: "file://" + filepath.ToSlash(sampleManifestPath),
			Tag:         "BATTLE ROYALE • SÚNG 3D",
			IsFeatured:  true,
		})
	}

	_ = s.Save()
}

// Account methods
func (s *Store) GetAccounts() []models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Account{}, s.Accounts...)
}

func (s *Store) GetActiveAccount() *models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.Accounts {
		if s.Accounts[i].IsActive {
			acc := s.Accounts[i]
			return &acc
		}
	}
	if len(s.Accounts) > 0 {
		acc := s.Accounts[0]
		return &acc
	}
	return nil
}

func (s *Store) SetActiveAccount(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Accounts {
		s.Accounts[i].IsActive = (s.Accounts[i].ID == id)
	}
	_ = s.saveLocked()
}

func (s *Store) AddAccount(account models.Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// deactivate others if this is set active
	if account.IsActive {
		for i := range s.Accounts {
			s.Accounts[i].IsActive = false
		}
	}
	s.Accounts = append(s.Accounts, account)
	_ = s.saveLocked()
}

func (s *Store) UpdateAccount(account models.Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Accounts {
		if s.Accounts[i].ID == account.ID {
			s.Accounts[i] = account
			break
		}
	}
	_ = s.saveLocked()
}

func (s *Store) RemoveAccount(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var updated []models.Account
	for _, acc := range s.Accounts {
		if acc.ID != id {
			updated = append(updated, acc)
		}
	}
	s.Accounts = updated
	if len(s.Accounts) > 0 {
		hasActive := false
		for _, acc := range s.Accounts {
			if acc.IsActive {
				hasActive = true
				break
			}
		}
		if !hasActive {
			s.Accounts[0].IsActive = true
		}
	}
	_ = s.saveLocked()
}

// SanitizeFolderName cleans a string to be a safe, valid folder name on Windows/Linux/macOS
func SanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Instance"
	}

	// Windows invalid folder characters: \ / : * ? " < > |
	invalidChars := regexp.MustCompile(`[\\/:*?"<>|]`)
	cleaned := invalidChars.ReplaceAllString(name, "_")

	// Strip non-printable ASCII control characters (0-31)
	controlChars := regexp.MustCompile(`[\x00-\x1f]`)
	cleaned = controlChars.ReplaceAllLiteralString(cleaned, "")

	// Windows cannot end folder names with spaces or dots
	cleaned = strings.TrimRight(cleaned, ". ")

	if cleaned == "" {
		cleaned = "Instance"
	}

	// Check reserved Windows device names
	upper := strings.ToUpper(cleaned)
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
		"COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
		"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
	if reserved[upper] {
		cleaned = cleaned + "_inst"
	}

	// Limit length to avoid path length limits
	if len(cleaned) > 64 {
		cleaned = strings.TrimRight(cleaned[:64], ". ")
	}

	return cleaned
}

// GetUniqueInstancePath returns a unique directory path for an instance under LauncherDir/instances
func (s *Store) GetUniqueInstancePath(name, instanceID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getUniqueInstancePathLocked(name, instanceID)
}

func (s *Store) getUniqueInstancePathLocked(name, instanceID string) string {
	baseFolder := SanitizeFolderName(name)
	instancesDir := filepath.Join(s.LauncherDir, "instances")
	_ = os.MkdirAll(instancesDir, 0755)

	candidateName := baseFolder
	counter := 1

	for {
		candidatePath := filepath.Join(instancesDir, candidateName)

		// Check if any other instance in memory already uses this path
		inUseByOther := false
		for _, inst := range s.Instances {
			if inst.ID != instanceID && filepath.Clean(inst.Path) == filepath.Clean(candidatePath) {
				inUseByOther = true
				break
			}
		}

		if !inUseByOther {
			// Check if file/folder exists on disk
			info, err := os.Stat(candidatePath)
			if os.IsNotExist(err) {
				return candidatePath
			}
			if err == nil && info.IsDir() {
				// If this candidatePath is already the folder of current instanceID, it is valid to reuse
				var isCurrentInstancePath bool
				for _, inst := range s.Instances {
					if inst.ID == instanceID && filepath.Clean(inst.Path) == filepath.Clean(candidatePath) {
						isCurrentInstancePath = true
						break
					}
				}
				if isCurrentInstancePath {
					return candidatePath
				}
			}
		}

		counter++
		candidateName = fmt.Sprintf("%s (%d)", baseFolder, counter)
	}
}

// MigrateInstanceFolderNames converts older UUID-based instance folders to clean, human-readable folders
// matching the instance names. Runs once on startup.
func (s *Store) MigrateInstanceFolderNames() {
	s.mu.Lock()
	defer s.mu.Unlock()

	hasChanges := false
	for i := range s.Instances {
		inst := &s.Instances[i]
		if inst.Name == "" {
			continue
		}

		currentFolder := filepath.Base(inst.Path)
		// Check if currentFolder is a UUID or empty
		_, err := uuid.Parse(currentFolder)
		isUUID := (err == nil)

		if isUUID || inst.Path == "" {
			targetPath := s.getUniqueInstancePathLocked(inst.Name, inst.ID)
			if inst.Path != "" && filepath.Clean(inst.Path) != filepath.Clean(targetPath) {
				// If old folder exists on disk, rename it
				if info, err := os.Stat(inst.Path); err == nil && info.IsDir() {
					// Check if target doesn't exist
					if _, err := os.Stat(targetPath); os.IsNotExist(err) {
						if err := os.Rename(inst.Path, targetPath); err == nil {
							inst.Path = targetPath
							hasChanges = true
							continue
						}
					}
				}
			}
			if inst.Path != targetPath {
				inst.Path = targetPath
				hasChanges = true
			}
		}
	}

	if hasChanges {
		_ = s.saveLocked()
	}
}

// ScanUnregisteredInstances scans LauncherDir/instances for any folders not yet registered
// in the store, automatically detects their game version and modloader, standardizes their folders,
// and adds them to the store. It also cleans up instances whose folders were removed from disk.
func (s *Store) ScanUnregisteredInstances() []models.Instance {
	instancesDir := filepath.Join(s.LauncherDir, "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hasChanges := false

	// Step 1: Clean up any instances whose folders were removed from disk
	var validInstances []models.Instance
	for _, inst := range s.Instances {
		if inst.Path != "" {
			if info, err := os.Stat(inst.Path); err == nil && info.IsDir() {
				validInstances = append(validInstances, inst)
			} else {
				hasChanges = true
			}
		} else {
			validInstances = append(validInstances, inst)
		}
	}
	s.Instances = validInstances

	// Step 2: Discover and register untracked folders
	var newlyAdded []models.Instance
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folderName := entry.Name()
		if strings.HasPrefix(folderName, ".") {
			continue
		}

		folderPath := filepath.Join(instancesDir, folderName)

		alreadyTracked := false
		for _, inst := range s.Instances {
			if filepath.Clean(inst.Path) == filepath.Clean(folderPath) ||
				strings.EqualFold(filepath.Base(inst.Path), folderName) {
				alreadyTracked = true
				break
			}
		}

		if !alreadyTracked {
			newInstance := s.detectAndNormalizeInstance(folderPath, folderName)
			s.Instances = append(s.Instances, newInstance)
			newlyAdded = append(newlyAdded, newInstance)
			hasChanges = true
		}
	}

	if hasChanges {
		_ = s.saveLocked()
	}

	return newlyAdded
}

// detectAndNormalizeInstance analyzes an untracked directory in instances/ and creates a normalized models.Instance
func (s *Store) detectAndNormalizeInstance(folderPath, folderName string) models.Instance {
	inst := models.Instance{
		ID:          uuid.New().String(),
		Name:        folderName,
		GameVersion: "1.21.1",
		VersionType: "release",
		Loader:      "vanilla",
		Icon:        "cube",
		MinMemoryMB: 2048,
		MaxMemoryMB: 4096,
		Path:        folderPath,
		CreatedAt:   time.Now(),
		LastPlayed:  time.Now(),
	}

	// 1. Check if our own ticalauncher_instance.json exists
	metaFile := filepath.Join(folderPath, "ticalauncher_instance.json")
	if data, err := os.ReadFile(metaFile); err == nil {
		var meta models.Instance
		if err := json.Unmarshal(data, &meta); err == nil {
			if meta.Name != "" {
				inst.Name = meta.Name
			}
			if meta.GameVersion != "" {
				inst.GameVersion = meta.GameVersion
			}
			if meta.Loader != "" {
				inst.Loader = meta.Loader
			}
			if meta.LoaderVersion != "" {
				inst.LoaderVersion = meta.LoaderVersion
			}
			if meta.Icon != "" {
				inst.Icon = meta.Icon
			}
			if meta.MinMemoryMB > 0 {
				inst.MinMemoryMB = meta.MinMemoryMB
			}
			if meta.MaxMemoryMB > 0 {
				inst.MaxMemoryMB = meta.MaxMemoryMB
			}
			if meta.JVMArgs != "" {
				inst.JVMArgs = meta.JVMArgs
			}
			if meta.JavaPath != "" {
				inst.JavaPath = meta.JavaPath
			}
			inst.Path = folderPath
			s.standardizeInstanceFolder(folderPath, inst)
			return inst
		}
	}

	// 2. Check CurseForge / Modpack manifest: minecraftinstance.json
	curseFile := filepath.Join(folderPath, "minecraftinstance.json")
	if data, err := os.ReadFile(curseFile); err == nil {
		type CurseModLoader struct {
			Name         string `json:"name"`
			Type         int    `json:"type"` // 1=Forge, 4=Fabric, 5=Quilt, 6=NeoForge
			ForgeVersion string `json:"forgeVersion"`
		}
		type CurseInstance struct {
			Name               string         `json:"name"`
			GameVersion        string         `json:"gameVersion"`
			BaseModLoader      CurseModLoader `json:"baseModLoader"`
			InstalledModLoader CurseModLoader `json:"installedModLoader"`
			AllocatedMemory    int            `json:"allocatedMemory"`
		}
		var curse CurseInstance
		if err := json.Unmarshal(data, &curse); err == nil {
			if strings.TrimSpace(curse.Name) != "" {
				inst.Name = strings.TrimSpace(curse.Name)
			}
			if strings.TrimSpace(curse.GameVersion) != "" {
				inst.GameVersion = strings.TrimSpace(curse.GameVersion)
			}

			loaderRaw := curse.InstalledModLoader.Name
			if loaderRaw == "" {
				loaderRaw = curse.BaseModLoader.Name
			}
			loaderType := curse.BaseModLoader.Type
			if loaderType == 0 {
				loaderType = curse.InstalledModLoader.Type
			}

			loaderLower := strings.ToLower(loaderRaw)
			if strings.Contains(loaderLower, "neoforge") || loaderType == 6 {
				inst.Loader = "neoforge"
				inst.Icon = "zap"
				inst.LoaderVersion = extractLoaderVersion(loaderRaw, "neoforge")
			} else if strings.Contains(loaderLower, "forge") || loaderType == 1 {
				inst.Loader = "forge"
				inst.Icon = "skull"
				inst.LoaderVersion = curse.BaseModLoader.ForgeVersion
				if inst.LoaderVersion == "" {
					inst.LoaderVersion = extractLoaderVersion(loaderRaw, "forge")
				}
			} else if strings.Contains(loaderLower, "fabric") || loaderType == 4 {
				inst.Loader = "fabric"
				inst.Icon = "flask"
				inst.LoaderVersion = extractLoaderVersion(loaderRaw, "fabric")
			} else if strings.Contains(loaderLower, "quilt") || loaderType == 5 {
				inst.Loader = "quilt"
				inst.Icon = "sparkles"
				inst.LoaderVersion = extractLoaderVersion(loaderRaw, "quilt")
			}

			if curse.AllocatedMemory >= 2048 {
				inst.MaxMemoryMB = curse.AllocatedMemory
			}
		}
	}

	// 3. Check MultiMC / Prism: mmc-pack.json or instance.cfg
	mmcFile := filepath.Join(folderPath, "mmc-pack.json")
	if data, err := os.ReadFile(mmcFile); err == nil {
		type MMCComponent struct {
			CachedVersion string `json:"cachedVersion"`
			Version       string `json:"version"`
			UID           string `json:"uid"`
		}
		type MMCPack struct {
			Components []MMCComponent `json:"components"`
		}
		var mmc MMCPack
		if err := json.Unmarshal(data, &mmc); err == nil {
			for _, comp := range mmc.Components {
				ver := comp.Version
				if ver == "" {
					ver = comp.CachedVersion
				}
				switch comp.UID {
				case "net.minecraft":
					if ver != "" {
						inst.GameVersion = ver
					}
				case "net.fabricmc.fabric-loader":
					inst.Loader = "fabric"
					inst.LoaderVersion = ver
					inst.Icon = "flask"
				case "net.minecraftforge":
					inst.Loader = "forge"
					inst.LoaderVersion = ver
					inst.Icon = "skull"
				case "net.neoforged":
					inst.Loader = "neoforge"
					inst.LoaderVersion = ver
					inst.Icon = "zap"
				case "org.quiltmc.quilt-loader":
					inst.Loader = "quilt"
					inst.LoaderVersion = ver
					inst.Icon = "sparkles"
				}
			}
		}
	}

	cfgFile := filepath.Join(folderPath, "instance.cfg")
	if f, err := os.Open(cfgFile); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "name=") {
				val := strings.TrimPrefix(line, "name=")
				if val != "" {
					inst.Name = val
				}
			}
		}
		_ = f.Close()
	}

	// 4. Check Modrinth: modrinth.index.json
	modrinthFile := filepath.Join(folderPath, "modrinth.index.json")
	if data, err := os.ReadFile(modrinthFile); err == nil {
		type ModrinthIndex struct {
			Game         string            `json:"game"`
			Name         string            `json:"name"`
			Dependencies map[string]string `json:"dependencies"`
		}
		var mIdx ModrinthIndex
		if err := json.Unmarshal(data, &mIdx); err == nil {
			if mIdx.Name != "" {
				inst.Name = mIdx.Name
			}
			if mcVer, ok := mIdx.Dependencies["minecraft"]; ok && mcVer != "" {
				inst.GameVersion = mcVer
			}
			if fabVer, ok := mIdx.Dependencies["fabric-loader"]; ok && fabVer != "" {
				inst.Loader = "fabric"
				inst.LoaderVersion = fabVer
				inst.Icon = "flask"
			} else if forgeVer, ok := mIdx.Dependencies["forge"]; ok && forgeVer != "" {
				inst.Loader = "forge"
				inst.LoaderVersion = forgeVer
				inst.Icon = "skull"
			} else if neoVer, ok := mIdx.Dependencies["neoforge"]; ok && neoVer != "" {
				inst.Loader = "neoforge"
				inst.LoaderVersion = neoVer
				inst.Icon = "zap"
			} else if quiltVer, ok := mIdx.Dependencies["quilt-loader"]; ok && quiltVer != "" {
				inst.Loader = "quilt"
				inst.LoaderVersion = quiltVer
				inst.Icon = "sparkles"
			}
		}
	}

	// 5. Check version json in root or versions/
	rootEntries, _ := os.ReadDir(folderPath)
	for _, entry := range rootEntries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			nameLower := strings.ToLower(entry.Name())
			if nameLower == "minecraftinstance.json" || nameLower == "modrinth.index.json" || nameLower == "ticalauncher_instance.json" {
				continue
			}
			vPath := filepath.Join(folderPath, entry.Name())
			if vData, err := os.ReadFile(vPath); err == nil {
				type SimpleVersionInfo struct {
					ID           string `json:"id"`
					MainClass    string `json:"mainClass"`
					InheritsFrom string `json:"inheritsFrom"`
				}
				var vi SimpleVersionInfo
				if err := json.Unmarshal(vData, &vi); err == nil && (vi.MainClass != "" || vi.InheritsFrom != "") {
					if inst.GameVersion == "1.21.1" && vi.InheritsFrom != "" {
						if m := regexp.MustCompile(`1\.\d+(\.\d+)?`).FindString(vi.InheritsFrom); m != "" {
							inst.GameVersion = m
						}
					}
					mainLower := strings.ToLower(vi.MainClass)
					if strings.Contains(mainLower, "bootstraplauncher") || strings.Contains(mainLower, "fml") || strings.Contains(mainLower, "forge") {
						inst.Loader = "forge"
						inst.Icon = "skull"
					} else if strings.Contains(mainLower, "fabric") || strings.Contains(mainLower, "knot") {
						inst.Loader = "fabric"
						inst.Icon = "flask"
					}
				}
			}
		}
	}

	// 6. Inspect mods/ folder
	modsDir := filepath.Join(folderPath, "mods")
	if modEntries, err := os.ReadDir(modsDir); err == nil && len(modEntries) > 0 {
		jarCount := 0
		fabricCount := 0
		forgeCount := 0
		neoforgeCount := 0
		versionCounts := make(map[string]int)

		verRegex := regexp.MustCompile(`1\.\d+(\.\d+)?`)

		for _, m := range modEntries {
			if !m.IsDir() && strings.HasSuffix(strings.ToLower(m.Name()), ".jar") {
				jarCount++
				mLower := strings.ToLower(m.Name())

				if strings.Contains(mLower, "fabric") {
					fabricCount++
				}
				if strings.Contains(mLower, "forge") && !strings.Contains(mLower, "neoforge") {
					forgeCount++
				}
				if strings.Contains(mLower, "neoforge") {
					neoforgeCount++
				}

				if matches := verRegex.FindString(m.Name()); matches != "" {
					versionCounts[matches]++
				}
			}
		}

		if jarCount > 0 {
			// Large modpack memory recommendation
			if jarCount > 40 && inst.MaxMemoryMB < 6144 {
				inst.MaxMemoryMB = 6144
			}

			// If loader is still vanilla, infer from mod files
			if inst.Loader == "vanilla" {
				if neoforgeCount > forgeCount && neoforgeCount > fabricCount {
					inst.Loader = "neoforge"
					inst.Icon = "zap"
				} else if forgeCount >= fabricCount && forgeCount > 0 {
					inst.Loader = "forge"
					inst.Icon = "skull"
				} else if fabricCount > 0 {
					inst.Loader = "fabric"
					inst.Icon = "flask"
				} else {
					inst.Loader = "fabric"
					inst.Icon = "flask"
				}
			}

			// If game version is still default 1.21.1, infer from mods
			if inst.GameVersion == "1.21.1" {
				topVer := ""
				topCount := 0
				for ver, count := range versionCounts {
					if count > topCount {
						topCount = count
						topVer = ver
					}
				}
				if topCount >= 2 && topVer != "" {
					inst.GameVersion = topVer
				}
			}
		}
	}

	// 7. Check folder name for version hints if still 1.21.1
	if inst.GameVersion == "1.21.1" {
		if matched := regexp.MustCompile(`1\.\d+(\.\d+)?`).FindString(folderName); matched != "" {
			inst.GameVersion = matched
		}
	}

	// Standardize instance directory structure
	s.standardizeInstanceFolder(folderPath, inst)

	return inst
}

// standardizeInstanceFolder ensures required folders and ticalauncher_instance.json exist
func (s *Store) standardizeInstanceFolder(folderPath string, inst models.Instance) {
	_ = os.MkdirAll(filepath.Join(folderPath, "mods"), 0755)
	_ = os.MkdirAll(filepath.Join(folderPath, "resourcepacks"), 0755)
	_ = os.MkdirAll(filepath.Join(folderPath, "shaderpacks"), 0755)
	_ = os.MkdirAll(filepath.Join(folderPath, "saves"), 0755)
	_ = os.MkdirAll(filepath.Join(folderPath, "config"), 0755)

	metaPath := filepath.Join(folderPath, "ticalauncher_instance.json")
	if data, err := json.MarshalIndent(inst, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, data, 0644)
	}
}

func extractLoaderVersion(raw, loaderType string) string {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	idx := strings.Index(lower, loaderType)
	if idx != -1 {
		part := raw[idx+len(loaderType):]
		part = strings.TrimLeft(part, "-_ /")
		return part
	}
	return ""
}

// Instance methods
func (s *Store) GetInstances() []models.Instance {
	s.ScanUnregisteredInstances()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Instance{}, s.Instances...)
}

func (s *Store) GetInstance(id string) *models.Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.Instances {
		if s.Instances[i].ID == id {
			inst := s.Instances[i]
			return &inst
		}
	}
	return nil
}

func (s *Store) AddInstance(inst models.Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inst.ID == "" {
		inst.ID = uuid.New().String()
	}
	if inst.Path == "" {
		inst.Path = s.getUniqueInstancePathLocked(inst.Name, inst.ID)
	}
	_ = os.MkdirAll(inst.Path, 0755)
	inst.CreatedAt = time.Now()
	s.Instances = append([]models.Instance{inst}, s.Instances...)
	_ = s.saveLocked()
}

func (s *Store) UpdateInstance(inst models.Instance) (*models.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var existingIndex = -1
	for i := range s.Instances {
		if s.Instances[i].ID == inst.ID {
			existingIndex = i
			break
		}
	}
	if existingIndex == -1 {
		return nil, fmt.Errorf("không tìm thấy instance với ID %s", inst.ID)
	}

	oldInst := s.Instances[existingIndex]
	trimmedNewName := strings.TrimSpace(inst.Name)
	if trimmedNewName == "" {
		trimmedNewName = oldInst.Name
	}
	inst.Name = trimmedNewName

	// Determine if folder rename is necessary
	expectedFolderName := SanitizeFolderName(inst.Name)
	currentFolderName := filepath.Base(oldInst.Path)

	needsRename := false
	if trimmedNewName != oldInst.Name || currentFolderName != expectedFolderName || inst.Path == "" {
		needsRename = true
	}

	if needsRename {
		newPath := s.getUniqueInstancePathLocked(inst.Name, inst.ID)
		if oldInst.Path != "" && filepath.Clean(oldInst.Path) != filepath.Clean(newPath) {
			if info, err := os.Stat(oldInst.Path); err == nil && info.IsDir() {
				_ = os.MkdirAll(filepath.Dir(newPath), 0755)

				// Windows case-only rename handling
				if strings.EqualFold(oldInst.Path, newPath) {
					tempPath := oldInst.Path + "_renametmp"
					if err := os.Rename(oldInst.Path, tempPath); err != nil {
						return nil, fmt.Errorf("lỗi đổi tên tạm: %w", err)
					}
					if err := os.Rename(tempPath, newPath); err != nil {
						_ = os.Rename(tempPath, oldInst.Path) // rollback
						return nil, fmt.Errorf("lỗi đổi tên thư mục sang '%s': %w", filepath.Base(newPath), err)
					}
				} else {
					if err := os.Rename(oldInst.Path, newPath); err != nil {
						return nil, fmt.Errorf("lỗi đổi tên thư mục từ '%s' sang '%s': %w (vui lòng đảm bảo không có file đang mở)", filepath.Base(oldInst.Path), filepath.Base(newPath), err)
					}
				}
			} else {
				_ = os.MkdirAll(newPath, 0755)
			}
		} else {
			_ = os.MkdirAll(newPath, 0755)
		}
		inst.Path = newPath
	} else if inst.Path == "" {
		inst.Path = oldInst.Path
	}

	s.Instances[existingIndex] = inst
	s.standardizeInstanceFolder(inst.Path, inst)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	return &inst, nil
}

func (s *Store) RemoveInstance(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var updated []models.Instance
	for _, inst := range s.Instances {
		if inst.ID != id {
			updated = append(updated, inst)
		}
	}
	s.Instances = updated
	_ = s.saveLocked()
}

func (s *Store) GetSettings() models.AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Settings
}

func (s *Store) UpdateSettings(settings models.AppSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Settings = settings
	_ = s.saveLocked()
}

// Community Server methods
func (s *Store) GetCommunityServers() []models.CommunityServerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.CommunityServerEntry{}, s.CommunityServers...)
}

func (s *Store) AddCommunityServer(srv models.CommunityServerEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if srv.ID == "" {
		srv.ID = uuid.New().String()
	}
	s.CommunityServers = append(s.CommunityServers, srv)
	_ = s.saveLocked()
}

func (s *Store) RemoveCommunityServer(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var updated []models.CommunityServerEntry
	for _, srv := range s.CommunityServers {
		if srv.ID != id {
			updated = append(updated, srv)
		}
	}
	s.CommunityServers = updated
	_ = s.saveLocked()
}

func (s *Store) FindInstanceByServerID(serverID string) *models.Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.Instances {
		if s.Instances[i].ServerID == serverID {
			inst := s.Instances[i]
			return &inst
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	type StoredData struct {
		Accounts         []models.Account              `json:"accounts"`
		Instances        []models.Instance             `json:"instances"`
		CommunityServers []models.CommunityServerEntry `json:"communityServers"`
		Settings         models.AppSettings            `json:"settings"`
	}

	stored := StoredData{
		Accounts:         s.Accounts,
		Instances:        s.Instances,
		CommunityServers: s.CommunityServers,
		Settings:         s.Settings,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.getStoreFilePath(), data, 0644)
}

func createSampleServerManifest(filePath string) {
	if _, err := os.Stat(filePath); err == nil {
		return // already exists
	}

	manifest := models.ServerManifest{
		ID:          "tica-battle-royale",
		Name:        "TiCa Battle Royale",
		Tagline:     "Chế độ nhảy dù, bắn súng 3D và bo thu kịch tính",
		Description: "Máy chủ Battle Royale chính thức của cộng đồng TiCa. Tích hợp súng 3D, xe cộ, giáp chống đạn và bản đồ sinh tồn mở rộng.",
		Version:     "1.0.0",
		Changelog:   "Khởi động Mùa giải S1! Cập nhật đầy đủ súng trường, scope ngắm và tối ưu hóa giật lag.",
		GameVersion: "1.20.1",
		Loader:      "fabric",
		LoaderVersion: "0.16.5",
		ServerIP:    "127.0.0.1",
		ServerPort:  25565,
		MinMemoryMB: 2048,
		MaxMemoryMB: 4096,
		Files: []models.ModpackFile{
			{
				Path: "mods/sodium-fabric-0.5.11+mc1.20.1.jar",
				URL:  "https://cdn.modrinth.com/data/AANobbMI/versions/gG0rS7T4/sodium-fabric-0.5.11%2Bmc1.20.1.jar",
				SHA1: "4f70ecaa33e9b11e2bfba728ad730ecfa97ec2ea",
				Size: 1146747,
			},
			{
				Path: "mods/iris-1.7.5+mc1.20.1.jar",
				URL:  "https://cdn.modrinth.com/data/YL57xq9U/versions/F8gM8gL2/iris-1.7.5%2Bmc1.20.1.jar",
				SHA1: "1ec5aaeb11461ff635b719448831969d27bf9d64",
				Size: 2617757,
			},
		},
	}

	data, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filePath, data, 0644)
}
