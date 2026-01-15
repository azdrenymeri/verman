package languages

import (
	"fmt"
	"regexp"
)

func init() {
	Register(&Node{})
}

type Node struct{}

func (n *Node) Name() string {
	return "node"
}

func (n *Node) EnvVars() map[string]string {
	// Node doesn't use a HOME variable, just PATH
	return map[string]string{}
}

func (n *Node) PathDirs() []string {
	return []string{"."} // node.exe is in the root
}

func (n *Node) VersionFiles() []string {
	return []string{".nvmrc", ".node-version"}
}

func (n *Node) ValidateVersion(version string) bool {
	// Matches versions like: 18, 20, 18.19.0, v20.10.0
	match, _ := regexp.MatchString(`^v?\d+(\.\d+(\.\d+)?)?$`, version)
	return match
}

func (n *Node) GetDownloadURL(version string) (string, error) {
	// Strip leading 'v' if present for consistency
	v := version
	if len(v) > 0 && v[0] == 'v' {
		v = v[1:]
	}

	// https://nodejs.org/dist/v20.10.0/node-v20.10.0-win-x64.zip
	return fmt.Sprintf(
		"https://nodejs.org/dist/v%s/node-v%s-win-x64.zip",
		v, v,
	), nil
}

func (n *Node) PostInstall(versionPath string) error {
	return nil
}

func (n *Node) VersionCommand() string {
	return "node --version"
}
