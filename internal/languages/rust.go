package languages

import (
	"regexp"
)

func init() {
	Register(&Rust{})
}

type Rust struct{}

func (r *Rust) Name() string {
	return "rust"
}

func (r *Rust) EnvVars() map[string]string {
	return map[string]string{
		"RUSTUP_HOME": "rustup",
		"CARGO_HOME":  "cargo",
	}
}

func (r *Rust) PathDirs() []string {
	return []string{"cargo/bin"}
}

func (r *Rust) VersionFiles() []string {
	return []string{"rust-toolchain.toml", "rust-toolchain"}
}

func (r *Rust) ValidateVersion(version string) bool {
	// Matches: stable, nightly, 1.75.0, nightly-2024-01-01
	match, _ := regexp.MatchString(`^(stable|beta|nightly(-\d{4}-\d{2}-\d{2})?|\d+\.\d+\.\d+)$`, version)
	return match
}

func (r *Rust) GetDownloadURL(version string) (string, error) {
	// Rust uses rustup for version management
	// We'll handle this specially - install rustup and use it to manage versions
	// For standalone: https://static.rust-lang.org/dist/rust-1.75.0-x86_64-pc-windows-msvc.tar.gz
	return "https://win.rustup.rs/x86_64", nil // rustup-init.exe
}

func (r *Rust) PostInstall(versionPath string) error {
	// Rust installation is handled via rustup
	return nil
}

func (r *Rust) VersionCommand() string {
	return "rustc --version"
}
