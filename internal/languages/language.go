package languages

// Language defines the interface for a managed language/runtime
type Language interface {
	// Name returns the language identifier (java, node, etc.)
	Name() string

	// EnvVars returns environment variables to set for this language
	// Key is the env var name, value is relative to version root
	EnvVars() map[string]string

	// PathDirs returns directories to add to PATH (relative to version root)
	PathDirs() []string

	// VersionFile returns the filename used for per-project version detection
	VersionFiles() []string

	// ValidateVersion checks if a version string is valid
	ValidateVersion(version string) bool

	// GetDownloadURL returns the download URL for a specific version
	GetDownloadURL(version string) (string, error)

	// PostInstall runs any post-installation steps
	PostInstall(versionPath string) error

	// VersionCommand returns the command to check the installed version
	VersionCommand() string
}

// Registry holds all supported languages
var Registry = make(map[string]Language)

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
