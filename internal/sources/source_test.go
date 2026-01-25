package sources

import (
	"testing"
)

func init() {
	_ = Load("")
}

func TestLoadSources(t *testing.T) {
	sources := All()
	if len(sources) == 0 {
		t.Fatal("Expected at least one source to be loaded")
	}

	// Check that key sources are loaded
	expected := []string{"java", "node", "scala", "scala3", "go", "gradle", "maven", "sbt"}
	for _, name := range expected {
		if _, ok := Get(name); !ok {
			t.Errorf("Expected source %s to be loaded", name)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		source  string
		version string
		valid   bool
	}{
		// Java versions
		{"java", "21", true},
		{"java", "17.0.9", true},
		{"java", "8", true},
		{"java", "21-tem", true},
		{"java", "21-amzn", true},
		{"java", "21-zulu", true},
		{"java", "latest", false},

		// Node versions
		{"node", "20", true},
		{"node", "20.10.0", true},
		{"node", "v20.10.0", true},
		{"node", "lts", false},

		// Scala 2 versions
		{"scala", "2.13.12", true},
		{"scala", "2.12.18", true},
		{"scala", "3.3.1", false}, // Scala 3 should not match scala 2 regex

		// Scala 3 versions
		{"scala3", "3.3.1", true},
		{"scala3", "3.4.0", true},
		{"scala3", "2.13.12", false}, // Scala 2 should not match scala 3 regex

		// Go versions
		{"go", "1.21.5", true},
		{"go", "1.22", true},
		{"go", "go1.21", false},

		// Gradle versions
		{"gradle", "8.5", true},
		{"gradle", "8.4.1", true},
		{"gradle", "gradle-8.5", false},
	}

	for _, tt := range tests {
		src, ok := Get(tt.source)
		if !ok {
			t.Fatalf("Source %s not found", tt.source)
		}

		result := src.ValidateVersion(tt.version)
		if result != tt.valid {
			t.Errorf("%s version %q: expected valid=%v, got valid=%v", tt.source, tt.version, tt.valid, result)
		}
	}
}

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		source   string
		version  string
		contains string
	}{
		{"java", "21", "adoptium.net"},
		{"node", "20.10.0", "nodejs.org"},
		{"scala", "2.13.12", "github.com/scala/scala"},
		{"scala3", "3.3.1", "github.com/scala/scala3"},
		{"go", "1.21.5", "go.dev"},
		{"gradle", "8.5", "gradle.org"},
	}

	for _, tt := range tests {
		src, ok := Get(tt.source)
		if !ok {
			t.Fatalf("Source %s not found", tt.source)
		}

		url := src.GetDownloadURL(tt.version)
		if url == "" {
			t.Errorf("%s %s: expected non-empty URL", tt.source, tt.version)
		}
		if tt.contains != "" && !containsStr(url, tt.contains) {
			t.Errorf("%s %s: URL %q should contain %q", tt.source, tt.version, url, tt.contains)
		}
	}
}

