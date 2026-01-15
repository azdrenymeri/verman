package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&DotNet{})
}

type DotNet struct{}

func (d *DotNet) Name() string {
	return "dotnet"
}

func (d *DotNet) EnvVars() map[string]string {
	return map[string]string{
		"DOTNET_ROOT": ".",
	}
}

func (d *DotNet) PathDirs() []string {
	return []string{"."}
}

func (d *DotNet) VersionFiles() []string {
	return []string{"global.json"}
}

func (d *DotNet) ValidateVersion(version string) bool {
	// Matches: 6.0, 7.0, 8.0, 6.0.417, 8.0.100
	match, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	return match
}

func (d *DotNet) GetDownloadURL(version string) (string, error) {
	// .NET SDK downloads from dotnet.microsoft.com
	// https://dotnet.microsoft.com/download/dotnet/scripts/v1/dotnet-install.ps1
	// Direct: needs channel selection, using install script is easier
	// https://dotnetcli.azureedge.net/dotnet/Sdk/8.0.100/dotnet-sdk-8.0.100-win-x64.zip
	return fmt.Sprintf(
		"https://dotnetcli.azureedge.net/dotnet/Sdk/%s/dotnet-sdk-%s-win-x64.zip",
		version, version,
	), nil
}

func (d *DotNet) PostInstall(versionPath string) error {
	return nil
}

func (d *DotNet) VersionCommand() string {
	return "dotnet --version"
}
