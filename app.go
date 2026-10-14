package main

import (
	"Aether/pkg/auth"
	"Aether/pkg/java"
	"Aether/pkg/minecraft"
	"Aether/pkg/models"
	"Aether/pkg/modrinth"
	"Aether/pkg/storage"
	modsync "Aether/pkg/sync"
	"Aether/pkg/system"
	"Aether/pkg/tunnel"
	"Aether/pkg/updater"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	store        *storage.Store
	activeCmd    *exec.Cmd
	cmdMu        sync.Mutex
	isRunning    bool
	msAuthMu     sync.Mutex
	msAuthCancel context.CancelFunc
	tunnelMgr    *tunnel.TunnelManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{
		store: storage.NewStore(),
	}
	a.tunnelMgr = tunnel.NewTunnelManager(func(status tunnel.TunnelStatus) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "tunnel:status", status)
		}
	})
	return a
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	if a.tunnelMgr != nil {
		_ = a.tunnelMgr.Stop()
	}
}

// Window controls
func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) ToggleMaximizeWindow() {
	runtime.WindowToggleMaximise(a.ctx)
}

func (a *App) CloseWindow() {
	runtime.Quit(a.ctx)
}

// System stats
func (a *App) GetSystemStats() models.SystemStats {
	return system.GetSystemStats(a.store.LauncherDir)
}

func (a *App) OpenFolder(folderPath string) error {
	if folderPath == "" {
		folderPath = a.store.LauncherDir
	}
	return system.OpenFolder(folderPath)
}

// BrowseJavaExecutable opens a native Windows file dialog to pick javaw.exe or java.exe
func (a *App) BrowseJavaExecutable() (string, error) {
	selected, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn file thực thi Java (javaw.exe hoặc java.exe)",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Java Executable (*.exe)",
				Pattern:     "*.exe",
			},
			{
				DisplayName: "Tất cả tập tin (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	return selected, err
}

// Account management
func (a *App) GetAccounts() []models.Account {
	return a.store.GetAccounts()
}

func (a *App) GetActiveAccount() *models.Account {
	return a.store.GetActiveAccount()
}

func (a *App) AddOfflineAccount(username string) (*models.Account, error) {
	acc, err := auth.GenerateOfflineAccount(username)
	if err != nil {
		return nil, err
	}
	a.store.AddAccount(*acc)
	return acc, nil
}

func (a *App) SetActiveAccount(id string) {
	a.store.SetActiveAccount(id)
}

func (a *App) RemoveAccount(id string) {
	a.store.RemoveAccount(id)
}

