//go:build windows

package version

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// setUserEnvVar sets a user environment variable persistently using multiple fallback methods
func setUserEnvVar(name, value string) error {
	var lastErr error
	var method string

	// Method 1: Try Windows Registry API (most reliable on full Windows)
	if err := setEnvViaRegistry(name, value); err == nil {
		// Verify it was actually written
		if v, e := getUserEnvVar(name); e == nil && v == value {
			method = "registry"
		} else {
			lastErr = fmt.Errorf("registry write succeeded but verification failed")
		}
	} else {
		lastErr = err
	}

	// Method 2: Try PowerShell Core (pwsh) - works on Nano Server
	if method == "" {
		if err := setEnvViaPowerShell("pwsh", name, value); err == nil {
			// Verify after PowerShell method
			if v, e := getUserEnvVar(name); e == nil && v == value {
				method = "pwsh"
			} else {
				lastErr = fmt.Errorf("pwsh write succeeded but verification failed")
			}
		} else {
			lastErr = err
		}
	}

	// Method 3: Try Windows PowerShell (powershell.exe)
	if method == "" {
		if err := setEnvViaPowerShell("powershell", name, value); err == nil {
			// Verify after PowerShell method
			if v, e := getUserEnvVar(name); e == nil && v == value {
				method = "powershell"
			} else {
				lastErr = fmt.Errorf("powershell write succeeded but verification failed")
			}
		} else {
			lastErr = err
		}
	}

	// Method 4: Try setx command (standard Windows, may not be in slim images)
	if method == "" {
		if err := setEnvViaSetx(name, value); err == nil {
			// Verify after setx
			if v, e := getUserEnvVar(name); e == nil && v == value {
				method = "setx"
			} else {
				lastErr = fmt.Errorf("setx write succeeded but verification failed")
			}
		} else {
			lastErr = err
		}
	}

	if method == "" {
		return fmt.Errorf("all methods failed: %v", lastErr)
	}

	// Also set in current process so it's immediately available
	os.Setenv(name, value)

	// For containers: also create a PowerShell profile entry as backup
	createPowerShellProfileEntry(name, value)

	return nil
}

// setEnvViaRegistry uses Windows Registry API directly
func setEnvViaRegistry(name, value string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue(name, value); err != nil {
		return err
	}

	// Try to broadcast change (will silently fail in containers without GUI)
	broadcastSettingChange()
	return nil
}

// setEnvViaPowerShell uses PowerShell to set the environment variable
func setEnvViaPowerShell(shell, name, value string) error {
	// Use a command that returns success/failure
	psCmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', '%s', 'User'); if ($?) { exit 0 } else { exit 1 }", name, value)
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// setEnvViaSetx uses the setx command
func setEnvViaSetx(name, value string) error {
	cmd := exec.Command("setx", name, value)
	return cmd.Run()
}

// createPowerShellProfileEntry adds env var to PowerShell profile for container persistence
func createPowerShellProfileEntry(name, value string) {
	// Get PowerShell profile path
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Try PowerShell Core profile first, then Windows PowerShell
	profiles := []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}

	for _, profilePath := range profiles {
		// Create directory if needed
		dir := filepath.Dir(profilePath)
		os.MkdirAll(dir, 0755)

		// Read existing profile
		content, _ := os.ReadFile(profilePath)
		contentStr := string(content)

		// Check if already set
		marker := fmt.Sprintf("# verman: %s", name)
		if contains(contentStr, marker) {
			// Already exists, skip
			continue
		}

		// Append to profile
		entry := fmt.Sprintf("\n%s\n$env:%s = \"%s\"\n", marker, name, value)
		f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		f.WriteString(entry)
		f.Close()
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// broadcastSettingChange notifies Windows that environment variables have changed
// This is a best-effort operation - it will silently fail in containers without GUI
func broadcastSettingChange() {
	defer func() {
		// Recover from any panic (e.g., if user32.dll is not available)
		recover()
	}()

	user32 := syscall.NewLazyDLL("user32.dll")
	if err := user32.Load(); err != nil {
		return // user32.dll not available (slim container)
	}

	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")
	if err := sendMessageTimeout.Find(); err != nil {
		return // proc not available
	}

	envPtr, _ := syscall.UTF16PtrFromString("Environment")
	sendMessageTimeout.Call(
		uintptr(0xFFFF), // HWND_BROADCAST
		uintptr(0x001A), // WM_SETTINGCHANGE
		uintptr(0),
		uintptr(unsafe.Pointer(envPtr)),
		uintptr(0x0002), // SMTO_ABORTIFHUNG
		uintptr(5000),
		uintptr(0),
	)
}

// getUserEnvVar gets a user environment variable from the registry
func getUserEnvVar(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err != nil {
		return "", err
	}
	return value, nil
}
