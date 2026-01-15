package version

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/azdren/verman/internal/languages" // Register languages
)

func TestDetectJavaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-java-project")
	os.MkdirAll(projectDir, 0755)

	// Create .java-version file
	versionFile := filepath.Join(projectDir, ".java-version")
	os.WriteFile(versionFile, []byte("21"), 0644)

	// Detect
	result := DetectForLanguage(projectDir, "java")

	if result == nil {
		t.Fatal("Expected to detect java version")
	}

	if result.Version != "21" {
		t.Errorf("Expected version 21, got %s", result.Version)
	}

	if result.Language != "java" {
		t.Errorf("Expected language java, got %s", result.Language)
	}
}

func TestDetectNodeVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{"nvmrc", ".nvmrc", "20.10.0", "20.10.0"},
		{"nvmrc with v prefix", ".nvmrc", "v18.19.0", "18.19.0"},
		{"node-version", ".node-version", "18", "18"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.WriteFile(filepath.Join(tmpDir, tt.filename), []byte(tt.content), 0644)

			result := DetectForLanguage(tmpDir, "node")

			if result == nil {
				t.Fatal("Expected to detect node version")
			}

			if result.Version != tt.expected {
				t.Errorf("Expected version %s, got %s", tt.expected, result.Version)
			}
		})
	}
}

func TestDetectScalaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".scala-version"), []byte("3.3.1"), 0644)

	result := DetectForLanguage(tmpDir, "scala")

	if result == nil {
		t.Fatal("Expected to detect scala version")
	}

	if result.Version != "3.3.1" {
		t.Errorf("Expected version 3.3.1, got %s", result.Version)
	}
}

func TestDetectGoVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Test go.mod parsing
	goMod := `module example.com/myapp

go 1.21

require (
	github.com/some/package v1.0.0
)
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	result := DetectForLanguage(tmpDir, "go")

	if result == nil {
		t.Fatal("Expected to detect go version")
	}

	if result.Version != "1.21" {
		t.Errorf("Expected version 1.21, got %s", result.Version)
	}
}

func TestDetectDotNetVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Test global.json parsing
	globalJson := `{
  "sdk": {
    "version": "8.0.100"
  }
}`
	os.WriteFile(filepath.Join(tmpDir, "global.json"), []byte(globalJson), 0644)

	result := DetectForLanguage(tmpDir, "dotnet")

	if result == nil {
		t.Fatal("Expected to detect dotnet version")
	}

	if result.Version != "8.0.100" {
		t.Errorf("Expected version 8.0.100, got %s", result.Version)
	}
}

func TestDetectRustVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Test rust-toolchain.toml parsing
	toolchain := `[toolchain]
channel = "1.75.0"
`
	os.WriteFile(filepath.Join(tmpDir, "rust-toolchain.toml"), []byte(toolchain), 0644)

	result := DetectForLanguage(tmpDir, "rust")

	if result == nil {
		t.Fatal("Expected to detect rust version")
	}

	if result.Version != "1.75.0" {
		t.Errorf("Expected version 1.75.0, got %s", result.Version)
	}
}

func TestDetectAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple version files
	os.WriteFile(filepath.Join(tmpDir, ".java-version"), []byte("21"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".nvmrc"), []byte("20"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".python-version"), []byte("3.12.0"), 0644)

	results, err := DetectAll(tmpDir)
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	// Should find all three
	if len(results) != 3 {
		t.Errorf("Expected 3 detected versions, got %d", len(results))
	}

	// Verify each language was detected
	found := make(map[string]string)
	for _, r := range results {
		found[r.Language] = r.Version
	}

	if found["java"] != "21" {
		t.Errorf("Java version mismatch: expected 21, got %s", found["java"])
	}
	if found["node"] != "20" {
		t.Errorf("Node version mismatch: expected 20, got %s", found["node"])
	}
	if found["python"] != "3.12.0" {
		t.Errorf("Python version mismatch: expected 3.12.0, got %s", found["python"])
	}
}

func TestDetectWalksUpDirectoryTree(t *testing.T) {
	tmpDir := t.TempDir()

	// Create version file in parent directory
	os.WriteFile(filepath.Join(tmpDir, ".java-version"), []byte("17"), 0644)

	// Create nested project directory
	nestedDir := filepath.Join(tmpDir, "src", "main", "java")
	os.MkdirAll(nestedDir, 0755)

	// Detect from nested directory - should find parent's version file
	result := DetectForLanguage(nestedDir, "java")

	if result == nil {
		t.Fatal("Expected to detect java version from parent directory")
	}

	if result.Version != "17" {
		t.Errorf("Expected version 17, got %s", result.Version)
	}
}

func TestDetectNoVersionFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory - no version files
	result := DetectForLanguage(tmpDir, "java")

	if result != nil {
		t.Errorf("Expected nil result for missing version file, got %+v", result)
	}
}

func TestDetectSdkmanrc(t *testing.T) {
	tmpDir := t.TempDir()

	sdkmanrc := `java=17.0.9-tem
scala=3.3.1
`
	os.WriteFile(filepath.Join(tmpDir, ".sdkmanrc"), []byte(sdkmanrc), 0644)

	result := DetectForLanguage(tmpDir, "java")

	if result == nil {
		t.Fatal("Expected to detect java version from .sdkmanrc")
	}

	// Should extract major version
	if result.Version != "17" {
		t.Errorf("Expected version 17, got %s", result.Version)
	}
}
