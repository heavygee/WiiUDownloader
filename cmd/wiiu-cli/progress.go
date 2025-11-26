package main

import (
	"fmt"
	"sync"
	"time"
)

type CLIProgressReporter struct {
	gameTitle             string
	totalSize             int64
	downloadedSize        int64
	startTime             time.Time
	cancelled             bool
	mu                    sync.RWMutex
	fileProgress          map[string]int64
	totalFiles            int
	completedFiles        int
}

func NewCLIProgressReporter() *CLIProgressReporter {
	return &CLIProgressReporter{
		fileProgress: make(map[string]int64),
	}
}

func (c *CLIProgressReporter) SetGameTitle(title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gameTitle = title
	fmt.Printf("Downloading: %s\n", title)
}

func (c *CLIProgressReporter) UpdateDownloadProgress(downloaded int64, filename string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fileProgress[filename] = downloaded
	c.downloadedSize = 0
	for _, size := range c.fileProgress {
		c.downloadedSize += size
	}
	c.printProgress()
}

func (c *CLIProgressReporter) UpdateDecryptionProgress(progress float64) {
	fmt.Printf("Decryption progress: %.1f%%\n", progress*100)
}

func (c *CLIProgressReporter) Cancelled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cancelled
}

func (c *CLIProgressReporter) SetCancelled() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelled = true
}

func (c *CLIProgressReporter) SetDownloadSize(size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalSize = size
}

func (c *CLIProgressReporter) ResetTotals() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloadedSize = 0
	c.totalSize = 0
	c.completedFiles = 0
	c.totalFiles = 0
	c.fileProgress = make(map[string]int64)
}

func (c *CLIProgressReporter) MarkFileAsDone(filename string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completedFiles++
	fmt.Printf("Completed: %s (%d/%d files)\n", filename, c.completedFiles, c.totalFiles)
}

func (c *CLIProgressReporter) SetTotalDownloadedForFile(filename string, downloaded int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fileProgress[filename] = downloaded
}

func (c *CLIProgressReporter) SetStartTime(startTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startTime = startTime
	c.totalFiles = len(c.fileProgress)
}

func (c *CLIProgressReporter) printProgress() {
	if c.totalSize == 0 {
		return
	}

	percentage := float64(c.downloadedSize) / float64(c.totalSize) * 100
	elapsed := time.Since(c.startTime)

	// Calculate ETA
	var eta time.Duration
	if c.downloadedSize > 0 && elapsed.Seconds() > 0 {
		totalSeconds := elapsed.Seconds() / (float64(c.downloadedSize) / float64(c.totalSize))
		eta = time.Duration(totalSeconds-elapsed.Seconds()) * time.Second
	}

	fmt.Printf("\rProgress: %.1f%% (%s/%s) ETA: %s",
		percentage,
		formatBytes(c.downloadedSize),
		formatBytes(c.totalSize),
		formatDuration(eta))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "unknown"
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
