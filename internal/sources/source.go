package sources

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

//go:embed definitions/*.json
var embeddedSources embed.FS

// Distribution represents a vendor-specific distribution
type Distribution struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	DownloadURL string `json:"downloadUrl"`
}

// Source represents a language/tool source configuration
type Source struct {
	Name           string                   `json:"name"`
	DisplayName    string                   `json:"displayName"`
	ReleasesURL    string                   `json:"releasesUrl"`
	ReleasesPath   string                   `json:"releasesPath,omitempty"`  // JSON path to versions array
	VersionField   string                   `json:"versionField,omitempty"`  // Field name for version in releases
	VersionPrefix  string                   `json:"versionPrefix,omitempty"` // Prefix to strip from versions (e.g., "maven-")
	DownloadURL    string                   `json:"downloadUrl"`
	DownloadType   string                   `json:"downloadType,omitempty"`   // "zip" (default), "file" for single file downloads
	ExtractPattern string                   `json:"extractPattern,omitempty"` // Folder name inside archive
	VersionRegex   string                   `json:"versionRegex"`
	VersionFiles   []string                 `json:"versionFiles"`
	EnvVars        map[string]string        `json:"envVars"`
	PathDirs       []string                 `json:"pathDirs"`
	PostInstall    []string                 `json:"postInstall,omitempty"`         // Commands to run after install
	Dependencies   []string                 `json:"dependencies,omitempty"`        // Other tools this depends on (e.g., ["java"])
	Distributions  map[string]*Distribution `json:"distributions,omitempty"`       // Vendor distributions (for Java: tem, amzn, zulu)
	DefaultDist    string                   `json:"defaultDistribution,omitempty"` // Default distribution key
	StaticVersions []string                 `json:"staticVersions,omitempty"`      // Additional versions not in API (e.g., legacy versions)
}

