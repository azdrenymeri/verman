package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Ruby{})
}

type Ruby struct{}

func (r *Ruby) Name() string {
	return "ruby"
}

func (r *Ruby) EnvVars() map[string]string {
	return map[string]string{
		"GEM_HOME": "gems",
		"GEM_PATH": "gems",
	}
}

func (r *Ruby) PathDirs() []string {
	return []string{"bin", "gems/bin"}
}

func (r *Ruby) VersionFiles() []string {
	return []string{".ruby-version"}
}

func (r *Ruby) ValidateVersion(version string) bool {
	match, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, version)
	return match
}

func (r *Ruby) GetDownloadURL(version string) (string, error) {
	// RubyInstaller: https://github.com/oneclick/rubyinstaller2/releases
	// 7z archive: https://github.com/oneclick/rubyinstaller2/releases/download/RubyInstaller-3.2.2-1/rubyinstaller-3.2.2-1-x64.7z
	return fmt.Sprintf(
		"https://github.com/oneclick/rubyinstaller2/releases/download/RubyInstaller-%s-1/rubyinstaller-%s-1-x64.7z",
		version, version,
	), nil
}

func (r *Ruby) PostInstall(versionPath string) error {
	return nil
}

func (r *Ruby) VersionCommand() string {
	return "ruby --version"
}