// StartMicrosoftAuth initiates device code flow
func (a *App) StartMicrosoftAuth() (*auth.DeviceCodeResult, error) {
	a.msAuthMu.Lock()
	if a.msAuthCancel != nil {
		a.msAuthCancel()
		a.msAuthCancel = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.msAuthCancel = cancel
	a.msAuthMu.Unlock()

	res, err := auth.RequestDeviceCode(ctx)
	if err != nil {
		a.msAuthMu.Lock()
		if a.msAuthCancel != nil {
			a.msAuthCancel()
			a.msAuthCancel = nil
		}
		a.msAuthMu.Unlock()
		return nil, err
	}
	return res, nil
}

// PollMicrosoftAuth polls Microsoft for approval and completes login
func (a *App) PollMicrosoftAuth(deviceCode string, interval, expiresIn int) (*models.Account, error) {
	a.msAuthMu.Lock()
	cancel := a.msAuthCancel
	a.msAuthMu.Unlock()

	ctx := context.Background()
	if cancel != nil {
		// Use active context from StartMicrosoftAuth
		a.msAuthMu.Lock()
		ctx = a.ctx
		a.msAuthMu.Unlock()
	}

	tokRes, err := auth.PollForMSToken(ctx, deviceCode, interval, expiresIn)
	if err != nil {
		return nil, err
	}

	account, err := auth.CompleteMicrosoftAuth(ctx, tokRes.AccessToken, tokRes.RefreshToken)
	if err != nil {
		return nil, err
	}

	a.store.AddAccount(*account)
	a.store.SetActiveAccount(account.ID)

	a.msAuthMu.Lock()
	if a.msAuthCancel != nil {
		a.msAuthCancel = nil
	}
	a.msAuthMu.Unlock()

	return account, nil
}

// CancelMicrosoftAuth aborts any active Microsoft auth polling
func (a *App) CancelMicrosoftAuth() {
	a.msAuthMu.Lock()
	defer a.msAuthMu.Unlock()
	if a.msAuthCancel != nil {
		a.msAuthCancel()
		a.msAuthCancel = nil
	}
}

// Tailscale P2P Tunnel methods
func (a *App) GetTunnelStatus() tunnel.TunnelStatus {
	if a.tunnelMgr == nil {
		return tunnel.TunnelStatus{StatusText: "Chưa khởi tạo"}
	}
	return a.tunnelMgr.GetStatus()
}

func (a *App) StartTunnel() (*tunnel.TunnelStatus, error) {
	if a.tunnelMgr == nil {
		return nil, fmt.Errorf("tunnel manager chưa khởi tạo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	authKey, target, err := tunnel.RequestBrokerCredential(ctx)
	if err != nil {
		return nil, err
	}
	return a.tunnelMgr.Start(authKey, target)
}

func (a *App) StopTunnel() (*tunnel.TunnelStatus, error) {
	if a.tunnelMgr == nil {
		return nil, nil
	}
	err := a.tunnelMgr.Stop()
	status := a.tunnelMgr.GetStatus()
	return &status, err
}

func (a *App) ToggleTunnel() (*tunnel.TunnelStatus, error) {
	if a.tunnelMgr == nil {
		return nil, fmt.Errorf("tunnel manager chưa khởi tạo")
	}
	cur := a.tunnelMgr.GetStatus()
	if cur.Running {
		return a.StopTunnel()
	}
	return a.StartTunnel()
}

// Instance management
func (a *App) GetInstances() []models.Instance {
	return a.store.GetInstances()
}

func (a *App) ScanInstancesFolder() []models.Instance {
	return a.store.ScanUnregisteredInstances()
}

func (a *App) CreateInstance(name, version, loader, loaderVersion, icon string, minRAM, maxRAM int) (*models.Instance, error) {
	if name == "" {
		loaderSuffix := ""
		if loader != "" && loader != "vanilla" {
			loaderSuffix = " " + strings.Title(loader)
		}
		name = fmt.Sprintf("Minecraft %s%s", version, loaderSuffix)
	}
	if minRAM <= 0 {
		minRAM = 2048
	}
	if maxRAM <= 0 {
		maxRAM = 4096
	}

	id := uuid.New().String()
	instPath := a.store.GetUniqueInstancePath(name, id)
	_ = os.MkdirAll(instPath, 0755)

	inst := models.Instance{
		ID:            id,
		Name:          name,
		GameVersion:   version,
		VersionType:   "release",
		Loader:        loader,
		LoaderVersion: loaderVersion,
		Icon:          icon,
		MinMemoryMB:   minRAM,
		MaxMemoryMB:   maxRAM,
		Path:          instPath,
		CreatedAt:     time.Now(),
		LastPlayed:    time.Now(),
	}

	a.store.AddInstance(inst)
	return &inst, nil
}

// GetLoaderVersions fetches available loader versions for a given Minecraft version and loader
func (a *App) GetLoaderVersions(gameVersion, loader string) (*models.LoaderVersionResult, error) {
	return minecraft.GetLoaderVersions(gameVersion, loader)
}

// GetServerLiveStatus returns real-time SLP ping and player count from Minecraft server
func (a *App) GetServerLiveStatus() *models.ServerLiveStatus {
	return minecraft.GetServerLiveStatus()
}

const OfficialChillTownManifestURL = "https://raw.githubusercontent.com/TieuLong07/camlauncher/main/manifests/chilltown.json"

// EnsureOfficialChillTown creates or migrates the official Cooking TQ profile,
// then applies the latest Chill Town modpack before it can launch.
func (a *App) EnsureOfficialChillTown() (*models.Instance, error) {
	manifest, err := modsync.FetchServerManifest(OfficialChillTownManifestURL)
	if err != nil {
		return nil, fmt.Errorf("không thể tải thông tin Chill Town: %w", err)
	}
	if manifest.ID != "chilltown" || manifest.GameVersion != "1.20.1" || !strings.EqualFold(manifest.Loader, "forge") {
		return nil, fmt.Errorf("manifest Chill Town không hợp lệ")
	}

	instances := a.store.GetInstances()
	target := storage.FindOfficialChillTown(instances)
	if target == nil {
		id := uuid.New().String()
		target = &models.Instance{
			ID:            id,
			Name:          "Chill Town",
			GameVersion:   manifest.GameVersion,
			VersionType:   "release",
			Loader:        manifest.Loader,
			LoaderVersion: manifest.LoaderVersion,
			Icon:          "flask",
			MinMemoryMB:   manifest.MinMemoryMB,
			MaxMemoryMB:   manifest.MaxMemoryMB,
			Path:          a.store.GetUniqueInstancePath("Chill Town", id),
			ServerSyncURL: OfficialChillTownManifestURL,
			ServerID:      "chilltown",
			CreatedAt:     time.Now(),
			LastPlayed:    time.Now(),
		}
		if target.MinMemoryMB <= 0 {
			target.MinMemoryMB = 2048
		}
		if target.MaxMemoryMB <= 0 {
			target.MaxMemoryMB = 8192
		}

		// Rename the complete old Cooking TQ folder. No content is copied or omitted.
		legacyPath := filepath.Join(os.Getenv("APPDATA"), "TiCaLauncher", "instances", "Cooking TQ")
		if info, statErr := os.Stat(legacyPath); statErr == nil && info.IsDir() {
			if err := storage.RenameDirectory(legacyPath, target.Path); err != nil {
				return nil, fmt.Errorf("không thể đổi tên nguyên instance Cooking TQ sang Chill Town: %w", err)
			}
			state, _ := json.Marshal(models.LocalSyncState{ServerID: "chilltown", SyncedVersion: manifest.Version, SyncedAt: time.Now(), LegacyMigrated: true})
			if err := os.WriteFile(filepath.Join(target.Path, "sync_state.json"), state, 0644); err != nil {
				return nil, err
			}
		}
		a.store.AddInstance(*target)
	} else {
		// Upgrade the prior partial copy by renaming the complete legacy instance in its place.
		// The partial target is retained as a timestamped sibling backup.
		legacyPath := filepath.Join(os.Getenv("APPDATA"), "TiCaLauncher", "instances", "Cooking TQ")
		statePath := filepath.Join(target.Path, "sync_state.json")
		var state models.LocalSyncState
		if data, readErr := os.ReadFile(statePath); readErr == nil {
			_ = json.Unmarshal(data, &state)
		}
		if !state.LegacyMigrated {
			if info, statErr := os.Stat(legacyPath); statErr == nil && info.IsDir() {
				if _, err := storage.MigrateLegacyInstance(legacyPath, target.Path); err != nil {
					return nil, fmt.Errorf("không thể đổi tên nguyên instance Cooking TQ sang Chill Town: %w", err)
				}
				state = models.LocalSyncState{ServerID: "chilltown", SyncedVersion: manifest.Version, SyncedAt: time.Now(), LegacyMigrated: true}
				stateBytes, _ := json.Marshal(state)
				if err := os.WriteFile(statePath, stateBytes, 0644); err != nil {
					return nil, err
				}
			}
		}
		target.Name = "Chill Town"
		target.GameVersion = manifest.GameVersion
		target.Loader = manifest.Loader
		target.LoaderVersion = manifest.LoaderVersion
		target.ServerID = "chilltown"
		target.ServerSyncURL = OfficialChillTownManifestURL
		if updated, updateErr := a.store.UpdateInstance(*target); updateErr != nil {
			return nil, updateErr
		} else {
			target = updated
		}
	}

	progressCallback := func(evt models.ProgressEvent) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "game:progress", evt)
		}
	}
	if err := modsync.SyncServerModpack(target.Path, manifest, progressCallback); err != nil {
		return nil, err
	}
	target.ServerVersion = manifest.Version
	if updated, err := a.store.UpdateInstance(*target); err == nil {
		target = updated
	}
	return target, nil
}

