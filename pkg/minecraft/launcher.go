package minecraft

import (
	"Aether/pkg/java"
	"Aether/pkg/models"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type LaunchOptions struct {
	Instance       *models.Instance
	Account        *models.Account
	LauncherDir    string
	JavaExecutable string
	ProgressFn     func(models.ProgressEvent)
	LogFn          func(models.LogEvent)
}

// PrepareAndLaunch executes the full preparation pipeline and launches the game
func PrepareAndLaunch(opts LaunchOptions) (*exec.Cmd, error) {
	inst := opts.Instance
	acc := opts.Account
	sharedDir := opts.LauncherDir
	instanceDir := inst.Path
	if instanceDir == "" {
		instanceDir = filepath.Join(sharedDir, "instances", inst.ID)
	}

	_ = os.MkdirAll(instanceDir, 0755)
	_ = os.MkdirAll(filepath.Join(instanceDir, "mods"), 0755)
	_ = os.MkdirAll(filepath.Join(instanceDir, "saves"), 0755)
	_ = os.MkdirAll(filepath.Join(instanceDir, "resourcepacks"), 0755)
	_ = os.MkdirAll(filepath.Join(instanceDir, "shaderpacks"), 0755)

	versionsDir := filepath.Join(sharedDir, "versions")
	librariesDir := filepath.Join(sharedDir, "libraries")
	assetsDir := filepath.Join(sharedDir, "assets")
	nativesDir := filepath.Join(instanceDir, "natives")
	_ = os.MkdirAll(nativesDir, 0755)

	// Step 1: Resolve Version Profile (Local instance JSON, Fabric, Forge/NeoForge, or Vanilla)
	var versionDetail *models.VersionDetail
	var localProfilePath string

	// 1a. Check if instance directory contains a custom version JSON (e.g. Cooking TQ / TLauncher / CurseForge profile)
	if localDetail, pPath, err := FindInstanceVersionProfile(instanceDir); err == nil && localDetail != nil {
		versionDetail = localDetail
		localProfilePath = pPath
		if opts.LogFn != nil {
			opts.LogFn(models.LogEvent{
				InstanceID: inst.ID,
				Level:      "INFO",
				Line:       fmt.Sprintf("[CamLauncher] Sử dụng cấu hình profile cục bộ: %s", filepath.Base(pPath)),
				Timestamp:  time.Now().Format("15:04:05"),
			})
		}
	}

	// 1b. If no local profile found, fetch Mojang Vanilla manifest and base profile
	if versionDetail == nil {
		if opts.ProgressFn != nil {
			opts.ProgressFn(models.ProgressEvent{
				InstanceID: inst.ID,
				Stage:      "checking",
				Message:    "Đang kiểm tra thông tin phiên bản...",
				Percentage: 5,
			})
		}

		manifest, err := FetchVersionManifest(sharedDir)
		if err != nil {
			return nil, fmt.Errorf("lỗi lấy danh sách phiên bản: %w", err)
		}

		var targetVersionURL string
		for _, v := range manifest.Versions {
			if v.ID == inst.GameVersion {
				targetVersionURL = v.URL
				break
			}
		}
		if targetVersionURL == "" {
			return nil, fmt.Errorf("không tìm thấy phiên bản Minecraft: %s", inst.GameVersion)
		}

		// Fetch Version Detail JSON
		vanillaVersionFile := filepath.Join(versionsDir, inst.GameVersion, inst.GameVersion+".json")
		vanillaDetail, err := FetchVersionDetail(targetVersionURL, vanillaVersionFile)
		if err != nil {
			return nil, fmt.Errorf("lỗi lấy chi tiết phiên bản: %w", err)
		}
		versionDetail = vanillaDetail

		// 1c. If Fabric loader
		if strings.EqualFold(inst.Loader, "fabric") {
			loaderVer := inst.LoaderVersion
			if loaderVer == "" {
				latestLoader, err := GetLatestFabricLoader(inst.GameVersion)
				if err != nil {
					return nil, fmt.Errorf("lỗi lấy Fabric loader: %w", err)
				}
				loaderVer = latestLoader
			}
			fabDetail, err := FetchFabricVersionProfile(inst.GameVersion, loaderVer)
			if err != nil {
				return nil, fmt.Errorf("lỗi tải cấu hình Fabric: %w", err)
			}
			if fabDetail.MainClass != "" {
				versionDetail.MainClass = fabDetail.MainClass
			}
			if fabDetail.Arguments != nil {
				versionDetail.Arguments = fabDetail.Arguments
			}
			versionDetail.Libraries = append(versionDetail.Libraries, fabDetail.Libraries...)
		} else if strings.EqualFold(inst.Loader, "forge") || strings.EqualFold(inst.Loader, "neoforge") {
			// 1d. If Forge / NeoForge loader without local JSON profile
			loaderVer := inst.LoaderVersion
			if loaderVer == "" {
				recVer, err := GetRecommendedForgeVersion(inst.GameVersion)
				if err == nil && recVer != "" {
					loaderVer = recVer
				}
			}
			if loaderVer != "" {
				forgeDetail, err := GetForgeVersionProfile(sharedDir, inst.GameVersion, loaderVer, opts.ProgressFn)
				if err != nil {
					return nil, fmt.Errorf("lỗi tải cấu hình Forge: %w", err)
				}
				if forgeDetail.MainClass != "" {
					versionDetail.MainClass = forgeDetail.MainClass
				}
				if forgeDetail.Arguments != nil {
					versionDetail.Arguments = forgeDetail.Arguments
				}
				versionDetail.Libraries = append(versionDetail.Libraries, forgeDetail.Libraries...)
			}
		}
	} else {
		// If local profile was used, ensure Downloads["client"] and AssetIndex are filled if missing
		if _, hasClient := versionDetail.Downloads["client"]; !hasClient || versionDetail.AssetIndex.ID == "" {
			manifest, err := FetchVersionManifest(sharedDir)
			if err == nil {
				for _, v := range manifest.Versions {
					if v.ID == inst.GameVersion {
						vanillaVersionFile := filepath.Join(versionsDir, inst.GameVersion, inst.GameVersion+".json")
						if vDet, err := FetchVersionDetail(v.URL, vanillaVersionFile); err == nil {
							if !hasClient && vDet.Downloads != nil {
								if versionDetail.Downloads == nil {
									versionDetail.Downloads = make(map[string]models.DownloadInfo)
								}
								if cDl, ok := vDet.Downloads["client"]; ok {
									versionDetail.Downloads["client"] = cDl
								}
							}
							if versionDetail.AssetIndex.ID == "" {
								versionDetail.AssetIndex = vDet.AssetIndex
							}
						}
						break
					}
				}
			}
		}
	}

	// Step 3: Download Client.jar
	clientJarPath := filepath.Join(versionsDir, inst.GameVersion, inst.GameVersion+".jar")
	if localProfilePath != "" {
		possibleLocalJar := strings.TrimSuffix(localProfilePath, filepath.Ext(localProfilePath)) + ".jar"
		if fi, err := os.Stat(possibleLocalJar); err == nil && fi.Size() > 10000000 {
			if _, err := os.Stat(clientJarPath); os.IsNotExist(err) {
				_ = os.MkdirAll(filepath.Dir(clientJarPath), 0755)
				_ = copyFile(possibleLocalJar, clientJarPath)
			}
		}
	}

	clientDownload, ok := versionDetail.Downloads["client"]
	if ok && clientDownload.URL != "" {
		if !verifyFileSHA1(clientJarPath, clientDownload.SHA1) {
			if opts.ProgressFn != nil {
				opts.ProgressFn(models.ProgressEvent{
					InstanceID: inst.ID,
					Stage:      "downloading_client",
					Message:    fmt.Sprintf("Đang tải client.jar cho Minecraft %s...", inst.GameVersion),
					Percentage: 15,
				})
			}
			tasks := []DownloadTask{
				{
					URL:  clientDownload.URL,
					Path: clientJarPath,
					SHA1: clientDownload.SHA1,
					Size: clientDownload.Size,
				},
			}
			if err := DownloadBatch(tasks, 1, inst.ID, "downloading_client", opts.ProgressFn); err != nil {
				return nil, fmt.Errorf("lỗi tải client.jar: %w", err)
			}
		}
	}

	// Step 4: Resolve & Download Libraries
	allLibraries := versionDetail.Libraries

	var libTasks []DownloadTask
	var classpathEntries []string

	for _, lib := range allLibraries {
		if !isLibraryAllowedOnOS(lib.Rules, "windows") {
			continue
		}

		// Standard Mojang / Forge artifact
		if lib.Downloads != nil && lib.Downloads.Artifact != nil {
			art := lib.Downloads.Artifact
			targetPath := filepath.Join(librariesDir, filepath.FromSlash(art.Path))
			classpathEntries = append(classpathEntries, targetPath)

			if !verifyFileSHA1(targetPath, art.SHA1) {
				libTasks = append(libTasks, DownloadTask{
					URL:  art.URL,
					Path: targetPath,
					SHA1: art.SHA1,
					Size: art.Size,
				})
			}
		} else if lib.Name != "" {
			// Maven coordinate: group:name:version
			relPath := mavenCoordinateToPath(lib.Name)
			targetPath := filepath.Join(librariesDir, filepath.FromSlash(relPath))
			classpathEntries = append(classpathEntries, targetPath)

			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				baseURL := "https://libraries.minecraft.net/"
				if lib.URL != "" {
					baseURL = lib.URL
				} else if strings.EqualFold(inst.Loader, "fabric") {
					baseURL = "https://maven.fabricmc.net/"
				} else if strings.EqualFold(inst.Loader, "forge") || strings.EqualFold(inst.Loader, "neoforge") {
					baseURL = "https://maven.minecraftforge.net/"
				}
				fullURL := baseURL + relPath
				libTasks = append(libTasks, DownloadTask{
					URL:  fullURL,
					Path: targetPath,
				})
			}
		}
	}

	if len(libTasks) > 0 {
		if opts.ProgressFn != nil {
			opts.ProgressFn(models.ProgressEvent{
				InstanceID: inst.ID,
				Stage:      "downloading_libraries",
				Message:    fmt.Sprintf("Đang tải %d thư viện...", len(libTasks)),
				Percentage: 30,
			})
		}
		if err := DownloadBatch(libTasks, 16, inst.ID, "downloading_libraries", opts.ProgressFn); err != nil {
			return nil, fmt.Errorf("lỗi tải thư viện: %w", err)
		}
	}

	// Step 5: Download Asset Index & Objects
	assetIndexPath := filepath.Join(assetsDir, "indexes", versionDetail.AssetIndex.ID+".json")
	if !verifyFileSHA1(assetIndexPath, versionDetail.AssetIndex.SHA1) {
		tasks := []DownloadTask{
			{
				URL:  versionDetail.AssetIndex.URL,
				Path: assetIndexPath,
				SHA1: versionDetail.AssetIndex.SHA1,
			},
		}
		_ = DownloadBatch(tasks, 1, inst.ID, "downloading_assets", opts.ProgressFn)
	}

	// Parse Asset Index
	var assetTasks []DownloadTask
	if indexData, err := os.ReadFile(assetIndexPath); err == nil {
		type AssetObject struct {
			Hash string `json:"hash"`
			Size int64  `json:"size"`
		}
		type AssetIndex struct {
			Objects map[string]AssetObject `json:"objects"`
		}
		var parsedIndex AssetIndex
		if err := json.Unmarshal(indexData, &parsedIndex); err == nil {
			for _, obj := range parsedIndex.Objects {
				prefix := obj.Hash[0:2]
				objPath := filepath.Join(assetsDir, "objects", prefix, obj.Hash)
				if !verifyFileSHA1(objPath, obj.Hash) {
					assetTasks = append(assetTasks, DownloadTask{
						URL:  fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", prefix, obj.Hash),
						Path: objPath,
						SHA1: obj.Hash,
						Size: obj.Size,
					})
				}
			}
		}
	}

	if len(assetTasks) > 0 {
		if opts.ProgressFn != nil {
			opts.ProgressFn(models.ProgressEvent{
				InstanceID: inst.ID,
				Stage:      "downloading_assets",
				Message:    fmt.Sprintf("Đang đồng bộ %d tài nguyên âm thanh/hình ảnh...", len(assetTasks)),
				Percentage: 60,
			})
		}
		_ = DownloadBatch(assetTasks, 24, inst.ID, "downloading_assets", opts.ProgressFn)
	}

	// Step 6: Construct Classpath
	classpathEntries = append(classpathEntries, clientJarPath)
	fullClasspath := strings.Join(classpathEntries, string(filepath.ListSeparator))

	// Step 7: Java and Arguments
	javaExe := opts.JavaExecutable
	needsJavaResolution := false
	if javaExe == "" {
		needsJavaResolution = true
	} else if fi, statErr := os.Stat(javaExe); statErr != nil || fi.IsDir() {
		needsJavaResolution = true
	}

	if needsJavaResolution {
		reqVer := java.GetRequiredJavaVersion(inst.GameVersion, versionDetail)
		if opts.ProgressFn != nil {
			opts.ProgressFn(models.ProgressEvent{
				InstanceID: inst.ID,
				Stage:      "checking_java",
				Message:    fmt.Sprintf("Đang chuẩn bị Java %d cho Minecraft %s...", reqVer, inst.GameVersion),
				Percentage: 70,
			})
		}
		ensuredJava, err := java.EnsureJavaRuntime(sharedDir, reqVer, opts.ProgressFn)
		if err != nil {
			return nil, fmt.Errorf("không thể tự động chuẩn bị Java %d: %w", reqVer, err)
		}
		javaExe = ensuredJava
	}

	// Ensure Forge Client Libraries are installed if Forge loader is used
	isForge := strings.EqualFold(inst.Loader, "forge") ||
		strings.Contains(strings.ToLower(versionDetail.MainClass), "bootstraplauncher") ||
		strings.Contains(strings.ToLower(versionDetail.MainClass), "modlauncher")
	if isForge {
		forgeVer := ExtractForgeVersion(inst, versionDetail)
		_ = EnsureForgeClientInstalled(sharedDir, javaExe, inst.GameVersion, forgeVer, opts.ProgressFn)
	}

	minRAM := inst.MinMemoryMB
	if minRAM <= 0 {
		minRAM = 1024
	}
	maxRAM := inst.MaxMemoryMB
	if maxRAM <= 0 {
		maxRAM = 4096
	}

	userType := "mojang"
	if acc.Type == models.AccountMicrosoft {
		userType = "msa"
	}

	replacements := map[string]string{
		"${auth_player_name}":    acc.Username,
		"${version_name}":        inst.GameVersion,
		"${game_directory}":      instanceDir,
		"${assets_root}":         assetsDir,
		"${assets_index_name}":   versionDetail.AssetIndex.ID,
		"${auth_uuid}":           acc.UUID,
		"${auth_access_token}":   acc.AccessToken,
		"${clientid}":            "ticalauncher",
		"${auth_xuid}":           "",
		"${user_type}":           userType,
		"${version_type}":        "CamLauncher",
		"${resolution_width}":    "854",
		"${resolution_height}":   "480",
		"${natives_directory}":   nativesDir,
		"${launcher_name}":       "CamLauncher",
		"${launcher_version}":    "1.0.0",
		"${classpath}":           fullClasspath,
		"${library_directory}":   filepath.ToSlash(librariesDir),
		"${classpath_separator}": string(filepath.ListSeparator),
		"${path}":                clientJarPath,
	}

	var evaluatedJVMArgs []string
	var evaluatedGameArgs []string

	if versionDetail.Arguments != nil {
		if len(versionDetail.Arguments.JVM) > 0 {
			evaluatedJVMArgs = evaluateArgumentList(versionDetail.Arguments.JVM, replacements)
		}
		if len(versionDetail.Arguments.Game) > 0 {
			evaluatedGameArgs = evaluateArgumentList(versionDetail.Arguments.Game, replacements)
		}
	}

	if len(evaluatedGameArgs) == 0 {
		if versionDetail.MinecraftArguments != "" {
			legacyParts := strings.Fields(versionDetail.MinecraftArguments)
			for _, p := range legacyParts {
				val := p
				for k, v := range replacements {
					val = strings.ReplaceAll(val, k, v)
				}
				evaluatedGameArgs = append(evaluatedGameArgs, val)
			}
		} else {
			evaluatedGameArgs = []string{
				"--username", acc.Username,
				"--version", inst.GameVersion,
				"--gameDir", instanceDir,
				"--assetsDir", assetsDir,
				"--assetIndex", versionDetail.AssetIndex.ID,
				"--uuid", acc.UUID,
				"--accessToken", acc.AccessToken,
				"--userType", userType,
				"--versionType", "CamLauncher",
			}
		}
	}

	var cmdArgs []string

	// High Performance JVM GC Flags & RAM
	cmdArgs = append(cmdArgs, fmt.Sprintf("-Xms%dM", minRAM))
	cmdArgs = append(cmdArgs, fmt.Sprintf("-Xmx%dM", maxRAM))
	cmdArgs = append(cmdArgs,
		"-XX:+UseG1GC",
		"-XX:+ParallelRefProcEnabled",
		"-XX:MaxGCPauseMillis=200",
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+DisableExplicitGC",
		"-XX:+AlwaysPreTouch",
		"-XX:G1NewSizePercent=30",
		"-XX:G1MaxNewSizePercent=40",
		"-XX:G1ReservePercent=20",
		"-XX:G1HeapWastePercent=5",
		"-XX:G1MixedGCCountTarget=4",
		"-XX:InitiatingHeapOccupancyPercent=15",
		"-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:G1RSetUpdatingPauseTimePercent=5",
		"-XX:SurvivorRatio=32",
		"-XX:+PerfDisableSharedMem",
		"-XX:MaxTenuringThreshold=1",
	)

	// User custom JVM args
	if inst.JVMArgs != "" {
		userTokens := strings.Fields(inst.JVMArgs)
		cmdArgs = append(cmdArgs, userTokens...)
	}

	// Evaluated JVM Arguments from profile
	cmdArgs = append(cmdArgs, evaluatedJVMArgs...)

	// Ensure Classpath flag exists
	hasClasspath := false
	for _, a := range evaluatedJVMArgs {
		if a == "-cp" || a == "-classpath" {
			hasClasspath = true
			break
		}
	}
	if !hasClasspath {
		cmdArgs = append(cmdArgs, "-cp", fullClasspath)
	}

	// Ensure Native Library Path exists if not in evaluated args
	hasNativePath := false
	for _, a := range evaluatedJVMArgs {
		if strings.HasPrefix(a, "-Djava.library.path=") {
			hasNativePath = true
			break
		}
	}
	if !hasNativePath {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-Djava.library.path=%s", nativesDir))
	}

	// Main Class
	mainClass := versionDetail.MainClass
	if mainClass == "" {
		mainClass = "net.minecraft.client.main.Main"
	}
	cmdArgs = append(cmdArgs, mainClass)

	// Game Arguments
	cmdArgs = append(cmdArgs, evaluatedGameArgs...)

	if opts.ProgressFn != nil {
		opts.ProgressFn(models.ProgressEvent{
			InstanceID: inst.ID,
			Stage:      "ready",
			Message:    "Khởi chạy Minecraft!",
			Percentage: 100,
		})
	}

	if opts.LogFn != nil {
		opts.LogFn(models.LogEvent{
			InstanceID: inst.ID,
			Level:      "INFO",
			Line:       fmt.Sprintf("[CamLauncher] Khởi chạy: %s %s", javaExe, strings.Join(cmdArgs, " ")),
			Timestamp:  time.Now().Format("15:04:05"),
		})
	}

	// Step 8: Spawn Process
	// On Windows, prefer javaw.exe if available so Windows does not allocate a console window
	if runtime.GOOS == "windows" && strings.EqualFold(filepath.Base(javaExe), "java.exe") {
		javawCandidate := filepath.Join(filepath.Dir(javaExe), "javaw.exe")
		if _, err := os.Stat(javawCandidate); err == nil {
			javaExe = javawCandidate
		}
	}

	cmd := exec.Command(javaExe, cmdArgs...)
	cmd.Dir = instanceDir
	if runtime.GOOS == "windows" {
		if strings.EqualFold(filepath.Base(javaExe), "java.exe") {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: 0x08000000, // CREATE_NO_WINDOW prevents console popup without hiding GUI
			}
		}
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("không thể khởi chạy tiến trình Java: %w", err)
	}

	logFile, _ := os.OpenFile(filepath.Join(instanceDir, "launcher_output.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	// Stream stdout & stderr
	go streamLog(inst.ID, stdout, "STDOUT", opts.LogFn, logFile)
	go streamLog(inst.ID, stderr, "STDERR", opts.LogFn, logFile)

	return cmd, nil
}

func streamLog(instanceID string, r io.Reader, defaultLevel string, logFn func(models.LogEvent), logFile *os.File) {
	if r == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if logFile != nil {
			_, _ = logFile.WriteString(text + "\n")
		}
		if logFn != nil {
			level := defaultLevel
			if strings.Contains(text, "WARN") {
				level = "WARN"
			} else if strings.Contains(text, "ERROR") || strings.Contains(text, "Exception") || strings.Contains(text, "Crash") {
				level = "ERROR"
			} else if strings.Contains(text, "[CHAT]") {
				level = "CHAT"
			}

			logFn(models.LogEvent{
				InstanceID: instanceID,
				Level:      level,
				Line:       text,
				Timestamp:  time.Now().Format("15:04:05"),
			})
		}
	}
}

