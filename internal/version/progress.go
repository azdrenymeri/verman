package version

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressWriter wraps an io.Writer and displays download progress
type ProgressWriter struct {
	writer      io.Writer
	total       int64
	current     int64
	startTime   time.Time
	lastUpdate  time.Time
	mu          sync.Mutex
	description string
}

// NewProgressWriter creates a new progress writer
func NewProgressWriter(w io.Writer, total int64, description string) *ProgressWriter {
	return &ProgressWriter{
		writer:      w,
		total:       total,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		description: description,
	}
}

// Write implements io.Writer and updates progress
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.mu.Lock()
	pw.current += int64(n)

	// Update display at most every 100ms to avoid flickering
	if time.Since(pw.lastUpdate) >= 100*time.Millisecond {
		pw.display()
		pw.lastUpdate = time.Now()
	}
	pw.mu.Unlock()

	return n, nil
}

// display shows the current progress
func (pw *ProgressWriter) display() {
	elapsed := time.Since(pw.startTime).Seconds()
	if elapsed == 0 {
		elapsed = 0.001
	}

	speed := float64(pw.current) / elapsed

	if pw.total > 0 {
		// Known total size - show percentage and ETA
		percent := float64(pw.current) / float64(pw.total) * 100

		// Calculate ETA
		remaining := pw.total - pw.current
		eta := time.Duration(float64(remaining)/speed) * time.Second

		// Build progress bar
		barWidth := 30
		filled := int(percent / 100 * float64(barWidth))
		bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)

		fmt.Printf("\r[%s] %5.1f%% %s/%s %s/s ETA %s   ",
			bar,
			percent,
			formatBytes(pw.current),
			formatBytes(pw.total),
			formatBytes(int64(speed)),
			formatDuration(eta))
	} else {
		// Unknown total size - show downloaded amount and speed
		fmt.Printf("\r%s downloaded %s/s   ",
			formatBytes(pw.current),
			formatBytes(int64(speed)))
	}
}

// Finish completes the progress display
func (pw *ProgressWriter) Finish() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	elapsed := time.Since(pw.startTime)
	speed := float64(pw.current) / elapsed.Seconds()

	// Clear the progress line and print completion message
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	fmt.Printf("Downloaded %s in %s (%s/s)\n",
		formatBytes(pw.current),
		formatDuration(elapsed),
		formatBytes(int64(speed)))
}

// formatBytes formats bytes into human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human-readable format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// DownloadWithProgress downloads from resp.Body to writer with progress display
func DownloadWithProgress(dst io.Writer, src io.Reader, total int64, description string) error {
	pw := NewProgressWriter(dst, total, description)
	_, err := io.Copy(pw, src)
	pw.Finish()
	return err
}
