package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	server := NewServer("/tmp/downloads")

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleHealth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

// TestPlatformFromTitleID tests platform detection from title IDs
func TestPlatformFromTitleID(t *testing.T) {
	tests := []struct {
		titleID   uint64
		expected  string
		testName  string
	}{
		{0x00050000101C9500, "Wii U", "Wii U game"},
		{0x00040000000B8B00, "3DS", "3DS game"},
		{0x0100000000010000, "Switch", "Switch game"},
		{0x0000000700000000, "vWii", "vWii IOS"},
		{0x0007000800000000, "vWii", "vWii system"},
		{0x0001000000000000, "Wii", "Wii game"},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result := getPlatformFromTitleID(test.titleID)
			if result != test.expected {
				t.Errorf("getPlatformFromTitleID(%016X) = %v, want %v", test.titleID, result, test.expected)
			}
		})
	}
}

// TestFormatFromTitleID tests format detection from title IDs
func TestFormatFromTitleID(t *testing.T) {
	tests := []struct {
		titleID   uint64
		expected  string
		testName  string
	}{
		{0x00050000101C9500, "Content", "Wii U game - Content"},
		{0x00040000000B8B00, "CIA", "3DS game - CIA"},
		{0x0100000000010000, "NSP", "Switch game - NSP"},
		{0x0000000700000000, "Content", "vWii IOS - Content"},
		{0x0001000000000000, "ISO", "Wii game - ISO"},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result := getFormatFromTitleID(test.titleID)
			if result != test.expected {
				t.Errorf("getFormatFromTitleID(%016X) = %v, want %v", test.titleID, result, test.expected)
			}
		})
	}
}

// TestListTitlesEndpoint tests the list titles endpoint with various filters
func TestListTitlesEndpoint(t *testing.T) {
	server := NewServer("/tmp/downloads")

	tests := []struct {
		url         string
		expectedCode int
		testName    string
	}{
		{"/api/titles", http.StatusOK, "list all titles"},
		{"/api/titles?platform=wiiu", http.StatusOK, "filter by platform"},
		{"/api/titles?format=cia", http.StatusOK, "filter by format"},
		{"/api/titles?category=game", http.StatusOK, "filter by category"},
		{"/api/titles?region=usa", http.StatusOK, "filter by region"},
		{"/api/titles?search=mario", http.StatusOK, "search titles"},
		{"/api/titles?platform=invalid", http.StatusBadRequest, "invalid platform"},
		{"/api/titles?format=invalid", http.StatusBadRequest, "invalid format"},
		{"/api/titles?category=invalid", http.StatusBadRequest, "invalid category"},
		{"/api/titles?region=invalid", http.StatusBadRequest, "invalid region"},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			req, err := http.NewRequest("GET", test.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			server.router.ServeHTTP(rr, req)

			if status := rr.Code; status != test.expectedCode {
				t.Errorf("handler returned wrong status code for %s: got %v want %v", test.url, status, test.expectedCode)
			}

			// If successful, verify JSON response
			if test.expectedCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to parse JSON response for %s: %v", test.url, err)
				}

				if _, exists := response["count"]; !exists {
					t.Errorf("Response missing 'count' field for %s", test.url)
				}

				if _, exists := response["titles"]; !exists {
					t.Errorf("Response missing 'titles' field for %s", test.url)
				}
			}
		})
	}
}

// TestStartDownloadEndpoint tests the download endpoint
func TestStartDownloadEndpoint(t *testing.T) {
	server := NewServer("/tmp/downloads")

	// Test valid download request
	downloadReq := map[string]interface{}{
		"title_id":         "00050000101C9500",
		"decrypt":          true,
		"delete_encrypted": false,
	}

	reqBody, _ := json.Marshal(downloadReq)
	req, err := http.NewRequest("POST", "/api/download", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// Should return 202 Accepted for valid request
	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}

	// Test invalid requests
	invalidTests := []struct {
		body     map[string]interface{}
		testName string
	}{
		{map[string]interface{}{}, "missing title_id"},
		{map[string]interface{}{"title_id": "invalid"}, "invalid title_id format"},
		{map[string]interface{}{"title_id": "1234567890123456"}, "non-existent title"},
	}

	for _, test := range invalidTests {
		t.Run(test.testName, func(t *testing.T) {
			reqBody, _ := json.Marshal(test.body)
			req, _ := http.NewRequest("POST", "/api/download", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %v", test.testName, status)
			}
		})
	}
}

// TestGetTitleEndpoint tests the get single title endpoint
func TestGetTitleEndpoint(t *testing.T) {
	server := NewServer("/tmp/downloads")

	// Test valid title ID
	req, err := http.NewRequest("GET", "/api/titles/00050000101C9500", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// Should return 404 for non-existent title (since we don't have real data)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test invalid title ID format
	req, err = http.NewRequest("GET", "/api/titles/invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid ID: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestOpenAPIEndpoint tests the OpenAPI spec endpoint
func TestOpenAPIEndpoint(t *testing.T) {
	server := NewServer("/tmp/downloads")

	req, err := http.NewRequest("GET", "/api/openapi.json", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify it's valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse OpenAPI JSON response: %v", err)
	}

	// Check for required OpenAPI fields
	if response["openapi"] != "3.0.3" {
		t.Errorf("Expected OpenAPI version 3.0.3, got %v", response["openapi"])
	}
}

// TestCORSHeaders tests CORS headers are present
func TestCORSHeaders(t *testing.T) {
	server := NewServer("/tmp/downloads")

	req, err := http.NewRequest("OPTIONS", "/api/titles", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expectedValue := range expectedHeaders {
		if actualValue := rr.Header().Get(header); actualValue != expectedValue {
			t.Errorf("Expected %s: %s, got %s", header, expectedValue, actualValue)
		}
	}
}
