package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/sources"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var listAll bool

var listCmd = &cobra.Command{
	Use:   "list [language]",
	Short: "List installed versions",
	Long: `List installed versions, or available remote versions with --all.

Examples:
  verman list              # List all installed versions
  verman list java         # List installed Java versions
  verman list java --all   # List all available Java versions (like SDKMAN)
  verman list node -a      # List all available Node.js versions`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mgr := version.NewManager(cfg)

		if len(args) == 1 {
			if listAll {
				listRemoteVersions(mgr, args[0])
			} else {
				listLanguage(mgr, args[0])
			}
		} else {
			if listAll {
				fmt.Println("Please specify a language to list available versions")
				fmt.Println("Example: verman list java --all")
				os.Exit(1)
			}
			listAllInstalled(mgr)
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

func listAllInstalled(mgr *version.Manager) {
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

func listRemoteVersions(mgr *version.Manager, langName string) {
	src, ok := sources.Get(langName)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown language: %s\n", langName)
		fmt.Fprintf(os.Stderr, "Available: %v\n", languages.Names())
		os.Exit(1)
	}

	fmt.Printf("Fetching available %s versions...\n\n", src.DisplayName)

	versions, err := src.FetchVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching versions: %v\n", err)
		os.Exit(1)
	}

	if len(versions) == 0 {
		fmt.Printf("No versions found for %s\n", langName)
		return
	}

	// Get installed versions for marking
	installed, _ := mgr.ListInstalled(langName)
	installedMap := make(map[string]bool)
	for _, v := range installed {
		installedMap[v] = true
	}
	current, _ := mgr.GetCurrent(langName)

	// Sort versions (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	// Use SDKMAN-style table format for Java (with distributions)
	if langName == "java" && len(src.Distributions) > 0 {
		listJavaVersions(src, versions, installedMap, current)
	} else {
		listSimpleVersions(src, langName, versions, installedMap, current)
	}
}

// listJavaVersions displays Java versions in SDKMAN table format with distributions
func listJavaVersions(src *sources.Source, versions []string, installedMap map[string]bool, current string) {
	const width = 80

	// Header
	fmt.Println(strings.Repeat("=", width))
	fmt.Println("Available Java Versions for Windows x64")
	fmt.Println(strings.Repeat("=", width))
	fmt.Printf(" %-14s| %-4s| %-13s| %-8s| %-10s| %s\n",
		"Vendor", "Use", "Version", "Dist", "Status", "Identifier")
	fmt.Println(strings.Repeat("-", width))

	// Group by distribution
	distOrder := []struct {
		key  string
		name string
	}{
		{"temurin", "Temurin"},
		{"corretto", "Corretto"},
		{"zulu", "Zulu"},
	}

	for _, dist := range distOrder {
		d, ok := src.Distributions[dist.key]
		if !ok {
			continue
		}

		firstRow := true
		for _, v := range versions {
			// Build identifier with short distribution name
			distShortId := dist.key
			if dist.key == "temurin" {
				distShortId = "tem"
			} else if dist.key == "corretto" {
				distShortId = "amzn"
			}
			identifier := v + "-" + distShortId // e.g., "21-tem", "21-amzn", "21-zulu"

			// Determine status
			use := "   "
			status := ""

			// Check if this specific distribution is installed
			fullIdentifier := v + "-" + sources.NormalizeDistribution(dist.key[:3])
			if current == fullIdentifier {
				use = ">>>"
				status = "installed"
			} else if installedMap[fullIdentifier] {
				status = "installed"
			} else if current == v || installedMap[v] {
				// Check base version (without distribution)
				if current == v {
					use = ">>>"
					status = "installed"
				} else if installedMap[v] {
					status = "installed"
				}
			}

			vendor := ""
			if firstRow {
				vendor = d.DisplayName
				if len(vendor) > 14 {
					vendor = dist.name
				}
				firstRow = false
			}

			distShort := dist.key
			if dist.key == "temurin" {
				distShort = "tem"
			} else if dist.key == "corretto" {
				distShort = "amzn"
			}

			fmt.Printf(" %-14s| %-4s| %-13s| %-8s| %-10s| %s\n",
				vendor, use, v, distShort, status, identifier)
		}
		fmt.Println(strings.Repeat("-", width))
	}

	// Footer
	fmt.Println(strings.Repeat("=", width))
	fmt.Println("Use the Identifier for installation:")
	fmt.Println()
	fmt.Println("    $ verman install java 21-tem")
	fmt.Println("    $ verman install java 17-amzn")
	fmt.Println("    $ verman install java 21-zulu")
	fmt.Println()
	fmt.Println(strings.Repeat("=", width))
}

// listSimpleVersions displays versions in a simple column format (for non-Java tools)
func listSimpleVersions(src *sources.Source, langName string, versions []string, installedMap map[string]bool, current string) {
	const width = 80

	// Header
	fmt.Println(strings.Repeat("=", width))
	fmt.Printf("Available %s Versions\n", src.DisplayName)
	fmt.Println(strings.Repeat("=", width))

	// Print in columns
	const cols = 5
	const colWidth = 15

	for i, v := range versions {
		marker := "    "
		if v == current {
			marker = "> * "
		} else if installedMap[v] {
			marker = "  * "
		}

		fmt.Printf("%s%-*s", marker, colWidth-4, v)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(versions)%cols != 0 {
		fmt.Println()
	}

	// Footer
	fmt.Println(strings.Repeat("=", width))
	fmt.Println("* - installed")
	fmt.Println("> - currently in use")
	fmt.Println(strings.Repeat("=", width))
	fmt.Println()
	fmt.Printf("Use: verman install %s <version>\n", langName)
}

// compareVersions compares two version strings semantically
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bNum)
		}
		if aNum != bNum {
			return aNum - bNum
		}
	}
	return 0
}

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "List all available versions (remote)")
	rootCmd.AddCommand(listCmd)
}
