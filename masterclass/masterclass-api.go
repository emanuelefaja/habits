package masterclass

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mad/middleware"
)

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type LessonResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Emoji       string `json:"emoji"`
	Type        string `json:"type"`
	ModuleSlug  string `json:"moduleSlug"`
	Order       int    `json:"order"`
	Description string `json:"description"`
	Content     string `json:"content,omitempty"`
	Completed   bool   `json:"completed"`
}

type ModuleResponse struct {
	ID          string           `json:"id"`
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Emoji       string           `json:"emoji"`
	Order       int              `json:"order"`
	Lessons     []LessonResponse `json:"lessons"`
	Completed   bool             `json:"completed"`
	Progress    float64          `json:"progress"`
}

type CourseResponse struct {
	Modules        []ModuleResponse `json:"modules"`
	CompletedCount int              `json:"completedCount"`
	TotalCount     int              `json:"totalCount"`
	Progress       float64          `json:"progress"`
}

type ProgressResponse struct {
	CompletedLessons int     `json:"completedLessons"`
	TotalLessons     int     `json:"totalLessons"`
	Percentage       float64 `json:"percentage"`
}

type LessonCompletionResponse struct {
	LessonID  string    `json:"lessonId"`
	ModuleID  string    `json:"moduleId"`
	Completed bool      `json:"completed"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type CourseAccessResponse struct {
	HasAccess    bool   `json:"hasAccess"`
	Status       string `json:"status,omitempty"`
	PurchasedAt  string `json:"purchasedAt,omitempty"`
	CourseID     string `json:"courseId"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// Helper functions

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response in JSON format
func writeErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	writeJSON(w, status, ErrorResponse{Error: errMsg})
}

// getUserID extracts the user ID from the request using the middleware
func getUserID(r *http.Request) (int, error) {
	userID := middleware.GetUserID(r)
	if userID == 0 {
		return 0, ErrNoAccess
	}
	return userID, nil
}

// API Handlers