var loadedSources map[string]*Source
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Load loads all source definitions
func Load(userSourcesDir string) error {
	loadedSources = make(map[string]*Source)

	// Load embedded sources first
	entries, err := embeddedSources.ReadDir("definitions")
	if err != nil {
		return fmt.Errorf("reading embedded sources: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := embeddedSources.ReadFile("definitions/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var src Source
		if err := json.Unmarshal(data, &src); err != nil {
			return fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		loadedSources[src.Name] = &src
	}

	// Override with user sources if they exist
	if userSourcesDir != "" {
		if _, err := os.Stat(userSourcesDir); err == nil {
			files, _ := filepath.Glob(filepath.Join(userSourcesDir, "*.json"))
			for _, file := range files {
				data, err := os.ReadFile(file)
				if err != nil {
					continue
				}

				var src Source
				if err := json.Unmarshal(data, &src); err != nil {
					continue
				}

				loadedSources[src.Name] = &src
			}
		}
	}

	return nil
}

// Get returns a source by name
func Get(name string) (*Source, bool) {
	src, ok := loadedSources[name]
	return src, ok
}

// All returns all loaded sources
func All() []*Source {
	result := make([]*Source, 0, len(loadedSources))
	for _, src := range loadedSources {
		result = append(result, src)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all source names
func Names() []string {
	names := make([]string, 0, len(loadedSources))
	for name := range loadedSources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateVersion checks if a version string is valid for this source
func (s *Source) ValidateVersion(version string) bool {
	if s.VersionRegex == "" {
		return true
	}
	match, _ := regexp.MatchString(s.VersionRegex, version)
	return match
}

// ParseVersionAndDistribution extracts version and distribution from a version string
// e.g., "21-tem" -> ("21", "tem"), "21" -> ("21", "")
func ParseVersionAndDistribution(version string) (string, string) {
	// Check for distribution suffix (SDKMAN-style: VERSION-VENDOR)
	parts := strings.Split(version, "-")
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		// Check if last part is a known distribution suffix
		distSuffixes := []string{"tem", "temurin", "amzn", "corretto", "zulu", "graal", "graalce"}
		for _, suffix := range distSuffixes {
			if strings.EqualFold(lastPart, suffix) {
				ver := strings.Join(parts[:len(parts)-1], "-")
				return ver, strings.ToLower(lastPart)
			}
		}
	}
	return version, ""
}

// NormalizeDistribution converts distribution aliases to canonical names
func NormalizeDistribution(dist string) string {
	switch strings.ToLower(dist) {
	case "tem", "temurin":
		return "temurin"
	case "amzn", "corretto":
		return "corretto"
	case "zulu":
		return "zulu"
	case "graal", "graalce":
		return "graalce"
	default:
		return dist
	}
}

// GetDownloadURL returns the download URL for a specific version
func (s *Source) GetDownloadURL(version string) string {
	return s.GetDownloadURLWithDist(version, "")
}

// GetDownloadURLWithDist returns the download URL for a specific version and distribution
func (s *Source) GetDownloadURLWithDist(version, dist string) string {
	var url string

	// If distributions are available, use them
	if len(s.Distributions) > 0 {
		// Normalize distribution name
		dist = NormalizeDistribution(dist)

		// Use default distribution if none specified
		if dist == "" {
			dist = s.DefaultDist
			if dist == "" {
				// Fall back to first available distribution
				for k := range s.Distributions {
					dist = k
					break
				}
			}
		}

		if d, ok := s.Distributions[dist]; ok {
			url = d.DownloadURL
		} else {
			// Fall back to default URL if distribution not found
			url = s.DownloadURL
		}
	} else {
		url = s.DownloadURL
	}

	url = strings.ReplaceAll(url, "{version}", version)
	// Handle version without dots (e.g., "21" for Java)
	url = strings.ReplaceAll(url, "{majorVersion}", strings.Split(version, ".")[0])
	return url
}

// GetDistributionDisplayName returns the display name for a distribution
func (s *Source) GetDistributionDisplayName(dist string) string {
	dist = NormalizeDistribution(dist)
	if d, ok := s.Distributions[dist]; ok {
		return d.DisplayName
	}
	return dist
}

// GetExtractPattern returns the expected folder name inside the archive
func (s *Source) GetExtractPattern(version string) string {
	if s.ExtractPattern == "" {
		return ""
	}
	pattern := s.ExtractPattern
	pattern = strings.ReplaceAll(pattern, "{version}", version)
	pattern = strings.ReplaceAll(pattern, "{majorVersion}", strings.Split(version, ".")[0])
	return pattern
}

// ResolveVersion resolves a partial version to a full version
// e.g., "20" -> "20.18.0" for Node.js
func (s *Source) ResolveVersion(partial string) (string, error) {
	if s.ReleasesURL == "" {
		// No releases URL, assume version is complete
		return partial, nil
	}

	versions, err := s.FetchVersions()
	if err != nil {
		// If we can't fetch versions, return error for partial versions
		// but allow exact-looking versions through
		if looksLikePartialVersion(partial) {
			return "", fmt.Errorf("could not fetch available versions to resolve %s: %w", partial, err)
		}
		return partial, nil
	}

	// Filter versions by regex if specified
	if s.VersionRegex != "" {
		re, err := regexp.Compile(s.VersionRegex)
		if err == nil {
			var filtered []string
			for _, v := range versions {
				if re.MatchString(v) || re.MatchString("v"+v) {
					filtered = append(filtered, v)
				}
			}
			versions = filtered
		}
	}

	// Find best matching version
	return s.findBestMatch(partial, versions)
}

// isWildcardVersion checks if version ends with .x, .X, or .*
func isWildcardVersion(v string) bool {
	v = strings.ToLower(v)
	return strings.HasSuffix(v, ".x") || strings.HasSuffix(v, ".*")
}

// stripWildcard removes the wildcard suffix from version
func stripWildcard(v string) string {
	v = strings.TrimSuffix(v, ".x")
	v = strings.TrimSuffix(v, ".X")
	v = strings.TrimSuffix(v, ".*")
	return v
}

// looksLikePartialVersion checks if a version string looks incomplete
// e.g., "2.13" looks partial, "2.13.12" looks complete, "2.13.x" is wildcard
func looksLikePartialVersion(v string) bool {
	v = strings.TrimPrefix(v, "v")

	// Wildcard patterns are always partial
	if isWildcardVersion(v) {
		return true
	}

	parts := strings.Split(v, ".")

	// Single number (like "21" for Java) - could be complete for some tools
	if len(parts) == 1 {
		return false // Allow single numbers through (Java uses major versions)
	}

	// Two parts like "2.13" - likely partial for most tools
	if len(parts) == 2 {
		return true
	}

	// Three or more parts - likely complete
	return false
}

// FetchVersions fetches available versions from the releases URL
func (s *Source) FetchVersions() ([]string, error) {
	var versions []string

	// Start with static versions if available
	if len(s.StaticVersions) > 0 {
		versions = append(versions, s.StaticVersions...)
	}

	// Fetch from API if URL is configured
	if s.ReleasesURL != "" {
		resp, err := httpClient.Get(s.ReleasesURL)
		if err != nil {
			// If we have static versions, return those even if API fails
			if len(versions) > 0 {
				return versions, nil
			}
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if len(versions) > 0 {
				return versions, nil
			}
			return nil, err
		}

		apiVersions, err := s.parseVersions(body)
		if err != nil {
			if len(versions) > 0 {
				return versions, nil
			}
			return nil, err
		}
		versions = append(versions, apiVersions...)
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(versions))
	for _, v := range versions {
		if !seen[v] {
			seen[v] = true
			unique = append(unique, v)
		}
	}

	return unique, nil
}

func (s *Source) parseVersions(data []byte) ([]string, error) {
	var versions []string

	// Try parsing as array first
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		for _, item := range arr {
			if v := s.extractVersion(item); v != "" {
				versions = append(versions, v)
			}
		}
		return versions, nil
	}

	// Try parsing as object with versions field
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// Check for common version array fields
		for _, field := range []string{"versions", "releases", "available_releases", "available_lts_releases"} {
			if arr, ok := obj[field].([]interface{}); ok {
				for _, item := range arr {
					if v := s.extractVersion(item); v != "" {
						versions = append(versions, v)
					}
				}
				if len(versions) > 0 {
					return versions, nil
				}
			}
		}
	}

	return versions, nil
}

func (s *Source) extractVersion(item interface{}) string {
	versionField := s.VersionField
	if versionField == "" {
		versionField = "version"
	}

	var ver string
	switch v := item.(type) {
	case string:
		ver = v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case map[string]interface{}:
		if val, ok := v[versionField]; ok {
			switch vv := val.(type) {
			case string:
				ver = vv
			case float64:
				return fmt.Sprintf("%.0f", vv)
			}
		}
	}

	if ver == "" {
		return ""
	}

	// Strip common prefixes
	ver = strings.TrimPrefix(ver, "v")
	if s.VersionPrefix != "" {
		ver = strings.TrimPrefix(ver, s.VersionPrefix)
	}

	return ver
}

func (s *Source) findBestMatch(partial string, versions []string) (string, error) {
	partial = strings.TrimPrefix(partial, "v")
	originalPartial := partial

	// Strip wildcard suffix (.x, .X, .*) before matching
	partial = stripWildcard(partial)

	// Exact match first (only if not originally a wildcard)
	if !isWildcardVersion(originalPartial) {
		for _, v := range versions {
			if v == partial {
				return v, nil
			}
		}
	}

	// Find latest version that starts with partial
	var matches []string
	for _, v := range versions {
		if v == partial || strings.HasPrefix(v, partial+".") || strings.HasPrefix(v, partial+"-") {
			matches = append(matches, v)
		}
	}

	// If no matches with prefix, also try matching major version for simple numbers
	if len(matches) == 0 && !strings.Contains(partial, ".") {
		for _, v := range versions {
			parts := strings.Split(v, ".")
			if len(parts) > 0 && parts[0] == partial {
				matches = append(matches, v)
			}
		}
	}

	if len(matches) == 0 {
		// For partial/wildcard versions, return error if no match found
		if looksLikePartialVersion(originalPartial) || isWildcardVersion(originalPartial) {
			return "", fmt.Errorf("no version found matching %s (available: check with list command)", originalPartial)
		}
		// For complete-looking versions, allow trying it
		return partial, nil
	}

	// Sort by version (semantic) and return highest
	sort.Slice(matches, func(i, j int) bool {
		return compareVersions(matches[i], matches[j]) > 0
	})

	return matches[0], nil
}

// compareVersions compares two version strings
// Returns >0 if a > b, <0 if a < b, 0 if equal
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			_, _ = fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			_, _ = fmt.Sscanf(bParts[i], "%d", &bNum)
		}
		if aNum != bNum {
			return aNum - bNum
		}
	}
	return 0
}

// GetDependencies returns the list of dependencies for this source
func (s *Source) GetDependencies() []string {
	return s.Dependencies
}

// DependencyStatus represents whether a dependency is satisfied
type DependencyStatus struct {
	Name      string
	Installed bool
	Version   string
}

// CheckDependencies checks if all dependencies are installed
// Returns a list of missing dependencies
func (s *Source) CheckDependencies(installedVersionsFunc func(string) ([]string, error)) []DependencyStatus {
	var statuses []DependencyStatus
	for _, dep := range s.Dependencies {
		status := DependencyStatus{Name: dep, Installed: false}
		if versions, err := installedVersionsFunc(dep); err == nil && len(versions) > 0 {
			status.Installed = true
			status.Version = versions[0] // First installed version
		}
		statuses = append(statuses, status)
	}
	return statuses
}
