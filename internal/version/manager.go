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

	// Check dependencies and warn if missing
	m.checkAndWarnDependencies(lang)

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

	// Create shims for this language
	if err := m.CreateShims(langName, currentPath, lang.PathDirs()); err != nil {
		// Non-fatal, just warn
		fmt.Fprintf(os.Stderr, "Warning: failed to create shims: %v\n", err)
	}

	// If global, update persistent environment variables
	if global {
		return m.SetGlobalEnv(lang, currentPath)
	}

	// For Java, set JAVA_HOME in current process and check if globally set
	if langName == "java" && runtime.GOOS == "windows" {
		absPath, _ := filepath.Abs(versionPath)

		// Check if JAVA_HOME was already set in current session (e.g., by install)
		existingJavaHome := os.Getenv("JAVA_HOME")

		// Always set in current process for immediate use
		os.Setenv("JAVA_HOME", absPath)

		// Only show the warning if JAVA_HOME wasn't already set in this session
		// (meaning we didn't just set it globally during install)
		if existingJavaHome == "" {
			// Check if globally set (in registry/profile)
			globalJavaHome, _ := getUserEnvVar("JAVA_HOME")
			if globalJavaHome == "" {
				fmt.Printf("\nNote: JAVA_HOME is set for this session but not globally.\n")
				fmt.Printf("  To set globally, run: setx JAVA_HOME \"%s\"\n", absPath)
				fmt.Printf("  Then restart your terminal.\n")
			}
		}
	}

	return nil
}

// CreateShims creates wrapper scripts in ~/.verman/bin for all executables
func (m *Manager) CreateShims(langName, currentPath string, pathDirs []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	shimDir := filepath.Join(home, ".verman", "bin")

	// Ensure shim directory exists
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return err
	}

	// Find all executables in the path dirs
	for _, relDir := range pathDirs {
		binDir := currentPath
		if relDir != "." {
			binDir = filepath.Join(currentPath, relDir)
		}

		entries, err := os.ReadDir(binDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			// Only create shims for .exe, .cmd, .bat files on Windows
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".exe" && ext != ".cmd" && ext != ".bat" {
				continue
			}

			baseName := strings.TrimSuffix(name, ext)
			shimPath := filepath.Join(shimDir, baseName+".cmd")
			targetPath := filepath.Join(binDir, name)

			// Create shim script
			shimContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", targetPath)
			if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
				continue // Skip on error, non-fatal
			}
		}
	}

	return nil
}

