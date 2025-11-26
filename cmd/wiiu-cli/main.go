package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	wiiudownloader "github.com/Xpl0itU/WiiUDownloader"
)

func main() {
	// Parse command line arguments
	titleID := flag.String("title", "", "Title ID to download (hexadecimal, e.g., 00050000101C9500)")
	outputDir := flag.String("output", "", "Output directory for downloaded files")
	decrypt := flag.Bool("decrypt", false, "Decrypt downloaded contents")
	deleteEncrypted := flag.Bool("delete-encrypted", false, "Delete encrypted contents after decryption")
	list := flag.Bool("list", false, "List available titles")
	search := flag.String("search", "", "Search for titles by name")
	category := flag.String("category", "game", "Category to list/search: game, update, dlc, demo, all")
	region := flag.String("region", "all", "Region filter: japan, usa, europe, all")
	flag.Parse()

	// Initialize HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Initialize threading
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Handle list/search commands
	if *list || *search != "" {
		handleListSearch(*list, *search, *category, *region)
		return
	}

	// Validate required arguments for download
	if *titleID == "" {
		fmt.Println("Error: title ID is required for download")
		flag.Usage()
		os.Exit(1)
	}

	if *outputDir == "" {
		fmt.Println("Error: output directory is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory
	absOutputDir, err := filepath.Abs(*outputDir)
	if err != nil {
		log.Fatal("Failed to get absolute path:", err)
	}

	if err := os.MkdirAll(absOutputDir, os.ModePerm); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Create progress reporter
	progress := NewCLIProgressReporter()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nDownload cancelled by user")
		progress.SetCancelled()
	}()

	// Start download
	fmt.Printf("Starting download of title %s to %s\n", *titleID, absOutputDir)
	if *decrypt {
		fmt.Println("Decryption enabled")
		if *deleteEncrypted {
			fmt.Println("Will delete encrypted contents after decryption")
		}
	}

	err = wiiudownloader.DownloadTitle(*titleID, absOutputDir, *decrypt, progress, *deleteEncrypted, client)
	if err != nil {
		if progress.Cancelled() {
			fmt.Println("Download cancelled")
			os.Exit(130)
		}
		log.Fatal("Download failed:", err)
	}

	if progress.Cancelled() {
		fmt.Println("Download cancelled")
		os.Exit(130)
	}

	fmt.Println("\nDownload completed successfully!")
}

func handleListSearch(list bool, search, categoryStr, regionStr string) {
	var category uint8
	switch categoryStr {
	case "game":
		category = wiiudownloader.TITLE_CATEGORY_GAME
	case "update":
		category = wiiudownloader.TITLE_CATEGORY_UPDATE
	case "dlc":
		category = wiiudownloader.TITLE_CATEGORY_DLC
	case "demo":
		category = wiiudownloader.TITLE_CATEGORY_DEMO
	case "all":
		category = wiiudownloader.TITLE_CATEGORY_ALL
	default:
		fmt.Printf("Unknown category: %s\n", categoryStr)
		os.Exit(1)
	}

	entries := wiiudownloader.GetTitleEntries(category)

	// Filter by region if specified
	if regionStr != "all" {
		var regionMask uint8
		switch regionStr {
		case "japan":
			regionMask = wiiudownloader.MCP_REGION_JAPAN
		case "usa":
			regionMask = wiiudownloader.MCP_REGION_USA
		case "europe":
			regionMask = wiiudownloader.MCP_REGION_EUROPE
		default:
			fmt.Printf("Unknown region: %s\n", regionStr)
			os.Exit(1)
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

	// Display results
	if len(entries) == 0 {
		fmt.Println("No titles found matching the criteria")
		return
	}

	fmt.Printf("Found %d titles:\n\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("Title ID: %016X\n", entry.TitleID)
		fmt.Printf("Name: %s\n", entry.Name)
		fmt.Printf("Region: %s\n", wiiudownloader.GetFormattedRegion(entry.Region))
		fmt.Printf("Type: %s\n", wiiudownloader.GetFormattedKind(entry.TitleID))
		fmt.Println("---")
	}
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
