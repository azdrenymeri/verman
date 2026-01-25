package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadConfig holds configuration for downloads
type DownloadConfig struct {
	URL            string
	DestPath       string
	ExpectedSHA256 string // Optional: if set, verify after download
	MaxRetries     int
	RetryDelay     time.Duration
	Description    string
}

// DefaultDownloadConfig returns sensible defaults
func DefaultDownloadConfig() DownloadConfig {
	return DownloadConfig{
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}
}

// DownloadResult contains information about the completed download
type DownloadResult struct {
	Size       int64
	SHA256     string
	Duration   time.Duration
	Retries    int
	FromResume bool
}

// DownloadWithRetry downloads a file with retry logic and optional checksum verification
func DownloadWithRetry(cfg DownloadConfig) (*DownloadResult, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s...
			delay := cfg.RetryDelay * time.Duration(1<<(attempt-1))
			fmt.Printf("Retry %d/%d in %v...\n", attempt, cfg.MaxRetries, delay)
			time.Sleep(delay)
		}

		result, err := downloadOnce(cfg)
		if err == nil {
			result.Retries = attempt
			result.Duration = time.Since(startTime)

			// Verify checksum if provided
			if cfg.ExpectedSHA256 != "" {
				if err := verifyChecksum(cfg.DestPath, cfg.ExpectedSHA256); err != nil {
					// Checksum mismatch - delete the file and retry
					_ = os.Remove(cfg.DestPath)
					lastErr = err
					continue
				}
			}

			return result, nil
		}

		lastErr = err

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("download failed after %d attempts: %w", cfg.MaxRetries+1, lastErr)
}

// downloadOnce performs a single download attempt
func downloadOnce(cfg DownloadConfig) (*DownloadResult, error) {
	result := &DownloadResult{}

	// Check for partial download (resume support)
	var existingSize int64
	if info, err := os.Stat(cfg.DestPath); err == nil {
		existingSize = info.Size()
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add range header for resume
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	// Set a reasonable timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Large files can take time
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	switch resp.StatusCode {
	case http.StatusOK:
		// Full download (no resume or server doesn't support it)
		existingSize = 0
		result.FromResume = false
	case http.StatusPartialContent:
		// Resume supported
		result.FromResume = true
	case http.StatusRequestedRangeNotSatisfiable:
		// File already complete or server confused - start fresh
		existingSize = 0
		result.FromResume = false
		// Need to make a new request without Range header
		resp.Body.Close()
		req.Header.Del("Range")
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
	default:
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Determine total size
	totalSize := resp.ContentLength
	if totalSize > 0 && existingSize > 0 {
		totalSize += existingSize
	}

	// Open file for writing (append if resuming)
	flags := os.O_WRONLY | os.O_CREATE
	if existingSize > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(cfg.DestPath, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Download with progress
	description := cfg.Description
	if description == "" {
		description = "Downloading"
	}
	if result.FromResume {
		description += " (resuming)"
	}

	if err := DownloadWithProgress(file, resp.Body, totalSize, description); err != nil {
		return nil, fmt.Errorf("download interrupted: %w", err)
	}

	// Get final file size
	if info, err := file.Stat(); err == nil {
		result.Size = info.Size()
	}

	// Calculate SHA256
	result.SHA256, _ = calculateSHA256(cfg.DestPath)

	return result, nil
}

// isNonRetryableError checks if an error should not trigger a retry
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// HTTP 4xx errors (except 408 Request Timeout, 429 Too Many Requests)
	if strings.Contains(errStr, "HTTP 4") {
		if strings.Contains(errStr, "HTTP 408") || strings.Contains(errStr, "HTTP 429") {
			return false // These are retryable
		}
		return true // Other 4xx are not retryable
	}

	return false
}

// calculateSHA256 calculates the SHA256 hash of a file
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyChecksum verifies the SHA256 checksum of a file
func verifyChecksum(filePath, expected string) error {
	actual, err := calculateSHA256(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Normalize both to lowercase for comparison
	expected = strings.ToLower(strings.TrimSpace(expected))
	actual = strings.ToLower(actual)

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// FetchChecksum fetches a SHA256 checksum from a URL
// Many download sites provide checksums at URL.sha256 or similar
func FetchChecksum(checksumURL string) (string, error) {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read checksum (usually just the hex string, sometimes with filename)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}

	// Parse checksum - handle formats like:
	// "abc123..." (just hash)
	// "abc123  filename" (GNU coreutils format)
	// "SHA256 (filename) = abc123" (BSD format)
	line := strings.TrimSpace(string(body))

	// Try to extract just the hex hash
	parts := strings.Fields(line)
	for _, part := range parts {
		// Check if it looks like a SHA256 hash (64 hex chars)
		clean := strings.TrimPrefix(part, "=")
		clean = strings.TrimSpace(clean)
		if len(clean) == 64 && isHexString(clean) {
			return strings.ToLower(clean), nil
		}
	}

	// If no 64-char hex found, return the whole thing and let verification fail
	return line, nil
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
