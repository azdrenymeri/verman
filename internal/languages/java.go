package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Java{})
}

type Java struct{}

func (j *Java) Name() string {
	return "java"
}

func (j *Java) EnvVars() map[string]string {
	return map[string]string{
		"JAVA_HOME": ".", // Root of the version directory
	}
}

func (j *Java) PathDirs() []string {
	return []string{"bin"}
}

func (j *Java) VersionFiles() []string {
	return []string{".java-version", ".sdkmanrc"}
}

func (j *Java) ValidateVersion(version string) bool {
	// Matches versions like: 8, 11, 17, 21, 17.0.9, 21.0.1
	match, _ := regexp.MatchString(`^\d+(\.\d+(\.\d+)?)?$`, version)
	return match
}

func (j *Java) GetDownloadURL(version string) (string, error) {
	// Using Adoptium (Eclipse Temurin) builds
	// Example: https://api.adoptium.net/v3/binary/latest/21/ga/windows/x64/jdk/hotspot/normal/eclipse
	majorVersion := version
	if len(version) > 2 && version[2] == '.' {
		majorVersion = version[:2]
	} else if len(version) > 1 && version[1] == '.' {
		majorVersion = version[:1]
	}

	return fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%s/ga/windows/x64/jdk/hotspot/normal/eclipse",
		majorVersion,
	), nil
}

func (j *Java) PostInstall(versionPath string) error {
	// No post-install steps needed for Java
	return nil
}

func (j *Java) VersionCommand() string {
	return "java -version"
}
