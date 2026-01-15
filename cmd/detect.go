package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Auto-detect versions from project files",
	Long: `Scan the current directory for version files and show detected versions.

Supported files:
  Java:   .java-version, .sdkmanrc
  Scala:  .scala-version
  Node:   .nvmrc, .node-version
  Python: .python-version
  Ruby:   .ruby-version
  Go:     .go-version, go.mod
  Rust:   rust-toolchain.toml, rust-toolchain
  .NET:   global.json

Examples:
  verman detect              # Show detected versions
  verman detect --apply      # Detect and switch to those versions
  verman detect --json       # Output as JSON`,
	Run: func(cmd *cobra.Command, args []string) {
		apply, _ := cmd.Flags().GetBool("apply")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		quiet, _ := cmd.Flags().GetBool("quiet")

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		detected, err := version.DetectAll(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(detected) == 0 {
			if !quiet && !jsonOutput {
				fmt.Println("No version files detected")
			}
			if jsonOutput {
				fmt.Println("[]")
			}
			return
		}

		if jsonOutput {
			data, _ := json.Marshal(detected)
			fmt.Println(string(data))
			return
		}

		if !quiet {
			fmt.Println("Detected versions:")
			for _, d := range detected {
				fmt.Printf("  %-8s %s (from %s)\n", d.Language+":", d.Version, d.Source)
			}
		}

		if apply {
			mgr := version.NewManager(cfg)
			fmt.Println("\nApplying versions:")
			for _, d := range detected {
				if err := mgr.Use(d.Language, d.Version, false); err != nil {
					fmt.Printf("  %-8s %v\n", d.Language+":", err)
				} else if !quiet {
					fmt.Printf("  %-8s switched to %s\n", d.Language+":", d.Version)
				}
			}
		}
	},
}

func init() {
	detectCmd.Flags().Bool("apply", false, "Switch to detected versions")
	detectCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(detectCmd)
}