// CheckOfficialServerUpdate checks the live GitHub manifest against the official profile.
func (a *App) CheckOfficialServerUpdate() (*models.UpdateCheckResult, error) {
	target := storage.FindOfficialChillTown(a.store.GetInstances())
	if target == nil {
		manifest, err := modsync.FetchServerManifest(OfficialChillTownManifestURL)
		if err != nil {
			return nil, err
		}
		return &models.UpdateCheckResult{ServerID: manifest.ID, ServerName: manifest.Name, LatestVersion: manifest.Version, HasUpdate: true, Changelog: manifest.Changelog, Manifest: manifest}, nil
	}
	return modsync.CheckServerUpdate(target.Path, OfficialChillTownManifestURL)
}

// SyncOfficialChillTown downloads and applies the current official modpack.
func (a *App) SyncOfficialChillTown() (*models.Instance, error) { return a.EnsureOfficialChillTown() }

// CheckEnvironmentSetup scans Java installations and evaluates readiness for Chill Town
func (a *App) CheckEnvironmentSetup() *models.EnvironmentSetupStatus {
	settings := a.store.GetSettings()
	return java.CheckEnvironmentSetup(a.store.LauncherDir, settings.DefaultJavaPath)
}

// InstallJavaRuntime downloads and installs a portable Adoptium Eclipse Temurin JRE
func (a *App) InstallJavaRuntime(majorVersion int) (string, error) {
	if majorVersion <= 0 {
		majorVersion = 17
	}
	progressCallback := func(evt models.ProgressEvent) {
		runtime.EventsEmit(a.ctx, "setup:progress", evt)
		runtime.EventsEmit(a.ctx, "game:progress", evt)
	}

	javaPath, err := java.EnsureJavaRuntime(a.store.LauncherDir, majorVersion, progressCallback)
	if err != nil {
		return "", err
	}

	// Update settings default if not set or if installing Java 17 for Chill Town
	settings := a.store.GetSettings()
	if settings.DefaultJavaPath == "" || majorVersion == 17 {
		settings.DefaultJavaPath = javaPath
		a.store.UpdateSettings(settings)
	}

	return javaPath, nil
}

