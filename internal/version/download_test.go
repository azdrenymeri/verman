package version

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadWithRetry_Success(t *testing.T) {
	// Create a test server
	content := []byte("test file content for download")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "30")
		_, _ = w.Write(content)
	}))
	defer server.Close()

	// Create temp file
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download.txt")

	cfg := DefaultDownloadConfig()
	cfg.URL = server.URL
	cfg.DestPath = destPath
	cfg.Description = "test"

	result, err := DownloadWithRetry(cfg)
	if err != nil {
		t.Fatalf("DownloadWithRetry failed: %v", err)
	}

	if result.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), result.Size)
	}

	if result.Retries != 0 {
		t.Errorf("Expected 0 retries, got %d", result.Retries)
	}

	// Verify file content
	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(downloaded) != string(content) {
		t.Errorf("Content mismatch: expected %q, got %q", content, downloaded)
	}
}

func TestDownloadWithRetry_WithChecksum(t *testing.T) {
	content := []byte("test file with checksum verification")
	hash := sha256.Sum256(content)
	expectedChecksum := hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "checksum-test.txt")

	cfg := DefaultDownloadConfig()
	cfg.URL = server.URL
	cfg.DestPath = destPath
	cfg.ExpectedSHA256 = expectedChecksum

	result, err := DownloadWithRetry(cfg)
	if err != nil {
		t.Fatalf("DownloadWithRetry failed: %v", err)
	}

	if result.SHA256 != expectedChecksum {
		t.Errorf("SHA256 mismatch: expected %s, got %s", expectedChecksum, result.SHA256)
	}
}

func TestDownloadWithRetry_ChecksumMismatch(t *testing.T) {
	content := []byte("test file content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "bad-checksum.txt")

	cfg := DefaultDownloadConfig()
	cfg.URL = server.URL
	cfg.DestPath = destPath
	cfg.ExpectedSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	cfg.MaxRetries = 1 // Reduce retries for faster test

	_, err := DownloadWithRetry(cfg)
	if err == nil {
		t.Fatal("Expected error for checksum mismatch, got nil")
	}
}

func TestDownloadWithRetry_Retry(t *testing.T) {
	attempts := 0
	content := []byte("success after retry")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			// Fail first attempt
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "retry-test.txt")

	cfg := DefaultDownloadConfig()
	cfg.URL = server.URL
	cfg.DestPath = destPath
	cfg.RetryDelay = 10 * time.Millisecond // Fast retry for test

	result, err := DownloadWithRetry(cfg)
	if err != nil {
		t.Fatalf("DownloadWithRetry failed: %v", err)
	}

	if result.Retries != 1 {
		t.Errorf("Expected 1 retry, got %d", result.Retries)
	}
}

func TestDownloadWithRetry_NonRetryable404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "404-test.txt")

	cfg := DefaultDownloadConfig()
	cfg.URL = server.URL
	cfg.DestPath = destPath
	cfg.MaxRetries = 3

	_, err := DownloadWithRetry(cfg)
	if err == nil {
		t.Fatal("Expected error for 404, got nil")
	}
}

func TestCalculateSHA256(t *testing.T) {
	content := []byte("hello world")
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "hash-test.txt")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	hash, err := calculateSHA256(filePath)
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, hash)
	}
}

func TestVerifyChecksum(t *testing.T) {
	content := []byte("test content")
	hash := sha256.Sum256(content)
	correctChecksum := hex.EncodeToString(hash[:])

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "verify-test.txt")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test correct checksum
	if err := verifyChecksum(filePath, correctChecksum); err != nil {
		t.Errorf("verifyChecksum failed for correct checksum: %v", err)
	}

	// Test uppercase checksum (should still work)
	if err := verifyChecksum(filePath, "  "+correctChecksum+"  "); err != nil {
		t.Errorf("verifyChecksum failed for checksum with whitespace: %v", err)
	}

	// Test incorrect checksum
	if err := verifyChecksum(filePath, "badchecksum"); err == nil {
		t.Error("verifyChecksum should fail for incorrect checksum")
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"ABC123", true},
		{"0123456789abcdef", true},
		{"ABCDEF", true},
		{"ghijkl", false},
		{"abc 123", false},
		{"abc-123", false},
		{"", true}, // empty string is technically valid hex
	}

	for _, tt := range tests {
		result := isHexString(tt.input)
		if result != tt.expected {
			t.Errorf("isHexString(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}
