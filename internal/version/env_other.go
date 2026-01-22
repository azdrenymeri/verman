//go:build !windows

package version

import (
	"fmt"
	"os"
	"path/filepath"
)

// setUserEnvVar sets a user environment variable persistently on Unix-like systems
func setUserEnvVar(name, value string) error {
	// On Unix, we append to shell profile
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Try to detect the shell and update appropriate profile
	shell := os.Getenv("SHELL")
	var profilePath string

	switch {
	case contains(shell, "zsh"):
		profilePath = filepath.Join(home, ".zshrc")
	case contains(shell, "bash"):
		profilePath = filepath.Join(home, ".bashrc")
	default:
		profilePath = filepath.Join(home, ".profile")
	}

	// Append export statement
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf("\nexport %s=\"%s\"\n", name, value)
	if _, err := f.WriteString(line); err != nil {
		return err
	}

	return nil
}

// getUserEnvVar gets an environment variable (just uses os.Getenv on Unix)
func getUserEnvVar(name string) (string, error) {
	return os.Getenv(name), nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
