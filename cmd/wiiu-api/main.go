package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	wiiudownloader "github.com/Xpl0itU/WiiUDownloader"
	"github.com/gorilla/mux"
)

// DownloadJob represents a download task
type DownloadJob struct {
	ID            string                 `json:"id"`
	TitleID       string                 `json:"title_id"`
	TitleName     string                 `json:"title_name"`
	Status        string                 `json:"status"` // pending, downloading, completed, failed, cancelled
	Progress      float64                `json:"progress"`
	DownloadSize  int64                  `json:"download_size"`
	Downloaded    int64                  `json:"downloaded"`
	Speed         string                 `json:"speed"`
	ETA           string                 `json:"eta"`
	Error         string                 `json:"error,omitempty"`
	OutputDir     string                 `json:"output_dir"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Decrypt       bool                   `json:"decrypt"`
	DeleteEncrypted bool                 `json:"delete_encrypted"`
	ctx           context.Context        `json:"-"`
	cancel        context.CancelFunc     `json:"-"`
	progress      *APIProgressReporter   `json:"-"`
}

type APIProgressReporter struct {
	job       *DownloadJob
	startTime time.Time
	mu        sync.RWMutex
}

func NewAPIProgressReporter(job *DownloadJob) *APIProgressReporter {
	return &APIProgressReporter{
		job:       job,
		startTime: time.Now(),
	}
}

func (a *APIProgressReporter) SetGameTitle(title string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.TitleName = title
}

func (a *APIProgressReporter) UpdateDownloadProgress(downloaded int64, filename string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.Downloaded = downloaded

	if a.job.DownloadSize > 0 {
		a.job.Progress = float64(downloaded) / float64(a.job.DownloadSize) * 100

		// Calculate speed and ETA
		elapsed := time.Since(a.startTime).Seconds()
		if elapsed > 0 {
			speed := float64(downloaded) / elapsed
			a.job.Speed = formatBytes(int64(speed)) + "/s"

			if speed > 0 {
				remaining := float64(a.job.DownloadSize-downloaded) / speed
				a.job.ETA = formatDuration(time.Duration(remaining) * time.Second)
			}
		}
	}
}

func (a *APIProgressReporter) UpdateDecryptionProgress(progress float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.Progress = progress
}

func (a *APIProgressReporter) Cancelled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.job.Status == "cancelled"
}

func (a *APIProgressReporter) SetCancelled() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.Status = "cancelled"
}

func (a *APIProgressReporter) SetDownloadSize(size int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.DownloadSize = size
}

func (a *APIProgressReporter) ResetTotals() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.job.Downloaded = 0
	a.job.Progress = 0
}

func (a *APIProgressReporter) MarkFileAsDone(filename string) {}
func (a *APIProgressReporter) SetTotalDownloadedForFile(filename string, downloaded int64) {}
func (a *APIProgressReporter) SetStartTime(startTime time.Time) {}

type Server struct {
	router       *mux.Router
	jobs         map[string]*DownloadJob
	jobsMutex    sync.RWMutex
	downloadsDir string
	client       *http.Client
}

func NewServer(downloadsDir string) *Server {
	// Create HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
		},
	}

	server := &Server{
		router:       mux.NewRouter(),
		jobs:         make(map[string]*DownloadJob),
		downloadsDir: downloadsDir,
		client:       client,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Titles
	api.HandleFunc("/titles", s.handleListTitles).Methods("GET")
	api.HandleFunc("/titles/{id}", s.handleGetTitle).Methods("GET")

	// Downloads
	api.HandleFunc("/download", s.handleStartDownload).Methods("POST")
	api.HandleFunc("/download/{id}", s.handleGetDownloadStatus).Methods("GET")
	api.HandleFunc("/download/{id}", s.handleCancelDownload).Methods("DELETE")

	// CORS middleware
	s.router.Use(s.corsMiddleware)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListTitles(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "game"
	}

	search := r.URL.Query().Get("search")
	region := r.URL.Query().Get("region")

	var categoryFlag uint8
	switch category {
	case "game":
		categoryFlag = wiiudownloader.TITLE_CATEGORY_GAME
	case "update":
		categoryFlag = wiiudownloader.TITLE_CATEGORY_UPDATE
	case "dlc":
		categoryFlag = wiiudownloader.TITLE_CATEGORY_DLC
	case "demo":
		categoryFlag = wiiudownloader.TITLE_CATEGORY_DEMO
	case "all":
		categoryFlag = wiiudownloader.TITLE_CATEGORY_ALL
	default:
		http.Error(w, "Invalid category", http.StatusBadRequest)
		return
	}

	entries := wiiudownloader.GetTitleEntries(categoryFlag)

	// Filter by region if specified
	if region != "" && region != "all" {
		var regionMask uint8
		switch region {
		case "japan":
			regionMask = wiiudownloader.MCP_REGION_JAPAN
		case "usa":
			regionMask = wiiudownloader.MCP_REGION_USA
		case "europe":
			regionMask = wiiudownloader.MCP_REGION_EUROPE
		default:
			http.Error(w, "Invalid region", http.StatusBadRequest)
			return
		}

		filtered := make([]wiiudownloader.TitleEntry, 0)
		for _, entry := range entries {
			if entry.Region&regionMask != 0 {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	// Filter by search term if specified
	if search != "" {
		filtered := make([]wiiudownloader.TitleEntry, 0)
		for _, entry := range entries {
			if containsIgnoreCase(entry.Name, search) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	// Convert to JSON response
	titles := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		titles[i] = map[string]interface{}{
			"id":     fmt.Sprintf("%016X", entry.TitleID),
			"name":   entry.Name,
			"region": wiiudownloader.GetFormattedRegion(entry.Region),
			"type":   wiiudownloader.GetFormattedKind(entry.TitleID),
		}
	}

	response := map[string]interface{}{
		"count":  len(titles),
		"titles": titles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Parse title ID
	tid, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		http.Error(w, "Invalid title ID format", http.StatusBadRequest)
		return
	}

	entry := wiiudownloader.GetTitleEntryFromTid(tid)
	if entry.TitleID == 0 {
		http.Error(w, "Title not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":     fmt.Sprintf("%016X", entry.TitleID),
		"name":   entry.Name,
		"region": wiiudownloader.GetFormattedRegion(entry.Region),
		"type":   wiiudownloader.GetFormattedKind(entry.TitleID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStartDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TitleID         string `json:"title_id"`
		Decrypt         bool   `json:"decrypt,omitempty"`
		DeleteEncrypted bool   `json:"delete_encrypted,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TitleID == "" {
		http.Error(w, "title_id is required", http.StatusBadRequest)
		return
	}

	// Validate title ID exists
	tid, err := strconv.ParseUint(req.TitleID, 16, 64)
	if err != nil {
		http.Error(w, "Invalid title ID format", http.StatusBadRequest)
		return
	}

	entry := wiiudownloader.GetTitleEntryFromTid(tid)
	if entry.TitleID == 0 {
		http.Error(w, "Title not found", http.StatusNotFound)
		return
	}

	// Create job ID
	jobID := fmt.Sprintf("%s_%d", req.TitleID, time.Now().Unix())

	// Create output directory
	outputDir := filepath.Join(s.downloadsDir, jobID)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create output directory", http.StatusInternalServerError)
		return
	}

	// Create download job
	ctx, cancel := context.WithCancel(context.Background())
	job := &DownloadJob{
		ID:              jobID,
		TitleID:         req.TitleID,
		TitleName:       entry.Name,
		Status:          "pending",
		OutputDir:       outputDir,
		StartTime:       time.Now(),
		Decrypt:         req.Decrypt,
		DeleteEncrypted: req.DeleteEncrypted,
		ctx:             ctx,
		cancel:          cancel,
		progress:        NewAPIProgressReporter(nil), // Will be set after job creation
	}

	job.progress = NewAPIProgressReporter(job)

	// Store job
	s.jobsMutex.Lock()
	s.jobs[jobID] = job
	s.jobsMutex.Unlock()

	// Start download in background
	go s.processDownload(job)

	response := map[string]interface{}{
		"job_id": jobID,
		"status": "started",
		"title":  entry.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetDownloadStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	s.jobsMutex.RLock()
	job, exists := s.jobs[jobID]
	s.jobsMutex.RUnlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":               job.ID,
		"title_id":         job.TitleID,
		"title_name":       job.TitleName,
		"status":           job.Status,
		"progress":         job.Progress,
		"download_size":    job.DownloadSize,
		"downloaded":       job.Downloaded,
		"speed":            job.Speed,
		"eta":              job.ETA,
		"output_dir":       job.OutputDir,
		"start_time":       job.StartTime.Format(time.RFC3339),
		"decrypt":          job.Decrypt,
		"delete_encrypted": job.DeleteEncrypted,
	}

	if job.Error != "" {
		response["error"] = job.Error
	}

	if job.EndTime != nil {
		response["end_time"] = job.EndTime.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleCancelDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	s.jobsMutex.RLock()
	job, exists := s.jobs[jobID]
	s.jobsMutex.RUnlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.Status == "completed" || job.Status == "failed" {
		http.Error(w, "Cannot cancel completed or failed job", http.StatusBadRequest)
		return
	}

	job.cancel()
	job.Status = "cancelled"
	now := time.Now()
	job.EndTime = &now

	response := map[string]interface{}{
		"status": "cancelled",
		"job_id": jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) processDownload(job *DownloadJob) {
	job.Status = "downloading"

	err := wiiudownloader.DownloadTitle(
		job.TitleID,
		job.OutputDir,
		job.Decrypt,
		job.progress,
		job.DeleteEncrypted,
		s.client,
	)

	now := time.Now()
	job.EndTime = &now

	if err != nil {
		if job.Status == "cancelled" {
			job.Status = "cancelled"
		} else {
			job.Status = "failed"
			job.Error = err.Error()
		}
	} else {
		job.Status = "completed"
		job.Progress = 100.0
	}
}

func main() {
	port := flag.String("port", "8080", "Port to run the server on")
	downloadsDir := flag.String("downloads", "./downloads", "Directory to store downloads")
	flag.Parse()

	// Create downloads directory if it doesn't exist
	if err := os.MkdirAll(*downloadsDir, os.ModePerm); err != nil {
		log.Fatal("Failed to create downloads directory:", err)
	}

	// Create server
	server := NewServer(*downloadsDir)

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	log.Printf("Starting WiiU API server on port %s", *port)
	log.Printf("Downloads directory: %s", *downloadsDir)
	log.Fatal(http.ListenAndServe(":"+*port, server.router))
}

// Helper functions
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

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsIgnoreCaseHelper(s, substr))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] && a[i] != b[i]+32 && a[i] != b[i]-32 {
			return false
		}
	}
	return true
}
