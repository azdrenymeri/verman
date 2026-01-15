package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "verman",
	Short: "A universal version manager for development tools",
	Long: `Verman is a universal version manager for Windows that manages
multiple programming language runtimes including Java, Scala, Node.js,
Python, Ruby, Go, Rust, and .NET.

Examples:
  verman use java 21        # Switch to Java 21
  verman install node 20    # Install Node.js 20
  verman list java          # List installed Java versions
  verman current            # Show all current versions
  verman detect             # Auto-detect versions from project files`,
}

func Execute() {
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
