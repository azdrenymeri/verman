package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print environment variable commands for current shell",
	Long: `Prints commands to set environment variables for all active language versions.

Run this after installing a new version to refresh your current terminal:
  PowerShell:  verman env | Invoke-Expression
  CMD:         Not yet supported

Or copy and paste the output commands manually.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := version.NewManager(cfg)
		home, _ := os.UserHomeDir()

		// Get all registered languages
		allLangs := languages.Names()

		var pathAdditions []string

		for _, langName := range allLangs {
			lang, ok := languages.Get(langName)
			if !ok {
				continue
			}

			// Check if this language has a current version
			currentVersion, err := mgr.GetCurrent(langName)
			if err != nil || currentVersion == "" {
				continue
			}

			currentPath := cfg.GetCurrentPath(langName)

			// Output environment variable settings (like JAVA_HOME)
			for envVar, relPath := range lang.EnvVars() {
				fullPath := currentPath
				if relPath != "." {
					fullPath = filepath.Join(currentPath, relPath)
				}

				if runtime.GOOS == "windows" {
					fmt.Printf("$env:%s = \"%s\"\n", envVar, fullPath)
				} else {
					fmt.Printf("export %s=\"%s\"\n", envVar, fullPath)
				}
			}

			// Collect PATH additions for this language
			for _, relDir := range lang.PathDirs() {
				binPath := currentPath
				if relDir != "." {
					binPath = filepath.Join(currentPath, relDir)
				}
				pathAdditions = append(pathAdditions, binPath)
			}
		}

		// Add verman bin directory
		vermanBin := filepath.Join(home, ".verman", "bin")
		pathAdditions = append([]string{vermanBin}, pathAdditions...)

		// Output PATH update
		if len(pathAdditions) > 0 {
			if runtime.GOOS == "windows" {
				fmt.Printf("$env:PATH = \"%s;\" + $env:PATH\n", strings.Join(pathAdditions, ";"))
			} else {
				fmt.Printf("export PATH=\"%s:$PATH\"\n", strings.Join(pathAdditions, ":"))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
