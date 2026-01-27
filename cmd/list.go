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
	// Special handling for scala: show both scala (2.x) and scala3 versions
	if langName == "scala" {
		listScalaVersions(mgr)
		return
	}

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

func listScalaVersions(mgr *version.Manager) {
	// List both scala (2.x) and scala3 versions together
	scala2Versions, _ := mgr.ListInstalled("scala")
	scala3Versions, _ := mgr.ListInstalled("scala3")

	scala2Current, _ := mgr.GetCurrent("scala")
	scala3Current, _ := mgr.GetCurrent("scala3")

	if len(scala2Versions) == 0 && len(scala3Versions) == 0 {
		fmt.Println("No Scala versions installed")
		return
	}

	fmt.Println("scala:")
	for _, v := range scala3Versions {
		marker := "  "
		if v == scala3Current {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, v)
	}
	for _, v := range scala2Versions {
		marker := "  "
		if v == scala2Current {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, v)
	}
}

func listRemoteScalaVersions(mgr *version.Manager) {
	// Fetch both Scala 2 and Scala 3 versions
	scala2Src, _ := sources.Get("scala")
	scala3Src, _ := sources.Get("scala3")

	fmt.Println("Fetching available Scala versions...")
	fmt.Println()

	var allVersions []string

	// Fetch Scala 3 versions
	if scala3Src != nil {
		if versions, err := scala3Src.FetchVersions(); err == nil {
			allVersions = append(allVersions, versions...)
		}
	}

	// Fetch Scala 2 versions
	if scala2Src != nil {
		if versions, err := scala2Src.FetchVersions(); err == nil {
			allVersions = append(allVersions, versions...)
		}
	}

	if len(allVersions) == 0 {
		fmt.Println("No versions found")
		return
	}

	// Get installed versions (both scala and scala3)
	installedMap := make(map[string]bool)
	if installed, _ := mgr.ListInstalled("scala"); len(installed) > 0 {
		for _, v := range installed {
			installedMap[v] = true
		}
	}
	if installed, _ := mgr.ListInstalled("scala3"); len(installed) > 0 {
		for _, v := range installed {
			installedMap[v] = true
		}
	}

	scala2Current, _ := mgr.GetCurrent("scala")
	scala3Current, _ := mgr.GetCurrent("scala3")

	// Sort versions (newest first)
	sort.Slice(allVersions, func(i, j int) bool {
		return compareVersions(allVersions[i], allVersions[j]) > 0
	})

	const width = 80

	// Header
	fmt.Println(strings.Repeat("=", width))
	fmt.Println("Available Scala Versions")
	fmt.Println(strings.Repeat("=", width))

	// Print in columns
	const cols = 5
	const colWidth = 15

	for i, v := range allVersions {
		marker := "    "
		if v == scala2Current || v == scala3Current {
			marker = "> * "
		} else if installedMap[v] {
			marker = "  * "
		}

		fmt.Printf("%s%-*s", marker, colWidth-4, v)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(allVersions)%cols != 0 {
		fmt.Println()
	}

	// Footer
	fmt.Println(strings.Repeat("=", width))
	fmt.Println("* - installed")
	fmt.Println("> - currently in use")
	fmt.Println(strings.Repeat("=", width))
	fmt.Println()
	fmt.Println("Use: verman install scala <version>")
	fmt.Println("     (Automatically selects Scala 2 or 3 based on version)")
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
	// Special handling for scala: show both scala2 and scala3 remote versions
	if langName == "scala" {
		listRemoteScalaVersions(mgr)
		return
	}

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
			switch dist.key {
			case "temurin":
				distShortId = "tem"
			case "corretto":
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
			switch dist.key {
			case "temurin":
				distShort = "tem"
			case "corretto":
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
			_, _ = fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			_, _ = fmt.Sscanf(bParts[i], "%d", &bNum)
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
