package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".verman", "config.json")

	// Create config
	cfg := &Config{
		path:     configPath,
		RootPath: filepath.Join(tmpDir, ".verman", "versions"),
		Languages: map[string]LanguageConfig{
			"java": {CurrentVersion: "21", InstallPath: "java"},
			"node": {CurrentVersion: "20", InstallPath: "node"},
		},
	}

	// Save config
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify values
	if loaded.RootPath != cfg.RootPath {
		t.Errorf("RootPath mismatch: expected %s, got %s", cfg.RootPath, loaded.RootPath)
	}

	if loaded.Languages["java"].CurrentVersion != "21" {
		t.Errorf("Java version mismatch: expected 21, got %s", loaded.Languages["java"].CurrentVersion)
	}

	if loaded.Languages["node"].CurrentVersion != "20" {
		t.Errorf("Node version mismatch: expected 20, got %s", loaded.Languages["node"].CurrentVersion)
	}
}

func TestSetCurrentVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".verman", "config.json")

	cfg := &Config{
		path:     configPath,
		RootPath: filepath.Join(tmpDir, ".verman", "versions"),
		Languages: map[string]LanguageConfig{
			"java": {CurrentVersion: "17", InstallPath: "java"},
		},
	}

	// Set new version
	if err := cfg.SetCurrentVersion("java", "21"); err != nil {
		t.Fatalf("Failed to set current version: %v", err)
	}

	// Verify it was updated
	if cfg.Languages["java"].CurrentVersion != "21" {
		t.Errorf("Expected java version 21, got %s", cfg.Languages["java"].CurrentVersion)
	}

	// Reload from file and verify persistence
	data, _ := os.ReadFile(configPath)
	var reloaded Config
	json.Unmarshal(data, &reloaded)

	if reloaded.Languages["java"].CurrentVersion != "21" {
		t.Errorf("Persisted java version should be 21, got %s", reloaded.Languages["java"].CurrentVersion)
	}
}

func TestGetVersionPath(t *testing.T) {
	cfg := &Config{
		RootPath: "/home/user/.verman/versions",
	}

	path := cfg.GetVersionPath("java", "21")
	expected := filepath.Join("/home/user/.verman/versions", "java", "21")

	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestGetCurrentPath(t *testing.T) {
	cfg := &Config{
		RootPath: "/home/user/.verman/versions",
	}

	path := cfg.GetCurrentPath("node")
	expected := filepath.Join("/home/user/.verman/versions", "node", "current")

	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}
