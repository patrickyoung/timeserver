package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	// Get health check URL from environment or use default
	healthURL := os.Getenv("HEALTH_URL")
	if healthURL == "" {
		healthURL = "http://localhost:8080/health"
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Make health check request
	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check returned status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Success
	fmt.Println("Health check passed")
	os.Exit(0)
}
