package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/azdren/verman/internal/languages"
	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues with verman setup",
	Long: `Checks your verman installation for common issues:
  - Verman directory structure
  - PATH configuration
  - Installed tools and their dependencies
  - Environment variables (JAVA_HOME, etc.)`,
	Run: func(cmd *cobra.Command, args []string) {
		runDoctor()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor() {
	fmt.Println("Verman Doctor")
	fmt.Println("=============")
	fmt.Println()

	issues := 0

	// Check 1: Verman directory
	issues += checkVermanDirectory()

	// Check 2: PATH configuration
	issues += checkPath()

	// Check 3: Installed tools
	issues += checkInstalledTools()

	// Check 4: Environment variables
	issues += checkEnvironment()

	// Check 5: Dependencies
	issues += checkDependencies()

	// Summary
	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("-------")
	if issues == 0 {
		fmt.Println("No issues found. Verman is configured correctly.")
	} else {
		fmt.Printf("Found %d issue(s). See recommendations above.\n", issues)
	}
}

func checkVermanDirectory() int {
	fmt.Println("Checking verman directory...")
	issues := 0

	home, err := os.UserHomeDir()
	if err != nil {
		printFail("Cannot determine home directory: %v", err)
		return 1
	}

	vermanDir := filepath.Join(home, ".verman")
	versionsDir := filepath.Join(vermanDir, "versions")
	binDir := filepath.Join(vermanDir, "bin")

	// Check .verman exists
	if _, err := os.Stat(vermanDir); os.IsNotExist(err) {
		printFail(".verman directory not found at %s", vermanDir)
		printHint("Run: verman setup")
		issues++
	} else {
		printPass(".verman directory exists")
	}

	// Check versions directory
	if _, err := os.Stat(versionsDir); os.IsNotExist(err) {
		printFail("versions directory not found")
		printHint("Run: verman setup")
		issues++
	} else {
		printPass("versions directory exists")
	}

	// Check bin directory
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		printFail("bin directory not found (shims)")
		printHint("Run: verman setup")
		issues++
	} else {
		printPass("bin directory exists")
	}

	fmt.Println()
	return issues
}

func checkPath() int {
	fmt.Println("Checking PATH configuration...")
	issues := 0

	home, _ := os.UserHomeDir()
	binDir := filepath.Join(home, ".verman", "bin")

	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)

	found := false
	for _, p := range paths {
		if strings.EqualFold(p, binDir) {
			found = true
			break
		}
	}

	if !found {
		printFail("verman bin directory not in PATH")
		printHint("Add %s to your PATH", binDir)
		printHint("Or run: verman init --install")
		issues++
	} else {
		printPass("verman bin directory is in PATH")
	}

	fmt.Println()
	return issues
}

func checkInstalledTools() int {
	fmt.Println("Checking installed tools...")
	issues := 0

	mgr := version.NewManager(cfg)
	installedCount := 0

	for _, langName := range languages.Names() {
		versions, err := mgr.ListInstalled(langName)
		if err != nil {
			continue
		}

		if len(versions) > 0 {
			current, _ := mgr.GetCurrent(langName)
			if current != "" {
				printPass("%s: %d version(s) installed, current: %s", langName, len(versions), current)
			} else {
				printWarn("%s: %d version(s) installed, but none selected", langName, len(versions))
				printHint("Run: verman use %s <version>", langName)
				issues++
			}
			installedCount++
		}
	}

	if installedCount == 0 {
		printInfo("No tools installed yet")
		printHint("Get started with: verman install java 21")
	}

	fmt.Println()
	return issues
}

func checkEnvironment() int {
	fmt.Println("Checking environment variables...")
	issues := 0

	mgr := version.NewManager(cfg)

	// Check JAVA_HOME if Java is installed
	javaVersions, _ := mgr.ListInstalled("java")
	if len(javaVersions) > 0 {
		javaHome := os.Getenv("JAVA_HOME")
		if javaHome == "" {
			printWarn("JAVA_HOME is not set")
			printHint("Run: verman env | Invoke-Expression")
			issues++
		} else if _, err := os.Stat(javaHome); os.IsNotExist(err) {
			printFail("JAVA_HOME points to non-existent directory: %s", javaHome)
			printHint("Run: verman env | Invoke-Expression")
			issues++
		} else {
			// Check if java.exe exists
			javaExe := filepath.Join(javaHome, "bin", "java.exe")
			if _, err := os.Stat(javaExe); os.IsNotExist(err) {
				printFail("JAVA_HOME is set but java.exe not found at %s", javaExe)
				issues++
			} else {
				printPass("JAVA_HOME is set correctly: %s", javaHome)
			}
		}
	}

	// Check GOROOT if Go is installed
	goVersions, _ := mgr.ListInstalled("go")
	if len(goVersions) > 0 {
		goRoot := os.Getenv("GOROOT")
		if goRoot == "" {
			printInfo("GOROOT is not set (optional)")
		} else {
			printPass("GOROOT is set: %s", goRoot)
		}
	}

	fmt.Println()
	return issues
}

func checkDependencies() int {
	fmt.Println("Checking dependencies...")
	issues := 0

	mgr := version.NewManager(cfg)

	// Check if JVM tools have Java installed
	jvmTools := []string{"maven", "gradle", "scala", "scala3", "sbt", "kotlin", "mill"}
	javaVersions, _ := mgr.ListInstalled("java")
	javaInstalled := len(javaVersions) > 0

	for _, tool := range jvmTools {
		versions, _ := mgr.ListInstalled(tool)
		if len(versions) > 0 && !javaInstalled {
			printFail("%s is installed but Java is not", tool)
			printHint("Install Java: verman install java 21")
			issues++
		}
	}

	if issues == 0 {
		// Try to run java -version if Java is installed
		if javaInstalled {
			cmd := exec.Command("java", "-version")
			if err := cmd.Run(); err != nil {
				printWarn("Java is installed but 'java -version' failed")
				printHint("Run: verman env | Invoke-Expression")
				issues++
			} else {
				printPass("Java is accessible from PATH")
			}
		}
	}

	fmt.Println()
	return issues
}

func printPass(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if runtime.GOOS == "windows" {
		fmt.Printf("  [OK] %s\n", msg)
	} else {
		fmt.Printf("  \033[32m✓\033[0m %s\n", msg)
	}
}

func printFail(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if runtime.GOOS == "windows" {
		fmt.Printf("  [FAIL] %s\n", msg)
	} else {
		fmt.Printf("  \033[31m✗\033[0m %s\n", msg)
	}
}

func printWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if runtime.GOOS == "windows" {
		fmt.Printf("  [WARN] %s\n", msg)
	} else {
		fmt.Printf("  \033[33m!\033[0m %s\n", msg)
	}
}

func printInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if runtime.GOOS == "windows" {
		fmt.Printf("  [INFO] %s\n", msg)
	} else {
		fmt.Printf("  \033[34mi\033[0m %s\n", msg)
	}
}

func printHint(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("         Hint: %s\n", msg)
}