// CourseStructureHandler returns the entire course structure with progress info
func CourseStructureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get course structure
		modules := GetCourseStructure()

		// Create response with completion status
		completedCount := 0
		totalCount := 0

		moduleResponses := make([]ModuleResponse, len(modules))
		for i, module := range modules {
			// Get module completion status
			moduleComplete, err := IsModuleComplete(db, userID, module.Slug)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Error checking module completion")
				return
			}

			// Get module progress
			completed, total, progress, err := GetModuleProgress(db, userID, module.Slug)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Error getting module progress")
				return
			}

			// Add to totals
			completedCount += completed
			totalCount += total

			// Create lesson responses with completion status
			lessonResponses := make([]LessonResponse, len(module.Lessons))
			for j, lesson := range module.Lessons {
				// Get lesson completion status
				lessonComplete, err := GetLessonCompletionStatus(db, userID, lesson.ID)
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, "Error checking lesson completion")
					return
				}

				lessonResponses[j] = LessonResponse{
					ID:          lesson.ID,
					Slug:        lesson.Slug,
					Title:       lesson.Title,
					Emoji:       lesson.Emoji,
					Type:        lesson.Type,
					ModuleSlug:  lesson.ModuleSlug,
					Order:       lesson.Order,
					Description: lesson.Description,
					Completed:   lessonComplete,
				}
			}

			moduleResponses[i] = ModuleResponse{
				ID:          module.ID,
				Slug:        module.Slug,
				Title:       module.Title,
				Description: module.Description,
				Emoji:       module.Emoji,
				Order:       module.Order,
				Lessons:     lessonResponses,
				Completed:   moduleComplete,
				Progress:    progress,
			}
		}

		// Calculate overall progress
		var overallProgress float64 = 0
		if totalCount > 0 {
			overallProgress = float64(completedCount) / float64(totalCount) * 100
		}

		response := CourseResponse{
			Modules:        moduleResponses,
			CompletedCount: completedCount,
			TotalCount:     totalCount,
			Progress:       overallProgress,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// LessonHandler returns details for a specific lesson
func LessonHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get module and lesson slugs from query parameters
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Get lesson
		lesson, err := GetLessonBySlug(moduleSlug, lessonSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "Lesson not found")
			return
		}

		// Get lesson completion status
		completed, err := GetLessonCompletionStatus(db, userID, lesson.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error checking lesson completion")
			return
		}

		// Load content HTML based on lesson type
		content := ""

		// This is a simplified way to get content - in a real app, you'd have a better system
		// for retrieving and rendering lesson content from templates or a CMS
		if lesson.Type == "markdown" {
			// For markdown lessons, convert the stored markdown to HTML
			content = RenderLessonContent(lesson.ID)
		} else if lesson.Type == "video" {
			// For video lessons, embed the video player
			content = fmt.Sprintf(`
				<div class="aspect-w-16 aspect-h-9 mb-6">
					<iframe src="https://www.youtube.com/embed/%s" 
						frameborder="0" 
						allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" 
						allowfullscreen>
					</iframe>
				</div>
				<div>%s</div>
			`, lesson.VideoID, lesson.Description)
		} else {
			// Default content
			content = fmt.Sprintf("<p>%s</p>", lesson.Description)
		}

		// Create response
		response := LessonResponse{
			ID:          lesson.ID,
			Slug:        lesson.Slug,
			Title:       lesson.Title,
			Emoji:       lesson.Emoji,
			Type:        lesson.Type,
			ModuleSlug:  lesson.ModuleSlug,
			Order:       lesson.Order,
			Description: lesson.Description,
			Content:     content,
			Completed:   completed,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// NextLessonHandler returns the next lesson
func NextLessonHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get current module and lesson slugs
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Get next lesson
		nextLesson, nextModule, err := GetNextLesson(moduleSlug, lessonSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "No next lesson available")
			return
		}

		// Get lesson completion status
		completed, err := GetLessonCompletionStatus(db, userID, nextLesson.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error checking lesson completion")
			return
		}

		// Create response
		response := LessonResponse{
			ID:          nextLesson.ID,
			Slug:        nextLesson.Slug,
			Title:       nextLesson.Title,
			Emoji:       nextLesson.Emoji,
			Type:        nextLesson.Type,
			ModuleSlug:  nextModule.Slug,
			Order:       nextLesson.Order,
			Description: nextLesson.Description,
			Completed:   completed,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// PreviousLessonHandler returns the previous lesson
func PreviousLessonHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get current module and lesson slugs
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Get previous lesson
		prevLesson, prevModule, err := GetPreviousLesson(moduleSlug, lessonSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "No previous lesson available")
			return
		}

		// Get lesson completion status
		completed, err := GetLessonCompletionStatus(db, userID, prevLesson.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error checking lesson completion")
			return
		}

		// Create response
		response := LessonResponse{
			ID:          prevLesson.ID,
			Slug:        prevLesson.Slug,
			Title:       prevLesson.Title,
			Emoji:       prevLesson.Emoji,
			Type:        prevLesson.Type,
			ModuleSlug:  prevModule.Slug,
			Order:       prevLesson.Order,
			Description: prevLesson.Description,
			Completed:   completed,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// LessonExistsHandler checks if a lesson exists
func LessonExistsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get module and lesson slugs
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Check if lesson exists
		exists := LessonExists(moduleSlug, lessonSlug)

		// Create response
		response := map[string]bool{
			"exists": exists,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// MarkLessonCompleteHandler marks a lesson as complete
func MarkLessonCompleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get module and lesson slugs
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Get lesson to verify it exists and to get its ID
		lesson, err := GetLessonBySlug(moduleSlug, lessonSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "Lesson not found")
			return
		}

		// Mark lesson as complete
		err = MarkLessonComplete(db, userID, moduleSlug, lesson.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error marking lesson as complete")
			return
		}

		// Create response
		response := LessonCompletionResponse{
			LessonID:  lesson.ID,
			ModuleID:  moduleSlug,
			Completed: true,
			Timestamp: time.Now(),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// MarkLessonIncompleteHandler marks a lesson as incomplete
func MarkLessonIncompleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get module and lesson slugs
		moduleSlug := r.URL.Query().Get("moduleSlug")
		lessonSlug := r.URL.Query().Get("lessonSlug")

		if moduleSlug == "" || lessonSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug and lesson slug are required")
			return
		}

		// Get lesson to verify it exists and to get its ID
		lesson, err := GetLessonBySlug(moduleSlug, lessonSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "Lesson not found")
			return
		}

		// Mark lesson as incomplete
		err = MarkLessonIncomplete(db, userID, lesson.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error marking lesson as incomplete")
			return
		}

		// Create response
		response := LessonCompletionResponse{
			LessonID:  lesson.ID,
			ModuleID:  moduleSlug,
			Completed: false,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GetUserProgressHandler returns the user's progress through the course
func GetUserProgressHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get user progress
		completedLessons, totalLessons, percentage, err := GetUserProgress(db, userID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error getting user progress")
			return
		}

		// Create response
		response := ProgressResponse{
			CompletedLessons: completedLessons,
			TotalLessons:     totalLessons,
			Percentage:       percentage,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GetModuleProgressHandler returns progress through a specific module
func GetModuleProgressHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get module slug
		moduleSlug := r.URL.Query().Get("moduleSlug")
		if moduleSlug == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Module slug is required")
			return
		}

		// Get module progress
		completedLessons, totalLessons, percentage, err := GetModuleProgress(db, userID, moduleSlug)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error getting module progress")
			return
		}

		// Create response
		response := ProgressResponse{
			CompletedLessons: completedLessons,
			TotalLessons:     totalLessons,
			Percentage:       percentage,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GetUserCourseAccessHandler checks if a user has access to the course
func GetUserCourseAccessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Get course ID (using a default value for now)
		courseID := r.URL.Query().Get("courseId")
		if courseID == "" {
			courseID = "masterclass" // Default course ID
		}

		// Check if user has access
		hasAccess, err := HasCourseAccess(db, userID, courseID)

		// Create response
		response := CourseAccessResponse{
			CourseID: courseID,
		}

		if err != nil {
			if err == ErrNoAccess {
				response.HasAccess = false
				response.ErrorMessage = "User does not have access to this course"
				writeJSON(w, http.StatusOK, response)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Error checking course access")
			}
			return
		}

		response.HasAccess = hasAccess

		// Get more details if access exists
		if hasAccess {
			var status string
			var purchasedAt time.Time

			err := db.QueryRow(`
				SELECT status, purchased_at FROM user_course_access
				WHERE user_id = ? AND course_id = ?
			`, userID, courseID).Scan(&status, &purchasedAt)

			if err == nil {
				response.Status = status
				response.PurchasedAt = purchasedAt.Format(time.RFC3339)
			}
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GrantCourseAccessHandler grants a user access to the course
func GrantCourseAccessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// This endpoint may require admin privileges
		// For now, just check if the user is authenticated
		userID, err := getUserID(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		// Parse request
		var request struct {
			UserID        int     `json:"userId"`
			CourseID      string  `json:"courseId"`
			PurchasePrice float64 `json:"purchasePrice"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request format")
			return
		}

		// For security, validate that the requester has permission to grant access
		// This is a simplified check and should be replaced with proper authorization
		if userID != request.UserID {
			writeErrorResponse(w, http.StatusForbidden, "You don't have permission to grant access")
			return
		}

		// Grant access
		err = GrantCourseAccess(db, request.UserID, request.CourseID, request.PurchasePrice)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Error granting course access")
			return
		}

		// Create response
		response := map[string]interface{}{
			"success":  true,
			"userId":   request.UserID,
			"courseId": request.CourseID,
			"message":  "Course access granted successfully",
		}

		writeJSON(w, http.StatusOK, response)
	}
}
