package version

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/azdren/verman/internal/languages"
)

// DetectedVersion holds a detected version and its source
type DetectedVersion struct {
	Language string
	Version  string
	Source   string // file path that specified this version
}

// DetectAll scans the given directory for version files
func DetectAll(dir string) ([]DetectedVersion, error) {
	var detected []DetectedVersion

	for _, lang := range languages.All() {
		if dv := detectForLanguage(dir, lang); dv != nil {
			detected = append(detected, *dv)
		}
	}

	return detected, nil
}

// DetectForLanguage finds version for a specific language
func DetectForLanguage(dir, langName string) *DetectedVersion {
	lang, ok := languages.Get(langName)
	if !ok {
		return nil
	}
	return detectForLanguage(dir, lang)
}

func detectForLanguage(dir string, lang languages.Language) *DetectedVersion {
	for _, versionFile := range lang.VersionFiles() {
		// Search up the directory tree
		currentDir := dir
		for {
			filePath := filepath.Join(currentDir, versionFile)
			if version := readVersionFile(filePath, lang.Name()); version != "" {
				return &DetectedVersion{
					Language: lang.Name(),
					Version:  version,
					Source:   filePath,
				}
			}

			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break // reached root
			}
			currentDir = parent
		}
	}
	return nil
}

func readVersionFile(path string, langName string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(data))
	filename := filepath.Base(path)

	switch filename {
	case "global.json":
		// .NET global.json: {"sdk": {"version": "8.0.100"}}
		return parseGlobalJson(data)

	case "go.mod":
		// Go mod: look for "go 1.21" line
		return parseGoMod(content)

	case "rust-toolchain.toml":
		// Rust: [toolchain] channel = "1.75.0"
		return parseRustToolchain(content)

	case ".sdkmanrc":
		// SDKMAN: java=17.0.9-tem
		return parseSdkmanrc(content, langName)

	default:
		// Simple version files (.nvmrc, .java-version, etc.)
		// Just return first line, stripping 'v' prefix if present
		version := strings.Split(content, "\n")[0]
		version = strings.TrimPrefix(version, "v")
		return version
	}
}

func parseGlobalJson(data []byte) string {
	var gj struct {
		SDK struct {
			Version string `json:"version"`
		} `json:"sdk"`
	}
	if err := json.Unmarshal(data, &gj); err != nil {
		return ""
	}
	return gj.SDK.Version
}

func parseGoMod(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			return strings.TrimPrefix(line, "go ")
		}
	}
	return ""
}

func parseRustToolchain(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "channel") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			}
		}
	}
	return ""
}

func parseSdkmanrc(content, langName string) string {
	// Only parse for Java currently
	if langName != "java" {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "java=") {
			version := strings.TrimPrefix(line, "java=")
			// SDKMAN format: 17.0.9-tem -> extract major version
			if idx := strings.Index(version, "."); idx > 0 {
				return version[:idx]
			}
			return version
		}
	}
	return ""
}
