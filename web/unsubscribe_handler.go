package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"mad/models/email"
)

// UnsubscribeHandler handles the email campaign unsubscribe page and actions
func UnsubscribeHandler(db *sql.DB, emailService email.EmailService, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the campaign manager from the email service
		campaignManager := emailService.GetCampaignManager()
		if campaignManager == nil {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
			return
		}

		// For GET requests, just show the unsubscribe page
		if r.Method == http.MethodGet {
			userEmail := r.URL.Query().Get("email")
			campaignID := r.URL.Query().Get("campaign")
			token := r.URL.Query().Get("token")

			log.Printf("Unsubscribe GET request: email=%s, campaign=%s, token=%s", userEmail, campaignID, token)

			if userEmail == "" || campaignID == "" {
				log.Printf("Missing query parameters: email=%s, campaign=%s", userEmail, campaignID)
				http.Error(w, "Missing required parameters", http.StatusBadRequest)
				return
			}

			// Get campaign details for display
			campaign, err := email.GetCampaign(campaignID)
			if err != nil {
				log.Printf("Error getting campaign: %v", err)
				http.Error(w, "Invalid campaign", http.StatusNotFound)
				return
			}

			data := struct {
				Email         string
				CampaignID    string
				CampaignName  string
				CampaignEmoji string
				Token         string
				Quote         struct {
					Text   string
					Author string
				}
				Unsubscribed bool
			}{
				Email:         userEmail,
				CampaignID:    campaignID,
				CampaignName:  campaign.Name,
				CampaignEmoji: campaign.Emoji,
				Token:         token,
				Quote: struct {
					Text   string
					Author string
				}{
					Text:   "Small habits make big changes.",
					Author: "The Habits Company",
				},
				Unsubscribed: false,
			}
			renderTemplate(w, templates, "unsubscribe.html", data)
			return
		}

		// For POST requests, handle the unsubscribe action
		if r.Method == http.MethodPost {
			// Parse form data
			if err := r.ParseForm(); err != nil {
				log.Printf("Error parsing form data: %v", err)
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}

			// Get values from form data instead of URL query parameters
			formEmail := r.PostFormValue("email")
			formCampaignID := r.PostFormValue("campaign_id")
			formToken := r.PostFormValue("token")

			log.Printf("Unsubscribe POST request: email=%s, campaign=%s, token=%s", formEmail, formCampaignID, formToken)

			if formEmail == "" || formCampaignID == "" || formToken == "" {
				log.Printf("Missing form parameters: email=%s, campaign=%s, token=%s", formEmail, formCampaignID, formToken)
				http.Error(w, "Missing required parameters", http.StatusBadRequest)
				return
			}

			// Apply rate limiting - 10 attempts per hour per IP
			remaining, resetTime, err := UnsubscribeLimiter.CheckLimit(r)
			if err != nil {
				log.Printf("Rate limit check error: %v", err)
				http.Error(w, "Error processing request", http.StatusInternalServerError)
				return
			}
			if remaining == 0 {
				waitDuration := time.Until(resetTime)
				http.Error(w, fmt.Sprintf("Too many unsubscribe attempts. Please try again in %d minutes.", int(waitDuration.Minutes())+1), http.StatusTooManyRequests)
				return
			}

			// With token validation:
			valid, err := campaignManager.ValidateUnsubscribeToken(formEmail, formCampaignID, formToken)
			if err != nil {
				log.Printf("Error validating token: %v", err)
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}

			if !valid {
				log.Printf("Invalid token for email=%s, campaign=%s", formEmail, formCampaignID)
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}

			// Get campaign details for the response
			campaign, err := email.GetCampaign(formCampaignID)
			if err != nil {
				log.Printf("Error getting campaign: %v", err)
				http.Error(w, "Invalid campaign", http.StatusNotFound)
				return
			}

			err = campaignManager.UnsubscribeUser(formEmail, formCampaignID)
			if err != nil {
				log.Printf("Error unsubscribing user: %v", err)
				http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
				return
			}

			log.Printf("Successfully unsubscribed %s from campaign %s", formEmail, formCampaignID)

			data := struct {
				Success       bool
				Email         string
				CampaignID    string
				CampaignName  string
				CampaignEmoji string
				Quote         struct {
					Text   string
					Author string
				}
				Unsubscribed bool
				Token        string
			}{
				Success:       true,
				Email:         formEmail,
				CampaignID:    formCampaignID,
				CampaignName:  campaign.Name,
				CampaignEmoji: campaign.Emoji,
				Quote: struct {
					Text   string
					Author string
				}{
					Text:   "Small habits make big changes.",
					Author: "The Habits Company",
				},
				Unsubscribed: true,
				Token:        formToken,
			}
			renderTemplate(w, templates, "unsubscribe.html", data)
			return
		}

		HandleNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}
