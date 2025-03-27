package models

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"mad/models/email"
)

// Quote represents a motivational quote with text and author
type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

// GetRandomQuote returns a random quote from the quotes.json file
func GetRandomQuote() (Quote, error) {
	// Initialize with a default quote in case of error
	defaultQuote := Quote{
		Text:   "Success is the sum of small efforts, repeated day in and day out.",
		Author: "Robert Collier",
	}

	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return defaultQuote, err
	}

	// Construct the path to the quotes.json file
	quotesPath := filepath.Join(dir, "static", "quotes.json")

	// Read the file
	data, err := os.ReadFile(quotesPath)
	if err != nil {
		return defaultQuote, err
	}

	// Parse the JSON
	var quotes []Quote
	if err := json.Unmarshal(data, &quotes); err != nil {
		return defaultQuote, err
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Return a random quote
	if len(quotes) > 0 {
		return quotes[rand.Intn(len(quotes))], nil
	}

	return defaultQuote, nil
}

// GetRandomQuoteForEmail converts a Quote to the email.QuoteInfo format
func GetRandomQuoteForEmail() (email.QuoteInfo, error) {
	quote, err := GetRandomQuote()
	if err != nil {
		// Return default quote if there's an error
		return email.QuoteInfo{
			Text:   "Success is the sum of small efforts, repeated day in and day out.",
			Author: "Robert Collier",
		}, nil
	}

	return email.QuoteInfo{
		Text:   quote.Text,
		Author: quote.Author,
	}, nil
}
