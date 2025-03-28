package web

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"mad/middleware"
	"mad/models"
)

// SetupRoutes configures all application routes
func SetupRoutes(db *sql.DB, templates *template.Template) {
	// Only setting up the home route for now
	setupHomeRoute(db, templates)
}

// Set up just the home route for now
func setupHomeRoute(db *sql.DB, templates *template.Template) {
	http.Handle("/", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer with a timeout
		tw := newTimeoutResponseWriter(w, 10*time.Second)

		if r.URL.Path != "/" {
			http.NotFound(tw, r)
			return
		}

		if !middleware.IsAuthenticated(r) {
			// Guest
			if err := templates.ExecuteTemplate(tw, "guest-home.html", nil); err != nil {
				// Check if the error is due to a client disconnection
				if strings.Contains(err.Error(), "write: broken pipe") ||
					strings.Contains(err.Error(), "client disconnected") ||
					strings.Contains(err.Error(), "connection reset by peer") ||
					strings.Contains(err.Error(), "response timeout exceeded") {
					log.Printf("Client disconnected while rendering guest-home.html: %v", err)
					return
				}
				http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			log.Printf("Error getting authenticated user: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only get necessary habit data for the current view
		habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
		if err != nil {
			log.Printf("Error getting habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Limit the amount of data sent to the template
		habitsJSON, err := json.Marshal(habits)
		if err != nil {
			log.Printf("Error marshaling habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			User       *models.User
			HabitsJSON template.JS
			Flash      string
		}{
			User:       user,
			HabitsJSON: template.JS(habitsJSON),
			Flash:      middleware.GetFlash(r),
		}
		renderTemplate(tw, templates, "home.html", data)
	})))
}

// timeoutResponseWriter is a custom ResponseWriter that adds timeout functionality
type timeoutResponseWriter struct {
	http.ResponseWriter
	timeout time.Duration
	start   time.Time
}

func newTimeoutResponseWriter(w http.ResponseWriter, timeout time.Duration) *timeoutResponseWriter {
	return &timeoutResponseWriter{
		ResponseWriter: w,
		timeout:        timeout,
		start:          time.Now(),
	}
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	// Check if we've exceeded the timeout
	if time.Since(w.start) > w.timeout {
		return 0, fmt.Errorf("response timeout exceeded")
	}
	return w.ResponseWriter.Write(b)
}

func (w *timeoutResponseWriter) WriteHeader(statusCode int) {
	// Check if we've exceeded the timeout
	if time.Since(w.start) > w.timeout {
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Helper function to get authenticated user
func getAuthenticatedUser(r *http.Request, db *sql.DB) (*models.User, error) {
	if !middleware.IsAuthenticated(r) {
		return nil, nil
	}
	userID := middleware.GetUserID(r)
	return models.GetUserByID(db, int64(userID))
}

// Helper function to render templates
func renderTemplate(w http.ResponseWriter, templates *template.Template, name string, data interface{}) {
	// Use a buffer to render the template first
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log the size of the response
	responseSize := buf.Len()
	log.Printf("Template %s rendered with size: %d bytes", name, responseSize)

	// Then write the buffered content to the response writer
	_, err := buf.WriteTo(w)
	if err != nil {
		// Check if the error is due to a client disconnection
		if strings.Contains(err.Error(), "write: broken pipe") ||
			strings.Contains(err.Error(), "client disconnected") ||
			strings.Contains(err.Error(), "connection reset by peer") {
			log.Printf("Client disconnected while sending template %s: %v", name, err)
			return // Don't try to write an error response to a disconnected client
		}

		log.Printf("Error writing template %s to response: %v", name, err)
		// At this point, we may not be able to write an error response
		// since we've already started writing the response
	}
}
