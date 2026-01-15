package version

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/azdren/verman/internal/config"
	"github.com/azdren/verman/internal/languages"
)

type Manager struct {
	Config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{Config: cfg}
}

// ListInstalled returns all installed versions for a language
func (m *Manager) ListInstalled(langName string) ([]string, error) {
	langPath := filepath.Join(m.Config.RootPath, langName)

	entries, err := os.ReadDir(langPath)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			versions = append(versions, entry.Name())
		}
	}
	return versions, nil
}

// GetCurrent returns the current active version for a language
func (m *Manager) GetCurrent(langName string) (string, error) {
	currentPath := m.Config.GetCurrentPath(langName)

	// On Windows, check if it's a junction point
	target, err := os.Readlink(currentPath)
	if err != nil {
		// Try reading the version from config
		if langCfg, ok := m.Config.Languages[langName]; ok && langCfg.CurrentVersion != "" {
			return langCfg.CurrentVersion, nil
		}
		return "", nil
	}

	return filepath.Base(target), nil
}

// Use switches to a specific version
func (m *Manager) Use(langName, version string, global bool) error {
	lang, ok := languages.Get(langName)
	if !ok {
		return fmt.Errorf("unknown language: %s", langName)
	}

	versionPath := m.Config.GetVersionPath(langName, version)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s not installed for %s", version, langName)
	}

	currentPath := m.Config.GetCurrentPath(langName)

	// Remove existing junction/symlink (ignore errors if doesn't exist)
	removeJunction(currentPath)

	// Create junction point (Windows) or symlink (Unix)
	if err := createJunction(currentPath, versionPath); err != nil {
		return err
	}

	// Update config
	if err := m.Config.SetCurrentVersion(langName, version); err != nil {
		return err
	}

	// If global, update persistent environment variables
	if global {
		return m.SetGlobalEnv(lang, currentPath)
	}

	return nil
}

// removeJunction removes a junction point or symlink
func removeJunction(path string) {
	// On Windows, junction points are directories, use RemoveAll
	// But first check if it exists
	if _, err := os.Lstat(path); err == nil {
		if runtime.GOOS == "windows" {
			// Use rmdir for junctions to avoid deleting target contents
			exec.Command("cmd", "/c", "rmdir", path).Run()
		} else {
			os.Remove(path)
		}
	}
}

// createJunction creates a junction point (Windows) or symlink (Unix)
func createJunction(linkPath, targetPath string) error {
	if runtime.GOOS == "windows" {
		// Convert to absolute paths
		absLink, err := filepath.Abs(linkPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for link: %w", err)
		}
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for target: %w", err)
		}

		// Ensure Windows-style paths with backslashes
		absLink = filepath.Clean(absLink)
		absTarget = filepath.Clean(absTarget)

		// Convert any remaining forward slashes to backslashes
		absLink = strings.ReplaceAll(absLink, "/", "\\")
		absTarget = strings.ReplaceAll(absTarget, "/", "\\")

		// Use mklink /J for junction points (no admin required)
		cmd := exec.Command("cmd", "/c", "mklink", "/J", absLink, absTarget)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create junction: %w (output: %s, link: %s, target: %s)", err, string(output), absLink, absTarget)
		}
		return nil
	}

	// Unix: use symlink
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

// SetGlobalEnv updates persistent environment variables
func (m *Manager) SetGlobalEnv(lang languages.Language, currentPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("global env setting only supported on Windows")
	}

	// Set HOME-style variables
	for envVar, relPath := range lang.EnvVars() {
		fullPath := currentPath
		if relPath != "." {
			fullPath = filepath.Join(currentPath, relPath)
		}

		cmd := exec.Command("setx", envVar, fullPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set %s: %w", envVar, err)
		}
	}

	return nil
}

// Install downloads and installs a version
func (m *Manager) Install(langName, version string) error {
	lang, ok := languages.Get(langName)
	if !ok {
		return fmt.Errorf("unknown language: %s", langName)
	}

	if !lang.ValidateVersion(version) {
		return fmt.Errorf("invalid version format: %s", version)
	}

	versionPath := m.Config.GetVersionPath(langName, version)
	if _, err := os.Stat(versionPath); err == nil {
		return fmt.Errorf("version %s already installed", version)
	}

	url, err := lang.GetDownloadURL(version)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	fmt.Printf("Downloading %s %s from %s...\n", langName, version, url)

	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "verman-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("Extracting to %s...\n", versionPath)

	// Create version directory
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return err
	}

	// Extract zip
	if err := extractZip(tmpFile.Name(), versionPath); err != nil {
		os.RemoveAll(versionPath)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Run post-install
	if err := lang.PostInstall(versionPath); err != nil {
		return fmt.Errorf("post-install failed: %w", err)
	}

	fmt.Printf("Successfully installed %s %s\n", langName, version)
	return nil
}

// Uninstall removes an installed version
func (m *Manager) Uninstall(langName, version string) error {
	versionPath := m.Config.GetVersionPath(langName, version)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s not installed", version)
	}

	current, _ := m.GetCurrent(langName)
	if current == version {
		// Remove the junction first
		removeJunction(m.Config.GetCurrentPath(langName))
	}

	return os.RemoveAll(versionPath)
}

func extractZip(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Find common prefix (many zips have a single root folder)
	var prefix string
	if len(r.File) > 0 {
		first := r.File[0].Name
		if idx := strings.Index(first, "/"); idx > 0 {
			prefix = first[:idx+1]
			// Verify all files have this prefix
			for _, f := range r.File {
				if !strings.HasPrefix(f.Name, prefix) {
					prefix = ""
					break
				}
			}
		}
	}

	for _, f := range r.File {
		name := f.Name
		if prefix != "" {
			name = strings.TrimPrefix(name, prefix)
		}
		if name == "" {
			continue
		}

		fpath := filepath.Join(destPath, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
