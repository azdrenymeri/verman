package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/azdren/verman/internal/config"
	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/sources"
	"github.com/spf13/cobra"
)

var cfg *config.Config

// SetVersion sets the version string (called from main)
func SetVersion(v string) {
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "verman",
	Short: "A universal version manager for development tools",
	Long: `Verman is a universal version manager for Windows focused on the
JVM ecosystem: Java, Scala, Kotlin, Gradle, Maven, SBT, Mill, plus Node.js and Go.

Examples:
  verman use java 21        # Switch to Java 21
  verman install node 20    # Install Node.js 20
  verman list java          # List installed Java versions
  verman current            # Show all current versions
  verman detect             # Auto-detect versions from project files`,
}

func Execute() {
	// Initialize sources
	home, _ := os.UserHomeDir()
	userSourcesDir := filepath.Join(home, ".verman", "sources")
	if err := sources.Load(userSourcesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading sources: %v\n", err)
		os.Exit(1)
	}

	// Load languages from sources
	if err := languages.LoadFromSources(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading languages: %v\n", err)
		os.Exit(1)
	}

	// Load config
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("global", "g", false, "Apply changes globally (persistent ENV vars)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output")
}
