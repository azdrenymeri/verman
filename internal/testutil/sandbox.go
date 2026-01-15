// Package testutil provides testing utilities for verman
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Sandbox represents a test sandbox environment that simulates
// a user's home directory with verman installed
type Sandbox struct {
	T       *testing.T
	RootDir string // Temp directory acting as "home"

	// Simulated paths
	VermanDir   string // ~/.verman
	VersionsDir string // ~/.verman/versions
	ConfigFile  string // ~/.verman/config.json
}

// NewSandbox creates a new test sandbox with a temp directory structure
func NewSandbox(t *testing.T) *Sandbox {
	t.Helper()

	rootDir := t.TempDir() // Automatically cleaned up after test

	sb := &Sandbox{
		T:           t,
		RootDir:     rootDir,
		VermanDir:   filepath.Join(rootDir, ".verman"),
		VersionsDir: filepath.Join(rootDir, ".verman", "versions"),
		ConfigFile:  filepath.Join(rootDir, ".verman", "config.json"),
	}

	// Create directory structure
	dirs := []string{
		sb.VermanDir,
		sb.VersionsDir,
		filepath.Join(sb.VersionsDir, "java"),
		filepath.Join(sb.VersionsDir, "node"),
		filepath.Join(sb.VersionsDir, "scala"),
		filepath.Join(sb.VersionsDir, "python"),
		filepath.Join(sb.VersionsDir, "ruby"),
		filepath.Join(sb.VersionsDir, "go"),
		filepath.Join(sb.VersionsDir, "rust"),
		filepath.Join(sb.VersionsDir, "dotnet"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create sandbox directory %s: %v", dir, err)
		}
	}

	return sb
}

// CreateMockVersion creates a fake installed version in the sandbox
// This simulates what an installed version looks like without downloading
func (sb *Sandbox) CreateMockVersion(lang, version string) string {
	sb.T.Helper()

	versionDir := filepath.Join(sb.VersionsDir, lang, version)
	binDir := filepath.Join(versionDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		sb.T.Fatalf("Failed to create mock version dir: %v", err)
	}

	// Create mock executables based on language
	var exeName string
	switch lang {
	case "java":
		exeName = "java.exe"
	case "node":
		// Node has exe in root, not bin
		exeName = filepath.Join("..", "node.exe")
	case "scala":
		exeName = "scala.bat"
	case "python":
		exeName = filepath.Join("..", "python.exe")
	case "go":
		exeName = "go.exe"
	case "ruby":
		exeName = "ruby.exe"
	case "rust":
		binDir = filepath.Join(versionDir, "cargo", "bin")
		os.MkdirAll(binDir, 0755)
		exeName = "rustc.exe"
	case "dotnet":
		exeName = filepath.Join("..", "dotnet.exe")
	}

	if exeName != "" {
		exePath := filepath.Join(binDir, exeName)
		os.MkdirAll(filepath.Dir(exePath), 0755)
		// Create empty file as mock executable
		if err := os.WriteFile(exePath, []byte("mock"), 0755); err != nil {
			sb.T.Fatalf("Failed to create mock executable: %v", err)
		}
	}

	return versionDir
}

// CreateVersionFile creates a version file in a project directory
func (sb *Sandbox) CreateVersionFile(projectDir, filename, content string) string {
	sb.T.Helper()

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		sb.T.Fatalf("Failed to create project dir: %v", err)
	}

	filePath := filepath.Join(projectDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		sb.T.Fatalf("Failed to create version file: %v", err)
	}

	return filePath
}

// CreateProject creates a mock project directory with version files
func (sb *Sandbox) CreateProject(name string, versionFiles map[string]string) string {
	sb.T.Helper()

	projectDir := filepath.Join(sb.RootDir, "projects", name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		sb.T.Fatalf("Failed to create project dir: %v", err)
	}

	for filename, content := range versionFiles {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			sb.T.Fatalf("Failed to create version file %s: %v", filename, err)
		}
	}

	return projectDir
}

// GetVersionPath returns the path to a version directory
func (sb *Sandbox) GetVersionPath(lang, version string) string {
	return filepath.Join(sb.VersionsDir, lang, version)
}

// GetCurrentPath returns the path to the "current" symlink for a language
func (sb *Sandbox) GetCurrentPath(lang string) string {
	return filepath.Join(sb.VersionsDir, lang, "current")
}

// AssertDirExists fails the test if the directory doesn't exist
func (sb *Sandbox) AssertDirExists(path string) {
	sb.T.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		sb.T.Errorf("Expected directory to exist: %s", path)
		return
	}
	if err != nil {
		sb.T.Errorf("Error checking directory %s: %v", path, err)
		return
	}
	if !info.IsDir() {
		sb.T.Errorf("Expected %s to be a directory", path)
	}
}

// AssertFileExists fails the test if the file doesn't exist
func (sb *Sandbox) AssertFileExists(path string) {
	sb.T.Helper()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		sb.T.Errorf("Expected file to exist: %s", path)
	} else if err != nil {
		sb.T.Errorf("Error checking file %s: %v", path, err)
	}
}

// AssertFileContains fails if file doesn't contain expected content
func (sb *Sandbox) AssertFileContains(path, expected string) {
	sb.T.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		sb.T.Errorf("Failed to read file %s: %v", path, err)
		return
	}
	if string(content) != expected {
		sb.T.Errorf("File %s content mismatch:\nExpected: %s\nGot: %s", path, expected, string(content))
	}
}
