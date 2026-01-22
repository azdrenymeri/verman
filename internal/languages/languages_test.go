package languages

import (
	"testing"

	"github.com/azdren/verman/internal/sources"
)

func init() {
	// Load sources for testing
	sources.Load("")
	LoadFromSources()
}

func TestAllLanguagesRegistered(t *testing.T) {
	// Core languages that should always be available
	expected := []string{"java", "node", "scala", "go", "gradle"}

	for _, name := range expected {
		if _, ok := Get(name); !ok {
			t.Errorf("Language %s should be registered", name)
		}
	}
}

func TestLanguageNames(t *testing.T) {
	names := Names()

	// Should have at least the core languages
	if len(names) < 5 {
		t.Errorf("Expected at least 5 languages, got %d", len(names))
	}
}

func TestJavaValidateVersion(t *testing.T) {
	java, ok := Get("java")
	if !ok {
		t.Fatal("Java language not found")
	}

	valid := []string{"8", "11", "17", "21", "17.0.9", "21.0.1"}
	invalid := []string{"", "abc", "17.x", "latest"}

	for _, v := range valid {
		if !java.ValidateVersion(v) {
			t.Errorf("Java version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if java.ValidateVersion(v) {
			t.Errorf("Java version %s should be invalid", v)
		}
	}
}

func TestNodeValidateVersion(t *testing.T) {
	node, ok := Get("node")
	if !ok {
		t.Fatal("Node language not found")
	}

	valid := []string{"18", "20", "18.19.0", "v20.10.0", "20.10.0"}
	invalid := []string{"", "lts", "latest", "node18"}

	for _, v := range valid {
		if !node.ValidateVersion(v) {
			t.Errorf("Node version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if node.ValidateVersion(v) {
			t.Errorf("Node version %s should be invalid", v)
		}
	}
}

func TestScalaValidateVersion(t *testing.T) {
	scala, ok := Get("scala")
	if !ok {
		t.Fatal("Scala language not found")
	}

	// Scala 2.x versions (scala definition handles 2.x)
	valid := []string{"2.13.12", "2.12.18"}
	invalid := []string{"latest"}

	for _, v := range valid {
		if !scala.ValidateVersion(v) {
			t.Errorf("Scala 2 version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if scala.ValidateVersion(v) {
			t.Errorf("Scala version %s should be invalid", v)
		}
	}

	// Scala 3.x is a separate definition (scala3)
	scala3, ok := Get("scala3")
	if !ok {
		t.Fatal("Scala3 language not found")
	}

	scala3Valid := []string{"3.3.1", "3.4.0"}
	for _, v := range scala3Valid {
		if !scala3.ValidateVersion(v) {
			t.Errorf("Scala 3 version %s should be valid", v)
		}
	}
}

func TestGradleValidateVersion(t *testing.T) {
	gradle, ok := Get("gradle")
	if !ok {
		t.Fatal("Gradle language not found")
	}

	valid := []string{"8.5", "8.4.1", "7.6.3"}
	invalid := []string{"", "latest", "gradle-8.5"}

	for _, v := range valid {
		if !gradle.ValidateVersion(v) {
			t.Errorf("Gradle version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if gradle.ValidateVersion(v) {
			t.Errorf("Gradle version %s should be invalid", v)
		}
	}
}

func TestLanguageEnvVars(t *testing.T) {
	tests := []struct {
		lang     string
		expected map[string]string
	}{
		{"java", map[string]string{"JAVA_HOME": "."}},
		{"scala", map[string]string{"SCALA_HOME": "."}},
		{"go", map[string]string{"GOROOT": "."}},
		{"gradle", map[string]string{"GRADLE_HOME": "."}},
	}

	for _, tt := range tests {
		lang, ok := Get(tt.lang)
		if !ok {
			t.Fatalf("Language %s not found", tt.lang)
		}

		envVars := lang.EnvVars()
		for key, expected := range tt.expected {
			if envVars[key] != expected {
				t.Errorf("%s: ENV %s expected %s, got %s", tt.lang, key, expected, envVars[key])
			}
		}
	}
}

func TestLanguagePathDirs(t *testing.T) {
	tests := []struct {
		lang     string
		contains string
	}{
		{"java", "bin"},
		{"scala", "bin"},
		{"node", "."},
		{"go", "bin"},
		{"gradle", "bin"},
	}

	for _, tt := range tests {
		lang, ok := Get(tt.lang)
		if !ok {
			t.Fatalf("Language %s not found", tt.lang)
		}

		pathDirs := lang.PathDirs()
		found := false
		for _, dir := range pathDirs {
			if dir == tt.contains {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("%s: PathDirs should contain %s, got %v", tt.lang, tt.contains, pathDirs)
		}
	}
}

func TestLanguageVersionFiles(t *testing.T) {
	tests := []struct {
		lang     string
		contains string
	}{
		{"java", ".java-version"},
		{"node", ".nvmrc"},
		{"scala", ".scala-version"},
		{"go", "go.mod"},
		{"gradle", ".gradle-version"},
	}

	for _, tt := range tests {
		lang, ok := Get(tt.lang)
		if !ok {
			t.Fatalf("Language %s not found", tt.lang)
		}

		files := lang.VersionFiles()
		found := false
		for _, f := range files {
			if f == tt.contains {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("%s: VersionFiles should contain %s, got %v", tt.lang, tt.contains, files)
		}
	}
}

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		lang    string
		version string
		wantErr bool
	}{
		{"java", "21", false},
		{"node", "20.10.0", false},
		{"scala", "2.13.12", false},
		{"scala3", "3.3.1", false},
		{"go", "1.21.5", false},
		{"gradle", "8.5", false},
	}

	for _, tt := range tests {
		lang, ok := Get(tt.lang)
		if !ok {
			t.Fatalf("Language %s not found", tt.lang)
		}

		url, err := lang.GetDownloadURL(tt.version)
		if tt.wantErr && err == nil {
			t.Errorf("%s %s: expected error", tt.lang, tt.version)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s %s: unexpected error: %v", tt.lang, tt.version, err)
		}
		if !tt.wantErr && url == "" {
			t.Errorf("%s %s: expected non-empty URL", tt.lang, tt.version)
		}
	}
}
