package languages

import (
	"testing"
)

func TestAllLanguagesRegistered(t *testing.T) {
	expected := []string{"java", "node", "scala", "python", "ruby", "go", "rust", "dotnet"}

	for _, name := range expected {
		if _, ok := Get(name); !ok {
			t.Errorf("Language %s should be registered", name)
		}
	}
}

func TestLanguageNames(t *testing.T) {
	names := Names()

	if len(names) != 8 {
		t.Errorf("Expected 8 languages, got %d", len(names))
	}
}

func TestJavaValidateVersion(t *testing.T) {
	java := &Java{}

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
	node := &Node{}

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
	scala := &Scala{}

	valid := []string{"2.13.12", "3.3.1", "2.12.18"}
	invalid := []string{"3", "2.13", "latest"}

	for _, v := range valid {
		if !scala.ValidateVersion(v) {
			t.Errorf("Scala version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if scala.ValidateVersion(v) {
			t.Errorf("Scala version %s should be invalid", v)
		}
	}
}

func TestRustValidateVersion(t *testing.T) {
	rust := &Rust{}

	valid := []string{"stable", "beta", "nightly", "1.75.0", "nightly-2024-01-01"}
	invalid := []string{"", "latest", "1.75"}

	for _, v := range valid {
		if !rust.ValidateVersion(v) {
			t.Errorf("Rust version %s should be valid", v)
		}
	}

	for _, v := range invalid {
		if rust.ValidateVersion(v) {
			t.Errorf("Rust version %s should be invalid", v)
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
		{"dotnet", map[string]string{"DOTNET_ROOT": "."}},
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
		{"python", ".python-version"},
		{"ruby", ".ruby-version"},
		{"go", "go.mod"},
		{"rust", "rust-toolchain.toml"},
		{"dotnet", "global.json"},
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
		{"scala", "3.3.1", false},
		{"scala", "2.13.12", false},
		{"go", "1.21.5", false},
		{"dotnet", "8.0.100", false},
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
