package languages

import (
	"github.com/azdren/verman/internal/sources"
)

func init() {
	// Auto-load embedded sources when package is imported
	// This ensures tests and simple usage work without explicit initialization
	// CLI commands can call sources.Load() again with user sources dir to override
	sources.Load("")
	LoadFromSources()
}

// Language defines the interface for a managed language/runtime
type Language interface {
	// Name returns the language identifier (java, node, etc.)
	Name() string

	// DisplayName returns the human-readable name
	DisplayName() string

	// EnvVars returns environment variables to set for this language
	// Key is the env var name, value is relative to version root
	EnvVars() map[string]string

	// PathDirs returns directories to add to PATH (relative to version root)
	PathDirs() []string

	// VersionFile returns the filename used for per-project version detection
	VersionFiles() []string

	// ValidateVersion checks if a version string is valid
	ValidateVersion(version string) bool

	// ResolveVersion resolves a partial version to a full version
	// e.g., "20" -> "20.18.0" for Node.js
	ResolveVersion(version string) (string, error)

	// GetDownloadURL returns the download URL for a specific version
	GetDownloadURL(version string) (string, error)

	// GetDownloadURLWithDist returns the download URL for a specific version and distribution
	GetDownloadURLWithDist(version, distribution string) (string, error)

	// HasDistributions returns true if this language supports multiple distributions
	HasDistributions() bool

	// GetDistributionDisplayName returns the display name for a distribution
	GetDistributionDisplayName(dist string) string

	// GetExtractPattern returns the expected folder name inside the archive
	GetExtractPattern(version string) string

	// PostInstall runs any post-installation steps
	PostInstall(versionPath string) error

	// VersionCommand returns the command to check the installed version
	VersionCommand() string

	// GetDependencies returns the list of other tools this depends on
	GetDependencies() []string
}

// SourceLanguage adapts a Source to the Language interface
type SourceLanguage struct {
	source *sources.Source
}

func (sl *SourceLanguage) Name() string {
	return sl.source.Name
}

func (sl *SourceLanguage) DisplayName() string {
	return sl.source.DisplayName
}

func (sl *SourceLanguage) EnvVars() map[string]string {
	return sl.source.EnvVars
}

func (sl *SourceLanguage) PathDirs() []string {
	return sl.source.PathDirs
}

func (sl *SourceLanguage) VersionFiles() []string {
	return sl.source.VersionFiles
}

func (sl *SourceLanguage) ValidateVersion(version string) bool {
	return sl.source.ValidateVersion(version)
}

func (sl *SourceLanguage) ResolveVersion(version string) (string, error) {
	return sl.source.ResolveVersion(version)
}

func (sl *SourceLanguage) GetDownloadURL(version string) (string, error) {
	return sl.source.GetDownloadURL(version), nil
}

func (sl *SourceLanguage) GetDownloadURLWithDist(version, distribution string) (string, error) {
	return sl.source.GetDownloadURLWithDist(version, distribution), nil
}

func (sl *SourceLanguage) HasDistributions() bool {
	return len(sl.source.Distributions) > 0
}

func (sl *SourceLanguage) GetDistributionDisplayName(dist string) string {
	return sl.source.GetDistributionDisplayName(dist)
}

func (sl *SourceLanguage) GetExtractPattern(version string) string {
	return sl.source.GetExtractPattern(version)
}

func (sl *SourceLanguage) PostInstall(versionPath string) error {
	// TODO: Implement post-install commands if needed
	return nil
}

func (sl *SourceLanguage) VersionCommand() string {
	// Generate version command based on language name
	switch sl.source.Name {
	case "java":
		return "java -version"
	case "node":
		return "node --version"
	case "go":
		return "go version"
	case "scala", "scala3":
		return "scala -version"
	case "kotlin":
		return "kotlin -version"
	case "gradle":
		return "gradle --version"
	case "maven":
		return "mvn --version"
	case "sbt":
		return "sbt --version"
	case "mill":
		return "mill --version"
	default:
		return sl.source.Name + " --version"
	}
}

func (sl *SourceLanguage) GetDependencies() []string {
	return sl.source.GetDependencies()
}

// Registry holds all supported languages
var Registry = make(map[string]Language)

// LoadFromSources loads all languages from the sources package
func LoadFromSources() error {
	for _, src := range sources.All() {
		Registry[src.Name] = &SourceLanguage{source: src}
	}
	return nil
}

func Register(lang Language) {
	Registry[lang.Name()] = lang
}

func Get(name string) (Language, bool) {
	lang, ok := Registry[name]
	return lang, ok
}

func All() []Language {
	langs := make([]Language, 0, len(Registry))
	for _, lang := range Registry {
		langs = append(langs, lang)
	}
	return langs
}

func Names() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}
