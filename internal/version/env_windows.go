//go:build windows

package version

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// setUserEnvVar sets a user environment variable persistently using multiple fallback methods
func setUserEnvVar(name, value string) error {
	// Method 1: Try Windows Registry API (most reliable on full Windows)
	if err := setEnvViaRegistry(name, value); err == nil {
		return nil
	}

	// Method 2: Try PowerShell Core (pwsh) - works on Nano Server
	if err := setEnvViaPowerShell("pwsh", name, value); err == nil {
		return nil
	}

	// Method 3: Try Windows PowerShell (powershell.exe)
	if err := setEnvViaPowerShell("powershell", name, value); err == nil {
		return nil
	}

	// Method 4: Try setx command (standard Windows, may not be in slim images)
	if err := setEnvViaSetx(name, value); err == nil {
		return nil
	}

	return fmt.Errorf("all methods failed to set environment variable")
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
	psCmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', '%s', 'User')", name, value)
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", psCmd)
	return cmd.Run()
}

// setEnvViaSetx uses the setx command
func setEnvViaSetx(name, value string) error {
	cmd := exec.Command("setx", name, value)
	return cmd.Run()
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
