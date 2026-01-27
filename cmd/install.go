package cmd

import (
	"fmt"
	"os"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/sources"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <language> <version>",
	Short: "Install a specific version",
	Long: `Download and install a specific version of a language runtime.

Version can be partial (e.g., "20" for Node.js will resolve to latest 20.x.x).

For Java, you can specify a distribution using SDKMAN-style suffixes:
  - No suffix: Eclipse Temurin (default)
  - -tem or -temurin: Eclipse Temurin
  - -amzn or -corretto: Amazon Corretto
  - -zulu: Azul Zulu

Examples:
  verman install java 21           # Temurin (default)
  verman install java 21-tem       # Temurin (explicit)
  verman install java 21-amzn      # Amazon Corretto
  verman install java 21-zulu      # Azul Zulu
  verman install node 20           # Resolves to latest 20.x.x
  verman install node 20.10.0      # Exact version
  verman install scala 2.13.x      # Latest 2.13 patch version`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		langName := args[0]
		ver := args[1]

		// Smart routing: "scala 3.x" -> "scala3"
		if langName == "scala" && len(ver) > 0 && ver[0] == '3' {
			langName = "scala3"
		}

		lang, ok := languages.Get(langName)
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
			fmt.Fprintf(os.Stderr, "Available: %v\n", languages.Names())
			os.Exit(1)
		}

		// Parse distribution suffix if present (e.g., "21-tem" -> "21", "tem")
		baseVer, dist := sources.ParseVersionAndDistribution(ver)

		// Show distribution info for Java
		if lang.HasDistributions() && dist != "" {
			distName := lang.GetDistributionDisplayName(dist)
			fmt.Printf("Using distribution: %s\n", distName)
		}

		// Resolve partial version to full version
		resolvedVer, err := lang.ResolveVersion(baseVer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving version: %v\n", err)
			os.Exit(1)
		}

		if resolvedVer != baseVer {
			fmt.Printf("Resolved %s %s -> %s\n", langName, baseVer, resolvedVer)
		}

		// Construct the install version (include distribution suffix for identification)
		// Keep user's original input (e.g., "amzn" not "corretto") for consistency
		installVer := resolvedVer
		if dist != "" {
			installVer = resolvedVer + "-" + dist
		}

		mgr := version.NewManager(cfg)
		if err := mgr.InstallWithDist(langName, resolvedVer, dist); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Ask if user wants to use this version now
		fmt.Printf("\nSwitch to %s %s now? [Y/n] ", langName, installVer)
		var response string
		_, _ = fmt.Scanln(&response)
		if response == "" || response == "y" || response == "Y" {
			if err := mgr.Use(langName, installVer, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error switching version: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Now using %s %s\n", langName, installVer)
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

		// Smart routing: "scala 3.x" -> "scala3"
		if langName == "scala" && len(ver) > 0 && ver[0] == '3' {
			langName = "scala3"
		}

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
