package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type LanguageConfig struct {
	CurrentVersion string `json:"current_version"`
	InstallPath    string `json:"install_path"`
}

type Config struct {
	RootPath  string                    `json:"root_path"`
	Languages map[string]LanguageConfig `json:"languages"`
	path      string
}

var defaultLanguages = map[string]LanguageConfig{
	"java":   {InstallPath: "java"},
	"scala":  {InstallPath: "scala"},
	"node":   {InstallPath: "node"},
	"python": {InstallPath: "python"},
	"ruby":   {InstallPath: "ruby"},
	"go":     {InstallPath: "go"},
	"rust":   {InstallPath: "rust"},
	"dotnet": {InstallPath: "dotnet"},
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".verman", "config.json"), nil
}

func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		path:      configPath,
		Languages: make(map[string]LanguageConfig),
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Create default config
		home, _ := os.UserHomeDir()
		cfg.RootPath = filepath.Join(home, ".verman", "versions")
		cfg.Languages = defaultLanguages
		return cfg, cfg.Save()
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.path = configPath

	return cfg, nil
}

func (c *Config) Save() error {
	// Skip saving if no path is set (e.g., in tests)
	if c.path == "" {
		return nil
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// SetPath sets the config file path (useful for testing)
func (c *Config) SetPath(path string) {
	c.path = path
}

func (c *Config) GetVersionPath(lang, version string) string {
	return filepath.Join(c.RootPath, lang, version)
}

func (c *Config) GetCurrentPath(lang string) string {
	return filepath.Join(c.RootPath, lang, "current")
}

func (c *Config) SetCurrentVersion(lang, version string) error {
	if langCfg, ok := c.Languages[lang]; ok {
		langCfg.CurrentVersion = version
		c.Languages[lang] = langCfg
		return c.Save()
	}
	return nil
}
