package models

import "time"

// AccountType defines the type of Minecraft account
type AccountType string

const (
	AccountOffline   AccountType = "offline"
	AccountMicrosoft AccountType = "microsoft"
)

// Account represents a Minecraft player account
type Account struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	Type         AccountType `json:"type"`
	UUID         string      `json:"uuid"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken,omitempty"`
	ExpiresAt    int64       `json:"expiresAt,omitempty"`
	SkinURL      string      `json:"skinUrl"`
	CapeURL      string      `json:"capeUrl"`
	IsActive     bool        `json:"isActive"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// Instance represents an isolated Minecraft profile
type Instance struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	GameVersion     string    `json:"gameVersion"`
	VersionType     string    `json:"versionType"` // "release", "snapshot"
	Loader          string    `json:"loader"`      // "vanilla", "fabric", "forge", "neoforge"
	LoaderVersion   string    `json:"loaderVersion"`
	Icon            string    `json:"icon"` // icon preset or path
	MinMemoryMB     int       `json:"minMemoryMb"`
	MaxMemoryMB     int       `json:"maxMemoryMb"`
	JVMArgs         string    `json:"jvmArgs"`
	JavaPath        string    `json:"javaPath"`
	Path            string    `json:"path"`
	ServerSyncURL   string    `json:"serverSyncUrl,omitempty"`
	ServerID        string    `json:"serverId,omitempty"`
	ServerVersion   string    `json:"serverVersion,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	LastPlayed      time.Time `json:"lastPlayed"`
	PlayTimeMinutes int       `json:"playTimeMinutes"`
}

// VersionManifest represents the official Mojang version list
type VersionManifest struct {
	Latest   LatestVersion `json:"latest"`
	Versions []VersionItem `json:"versions"`
}

type LatestVersion struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type VersionItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

// Detailed Version JSON from Mojang
type VersionDetail struct {
	ID                 string                  `json:"id"`
	Type               string                  `json:"type"`
	MainClass          string                  `json:"mainClass"`
	MinecraftArguments string                  `json:"minecraftArguments,omitempty"`
	Arguments          *VersionArguments       `json:"arguments,omitempty"`
	AssetIndex         AssetIndexInfo          `json:"assetIndex"`
	Assets             string                  `json:"assets"`
	Downloads          map[string]DownloadInfo `json:"downloads"`
	Libraries          []Library               `json:"libraries"`
	JavaVersion        *JavaVersionInfo        `json:"javaVersion,omitempty"`
}

type JavaVersionInfo struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

type VersionArguments struct {
	Game []interface{} `json:"game"`
	JVM  []interface{} `json:"jvm"`
}

type AssetIndexInfo struct {
	ID        string `json:"id"`
	SHA1      string `json:"sha1"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"totalSize"`
	URL       string `json:"url"`
}

type DownloadInfo struct {
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
	Path string `json:"path,omitempty"`
}

type Library struct {
	Name      string            `json:"name"`
	Downloads *LibraryDownloads `json:"downloads,omitempty"`
	Rules     []Rule            `json:"rules,omitempty"`
	Natives   map[string]string `json:"natives,omitempty"`
	Extract   *ExtractRules     `json:"extract,omitempty"`
	URL       string            `json:"url,omitempty"`
}

type LibraryDownloads struct {
	Artifact    *DownloadInfo            `json:"artifact,omitempty"`
	Classifiers map[string]*DownloadInfo `json:"classifiers,omitempty"`
}

type Rule struct {
	Action string            `json:"action"` // "allow" or "disallow"
	OS     map[string]string `json:"os,omitempty"`
}

type ExtractRules struct {
	Exclude []string `json:"exclude"`
}

// LoaderVersionResult represents available loader versions for a Minecraft version
type LoaderVersionResult struct {
	Loader      string   `json:"loader"`
	GameVersion string   `json:"gameVersion"`
	Recommended string   `json:"recommended"`
	Versions    []string `json:"versions"`
}

