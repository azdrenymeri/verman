package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <language> <version>",
	Short: "Switch to a specific version",
	Long: `Switch to a specific version of a language runtime.

Examples:
  verman use java 21
  verman use node 20
  verman use -g scala 3.3.1   # Set globally (persistent)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		langName := args[0]
		ver := args[1]
		global, _ := cmd.Flags().GetBool("global")
		quiet, _ := cmd.Flags().GetBool("quiet")

		if _, ok := languages.Get(langName); !ok {
			fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
			fmt.Fprintf(os.Stderr, "Available: %v\n", languages.Names())
			os.Exit(1)
		}

		mgr := version.NewManager(cfg)
		if err := mgr.Use(langName, ver, global); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if !quiet {
			fmt.Printf("Now using %s %s\n", langName, ver)
			if global {
				fmt.Println("(Set globally - restart your terminal for changes to take effect)")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