// removeJunction removes a junction point or symlink
func removeJunction(path string) {
	// On Windows, junction points are directories, use RemoveAll
	// But first check if it exists
	if _, err := os.Lstat(path); err == nil {
		if runtime.GOOS == "windows" {
			// Try cmd first, fall back to PowerShell for Nano Server
			err := exec.Command("cmd", "/c", "rmdir", path).Run()
			if err != nil {
				// Fallback: Use PowerShell Remove-Item
				exec.Command("pwsh", "-NoProfile", "-Command",
					fmt.Sprintf("Remove-Item -Path '%s' -Force", path)).Run()
			}
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

		// Try cmd first (full Windows), fall back to PowerShell (Nano Server)
		cmd := exec.Command("cmd", "/c", "mklink", "/J", absLink, absTarget)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}

		// Fallback: Use PowerShell New-Item for Nano Server
		psCmd := fmt.Sprintf("New-Item -ItemType Junction -Path '%s' -Target '%s'", absLink, absTarget)
		cmd = exec.Command("pwsh", "-NoProfile", "-Command", psCmd)
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Try powershell.exe as last resort
			cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to create junction: %w (output: %s)", err, string(output))
			}
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

// checkAndWarnDependencies checks if dependencies are installed and warns if not
func (m *Manager) checkAndWarnDependencies(lang languages.Language) {
	deps := lang.GetDependencies()
	if len(deps) == 0 {
		return
	}

	var missing []string
	for _, dep := range deps {
		installed, err := m.ListInstalled(dep)
		if err != nil || len(installed) == 0 {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("\nWarning: %s requires %s, but %s is not installed.\n",
			lang.Name(), missing[0], missing[0])
		fmt.Printf("  Tip: Install %s first with: verman install %s <version>\n\n",
			missing[0], missing[0])
	}
}

// Install downloads and installs a version (uses default distribution)
func (m *Manager) Install(langName, version string) error {
	return m.InstallWithDist(langName, version, "")
}

// InstallWithDist downloads and installs a version with a specific distribution
func (m *Manager) InstallWithDist(langName, version, dist string) error {
	lang, ok := languages.Get(langName)
	if !ok {
		return fmt.Errorf("unknown language: %s", langName)
	}

	if !lang.ValidateVersion(version) {
		return fmt.Errorf("invalid version format: %s", version)
	}

	// Check dependencies and warn if missing
	m.checkAndWarnDependencies(lang)

	// Construct version path (include distribution in the folder name for Java)
	versionKey := version
	if dist != "" {
		versionKey = version + "-" + dist
	}
	versionPath := m.Config.GetVersionPath(langName, versionKey)
	if _, err := os.Stat(versionPath); err == nil {
		return fmt.Errorf("version %s already installed", versionKey)
	}

	url, err := lang.GetDownloadURLWithDist(version, dist)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	displayVer := version
	if dist != "" {
		displayVer = version + "-" + dist
	}
	fmt.Printf("Downloading %s %s from %s...\n", langName, displayVer, url)

	// Create version directory
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return err
	}

	// Check download type
	downloadType := lang.GetDownloadType()

	if downloadType == "file" {
		// Single file download - save directly to version directory
		fileName := filepath.Base(url)
		destPath := filepath.Join(versionPath, fileName)

		resp, err := http.Get(url)
		if err != nil {
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			os.RemoveAll(versionPath)
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(outFile, resp.Body); err != nil {
			outFile.Close()
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: %w", err)
		}
		outFile.Close()

		fmt.Printf("Downloaded to %s\n", versionPath)
	} else {
		// Zip archive download
		tmpFile, err := os.CreateTemp("", "verman-*.zip")
		if err != nil {
			os.RemoveAll(versionPath)
			return err
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Download
		resp, err := http.Get(url)
		if err != nil {
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}

		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			os.RemoveAll(versionPath)
			return fmt.Errorf("download failed: %w", err)
		}

		fmt.Printf("Extracting to %s...\n", versionPath)

		// Extract zip
		if err := extractZip(tmpFile.Name(), versionPath); err != nil {
			os.RemoveAll(versionPath)
			return fmt.Errorf("extraction failed: %w", err)
		}
	}

	// Run post-install
	if err := lang.PostInstall(versionPath); err != nil {
		return fmt.Errorf("post-install failed: %w", err)
	}

	fmt.Printf("Successfully installed %s %s\n", langName, displayVer)

	// For Java, offer to set JAVA_HOME globally
	if langName == "java" && runtime.GOOS == "windows" {
		m.offerJavaHomeSetup(versionPath)
	}

	// Remind about missing dependencies after install
	deps := lang.GetDependencies()
	for _, dep := range deps {
		installed, err := m.ListInstalled(dep)
		if err != nil || len(installed) == 0 {
			fmt.Printf("\nNote: %s requires %s to be configured. Run: verman install %s <version>\n",
				langName, strings.ToUpper(dep+"_HOME"), dep)
		}
	}

	return nil
}

// offerJavaHomeSetup prompts user to set JAVA_HOME globally after Java installation
func (m *Manager) offerJavaHomeSetup(javaPath string) {
	// Check if JAVA_HOME is already set
	currentJavaHome := os.Getenv("JAVA_HOME")
	if currentJavaHome != "" {
		fmt.Printf("\nJAVA_HOME is currently set to: %s\n", currentJavaHome)
	}

	fmt.Printf("\nSet JAVA_HOME globally? [Y/n] ")
	var response string
	fmt.Scanln(&response)
	if response == "" || response == "y" || response == "Y" {
		absPath, err := filepath.Abs(javaPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
			return
		}

		if err := setUserEnvVar("JAVA_HOME", absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting JAVA_HOME: %v\n", err)
			fmt.Printf("You can manually set it with:\n")
			fmt.Printf("  PowerShell: [Environment]::SetEnvironmentVariable('JAVA_HOME', '%s', 'User')\n", absPath)
			fmt.Printf("  Or: setx JAVA_HOME \"%s\"\n", absPath)
			return
		}

		// Verify the write succeeded
		verifyValue, verifyErr := getUserEnvVar("JAVA_HOME")
		if verifyErr != nil || verifyValue != absPath {
			fmt.Printf("JAVA_HOME was set, but verification failed.\n")
			fmt.Printf("The change may not persist. To ensure it works, run:\n")
			fmt.Printf("  setx JAVA_HOME \"%s\"\n", absPath)
		} else {
			fmt.Printf("JAVA_HOME set to: %s\n", absPath)
		}

		// Always show how to apply immediately
		fmt.Printf("\nTo apply in current terminal, run:\n")
		fmt.Printf("  $env:JAVA_HOME = \"%s\"\n", absPath)
		fmt.Printf("\nOr refresh all verman environment variables:\n")
		fmt.Printf("  verman env | Invoke-Expression\n")
	}
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