// EnvironmentSetupStatus represents the full check of Java runtimes on the machine
type EnvironmentSetupStatus struct {
	HasJava17      bool          `json:"hasJava17"`
	HasJava21      bool          `json:"hasJava21"`
	HasJava8       bool          `json:"hasJava8"`
	Java17Path     string        `json:"java17Path"`
	Java21Path     string        `json:"java21Path"`
	Java8Path      string        `json:"java8Path"`
	SelectedJava   string        `json:"selectedJava"`
	ChillTownReady bool          `json:"chillTownReady"`
	AllRuntimes    []JavaRuntime `json:"allRuntimes"`
	TotalRAMGB     int           `json:"totalRamGb"`
	RecommendedRAM int           `json:"recommendedRam"`
}

// Java Runtime Info
type JavaRuntime struct {
	Path         string `json:"path"`
	Version      string `json:"version"`
	MajorVersion int    `json:"majorVersion"`
	Is64Bit      bool   `json:"is64Bit"`
	IsDefault    bool   `json:"isDefault"`
}

// Hardware & System Stats
type SystemStats struct {
	CPUModel     string        `json:"cpuModel"`
	CPUCores     int           `json:"cpuCores"`
	TotalRAMMB   int           `json:"totalRamMb"`
	FreeRAMMB    int           `json:"freeRamMb"`
	GPUModel     string        `json:"gpuModel"`
	JavaRuntimes []JavaRuntime `json:"javaRuntimes"`
	LauncherDir  string        `json:"launcherDir"`
}

// Launcher Settings
type AppSettings struct {
	LauncherDir       string `json:"launcherDir"`
	DefaultMinRAM     int    `json:"defaultMinRam"`
	DefaultMaxRAM     int    `json:"defaultMaxRam"`
	DefaultJavaPath   string `json:"defaultJavaPath,omitempty"`
	AutoSetupDone     bool   `json:"autoSetupDone,omitempty"`
	DefaultJVMArgs    string `json:"defaultJvmArgs"`
	AutoCloseOnLaunch bool   `json:"autoCloseOnLaunch"`
	Language          string `json:"language"`
	Theme             string `json:"theme"`
}

// Realtime Progress for Downloads and Launch
type ProgressEvent struct {
	InstanceID     string  `json:"instanceId"`
	Stage          string  `json:"stage"` // "checking", "downloading_client", "downloading_libraries", "downloading_assets", "extracting", "ready"
	Message        string  `json:"message"`
	Percentage     float64 `json:"percentage"`
	CurrentItem    string  `json:"currentItem"`
	CompletedItems int     `json:"completedItems"`
	TotalItems     int     `json:"totalItems"`
	SpeedKbps      float64 `json:"speedKbps"`
}

// Realtime Game Console Log
type LogEvent struct {
	InstanceID string `json:"instanceId"`
	Level      string `json:"level"` // "INFO", "WARN", "ERROR", "STDOUT", "STDERR"
	Line       string `json:"line"`
	Timestamp  string `json:"timestamp"`
}

// Modrinth Project
type ModrinthProject struct {
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
	IsInstalled bool     `json:"is_installed"`
}

type ModrinthSearchResult struct {
	Hits      []ModrinthProject `json:"hits"`
	TotalHits int               `json:"total_hits"`
}