// SetDefaultJavaPath updates default Java runtime path in settings
func (a *App) SetDefaultJavaPath(path string) error {
	settings := a.store.GetSettings()
	settings.DefaultJavaPath = path
	a.store.UpdateSettings(settings)
	return nil
}

// AutoSetupEnvironment checks and auto-installs Java 17 for new machines
func (a *App) AutoSetupEnvironment() (*models.EnvironmentSetupStatus, error) {
	status := a.CheckEnvironmentSetup()
	if !status.HasJava17 {
		_, err := a.InstallJavaRuntime(17)
		if err != nil {
			return status, err
		}
		status = a.CheckEnvironmentSetup()
	}
	settings := a.store.GetSettings()
	settings.AutoSetupDone = true
	a.store.UpdateSettings(settings)
	return status, nil
}

func (a *App) DeleteInstance(id string) {
	a.store.RemoveInstance(id)
}

func (a *App) OpenInstanceFolder(id string) error {
	for _, inst := range a.store.GetInstances() {
		if inst.ID == id || inst.Name == id || (inst.ServerID != "" && inst.ServerID == id) {
			_ = os.MkdirAll(inst.Path, 0755)
			return system.OpenFolder(inst.Path)
		}
	}
	return fmt.Errorf("không tìm thấy instance")
}

func (a *App) OpenInstanceSubFolder(id, subFolder string) error {
	for _, inst := range a.store.GetInstances() {
		if inst.ID == id || inst.Name == id || (inst.ServerID != "" && inst.ServerID == id) {
			target := filepath.Join(inst.Path, subFolder)
			_ = os.MkdirAll(target, 0755)
			return system.OpenFolder(target)
		}
	}
	return fmt.Errorf("không tìm thấy instance")
}

func (a *App) UpdateInstance(inst models.Instance) (*models.Instance, error) {
	if a.IsGameRunning() {
		return nil, fmt.Errorf("không thể chỉnh sửa cấu hình hoặc đổi tên khi Minecraft đang chạy")
	}
	return a.store.UpdateInstance(inst)
}

