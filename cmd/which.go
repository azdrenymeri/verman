package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/languages"
	"github.com/spf13/cobra"
)

var whichCmd = &cobra.Command{
	Use:   "which <language>",
	Short: "Show path to current version",
	Long: `Show the filesystem path to the currently active version.

Examples:
  verman which java    # Show path to current Java
  verman which node    # Show path to current Node`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		langName := args[0]

		if _, ok := languages.Get(langName); !ok {
			fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
			os.Exit(1)
		}

		currentPath := cfg.GetCurrentPath(langName)
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			fmt.Printf("No %s version set\n", langName)
			os.Exit(1)
		}

		fmt.Println(currentPath)
	},
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
