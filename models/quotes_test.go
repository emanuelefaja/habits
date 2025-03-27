package models

import (
	"encoding/json"
	"mad/models/email"
	"os"
	"path/filepath"
	"testing"
)

// TestGetRandomQuote tests the GetRandomQuote function
func TestGetRandomQuote(t *testing.T) {
	// Create a temporary quotes.json file for testing
	tempDir, err := os.MkdirTemp("", "quotes_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the static directory within the temp directory
	staticDir := filepath.Join(tempDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Create test quotes
	quotes := []Quote{
		{Text: "Test quote 1", Author: "Author 1"},
		{Text: "Test quote 2", Author: "Author 2"},
		{Text: "Test quote 3", Author: "Author 3"},
	}

	// Write test quotes to a temporary quotes.json file
	quotesJSON, err := json.Marshal(quotes)
	if err != nil {
		t.Fatalf("Failed to marshal quotes: %v", err)
	}

	quotesPath := filepath.Join(staticDir, "quotes.json")
	if err := os.WriteFile(quotesPath, quotesJSON, 0644); err != nil {
		t.Fatalf("Failed to write quotes file: %v", err)
	}

	// Save and restore the original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Change to the temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test GetRandomQuote
	quote, err := GetRandomQuote()
	if err != nil {
		t.Fatalf("GetRandomQuote failed: %v", err)
	}

	// Verify the quote is one of our test quotes
	found := false
	for _, q := range quotes {
		if quote.Text == q.Text && quote.Author == q.Author {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Got unexpected quote: %v", quote)
	}
}

// TestGetRandomQuoteWithNoFile tests the fallback behavior when quotes.json doesn't exist
func TestGetRandomQuoteWithNoFile(t *testing.T) {
	// Create a temporary directory without a quotes.json file
	tempDir, err := os.MkdirTemp("", "quotes_test_no_file")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the static directory (but no quotes.json file)
	staticDir := filepath.Join(tempDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Save and restore the original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Change to the temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test GetRandomQuote when file doesn't exist
	quote, err := GetRandomQuote()
	if err != nil {
		// Error is expected, but we should still get the default quote
		if quote.Text != "Success is the sum of small efforts, repeated day in and day out." ||
			quote.Author != "Robert Collier" {
			t.Errorf("Expected default quote, got: %v", quote)
		}
	} else {
		// We should have the default quote
		if quote.Text != "Success is the sum of small efforts, repeated day in and day out." ||
			quote.Author != "Robert Collier" {
			t.Errorf("Expected default quote, got: %v", quote)
		}
	}
}

// TestGetRandomQuoteForEmail tests the GetRandomQuoteForEmail function
func TestGetRandomQuoteForEmail(t *testing.T) {
	// Create a temporary quotes.json file
	tempDir, err := os.MkdirTemp("", "quotes_email_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the static directory
	staticDir := filepath.Join(tempDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Create test quotes
	quotes := []Quote{
		{Text: "Email quote 1", Author: "Email Author 1"},
		{Text: "Email quote 2", Author: "Email Author 2"},
	}

	// Write test quotes to a temporary quotes.json file
	quotesJSON, err := json.Marshal(quotes)
	if err != nil {
		t.Fatalf("Failed to marshal quotes: %v", err)
	}

	quotesPath := filepath.Join(staticDir, "quotes.json")
	if err := os.WriteFile(quotesPath, quotesJSON, 0644); err != nil {
		t.Fatalf("Failed to write quotes file: %v", err)
	}

	// Save and restore the original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Change to the temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test GetRandomQuoteForEmail
	emailQuote, err := GetRandomQuoteForEmail()
	if err != nil {
		t.Fatalf("GetRandomQuoteForEmail failed: %v", err)
	}

	// Verify the quote is one of our test quotes and has been converted to email.QuoteInfo
	found := false
	for _, q := range quotes {
		if emailQuote.Text == q.Text && emailQuote.Author == q.Author {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Got unexpected email quote: %v", emailQuote)
	}

	// Verify the type is email.QuoteInfo
	var _ email.QuoteInfo = emailQuote
}

// TestGetRandomQuoteForEmailWithNoFile tests the fallback behavior for email quotes
func TestGetRandomQuoteForEmailWithNoFile(t *testing.T) {
	// Create a temporary directory without a quotes.json file
	tempDir, err := os.MkdirTemp("", "quotes_email_test_no_file")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the static directory (but no quotes.json file)
	staticDir := filepath.Join(tempDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Save and restore the original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Change to the temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test GetRandomQuoteForEmail when file doesn't exist
	emailQuote, err := GetRandomQuoteForEmail()

	// We should get the default quote regardless of error
	if emailQuote.Text != "Success is the sum of small efforts, repeated day in and day out." ||
		emailQuote.Author != "Robert Collier" {
		t.Errorf("Expected default email quote, got: %v", emailQuote)
	}

	// Verify the type is email.QuoteInfo
	var _ email.QuoteInfo = emailQuote
}
