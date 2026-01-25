package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install verman and configure PATH",
	Long: `Setup verman for first-time use.

This command will:
1. Copy verman.exe to ~/.verman/bin/
2. Add ~/.verman/bin to your PATH (permanently)
3. Create the versions directory structure

After running this command, restart your terminal and you can use 'verman' from anywhere.

Examples:
  verman setup              # Full setup
  verman setup --path-only  # Only add current location to PATH`,
	Run: func(cmd *cobra.Command, args []string) {
		pathOnly, _ := cmd.Flags().GetBool("path-only")

		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		vermanBinDir := filepath.Join(home, ".verman", "bin")
		vermanExePath := filepath.Join(vermanBinDir, "verman.exe")

		if pathOnly {
			// Just add current directory to PATH
			currentExe, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
				os.Exit(1)
			}
			currentDir := filepath.Dir(currentExe)
			addToPath(currentDir)
			return
		}

		// Full setup
		fmt.Println("Setting up verman...")

		// 1. Create directories
		fmt.Print("Creating directories... ")
		dirs := []string{
			vermanBinDir,
			filepath.Join(home, ".verman", "versions"),
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "\nError creating %s: %v\n", dir, err)
				os.Exit(1)
			}
		}
		fmt.Println("done")

		// 2. Copy executable
		fmt.Print("Installing verman.exe... ")
		currentExe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError getting executable path: %v\n", err)
			os.Exit(1)
		}

		// Don't copy if already in the right place
		if filepath.Clean(currentExe) != filepath.Clean(vermanExePath) {
			if err := copyFile(currentExe, vermanExePath); err != nil {
				fmt.Fprintf(os.Stderr, "\nError copying executable: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("done")
		} else {
			fmt.Println("already installed")
		}

		// 3. Add to PATH
		addToPath(vermanBinDir)

		// 4. Summary
		fmt.Println("\n✓ Setup complete!")
		fmt.Println("\nTo use verman now, refresh your PATH:")
		fmt.Printf("  $env:PATH = \"%s;\" + $env:PATH\n", vermanBinDir)
		fmt.Println("\nOr simply restart your terminal.")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Install a language:  verman install java 21")
		fmt.Println("  2. Set up shell integration for language runtimes:")
		fmt.Println("     verman init --install")
	},
}

func addToPath(dir string) {
	fmt.Print("Adding to PATH... ")

	// Get current user PATH
	currentPath := os.Getenv("PATH")

	// Check if already in PATH
	pathDirs := strings.Split(currentPath, ";")
	for _, p := range pathDirs {
		if strings.EqualFold(filepath.Clean(p), filepath.Clean(dir)) {
			fmt.Println("already in PATH")
			return
		}
	}

	// Add to user PATH permanently using PowerShell
	// We use [Environment]::SetEnvironmentVariable which modifies the registry
	newPath := getCurrentUserPath() + ";" + dir

	if err := setUserPath(newPath); err != nil {
		fmt.Fprintf(os.Stderr, "\nError updating PATH: %v\n", err)
		fmt.Println("\nManually add this to your PATH:")
		fmt.Printf("  %s\n", dir)
		return
	}

	fmt.Println("done")
	fmt.Printf("  Added: %s\n", dir)
}

func getCurrentUserPath() string {
	// Read from registry via PowerShell
	// This gets the User PATH, not the combined PATH
	cmd := `[Environment]::GetEnvironmentVariable("PATH", "User")`
	out, err := runPowerShell(cmd)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func setUserPath(newPath string) error {
	// Set user PATH via PowerShell
	cmd := fmt.Sprintf(`[Environment]::SetEnvironmentVariable("PATH", "%s", "User")`, newPath)
	_, err := runPowerShell(cmd)
	return err
}

func runPowerShell(command string) (string, error) {
	// Try pwsh (PowerShell Core) first, fall back to powershell (Windows PowerShell)
	shells := []string{"pwsh", "powershell"}
	for _, shell := range shells {
		cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", command)
		out, err := cmd.Output()
		if err == nil {
			return string(out), nil
		}
		// If shell not found, try next one
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			continue
		}
		// Other error (command failed), return it
		return string(out), err
	}
	return "", fmt.Errorf("no PowerShell found (tried pwsh, powershell)")
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()

	_, err = io.Copy(dest, source)
	return err
}

func init() {
	setupCmd.Flags().Bool("path-only", false, "Only add current location to PATH")
	rootCmd.AddCommand(setupCmd)
}
