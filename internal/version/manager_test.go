package version

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/azdren/verman/internal/config"
	_ "github.com/azdren/verman/internal/languages" // Register languages
)

// setupTestManager creates a test manager with a sandbox environment
func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, ".verman", "versions")

	// Create directories for all languages
	for _, lang := range []string{"java", "node", "scala", "python", "ruby", "go", "rust", "dotnet"} {
		_ = os.MkdirAll(filepath.Join(versionsDir, lang), 0755)
	}

	cfg := &config.Config{
		RootPath: versionsDir,
		Languages: map[string]config.LanguageConfig{
			"java":   {InstallPath: "java"},
			"node":   {InstallPath: "node"},
			"scala":  {InstallPath: "scala"},
			"python": {InstallPath: "python"},
			"ruby":   {InstallPath: "ruby"},
			"go":     {InstallPath: "go"},
			"rust":   {InstallPath: "rust"},
			"dotnet": {InstallPath: "dotnet"},
		},
	}

	return NewManager(cfg), tmpDir
}

// createMockVersion creates a fake installed version
func createMockVersion(t *testing.T, mgr *Manager, lang, version string) string {
	t.Helper()

	versionDir := mgr.Config.GetVersionPath(lang, version)
	binDir := filepath.Join(versionDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create mock version: %v", err)
	}

	// Create a mock executable
	mockExe := filepath.Join(binDir, "mock.exe")
	if err := os.WriteFile(mockExe, []byte("mock executable"), 0755); err != nil {
		t.Fatalf("Failed to create mock executable: %v", err)
	}

	return versionDir
}

func TestListInstalledEmpty(t *testing.T) {
	mgr, _ := setupTestManager(t)

	versions, err := mgr.ListInstalled("java")
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected empty list, got %v", versions)
	}
}

func TestListInstalledWithVersions(t *testing.T) {
	mgr, _ := setupTestManager(t)

	// Create mock versions
	createMockVersion(t, mgr, "java", "17")
	createMockVersion(t, mgr, "java", "21")
	createMockVersion(t, mgr, "java", "11")

	versions, err := mgr.ListInstalled("java")
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	// Check all versions are present
	versionSet := make(map[string]bool)
	for _, v := range versions {
		versionSet[v] = true
	}

	for _, expected := range []string{"17", "21", "11"} {
		if !versionSet[expected] {
			t.Errorf("Expected version %s to be listed", expected)
		}
	}
}

func TestUseVersion(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Junction point test only runs on Windows")
	}

	mgr, _ := setupTestManager(t)

	// Create mock version
	createMockVersion(t, mgr, "java", "21")

	// Use it
	err := mgr.Use("java", "21", false)
	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}

	// Verify current symlink/junction was created
	currentPath := mgr.Config.GetCurrentPath("java")
	info, err := os.Stat(currentPath)
	if os.IsNotExist(err) {
		t.Fatal("Current symlink was not created")
	}
	if err != nil {
		t.Fatalf("Error checking current symlink: %v", err)
	}
	if !info.IsDir() {
		t.Error("Current should be a directory (junction)")
	}

	// Verify GetCurrent returns correct version
	current, err := mgr.GetCurrent("java")
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if current != "21" {
		t.Errorf("Expected current version 21, got %s", current)
	}
}

func TestUseNonExistentVersion(t *testing.T) {
	mgr, _ := setupTestManager(t)

	err := mgr.Use("java", "99", false)
	if err == nil {
		t.Error("Expected error when using non-existent version")
	}
}

func TestUseUnknownLanguage(t *testing.T) {
	mgr, _ := setupTestManager(t)

	err := mgr.Use("cobol", "1.0", false)
	if err == nil {
		t.Error("Expected error for unknown language")
	}
}

func TestUninstall(t *testing.T) {
	mgr, _ := setupTestManager(t)

	// Create mock version
	createMockVersion(t, mgr, "node", "20")

	// Verify it exists
	versionPath := mgr.Config.GetVersionPath("node", "20")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		t.Fatal("Mock version should exist before uninstall")
	}

	// Uninstall
	err := mgr.Uninstall("node", "20")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(versionPath); !os.IsNotExist(err) {
		t.Error("Version directory should be removed after uninstall")
	}
}

func TestUninstallNonExistent(t *testing.T) {
	mgr, _ := setupTestManager(t)

	err := mgr.Uninstall("java", "999")
	if err == nil {
		t.Error("Expected error when uninstalling non-existent version")
	}
}

func TestGetCurrentNoVersion(t *testing.T) {
	mgr, _ := setupTestManager(t)

	current, err := mgr.GetCurrent("scala")
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}

	if current != "" {
		t.Errorf("Expected empty string for no current version, got %s", current)
	}
}

func TestSwitchBetweenVersions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Junction point test only runs on Windows")
	}

	mgr, _ := setupTestManager(t)

	// Create two versions
	createMockVersion(t, mgr, "java", "17")
	createMockVersion(t, mgr, "java", "21")

	// Use version 17
	if err := mgr.Use("java", "17", false); err != nil {
		t.Fatalf("Failed to use java 17: %v", err)
	}

	current, _ := mgr.GetCurrent("java")
	if current != "17" {
		t.Errorf("Expected current 17, got %s", current)
	}

	// Switch to version 21
	if err := mgr.Use("java", "21", false); err != nil {
		t.Fatalf("Failed to use java 21: %v", err)
	}

	current, _ = mgr.GetCurrent("java")
	if current != "21" {
		t.Errorf("Expected current 21, got %s", current)
	}
}

func TestMultipleLanguages(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Junction point test only runs on Windows")
	}

	mgr, _ := setupTestManager(t)

	// Create versions for multiple supported languages
	createMockVersion(t, mgr, "java", "21")
	createMockVersion(t, mgr, "node", "20")
	createMockVersion(t, mgr, "scala", "2.13.12")

	// Use each
	_ = mgr.Use("java", "21", false)
	_ = mgr.Use("node", "20", false)
	_ = mgr.Use("scala", "2.13.12", false)

	// Verify each has correct current
	tests := []struct {
		lang    string
		version string
	}{
		{"java", "21"},
		{"node", "20"},
		{"scala", "2.13.12"},
	}

	for _, tt := range tests {
		current, _ := mgr.GetCurrent(tt.lang)
		if current != tt.version {
			t.Errorf("%s: expected current %s, got %s", tt.lang, tt.version, current)
		}
	}
}
