package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Scala{})
}

type Scala struct{}

func (s *Scala) Name() string {
	return "scala"
}

func (s *Scala) EnvVars() map[string]string {
	return map[string]string{
		"SCALA_HOME": ".",
	}
}

func (s *Scala) PathDirs() []string {
	return []string{"bin"}
}

func (s *Scala) VersionFiles() []string {
	return []string{".scala-version"}
}

func (s *Scala) ValidateVersion(version string) bool {
	// Matches: 2.13.12, 3.3.1, 2.12.18
	match, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, version)
	return match
}

func (s *Scala) GetDownloadURL(version string) (string, error) {
	// Scala 3: https://github.com/lampepfl/dotty/releases/download/3.3.1/scala3-3.3.1.zip
	// Scala 2: https://downloads.lightbend.com/scala/2.13.12/scala-2.13.12.zip
	if version[0] == '3' {
		return fmt.Sprintf(
			"https://github.com/lampepfl/dotty/releases/download/%s/scala3-%s.zip",
			version, version,
		), nil
	}
	return fmt.Sprintf(
		"https://downloads.lightbend.com/scala/%s/scala-%s.zip",
		version, version,
	), nil
}

func (s *Scala) PostInstall(versionPath string) error {
	return nil
}

func (s *Scala) VersionCommand() string {
	return "scala -version"
}
