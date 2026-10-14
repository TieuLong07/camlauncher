package system

import (
	"Aether/pkg/models"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx  = kernel32.NewProc("GlobalMemoryStatusEx")
)

type memorystatusex struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

var (
	cachedCPU string
	cachedGPU string
	cpuGpuOnce sync.Once
)

// GetSystemStats detects hardware specs and Java runtimes
func GetSystemStats(launcherDir string) models.SystemStats {
	stats := models.SystemStats{
		CPUCores:    runtime.NumCPU(),
		LauncherDir: launcherDir,
	}

	// 1. Memory Detection via Win32 API
	var mem memorystatusex
	mem.cbSize = uint32(unsafe.Sizeof(mem))
	r1, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if r1 != 0 {
		stats.TotalRAMMB = int(mem.ullTotalPhys / (1024 * 1024))
		stats.FreeRAMMB = int(mem.ullAvailPhys / (1024 * 1024))
	} else {
		stats.TotalRAMMB = 16384 // fallback
		stats.FreeRAMMB = 8192
	}

	// 2. CPU and GPU Detection via Environment & PowerShell / WMI (Cached once)
	cpuGpuOnce.Do(func() {
		cachedCPU = os.Getenv("PROCESSOR_IDENTIFIER")
		if cachedCPU == "" {
			cachedCPU = fmt.Sprintf("%s x%d", runtime.GOARCH, runtime.NumCPU())
		}

		cmd := exec.Command("powershell", "-NoProfile", "-Command", `
			$cpu = (Get-CimInstance Win32_Processor | Select-Object -First 1).Name;
			$gpu = (Get-CimInstance Win32_VideoController | Select-Object -ExpandProperty Name) -join ', ';
			Write-Output "$cpu|||$gpu"
		`)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if out, err := cmd.Output(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "|||")
			if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {
				cachedCPU = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
				cachedGPU = strings.TrimSpace(parts[1])
			}
		}
	})

	stats.CPUModel = cachedCPU
	stats.GPUModel = cachedGPU

	// 3. Scan for Java Runtimes
	stats.JavaRuntimes = DetectJavaRuntimes()

	return stats
}

// DetectJavaRuntimes scans standard Windows directories for installed Java JRE/JDKs
func DetectJavaRuntimes() []models.JavaRuntime {
	var runtimes []models.JavaRuntime
	seenPaths := make(map[string]bool)

	// List of potential search roots
	appData := os.Getenv("APPDATA")
	roots := []string{
		filepath.Join(appData, "CamLauncher", "runtimes"),
		`C:\Program Files\Java`,
		`C:\Program Files\Eclipse Adoptium`,
		`C:\Program Files\Zulu`,
		`C:\Program Files\BellSoft`,
		`C:\Program Files\Microsoft`,
		`C:\Program Files\Amazon Corretto`,
		`C:\Program Files (x86)\Java`,
		os.Getenv("JAVA_HOME"),
	}

	// Check PATH java first
	if pathJava, err := exec.LookPath("java.exe"); err == nil {
		if abs, err := filepath.Abs(pathJava); err == nil {
			if rt := probeJavaExecutable(abs); rt != nil {
				rt.IsDefault = true
				runtimes = append(runtimes, *rt)
				seenPaths[strings.ToLower(abs)] = true
			}
		}
	}

	for _, root := range roots {
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.EqualFold(info.Name(), "java.exe") {
				norm := strings.ToLower(path)
				if !seenPaths[norm] {
					seenPaths[norm] = true
					if rt := probeJavaExecutable(path); rt != nil {
						runtimes = append(runtimes, *rt)
					}
				}
			}
			return nil
		})
	}

	return runtimes
}

func probeJavaExecutable(javaPath string) *models.JavaRuntime {
	cmd := exec.Command(javaPath, "-version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	strOut := string(out)
	// Example: openjdk version "21.0.2" 2024-01-16
	// or: java version "1.8.0_361"
	re := regexp.MustCompile(`version "([^"]+)"`)
	matches := re.FindStringSubmatch(strOut)
	if len(matches) < 2 {
		return nil
	}

	rawVer := matches[1]
	majorVer := parseJavaMajorVersion(rawVer)
	is64Bit := strings.Contains(strOut, "64-Bit") || strings.Contains(strOut, "x86_64") || strings.Contains(strOut, "amd64")

	return &models.JavaRuntime{
		Path:         javaPath,
		Version:      rawVer,
		MajorVersion: majorVer,
		Is64Bit:      is64Bit,
	}
}

func parseJavaMajorVersion(ver string) int {
	// "1.8.0_361" -> 8
	// "17.0.8" -> 17
	// "21.0.2" -> 21
	parts := strings.Split(ver, ".")
	if len(parts) == 0 {
		return 0
	}
	if parts[0] == "1" && len(parts) > 1 {
		v, _ := strconv.Atoi(parts[1])
		return v
	}
	v, _ := strconv.Atoi(parts[0])
	return v
}

// OpenFolder opens a folder in Windows File Explorer
func OpenFolder(dirPath string) error {
	if dirPath != "" {
		_ = os.MkdirAll(dirPath, 0755)
	}
	cmd := exec.Command("explorer.exe", dirPath)
	return cmd.Start()
}
