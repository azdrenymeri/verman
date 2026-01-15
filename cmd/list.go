package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [language]",
	Short: "List installed versions",
	Long: `List all installed versions for a language, or all languages if none specified.

Examples:
  verman list           # List all languages
  verman list java      # List Java versions only`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mgr := version.NewManager(cfg)

		if len(args) == 1 {
			listLanguage(mgr, args[0])
		} else {
			listAll(mgr)
		}
	},
}

func listLanguage(mgr *version.Manager, langName string) {
	if _, ok := languages.Get(langName); !ok {
		fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
		os.Exit(1)
	}

	versions, err := mgr.ListInstalled(langName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	current, _ := mgr.GetCurrent(langName)

	if len(versions) == 0 {
		fmt.Printf("No %s versions installed\n", langName)
		return
	}

	fmt.Printf("%s:\n", langName)
	for _, v := range versions {
		marker := "  "
		if v == current {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, v)
	}
}

func listAll(mgr *version.Manager) {
	hasAny := false

	for _, lang := range languages.All() {
		versions, err := mgr.ListInstalled(lang.Name())
		if err != nil || len(versions) == 0 {
			continue
		}

		hasAny = true
		current, _ := mgr.GetCurrent(lang.Name())

		fmt.Printf("%s:\n", lang.Name())
		for _, v := range versions {
			marker := "  "
			if v == current {
				marker = "* "
			}
			fmt.Printf("  %s%s\n", marker, v)
		}
		fmt.Println()
	}

	if !hasAny {
		fmt.Println("No versions installed")
		fmt.Println("Use 'verman install <language> <version>' to install")
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