func TestParseVersions(t *testing.T) {
	tests := []struct {
		name     string
		source   *Source
		data     string
		expected []string
	}{
		{
			name:     "node style array",
			source:   &Source{VersionField: "version"},
			data:     `[{"version":"v20.10.0"},{"version":"v18.19.0"}]`,
			expected: []string{"20.10.0", "18.19.0"},
		},
		{
			name:     "github releases style",
			source:   &Source{VersionField: "tag_name"},
			data:     `[{"tag_name":"v2.13.12"},{"tag_name":"v2.13.11"}]`,
			expected: []string{"2.13.12", "2.13.11"},
		},
		{
			name:     "gradle style",
			source:   &Source{VersionField: "version"},
			data:     `[{"version":"8.5"},{"version":"8.4"}]`,
			expected: []string{"8.5", "8.4"},
		},
		{
			name:     "adoptium style",
			source:   &Source{VersionField: "available_releases"},
			data:     `{"available_releases":[21,17,11,8]}`,
			expected: []string{"21", "17", "11", "8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions, err := tt.source.parseVersions([]byte(tt.data))
			if err != nil {
				t.Fatalf("parseVersions failed: %v", err)
			}

			if len(versions) != len(tt.expected) {
				t.Errorf("expected %d versions, got %d: %v", len(tt.expected), len(versions), versions)
				return
			}

			for i, v := range versions {
				if v != tt.expected[i] {
					t.Errorf("version[%d]: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	tests := []struct {
		partial  string
		versions []string
		expected string
	}{
		// Exact match
		{"20.10.0", []string{"20.10.0", "20.9.0", "18.19.0"}, "20.10.0"},
		// Partial major version
		{"20", []string{"20.10.0", "20.9.0", "18.19.0"}, "20.10.0"},
		// Partial minor version
		{"20.9", []string{"20.10.0", "20.9.1", "20.9.0"}, "20.9.1"},
		// With v prefix
		{"v20", []string{"20.10.0", "20.9.0"}, "20.10.0"},
		// No match returns partial
		{"99", []string{"20.10.0", "18.19.0"}, "99"},
	}

	src := &Source{}
	for _, tt := range tests {
		result, _ := src.findBestMatch(tt.partial, tt.versions)
		if result != tt.expected {
			t.Errorf("findBestMatch(%q, %v): expected %q, got %q", tt.partial, tt.versions, tt.expected, result)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestWildcardVersions(t *testing.T) {
	tests := []struct {
		version    string
		isWildcard bool
	}{
		{"2.13.x", true},
		{"2.13.X", true},
		{"2.13.*", true},
		{"20.x", true},
		{"2.13", false},
		{"2.13.12", false},
		{"21", false},
	}

	for _, tt := range tests {
		result := isWildcardVersion(tt.version)
		if result != tt.isWildcard {
			t.Errorf("isWildcardVersion(%q): expected %v, got %v", tt.version, tt.isWildcard, result)
		}
	}
}

func TestStripWildcard(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2.13.x", "2.13"},
		{"2.13.X", "2.13"},
		{"2.13.*", "2.13"},
		{"20.x", "20"},
		{"2.13", "2.13"},
		{"2.13.12", "2.13.12"},
	}

	for _, tt := range tests {
		result := stripWildcard(tt.input)
		if result != tt.expected {
			t.Errorf("stripWildcard(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestFindBestMatchWithWildcard(t *testing.T) {
	tests := []struct {
		partial  string
		versions []string
		expected string
	}{
		// Wildcard patterns
		{"2.13.x", []string{"2.13.18", "2.13.12", "2.12.19"}, "2.13.18"},
		{"2.13.X", []string{"2.13.18", "2.13.12", "2.12.19"}, "2.13.18"},
		{"2.13.*", []string{"2.13.18", "2.13.12", "2.12.19"}, "2.13.18"},
		{"20.x", []string{"20.18.0", "20.10.0", "18.19.0"}, "20.18.0"},
	}

	src := &Source{}
	for _, tt := range tests {
		result, err := src.findBestMatch(tt.partial, tt.versions)
		if err != nil {
			t.Errorf("findBestMatch(%q, %v): unexpected error %v", tt.partial, tt.versions, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("findBestMatch(%q, %v): expected %q, got %q", tt.partial, tt.versions, tt.expected, result)
		}
	}
}

func TestParseVersionAndDistribution(t *testing.T) {
	tests := []struct {
		input        string
		expectedVer  string
		expectedDist string
	}{
		{"21", "21", ""},
		{"21-tem", "21", "tem"},
		{"21-temurin", "21", "temurin"},
		{"21-amzn", "21", "amzn"},
		{"21-corretto", "21", "corretto"},
		{"21-zulu", "21", "zulu"},
		{"17.0.9-tem", "17.0.9", "tem"},
		{"21-unknown", "21-unknown", ""},  // Unknown suffix not stripped
		{"21-beta-tem", "21-beta", "tem"}, // Multi-part version
	}

	for _, tt := range tests {
		ver, dist := ParseVersionAndDistribution(tt.input)
		if ver != tt.expectedVer || dist != tt.expectedDist {
			t.Errorf("ParseVersionAndDistribution(%q): expected (%q, %q), got (%q, %q)",
				tt.input, tt.expectedVer, tt.expectedDist, ver, dist)
		}
	}
}

func TestNormalizeDistribution(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tem", "temurin"},
		{"temurin", "temurin"},
		{"amzn", "corretto"},
		{"corretto", "corretto"},
		{"zulu", "zulu"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := NormalizeDistribution(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeDistribution(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestJavaDistributions(t *testing.T) {
	java, ok := Get("java")
	if !ok {
		t.Fatal("Java source not found")
	}

	if len(java.Distributions) == 0 {
		t.Fatal("Java should have distributions defined")
	}

	// Check that key distributions exist
	expectedDists := []string{"temurin", "corretto", "zulu"}
	for _, dist := range expectedDists {
		if _, ok := java.Distributions[dist]; !ok {
			t.Errorf("Java should have %s distribution", dist)
		}
	}

	// Check default distribution
	if java.DefaultDist != "temurin" {
		t.Errorf("Java default distribution should be temurin, got %s", java.DefaultDist)
	}
}

func TestGetDownloadURLWithDist(t *testing.T) {
	java, ok := Get("java")
	if !ok {
		t.Fatal("Java source not found")
	}

	// Test default distribution
	url := java.GetDownloadURLWithDist("21", "")
	if !containsStr(url, "adoptium.net") {
		t.Errorf("Default Java URL should use adoptium.net, got %s", url)
	}

	// Test Temurin explicitly
	url = java.GetDownloadURLWithDist("21", "temurin")
	if !containsStr(url, "adoptium.net") {
		t.Errorf("Temurin URL should use adoptium.net, got %s", url)
	}

	// Test Corretto
	url = java.GetDownloadURLWithDist("21", "corretto")
	if !containsStr(url, "corretto.aws") {
		t.Errorf("Corretto URL should use corretto.aws, got %s", url)
	}

	// Test Zulu
	url = java.GetDownloadURLWithDist("21", "zulu")
	if !containsStr(url, "azul.com") {
		t.Errorf("Zulu URL should use azul.com, got %s", url)
	}
}

func TestDependencies(t *testing.T) {
	// Check that JVM tools have java dependency
	jvmTools := []string{"scala", "scala3", "maven", "gradle", "sbt", "kotlin", "mill"}
	for _, tool := range jvmTools {
		src, ok := Get(tool)
		if !ok {
			t.Fatalf("Source %s not found", tool)
		}

		deps := src.GetDependencies()
		hasJava := false
		for _, dep := range deps {
			if dep == "java" {
				hasJava = true
				break
			}
		}

		if !hasJava {
			t.Errorf("%s should depend on java", tool)
		}
	}

	// Check that java has no dependencies
	java, ok := Get("java")
	if !ok {
		t.Fatal("Java source not found")
	}
	if len(java.GetDependencies()) > 0 {
		t.Error("Java should have no dependencies")
	}
}
