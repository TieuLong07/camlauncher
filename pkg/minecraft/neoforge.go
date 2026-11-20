package minecraft

import (
	"Aether/pkg/models"
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func neoVersions(game string) ([]string, error) {
	parts := strings.Split(game, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("phiên bản Minecraft không hợp lệ: %s", game)
	}
	prefix := parts[1] + "."
	if len(parts) > 2 && parts[2] != "0" {
		prefix += parts[2] + "."
	} else {
		prefix += "0."
	}
	resp, err := http.Get("https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NeoForge metadata HTTP %d", resp.StatusCode)
	}
	var data struct {
		Versions []string `xml:"versioning>versions>version"`
	}
	if err = xml.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var versions []string
	for i := len(data.Versions) - 1; i >= 0; i-- {
		v := data.Versions[i]
		if strings.HasPrefix(v, prefix) && !strings.Contains(v, "-") {
			versions = append(versions, v)
		}
	}
	return versions, nil
}
func neoInstaller(root, version string) (string, error) {
	path := filepath.Join(root, "cache", "neoforge-"+version+"-installer.jar")
	if f, e := os.Stat(path); e == nil && f.Size() > 1000000 {
		return path, nil
	}
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return "", e
	}
	url := fmt.Sprintf("https://maven.neoforged.net/releases/net/neoforged/neoforge/%s/neoforge-%s-installer.jar", version, version)
	resp, e := http.Get(url)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("NeoForge installer HTTP %d", resp.StatusCode)
	}
	f, e := os.Create(path)
	if e != nil {
		return "", e
	}
	_, e = io.Copy(f, resp.Body)
	ce := f.Close()
	if e != nil {
		return "", e
	}
	return path, ce
}
func neoProfile(root, version string) (*models.VersionDetail, error) {
	path, e := neoInstaller(root, version)
	if e != nil {
		return nil, e
	}
	z, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == "version.json" {
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			defer r.Close()
			var p models.VersionDetail
			e = json.NewDecoder(r).Decode(&p)
			return &p, e
		}
	}
	return nil, fmt.Errorf("NeoForge installer thiếu version.json")
}
func ensureNeoInstalled(root, java, version string) error {
	marker := filepath.Join(root, "libraries", "net", "neoforged", "neoforge", version, "neoforge-"+version+"-client.jar")
	if f, e := os.Stat(marker); e == nil && f.Size() > 0 {
		return nil
	}
	p, e := neoInstaller(root, version)
	if e != nil {
		return e
	}
	if strings.EqualFold(filepath.Base(java), "javaw.exe") {
		java = filepath.Join(filepath.Dir(java), "java.exe")
	}
	cmd := exec.Command(java, "-jar", p, "--installClient", root)
	cmd.Dir = filepath.Dir(p)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	out, e := cmd.CombinedOutput()
	log := filepath.Join(root, "cache", "neoforge-"+version+"-install.log")
	_ = os.WriteFile(log, out, 0644)
	if e != nil {
		return fmt.Errorf("cài NeoForge thất bại: %w (log: %s)", e, log)
	}
	if f, e := os.Stat(marker); e != nil || f.Size() == 0 {
		return fmt.Errorf("cài NeoForge chưa đầy đủ (log: %s)", log)
	}
	return nil
}
