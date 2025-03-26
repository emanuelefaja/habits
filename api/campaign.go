package api

import (
	"encoding/json"
	"fmt"
	"log"
	"mad/middleware"
	"mad/models/email"
	"net/http"
	"strconv"
	"time"
)

// CampaignSubscriptionRequest represents the data needed to subscribe to a campaign
type CampaignSubscriptionRequest struct {
	FirstName  string `json:"first_name"`
	Email      string `json:"email"`
	CampaignID string `json:"campaign_id"`
	MathAnswer string `json:"math_answer"`
	MathNum1   int    `json:"math_num1"`
	MathNum2   int    `json:"math_num2"`
	CSRFToken  string `json:"_csrf"`
}

// CampaignUnsubscribeRequest represents the data needed to unsubscribe from a campaign
type CampaignUnsubscribeRequest struct {
	Email      string `json:"email"`
	CampaignID string `json:"campaign_id"`
	Token      string `json:"token"`
}

// SubscriptionResponse represents a subscription returned in the API
type SubscriptionResponse struct {
	ID            int       `json:"id"`
	CampaignID    string    `json:"campaign_id"`
	CampaignName  string    `json:"campaign_name"`
	CampaignEmoji string    `json:"campaign_emoji"`
	Status        string    `json:"status"`
	SubscribedAt  time.Time `json:"subscribed_at"`
}

// CampaignPreferenceRequest represents a request to update subscription preferences
type CampaignPreferenceRequest struct {
	CampaignID string `json:"campaign_id"`
	Status     string `json:"status"` // "active" or "paused"
}

// rateLimit tracks IP address request counts for rate limiting
var rateLimit = make(map[string]map[string]int)

// cleanupRateLimit periodically cleans up the rate limit map
func cleanupRateLimit() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			// Reset the rate limit map each hour
			rateLimit = make(map[string]map[string]int)
		}
	}()
}

func init() {
	// Start the rate limit cleanup routine
	cleanupRateLimit()
}

// checkRateLimit checks if the IP has exceeded the rate limit for the given action
func checkRateLimit(ip, action string, limit int) bool {
	if _, exists := rateLimit[ip]; !exists {
		rateLimit[ip] = make(map[string]int)
	}

	rateLimit[ip][action]++
	return rateLimit[ip][action] <= limit
}