func (a *App) GetInstanceFiles(instanceID, fileType string) ([]models.InstanceFileItem, error) {
	var targetInst *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID || inst.Name == instanceID || (inst.ServerID != "" && inst.ServerID == instanceID) {
			targetInst = &inst
			break
		}
	}
	if targetInst == nil {
		return nil, fmt.Errorf("không tìm thấy instance")
	}

	targetDir := filepath.Join(targetInst.Path, fileType)
	_ = os.MkdirAll(targetDir, 0755)

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return []models.InstanceFileItem{}, nil
	}

	var items []models.InstanceFileItem
	for _, entry := range entries {
		fileName := entry.Name()
		if strings.HasPrefix(fileName, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		var enabled bool
		var cleanName string

		if fileType == "mods" {
			if !strings.HasSuffix(fileName, ".jar") && !strings.HasSuffix(fileName, ".jar.disabled") && !strings.HasSuffix(fileName, ".disabled") {
				continue
			}
			enabled = strings.HasSuffix(fileName, ".jar")
			cleanName = strings.TrimSuffix(fileName, ".disabled")
			cleanName = strings.TrimSuffix(cleanName, ".jar")
		} else {
			enabled = !strings.HasSuffix(fileName, ".disabled")
			cleanName = strings.TrimSuffix(fileName, ".disabled")
			cleanName = strings.TrimSuffix(cleanName, ".zip")
		}

		items = append(items, models.InstanceFileItem{
			Name:      cleanName,
			FileName:  fileName,
			SizeBytes: info.Size(),
			ModTime:   info.ModTime().Format("2006-01-02 15:04"),
			Enabled:   enabled,
			FileType:  fileType,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items, nil
}

func (a *App) ToggleInstanceFile(instanceID, fileType, fileName string, enable bool) error {
	var targetInst *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID || inst.Name == instanceID || (inst.ServerID != "" && inst.ServerID == instanceID) {
			targetInst = &inst
			break
		}
	}
	if targetInst == nil {
		return fmt.Errorf("không tìm thấy instance")
	}

	targetDir := filepath.Join(targetInst.Path, fileType)
	oldPath := filepath.Join(targetDir, fileName)

	var newFileName string
	if enable {
		newFileName = strings.TrimSuffix(fileName, ".disabled")
	} else {
		if !strings.HasSuffix(fileName, ".disabled") {
			newFileName = fileName + ".disabled"
		} else {
			newFileName = fileName
		}
	}

	newPath := filepath.Join(targetDir, newFileName)
	if oldPath != newPath {
		return os.Rename(oldPath, newPath)
	}
	return nil
}

func (a *App) DeleteInstanceFile(instanceID, fileType, fileName string) error {
	var targetInst *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID || inst.Name == instanceID || (inst.ServerID != "" && inst.ServerID == instanceID) {
			targetInst = &inst
			break
		}
	}
	if targetInst == nil {
		return fmt.Errorf("không tìm thấy instance")
	}

	targetPath := filepath.Join(targetInst.Path, fileType, fileName)
	return os.RemoveAll(targetPath)
}

// Minecraft Manifest
func (a *App) GetVersionManifest() (*models.VersionManifest, error) {
	return minecraft.FetchVersionManifest(a.store.LauncherDir)
}

// Modrinth Mod Browser
func (a *App) SearchMods(query, gameVersion, loader, projectType string) (*models.ModrinthSearchResult, error) {
	return modrinth.SearchMods(query, gameVersion, loader, projectType, 24)
}

func (a *App) InstallModToInstance(instanceID, slug string) (string, error) {
	var target *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID {
			target = &inst
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("không tìm thấy instance")
	}

	modsDir := filepath.Join(target.Path, "mods")
	return modrinth.InstallMod(slug, target.GameVersion, target.Loader, modsDir)
}

func (a *App) InstallFPSBoostPack(instanceID string) (map[string]interface{}, error) {
	var target *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID {
			target = &inst
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("không tìm thấy instance")
	}

	modsDir := filepath.Join(target.Path, "mods")
	installed, failed := modrinth.InstallFPSBoostPack(target.GameVersion, target.Loader, modsDir)

	return map[string]interface{}{
		"installed": installed,
		"failed":    failed,
	}, nil
}

// Game Launch Engine
func (a *App) LaunchGame(instanceID string) error {
	a.cmdMu.Lock()
	if a.isRunning {
		a.cmdMu.Unlock()
		return fmt.Errorf("game đang chạy, vui lòng không mở trùng lặp")
	}
	a.cmdMu.Unlock()

	var target *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID {
			target = &inst
			break
		}
	}
	if target == nil {
		return fmt.Errorf("không tìm thấy instance")
	}

	activeAccount := a.store.GetActiveAccount()
	if activeAccount == nil {
		return fmt.Errorf("vui lòng chọn hoặc thêm tài khoản người chơi trước khi vào game")
	}

	// Auto-refresh Microsoft Account if token expired or near expiration (< 5 mins)
	if activeAccount.Type == models.AccountMicrosoft && activeAccount.RefreshToken != "" {
		if activeAccount.ExpiresAt == 0 || time.Now().Unix() > activeAccount.ExpiresAt-300 {
			refreshCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := auth.RefreshMicrosoftAccount(refreshCtx, activeAccount); err == nil {
				a.store.UpdateAccount(*activeAccount)
			}
		}
	}

	// Java will be automatically verified or downloaded by PrepareAndLaunch
	javaPath := target.JavaPath
	if javaPath == "" {
		settings := a.store.GetSettings()
		javaPath = settings.DefaultJavaPath
	}

	progressCallback := func(evt models.ProgressEvent) {
		runtime.EventsEmit(a.ctx, "game:progress", evt)
	}

	logCallback := func(evt models.LogEvent) {
		runtime.EventsEmit(a.ctx, "game:log", evt)
	}

	// Emit initial event
	progressCallback(models.ProgressEvent{
		InstanceID: target.ID,
		Stage:      "starting",
		Message:    "Chuẩn bị khởi chạy CamLauncher Engine...",
		Percentage: 0,
	})

	go func() {
		cmd, err := minecraft.PrepareAndLaunch(minecraft.LaunchOptions{
			Instance:       target,
			Account:        activeAccount,
			LauncherDir:    a.store.LauncherDir,
			JavaExecutable: javaPath,
			ProgressFn:     progressCallback,
			LogFn:          logCallback,
		})

		if err != nil {
			logCallback(models.LogEvent{
				InstanceID: target.ID,
				Level:      "ERROR",
				Line:       fmt.Sprintf("[CamLauncher] Lỗi khởi chạy: %v", err),
				Timestamp:  time.Now().Format("15:04:05"),
			})
			runtime.EventsEmit(a.ctx, "game:status", map[string]interface{}{
				"running": false,
				"error":   err.Error(),
			})
			return
		}

		a.cmdMu.Lock()
		a.activeCmd = cmd
		a.isRunning = true
		a.cmdMu.Unlock()

		runtime.EventsEmit(a.ctx, "game:status", map[string]interface{}{
			"running": true,
			"pid":     cmd.Process.Pid,
		})

		// Wait for game exit
		exitErr := cmd.Wait()

		a.cmdMu.Lock()
		a.isRunning = false
		a.activeCmd = nil
		a.cmdMu.Unlock()

		exitMsg := "Game đã thoát bình thường (code 0)"
		if exitErr != nil {
			exitMsg = fmt.Sprintf("Game đã thoát với mã lỗi: %v", exitErr)
		}

		logCallback(models.LogEvent{
			InstanceID: target.ID,
			Level:      "INFO",
			Line:       fmt.Sprintf("[CamLauncher] %s", exitMsg),
			Timestamp:  time.Now().Format("15:04:05"),
		})

		runtime.EventsEmit(a.ctx, "game:status", map[string]interface{}{
			"running": false,
			"message": exitMsg,
		})
	}()

	return nil
}