func isLibraryAllowedOnOS(rules []models.Rule, currentOS string) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false
	for _, rule := range rules {
		matchesOS := true
		if rule.OS != nil && rule.OS["name"] != "" {
			matchesOS = (rule.OS["name"] == currentOS)
		}
		if matchesOS {
			allowed = (rule.Action == "allow")
		}
	}
	return allowed
}

func mavenCoordinateToPath(coordinate string) string {
	parts := strings.Split(coordinate, ":")
	if len(parts) < 3 {
		return coordinate
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]
	classifier := ""
	if len(parts) >= 4 {
		classifier = "-" + parts[3]
	}

	return fmt.Sprintf("%s/%s/%s/%s-%s%s.jar", group, artifact, version, artifact, version, classifier)
}

func evaluateArgumentList(args []interface{}, replacements map[string]string) []string {
	var result []string

	replaceTokens := func(str string) string {
		res := str
		for k, v := range replacements {
			res = strings.ReplaceAll(res, k, v)
		}
		return res
	}

	for _, raw := range args {
		switch item := raw.(type) {
		case string:
			result = append(result, replaceTokens(item))
		case map[string]interface{}:
			if rulesRaw, ok := item["rules"]; ok {
				if !isArgumentAllowed(rulesRaw) {
					continue
				}
			}
			if valRaw, ok := item["value"]; ok {
				switch val := valRaw.(type) {
				case string:
					result = append(result, replaceTokens(val))
				case []interface{}:
					for _, sub := range val {
						if subStr, ok := sub.(string); ok {
							result = append(result, replaceTokens(subStr))
						}
					}
				}
			}
		}
	}

	return result
}