// SubscribeToCampaign handles POST /api/campaigns/subscribe
func SubscribeToCampaign(w http.ResponseWriter, r *http.Request) {
	// Rate limiting - 10 attempts per hour per IP
	ip := middleware.GetIPAddress(r)
	if !checkRateLimit(ip, "subscribe", 10) {
		http.Error(w, `{"error":"Too many subscription attempts. Please try again later."}`, http.StatusTooManyRequests)
		return
	}

	// Only accept POST method
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var req CampaignSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request format"}`, http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Email == "" || req.CampaignID == "" || req.FirstName == "" {
		http.Error(w, `{"error":"Missing required fields"}`, http.StatusBadRequest)
		return
	}

	// Verify math challenge
	mathSum := req.MathNum1 + req.MathNum2
	mathAnswer, err := strconv.Atoi(req.MathAnswer)
	if err != nil || mathAnswer != mathSum {
		http.Error(w, `{"error":"Incorrect answer to the math challenge"}`, http.StatusBadRequest)
		return
	}

	// Check if the campaign exists
	_, err = email.GetCampaign(req.CampaignID)
	if err != nil {
		http.Error(w, `{"error":"Campaign not found"}`, http.StatusNotFound)
		return
	}

	// Get the email service and campaign manager
	svc, ok := emailService.(*email.SMTPEmailService)
	if !ok || svc == nil {
		http.Error(w, `{"error":"Email service not available"}`, http.StatusInternalServerError)
		return
	}

	campaignManager := svc.GetCampaignManager()
	if campaignManager == nil {
		http.Error(w, `{"error":"Campaign manager not available"}`, http.StatusInternalServerError)
		return
	}

	// Get user ID if authenticated
	userID := 0
	if user := middleware.GetUser(r); user != nil {
		userID = int(user.ID)
	}

	// Subscribe the user to the campaign
	err = campaignManager.SubscribeUser(req.Email, req.CampaignID, userID)
	if err != nil {
		log.Printf("Error subscribing user: %v", err)
		http.Error(w, `{"error":"Failed to subscribe to campaign"}`, http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"success":true,"message":"Successfully subscribed to the campaign"}`)
}

// UnsubscribeFromCampaign handles POST /api/campaigns/unsubscribe
func UnsubscribeFromCampaign(w http.ResponseWriter, r *http.Request) {
	// Rate limiting - 10 attempts per hour per IP
	ip := middleware.GetIPAddress(r)
	if !checkRateLimit(ip, "unsubscribe", 10) {
		http.Error(w, `{"error":"Too many unsubscribe attempts. Please try again later."}`, http.StatusTooManyRequests)
		return
	}

	// Only accept POST method
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var req CampaignUnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request format"}`, http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Email == "" || req.CampaignID == "" || req.Token == "" {
		http.Error(w, `{"error":"Missing required fields"}`, http.StatusBadRequest)
		return
	}

	// Token validation would go here in a real implementation
	// For now, we'll accept any token as valid

	// Get the email service and campaign manager
	svc, ok := emailService.(*email.SMTPEmailService)
	if !ok || svc == nil {
		http.Error(w, `{"error":"Email service not available"}`, http.StatusInternalServerError)
		return
	}

	campaignManager := svc.GetCampaignManager()
	if campaignManager == nil {
		http.Error(w, `{"error":"Campaign manager not available"}`, http.StatusInternalServerError)
		return
	}

	// Unsubscribe the user from the campaign
	err := campaignManager.UnsubscribeUser(req.Email, req.CampaignID)
	if err != nil {
		// Return success even if not found to prevent email enumeration
		log.Printf("Error unsubscribing user: %v", err)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"success":true,"message":"Successfully unsubscribed from the campaign"}`)
}

// GetSubscriptions handles GET /api/campaigns/subscriptions
func GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Only accept GET method
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// This endpoint requires authentication
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"Authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Get the email service and campaign manager
	svc, ok := emailService.(*email.SMTPEmailService)
	if !ok || svc == nil {
		http.Error(w, `{"error":"Email service not available"}`, http.StatusInternalServerError)
		return
	}

	campaignManager := svc.GetCampaignManager()
	if campaignManager == nil {
		http.Error(w, `{"error":"Campaign manager not available"}`, http.StatusInternalServerError)
		return
	}

	// Get the user's subscriptions
	subscriptions, err := campaignManager.GetUserSubscriptions(int(user.ID))
	if err != nil {
		log.Printf("Error getting user subscriptions: %v", err)
		http.Error(w, `{"error":"Failed to retrieve subscriptions"}`, http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []SubscriptionResponse
	for _, sub := range subscriptions {
		campaign, err := email.GetCampaign(sub.CampaignID)
		if err != nil {
			// Skip campaigns that don't exist anymore
			continue
		}

		response = append(response, SubscriptionResponse{
			ID:            sub.ID,
			CampaignID:    sub.CampaignID,
			CampaignName:  campaign.Name,
			CampaignEmoji: campaign.Emoji,
			Status:        sub.Status,
			SubscribedAt:  sub.SubscribedAt,
		})
	}

	// Return the subscriptions as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, `{"error":"Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// UpdateSubscriptionPreferences handles PUT /api/campaigns/preferences
func UpdateSubscriptionPreferences(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT method
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// This endpoint requires authentication
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"Authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Parse the JSON request body
	var req CampaignPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request format"}`, http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.CampaignID == "" || (req.Status != "active" && req.Status != "paused") {
		http.Error(w, `{"error":"Invalid request parameters"}`, http.StatusBadRequest)
		return
	}

	// Get the email service and campaign manager
	svc, ok := emailService.(*email.SMTPEmailService)
	if !ok || svc == nil {
		http.Error(w, `{"error":"Email service not available"}`, http.StatusInternalServerError)
		return
	}

	campaignManager := svc.GetCampaignManager()
	if campaignManager == nil {
		http.Error(w, `{"error":"Campaign manager not available"}`, http.StatusInternalServerError)
		return
	}

	// Update the subscription preference
	err := campaignManager.UpdateSubscriptionStatus(int(user.ID), req.CampaignID, req.Status)
	if err != nil {
		log.Printf("Error updating subscription status: %v", err)
		http.Error(w, `{"error":"Failed to update subscription preferences"}`, http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"success":true,"message":"Successfully updated subscription preferences"}`)
}
