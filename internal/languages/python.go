package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Python{})
}

type Python struct{}

func (p *Python) Name() string {
	return "python"
}

func (p *Python) EnvVars() map[string]string {
	return map[string]string{
		"PYTHON_HOME": ".",
	}
}

func (p *Python) PathDirs() []string {
	return []string{".", "Scripts"}
}

func (p *Python) VersionFiles() []string {
	return []string{".python-version"}
}

func (p *Python) ValidateVersion(version string) bool {
	match, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	return match
}

func (p *Python) GetDownloadURL(version string) (string, error) {
	// https://www.python.org/ftp/python/3.12.1/python-3.12.1-amd64.exe
	// Note: Python distributes as installer, we'd need embedded zip
	// https://www.python.org/ftp/python/3.12.1/python-3.12.1-embed-amd64.zip
	return fmt.Sprintf(
		"https://www.python.org/ftp/python/%s/python-%s-embed-amd64.zip",
		version, version,
	), nil
}

func (p *Python) PostInstall(versionPath string) error {
	// Python embedded needs pip to be installed separately
	// For now, we'll handle this as a future enhancement
	return nil
}

func (p *Python) VersionCommand() string {
	return "python --version"
}