func (a *App) KillGame() error {
	a.cmdMu.Lock()
	defer a.cmdMu.Unlock()

	if !a.isRunning || a.activeCmd == nil || a.activeCmd.Process == nil {
		return fmt.Errorf("không có game nào đang chạy")
	}

	return a.activeCmd.Process.Kill()
}

func (a *App) IsGameRunning() bool {
	a.cmdMu.Lock()
	defer a.cmdMu.Unlock()
	return a.isRunning
}

// Settings
func (a *App) GetSettings() models.AppSettings {
	return a.store.GetSettings()
}

func (a *App) UpdateSettings(settings models.AppSettings) {
	a.store.UpdateSettings(settings)
}

// Community Server & Modpack Sync Methods
func (a *App) GetCommunityServers() []models.CommunityServerEntry {
	return a.store.GetCommunityServers()
}

func (a *App) AddCommunityServer(name, manifestURL, tag string) (*models.CommunityServerEntry, error) {
	if name == "" || manifestURL == "" {
		return nil, fmt.Errorf("tên và URL manifest không được để trống")
	}

	entry := models.CommunityServerEntry{
		ID:          uuid.New().String(),
		Name:        name,
		ManifestURL: manifestURL,
		Tag:         tag,
		IsFeatured:  false,
	}

	a.store.AddCommunityServer(entry)
	return &entry, nil
}

func (a *App) RemoveCommunityServer(id string) {
	a.store.RemoveCommunityServer(id)
}

