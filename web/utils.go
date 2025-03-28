package web

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"mad/middleware"
	"mad/models"
)

// TemplateData holds common data for templates
type TemplateData struct {
	Flash      string
	Error      string
	Token      string
	IsLoggedIn bool
	Email      string
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

// HandleNotAllowed sets appropriate headers for disallowed methods
func HandleNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
