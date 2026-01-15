package cmd

import (
	"fmt"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current [language]",
	Short: "Show current active versions",
	Long: `Show the currently active version for a language, or all languages.

Examples:
  verman current        # Show all current versions
  verman current java   # Show current Java version`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mgr := version.NewManager(cfg)

		if len(args) == 1 {
			showCurrent(mgr, args[0])
		} else {
			showAllCurrent(mgr)
		}
	},
}

func showCurrent(mgr *version.Manager, langName string) {
	current, err := mgr.GetCurrent(langName)
	if err != nil {
		fmt.Printf("%s: (error: %v)\n", langName, err)
		return
	}
	if current == "" {
		fmt.Printf("%s: (none)\n", langName)
	} else {
		fmt.Printf("%s: %s\n", langName, current)
	}
}

func showAllCurrent(mgr *version.Manager) {
	for _, lang := range languages.All() {
		current, _ := mgr.GetCurrent(lang.Name())
		if current != "" {
			fmt.Printf("%-8s %s\n", lang.Name()+":", current)
		}
	}
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