// CheckServerUpdate checks if the selected instance has an update from its community server
func (a *App) CheckServerUpdate(instanceID string) (*models.UpdateCheckResult, error) {
	var target *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID {
			target = &inst
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("không tìm thấy instance")
	}

	if target.ServerSyncURL == "" {
		// Not a synced instance
		return nil, nil
	}

	return modsync.CheckServerUpdate(target.Path, target.ServerSyncURL)
}

// InstallCommunityServer creates an instance for the server and downloads its modpack 1-click
func (a *App) InstallCommunityServer(serverID string) (*models.Instance, error) {
	var entry *models.CommunityServerEntry
	for _, srv := range a.store.GetCommunityServers() {
		if srv.ID == serverID {
			entry = &srv
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("không tìm thấy máy chủ cộng đồng")
	}

	manifest, err := modsync.FetchServerManifest(entry.ManifestURL)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc manifest máy chủ: %w", err)
	}

	// Check if instance already exists
	existing := a.store.FindInstanceByServerID(manifest.ID)
	if existing != nil {
		return existing, nil
	}

	id := uuid.New().String()
	instName := manifest.Name
	if instName == "" {
		instName = entry.Name
	}
	instPath := a.store.GetUniqueInstancePath(instName, id)
	_ = os.MkdirAll(instPath, 0755)

	minRAM := manifest.MinMemoryMB
	if minRAM <= 0 {
		minRAM = 2048
	}
	maxRAM := manifest.MaxMemoryMB
	if maxRAM <= 0 {
		maxRAM = 4096
	}

	inst := models.Instance{
		ID:            id,
		Name:          manifest.Name,
		GameVersion:   manifest.GameVersion,
		VersionType:   "release",
		Loader:        manifest.Loader,
		LoaderVersion: manifest.LoaderVersion,
		Icon:          "swords",
		MinMemoryMB:   minRAM,
		MaxMemoryMB:   maxRAM,
		Path:          instPath,
		ServerSyncURL: entry.ManifestURL,
		ServerID:      manifest.ID,
		ServerVersion: manifest.Version,
		CreatedAt:     time.Now(),
		LastPlayed:    time.Now(),
	}

	a.store.AddInstance(inst)

	// Sync mods immediately in background or synchronously
	progressCallback := func(evt models.ProgressEvent) {
		runtime.EventsEmit(a.ctx, "game:progress", evt)
	}

	go func() {
		_ = modsync.SyncServerModpack(instPath, manifest, progressCallback)
	}()

	return &inst, nil
}

// SyncCommunityServer manually triggers synchronization of mods for a community instance
func (a *App) SyncCommunityServer(instanceID string) error {
	var target *models.Instance
	for _, inst := range a.store.GetInstances() {
		if inst.ID == instanceID {
			target = &inst
			break
		}
	}
	if target == nil {
		return fmt.Errorf("không tìm thấy instance")
	}

	if target.ServerSyncURL == "" {
		return fmt.Errorf("instance này không liên kết với máy chủ modpack nào")
	}

	manifest, err := modsync.FetchServerManifest(target.ServerSyncURL)
	if err != nil {
		return fmt.Errorf("lỗi tải manifest: %w", err)
	}

	progressCallback := func(evt models.ProgressEvent) {
		runtime.EventsEmit(a.ctx, "game:progress", evt)
	}

	err = modsync.SyncServerModpack(target.Path, manifest, progressCallback)
	if err != nil {
		return err
	}

	target.ServerVersion = manifest.Version
	_, _ = a.store.UpdateInstance(*target)

	return nil
}

// Launcher Self-Update Methods
func (a *App) GetAppVersion() string {
	return updater.GetAppVersion()
}

func (a *App) CheckLauncherUpdate() (*models.LauncherUpdateInfo, error) {
	manifestURL := updater.DefaultUpdateManifestURL
	// Check if local test manifest exists in LauncherDir
	localTestManifest := filepath.Join(a.store.LauncherDir, "sample_launcher_update.json")
	if _, err := os.Stat(localTestManifest); err == nil {
		manifestURL = "file://" + filepath.ToSlash(localTestManifest)
	}

	return updater.CheckLauncherUpdate(manifestURL)
}

func (a *App) ApplyLauncherUpdate(downloadURL, sha256 string) error {
	progressCallback := func(evt models.ProgressEvent) {
		runtime.EventsEmit(a.ctx, "game:progress", evt)
	}
	return updater.ApplyLauncherUpdate(downloadURL, sha256, progressCallback)
}