func isArgumentAllowed(rulesRaw interface{}) bool {
	rulesList, ok := rulesRaw.([]interface{})
	if !ok || len(rulesList) == 0 {
		return true
	}

	allowed := false
	for _, rRaw := range rulesList {
		rMap, ok := rRaw.(map[string]interface{})
		if !ok {
			continue
		}

		action, _ := rMap["action"].(string)
		isAllow := action == "allow" || action == ""

		// Check features: is_demo_user, has_custom_resolution, has_quick_plays_support
		if featMap, ok := rMap["features"].(map[string]interface{}); ok && len(featMap) > 0 {
			featureMatches := false
			if !featureMatches {
				if isAllow {
					continue
				}
			}
		}

		// Check OS
		if osMap, ok := rMap["os"].(map[string]interface{}); ok {
			osName, _ := osMap["name"].(string)
			osArch, _ := osMap["arch"].(string)

			osMatches := true
			if osName != "" {
				currentOS := runtime.GOOS
				if osName == "osx" && currentOS == "darwin" {
					osMatches = true
				} else if osName != currentOS {
					osMatches = false
				}
			}

			if osArch != "" {
				currentArch := runtime.GOARCH
				if osArch == "x86" && currentArch != "386" {
					osMatches = false
				}
			}

			if !osMatches {
				continue
			}
		}

		allowed = isAllow
	}

	return allowed
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
