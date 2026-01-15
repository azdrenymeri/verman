package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azdren/verman/internal/config"
	"github.com/azdren/verman/internal/languages"
)

// EnvExport represents an environment variable to export
type EnvExport struct {
	Name  string
	Value string
}

// GetEnvExports returns environment exports for shell integration
func GetEnvExports(cfg *config.Config, langName string) ([]EnvExport, error) {
	lang, ok := languages.Get(langName)
	if !ok {
		return nil, fmt.Errorf("unknown language: %s", langName)
	}

	currentPath := cfg.GetCurrentPath(langName)
	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return nil, nil // No version set
	}

	var exports []EnvExport

	// Add HOME-style variables
	for envVar, relPath := range lang.EnvVars() {
		fullPath := currentPath
		if relPath != "." {
			fullPath = filepath.Join(currentPath, relPath)
		}
		exports = append(exports, EnvExport{Name: envVar, Value: fullPath})
	}

	return exports, nil
}

// GetPathAdditions returns PATH additions for a language
func GetPathAdditions(cfg *config.Config, langName string) ([]string, error) {
	lang, ok := languages.Get(langName)
	if !ok {
		return nil, fmt.Errorf("unknown language: %s", langName)
	}

	currentPath := cfg.GetCurrentPath(langName)
	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return nil, nil
	}

	var paths []string
	for _, dir := range lang.PathDirs() {
		if dir == "." {
			paths = append(paths, currentPath)
		} else {
			paths = append(paths, filepath.Join(currentPath, dir))
		}
	}

	return paths, nil
}

// GeneratePowerShellInit generates PowerShell script for shell integration
func GeneratePowerShellInit(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString("# Verman PowerShell Integration\n")
	sb.WriteString("# Add this to your PowerShell profile ($PROFILE)\n\n")

	// Add all language paths and env vars
	for _, lang := range languages.All() {
		currentPath := cfg.GetCurrentPath(lang.Name())
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			continue
		}

		sb.WriteString(fmt.Sprintf("# %s\n", lang.Name()))

		for envVar, relPath := range lang.EnvVars() {
			fullPath := currentPath
			if relPath != "." {
				fullPath = filepath.Join(currentPath, relPath)
			}
			sb.WriteString(fmt.Sprintf("$env:%s = '%s'\n", envVar, fullPath))
		}

		for _, dir := range lang.PathDirs() {
			pathDir := currentPath
			if dir != "." {
				pathDir = filepath.Join(currentPath, dir)
			}
			sb.WriteString(fmt.Sprintf("$env:PATH = '%s;' + $env:PATH\n", pathDir))
		}
		sb.WriteString("\n")
	}

	// Add auto-detect hook
	sb.WriteString(`# Auto-detect versions on directory change
function Set-VermanVersions {
    $detected = & verman detect --quiet --json 2>$null | ConvertFrom-Json
    if ($detected) {
        foreach ($item in $detected) {
            & verman use $item.language $item.version --quiet 2>$null
        }
    }
}

# Hook into directory change (optional - uncomment to enable)
# $ExecutionContext.SessionState.InvokeCommand.PreCommandLookupAction = {
#     param($CommandName, $CommandLookupEventArgs)
#     if ($CommandName -eq 'cd' -or $CommandName -eq 'Set-Location') {
#         Set-VermanVersions
#     }
# }
`)

	return sb.String()
}

// GenerateCmdInit generates batch script for CMD integration
func GenerateCmdInit(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString("@echo off\n")
	sb.WriteString("REM Verman CMD Integration\n\n")

	for _, lang := range languages.All() {
		currentPath := cfg.GetCurrentPath(lang.Name())
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			continue
		}

		for envVar, relPath := range lang.EnvVars() {
			fullPath := currentPath
			if relPath != "." {
				fullPath = filepath.Join(currentPath, relPath)
			}
			sb.WriteString(fmt.Sprintf("SET %s=%s\n", envVar, fullPath))
		}

		for _, dir := range lang.PathDirs() {
			pathDir := currentPath
			if dir != "." {
				pathDir = filepath.Join(currentPath, dir)
			}
			sb.WriteString(fmt.Sprintf("SET PATH=%s;%%PATH%%\n", pathDir))
		}
	}

	return sb.String()
}
