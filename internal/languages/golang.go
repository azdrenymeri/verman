package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Golang{})
}

type Golang struct{}

func (g *Golang) Name() string {
	return "go"
}

func (g *Golang) EnvVars() map[string]string {
	return map[string]string{
		"GOROOT": ".",
	}
}

func (g *Golang) PathDirs() []string {
	return []string{"bin"}
}

func (g *Golang) VersionFiles() []string {
	return []string{".go-version", "go.mod"} // go.mod contains version directive
}

func (g *Golang) ValidateVersion(version string) bool {
	match, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	return match
}

func (g *Golang) GetDownloadURL(version string) (string, error) {
	// https://go.dev/dl/go1.21.5.windows-amd64.zip
	return fmt.Sprintf(
		"https://go.dev/dl/go%s.windows-amd64.zip",
		version,
	), nil
}

func (g *Golang) PostInstall(versionPath string) error {
	return nil
}

func (g *Golang) VersionCommand() string {
	return "go version"
}
