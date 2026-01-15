package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <language> <version>",
	Short: "Install a specific version",
	Long: `Download and install a specific version of a language runtime.

Examples:
  verman install java 21
  verman install node 20.10.0
  verman install scala 3.3.1`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		langName := args[0]
		ver := args[1]

		if _, ok := languages.Get(langName); !ok {
			fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
			fmt.Fprintf(os.Stderr, "Available: %v\n", languages.Names())
			os.Exit(1)
		}

		mgr := version.NewManager(cfg)
		if err := mgr.Install(langName, ver); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Ask if user wants to use this version now
		fmt.Printf("\nSwitch to %s %s now? [Y/n] ", langName, ver)
		var response string
		fmt.Scanln(&response)
		if response == "" || response == "y" || response == "Y" {
			if err := mgr.Use(langName, ver, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error switching version: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Now using %s %s\n", langName, ver)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <language> <version>",
	Short: "Uninstall a specific version",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		langName := args[0]
		ver := args[1]

		mgr := version.NewManager(cfg)
		if err := mgr.Uninstall(langName, ver); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Uninstalled %s %s\n", langName, ver)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
