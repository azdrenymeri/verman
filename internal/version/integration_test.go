package version

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/azdren/verman/internal/config"
	_ "github.com/azdren/verman/internal/languages"
)

// TestIntegration_FullWorkflow tests a complete user workflow
// This simulates:
// 1. User installs multiple Java versions (mocked)
// 2. User switches between versions
// 3. User navigates to a project with .java-version
// 4. Version is detected and switched
func TestIntegration_FullWorkflow(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Integration test requires Windows for junction points")
	}

	// Setup sandbox
	homeDir := t.TempDir()
	vermanDir := filepath.Join(homeDir, ".verman")
	versionsDir := filepath.Join(vermanDir, "versions")
	projectsDir := filepath.Join(homeDir, "projects")

	// Create directory structure
	os.MkdirAll(filepath.Join(versionsDir, "java"), 0755)
	os.MkdirAll(projectsDir, 0755)

	// Create config
	cfg := &config.Config{
		RootPath: versionsDir,
		Languages: map[string]config.LanguageConfig{
			"java": {InstallPath: "java"},
		},
	}

	mgr := NewManager(cfg)

	// Step 1: "Install" Java 17 (mock)
	java17Dir := filepath.Join(versionsDir, "java", "17")
	os.MkdirAll(filepath.Join(java17Dir, "bin"), 0755)
	os.WriteFile(filepath.Join(java17Dir, "bin", "java.exe"), []byte("mock java 17"), 0755)

	// Step 2: "Install" Java 21 (mock)
	java21Dir := filepath.Join(versionsDir, "java", "21")
	os.MkdirAll(filepath.Join(java21Dir, "bin"), 0755)
	os.WriteFile(filepath.Join(java21Dir, "bin", "java.exe"), []byte("mock java 21"), 0755)

	// Verify both are listed
	versions, err := mgr.ListInstalled("java")
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("Expected 2 Java versions, got %d", len(versions))
	}

	// Step 3: Use Java 17
	if err := mgr.Use("java", "17", false); err != nil {
		t.Fatalf("Failed to use Java 17: %v", err)
	}

	current, _ := mgr.GetCurrent("java")
	if current != "17" {
		t.Errorf("Expected current Java 17, got %s", current)
	}

	// Step 4: Create a project that requires Java 21
	projectDir := filepath.Join(projectsDir, "my-spring-app")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, ".java-version"), []byte("21"), 0644)

	// Step 5: Detect version in project
	detected := DetectForLanguage(projectDir, "java")
	if detected == nil {
		t.Fatal("Failed to detect Java version in project")
	}
	if detected.Version != "21" {
		t.Errorf("Expected detected version 21, got %s", detected.Version)
	}

	// Step 6: Switch to detected version
	if err := mgr.Use("java", detected.Version, false); err != nil {
		t.Fatalf("Failed to switch to detected version: %v", err)
	}

	current, _ = mgr.GetCurrent("java")
	if current != "21" {
		t.Errorf("Expected current Java 21 after detection, got %s", current)
	}

	// Step 7: Verify the junction points to correct directory
	currentPath := cfg.GetCurrentPath("java")
	target, err := os.Readlink(currentPath)
	if err != nil {
		// On Windows, Readlink might not work for junctions, check if it's accessible
		info, statErr := os.Stat(currentPath)
		if statErr != nil {
			t.Fatalf("Current path not accessible: %v", statErr)
		}
		if !info.IsDir() {
			t.Error("Current should be a directory")
		}
	} else {
		if filepath.Base(target) != "21" {
			t.Errorf("Junction should point to 21, points to %s", target)
		}
	}
}

// TestIntegration_MultiLanguageProject tests a project using multiple languages
func TestIntegration_MultiLanguageProject(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Integration test requires Windows")
	}

	homeDir := t.TempDir()
	versionsDir := filepath.Join(homeDir, ".verman", "versions")

	// Create versions directories
	for _, lang := range []string{"java", "node", "python"} {
		os.MkdirAll(filepath.Join(versionsDir, lang), 0755)
	}

	cfg := &config.Config{
		RootPath: versionsDir,
		Languages: map[string]config.LanguageConfig{
			"java":   {InstallPath: "java"},
			"node":   {InstallPath: "node"},
			"python": {InstallPath: "python"},
		},
	}

	mgr := NewManager(cfg)

	// Install mock versions
	os.MkdirAll(filepath.Join(versionsDir, "java", "21", "bin"), 0755)
	os.MkdirAll(filepath.Join(versionsDir, "node", "20"), 0755)
	os.MkdirAll(filepath.Join(versionsDir, "python", "3.12", "Scripts"), 0755)

	// Create project with multiple version files
	projectDir := filepath.Join(homeDir, "fullstack-app")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, ".java-version"), []byte("21"), 0644)
	os.WriteFile(filepath.Join(projectDir, ".nvmrc"), []byte("20"), 0644)
	os.WriteFile(filepath.Join(projectDir, ".python-version"), []byte("3.12"), 0644)

	// Detect all versions
	detected, err := DetectAll(projectDir)
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	if len(detected) != 3 {
		t.Errorf("Expected 3 detected versions, got %d", len(detected))
	}

	// Apply all detected versions
	for _, d := range detected {
		if err := mgr.Use(d.Language, d.Version, false); err != nil {
			t.Errorf("Failed to use %s %s: %v", d.Language, d.Version, err)
		}
	}

	// Verify all are set correctly
	for _, d := range detected {
		current, _ := mgr.GetCurrent(d.Language)
		if current != d.Version {
			t.Errorf("%s: expected %s, got %s", d.Language, d.Version, current)
		}
	}
}

// TestIntegration_NestedProjects tests version detection in nested project structures
func TestIntegration_NestedProjects(t *testing.T) {
	homeDir := t.TempDir()

	// Create workspace with multiple nested projects
	// workspace/
	//   .java-version (17) - workspace default
	//   project-a/
	//     .java-version (21) - project override
	//     src/main/java/  - deep nesting
	//   project-b/
	//     (no .java-version - inherits workspace)

	workspaceDir := filepath.Join(homeDir, "workspace")
	projectADir := filepath.Join(workspaceDir, "project-a")
	projectASrcDir := filepath.Join(projectADir, "src", "main", "java")
	projectBDir := filepath.Join(workspaceDir, "project-b")
	projectBSrcDir := filepath.Join(projectBDir, "src", "main", "java")

	os.MkdirAll(projectASrcDir, 0755)
	os.MkdirAll(projectBSrcDir, 0755)

	// Write version files
	os.WriteFile(filepath.Join(workspaceDir, ".java-version"), []byte("17"), 0644)
	os.WriteFile(filepath.Join(projectADir, ".java-version"), []byte("21"), 0644)

	// Test detection from project-a deep directory
	resultA := DetectForLanguage(projectASrcDir, "java")
	if resultA == nil {
		t.Fatal("Should detect Java version in project-a")
	}
	if resultA.Version != "21" {
		t.Errorf("project-a should use version 21, got %s", resultA.Version)
	}

	// Test detection from project-b (should inherit workspace)
	resultB := DetectForLanguage(projectBSrcDir, "java")
	if resultB == nil {
		t.Fatal("Should detect Java version in project-b (inherited)")
	}
	if resultB.Version != "17" {
		t.Errorf("project-b should inherit workspace version 17, got %s", resultB.Version)
	}
}