// ModpackFile represents a single file to sync (mod, config, resourcepack)
type ModpackFile struct {
	Path string `json:"path"` // e.g. "mods/cgm-1.20.1.jar" or "config/cgm.json"
	URL  string `json:"url"`  // direct download link
	SHA1 string `json:"sha1,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// ModpackArchive is an optional complete pack archive for first-time installs.
type ModpackArchive struct {
	URL    string   `json:"url,omitempty"`
	URLs   []string `json:"urls,omitempty"` // split archive parts, in order
	SHA256 string   `json:"sha256,omitempty"`
	Size   int64    `json:"size,omitempty"`
}

// ServerManifest defines the modpack structure from the server admin
type ServerManifest struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Tagline       string          `json:"tagline"`
	Description   string          `json:"description"`
	Version       string          `json:"version"`     // e.g. "1.2.0"
	Changelog     string          `json:"changelog"`   // notes about this update
	GameVersion   string          `json:"gameVersion"` // e.g. "1.20.1"
	Loader        string          `json:"loader"`      // "fabric"
	LoaderVersion string          `json:"loaderVersion,omitempty"`
	ServerIP      string          `json:"serverIp,omitempty"`
	ServerPort    int             `json:"serverPort,omitempty"`
	BannerURL     string          `json:"bannerUrl,omitempty"`
	MinMemoryMB   int             `json:"minMemoryMb,omitempty"`
	MaxMemoryMB   int             `json:"maxMemoryMb,omitempty"`
	Archive       *ModpackArchive `json:"archive,omitempty"`
	Files         []ModpackFile   `json:"files"`
	ExpectedMods  []string        `json:"expectedMods,omitempty"`
}

// CommunityServerEntry is stored in the launcher's list of community servers
type CommunityServerEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ManifestURL string `json:"manifestUrl"`
	Tag         string `json:"tag"`
	IsFeatured  bool   `json:"isFeatured"`
}

// ServerLiveStatus represents real-time Minecraft SLP status
type ServerLiveStatus struct {
	Online     bool     `json:"online"`
	PingMs     int64    `json:"pingMs"`
	Players    int      `json:"players"`
	MaxPlayers int      `json:"maxPlayers"`
	Version    string   `json:"version"`
	MOTD       string   `json:"motd"`
	Favicon    string   `json:"favicon,omitempty"`
	PlayerList []string `json:"playerList,omitempty"`
}

// UpdateCheckResult tells the UI whether an update is available
type UpdateCheckResult struct {
	ServerID          string          `json:"serverId"`
	ServerName        string          `json:"serverName"`
	CurrentVersion    string          `json:"currentVersion"`
	LatestVersion     string          `json:"latestVersion"`
	HasUpdate         bool            `json:"hasUpdate"`
	IsDrifted         bool            `json:"isDrifted"`
	MissingMods       []string        `json:"missingMods,omitempty"`
	ExtraMods         []string        `json:"extraMods,omitempty"`
	Changelog         string          `json:"changelog"`
	FilesToDownload   []ModpackFile   `json:"filesToDownload"`
	FilesToDelete     []string        `json:"filesToDelete"`
	TotalDownloadSize int64           `json:"totalDownloadSize"`
	Manifest          *ServerManifest `json:"manifest,omitempty"`
}

// LocalSyncState tracks what files are currently managed for this instance
type LocalSyncState struct {
	ServerID       string    `json:"serverId"`
	SyncedVersion  string    `json:"syncedVersion"`
	SyncedAt       time.Time `json:"syncedAt"`
	ManagedFiles   []string  `json:"managedFiles"` // relative paths e.g. ["mods/a.jar", "mods/b.jar"]
	LegacyMigrated bool      `json:"legacyMigrated,omitempty"`
}

// LauncherUpdateInfo stores update metadata for the CamLauncher binary itself
type LauncherUpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	Title          string `json:"title"`
	Changelog      string `json:"changelog"`
	DownloadURL    string `json:"downloadUrl"`
	ReleaseDate    string `json:"releaseDate"`
	SHA256         string `json:"sha256,omitempty"`
}

// InstanceFileItem represents a mod, resourcepack, or shaderpack inside an instance
type InstanceFileItem struct {
	Name      string `json:"name"`
	FileName  string `json:"fileName"`
	SizeBytes int64  `json:"sizeBytes"`
	ModTime   string `json:"modTime"`
	Enabled   bool   `json:"enabled"`
	FileType  string `json:"fileType"` // "mods", "resourcepacks", "shaderpacks"
}
