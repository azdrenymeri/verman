package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/azdren/verman/internal/version"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Initialize shell integration",
	Long: `Generate shell integration scripts for verman.

Supported shells:
  powershell  - PowerShell (default on Windows)
  cmd         - Windows Command Prompt

Examples:
  verman init                    # Show PowerShell init script
  verman init powershell         # PowerShell integration
  verman init cmd                # CMD batch script
  verman init --install          # Install to profile automatically`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := "powershell"
		if len(args) > 0 {
			shell = args[0]
		}

		install, _ := cmd.Flags().GetBool("install")

		switch shell {
		case "powershell", "pwsh":
			script := version.GeneratePowerShellInit(cfg)
			if install {
				installPowerShellProfile(script)
			} else {
				fmt.Println(script)
				fmt.Println("\n# Add the above to your PowerShell profile:")
				fmt.Println("# notepad $PROFILE")
				fmt.Println("# Or run: verman init --install")
			}

		case "cmd":
			script := version.GenerateCmdInit(cfg)
			if install {
				installCmdScript(script)
			} else {
				fmt.Println(script)
				fmt.Println("\nREM Save this to a .bat file and run it in your shell")
				fmt.Println("REM Or run: verman init cmd --install")
			}

		default:
			fmt.Fprintf(os.Stderr, "Unknown shell: %s\n", shell)
			fmt.Fprintf(os.Stderr, "Supported: powershell, cmd\n")
			os.Exit(1)
		}
	},
}

func installPowerShellProfile(script string) {
	// Get PowerShell profile path
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	profilePath := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Read existing profile
	existing, _ := os.ReadFile(profilePath)

	// Check if already installed
	if len(existing) > 0 && contains(string(existing), "# Verman PowerShell Integration") {
		fmt.Println("Verman integration already installed in profile")
		fmt.Printf("Profile: %s\n", profilePath)
		return
	}

	// Append to profile
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening profile: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	if len(existing) > 0 {
		_, _ = f.WriteString("\n\n")
	}
	_, _ = f.WriteString(script)

	fmt.Printf("Installed to: %s\n", profilePath)
	fmt.Println("Restart PowerShell for changes to take effect")
}

func installCmdScript(script string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	scriptPath := filepath.Join(home, ".verman", "init.cmd")

	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Installed to: %s\n", scriptPath)
	fmt.Println("\nTo use, run this in CMD:")
	fmt.Printf("  %s\n", scriptPath)
	fmt.Println("\nOr add to your AUTOEXEC or create a shortcut")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func init() {
	initCmd.Flags().Bool("install", false, "Install to shell profile")
	rootCmd.AddCommand(initCmd)
}
