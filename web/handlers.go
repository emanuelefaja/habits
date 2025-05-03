package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"mad/api"
	"mad/masterclass"
	"mad/middleware"
	"mad/models"
)

// HomeHandler handles the home page route
func HomeHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Wrap with timeout
		tw := newTimeoutResponseWriter(w, 10*time.Second)

		if r.URL.Path != "/" {
			http.NotFound(tw, r)
			return
		}

		if !middleware.IsAuthenticated(r) {
			// Guest handler
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

		// Authenticated user handling
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
	}
}

// LoginHandler handles the login page
func LoginHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			data := TemplateData{
				Flash: middleware.GetFlash(r),
			}

			// Add quote to the template data
			templateData := map[string]interface{}{
				"Flash": data.Flash,
				"Error": data.Error,
				"Quote": quote,
			}

			renderTemplate(w, templates, "login.html", templateData)
		case http.MethodPost:
			api.LoginHandler(db, templates)(w, r)
		default:
			HandleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}
}

// LogoutHandler handles the logout functionality
func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			HandleNotAllowed(w, http.MethodPost)
			return
		}
		if err := middleware.ClearSession(r); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		middleware.SetFlash(r, "You have been logged out successfully!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// SettingsHandler handles the settings page route
func SettingsHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Debug: Print user settings
		log.Printf("User settings: confetti=%v, weekdays=%v, notifications=%v", user.ShowConfetti, user.ShowWeekdays, user.NotificationEnabled)

		data := struct {
			User  *models.User
			Flash string
		}{
			User:  user,
			Flash: middleware.GetFlash(r),
		}
		renderTemplate(w, templates, "settings.html", data)
	}
}

// BrandHandler handles the brand guidelines page route
func BrandHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Page  string
			User  *models.User
			Flash string
		}{
			Page:  "brand",
			Flash: middleware.GetFlash(r),
		}

		// If user is authenticated, get user data
		if middleware.IsAuthenticated(r) {
			user, err := getAuthenticatedUser(r, db)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			data.User = user
		}

		renderTemplate(w, templates, "brand.html", data)
	}
}

// AboutHandler handles the about page route
func AboutHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "about",
		}
		renderTemplate(w, templates, "about.html", data)
	}
}

// PrivacyHandler handles the privacy policy page route
func PrivacyHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)

		data := map[string]interface{}{
			"User":        user,
			"LastUpdated": time.Now().Format("January 2, 2006"),
		}
		renderTemplate(w, templates, "privacy.html", data)
	}
}

// TermsHandler handles the terms of service page route
func TermsHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)

		data := map[string]interface{}{
			"User":        user,
			"LastUpdated": time.Now().Format("January 2, 2006"),
		}
		renderTemplate(w, templates, "terms.html", data)
	}
}

// PricingHandler handles the pricing page route
func PricingHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "pricing",
		}
		renderTemplate(w, templates, "pricing.html", data)
	}
}

// TrackerHandler handles the tracker page route
func TrackerHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "tracker",
		}
		renderTemplate(w, templates, "tracker.html", data)
	}
}

// MasterclassHandler handles the masterclass landing page route
func MasterclassHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Template data with defaults for non-authenticated users
		data := struct {
			User      *models.User
			Page      string
			HasAccess bool
			Flash     string
		}{
			Page:  "masterclass",
			Flash: middleware.GetFlash(r),
		}

		// Check if user is authenticated
		if middleware.IsAuthenticated(r) {
			// Get authenticated user
			userID := middleware.GetUserID(r)
			user, err := models.GetUserByID(db, int64(userID))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			data.User = user

			// Check if user has access to the masterclass
			hasAccess, err := masterclass.HasCourseAccess(db, userID, "masterclass")
			if err == nil && hasAccess {
				data.HasAccess = true

				// Find first incomplete lesson, or default to first lesson if all complete
				firstIncompleteLesson, firstIncompleteModule, err := masterclass.GetFirstIncompleteLesson(db, userID)
				if err != nil {
					log.Printf("Error finding first incomplete lesson: %v", err)
					// Fallback to first lesson of first module if there's an error
					modules := masterclass.GetCourseStructure()
					if len(modules) > 0 && len(modules[0].Lessons) > 0 {
						firstIncompleteModule = &modules[0]
						firstIncompleteLesson = &modules[0].Lessons[0]
					}
				}

				// Redirect to the identified lesson
				if firstIncompleteLesson != nil && firstIncompleteModule != nil {
					redirectURL := fmt.Sprintf("/masterclass/%s/%s",
						firstIncompleteModule.Slug, firstIncompleteLesson.Slug)
					http.Redirect(w, r, redirectURL, http.StatusSeeOther)
					return
				}
			}
		}

		// If not authenticated or no access, show landing page
		renderTemplate(w, templates, "masterclass-lp.html", data)
	}
}

// MasterclassModuleHandler handles module and lesson pages
func MasterclassModuleHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First check if user has access
		userID := middleware.GetUserID(r)
		hasAccess, err := masterclass.HasCourseAccess(db, userID, "masterclass")
		if err != nil || !hasAccess {
			middleware.SetFlash(r, "You don't have access to this course")
			http.Redirect(w, r, "/masterclass", http.StatusSeeOther)
			return
		}

		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Extract module and lesson slugs from URL path
		path := strings.TrimPrefix(r.URL.Path, "/masterclass/")
		parts := strings.Split(path, "/")

		// Get all course data for the template
		modules := masterclass.GetCourseStructure()

		// Handle module-only URL (redirect to first lesson)
		if len(parts) == 1 && parts[0] != "" {
			moduleSlug := parts[0]

			// Find the module
			var targetModule *masterclass.Module
			for i := range modules {
				if modules[i].Slug == moduleSlug {
					targetModule = &modules[i]
					break
				}
			}

			if targetModule != nil && len(targetModule.Lessons) > 0 {
				// Redirect to first lesson
				firstLesson := targetModule.Lessons[0]
				redirectURL := fmt.Sprintf("/masterclass/%s/%s", moduleSlug, firstLesson.Slug)
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				return
			}
		}

		// Handle module+lesson URL
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			moduleSlug := parts[0]
			lessonSlug := parts[1]

			// Get the lesson first
			lesson, err := masterclass.GetLessonBySlug(moduleSlug, lessonSlug)
			if err != nil {
				// Lesson not found
				http.Redirect(w, r, "/masterclass", http.StatusSeeOther)
				return
			}

			// Get the module separately
			module, err := masterclass.GetModuleBySlug(moduleSlug)
			if err != nil {
				http.Redirect(w, r, "/masterclass", http.StatusSeeOther)
				return
			}

			// Pre-fetch lesson completion status
			completed, err := masterclass.GetLessonCompletionStatus(db, userID, lesson.ID)
			if err != nil {
				// Log error but continue with default value
				log.Printf("Error getting lesson completion status: %v", err)
				completed = false
			}

			// Get lesson content
			lessonContent := ""
			lessonPath := filepath.Join("ui", "masterclass", "lessons", moduleSlug, lessonSlug+".html")

			// Process the lesson content using the masterclass template processing function
			processedContent, err := masterclass.ProcessTemplate(lessonPath, nil)
			if err != nil {
				log.Printf("Error processing lesson template: %v", err)
				// Fallback to loading the raw content
				rawContent, err := masterclass.LoadLessonContent(moduleSlug, lessonSlug)
				if err != nil {
					log.Printf("Error loading lesson content: %v", err)
					lessonContent = fmt.Sprintf("<div class='prose dark:prose-invert'><p>Lesson content could not be loaded.</p><p class='text-sm text-gray-500'>Error: %s</p></div>", err.Error())
				} else {
					lessonContent = rawContent
				}
			} else {
				lessonContent = processedContent
			}

			// Pre-fetch course structure with completion data
			courseStructure := masterclass.GetCourseStructure()
			completedCount := 0
			totalCount := 0
			moduleResponses := make([]map[string]interface{}, len(courseStructure))

			for i, module := range courseStructure {
				// Get module completion status
				moduleComplete, err := masterclass.IsModuleComplete(db, userID, module.Slug)
				if err != nil {
					log.Printf("Error checking module completion: %v", err)
					moduleComplete = false
				}

				// Get module progress
				completed, total, progress, err := masterclass.GetModuleProgress(db, userID, module.Slug)
				if err != nil {
					log.Printf("Error getting module progress: %v", err)
					completed, total, progress = 0, len(module.Lessons), 0
				}

				// Add to totals
				completedCount += completed
				totalCount += total

				// Create lesson responses with completion status
				lessonResponses := make([]map[string]interface{}, len(module.Lessons))
				for j, lesson := range module.Lessons {
					// Get lesson completion status
					lessonComplete, err := masterclass.GetLessonCompletionStatus(db, userID, lesson.ID)
					if err != nil {
						log.Printf("Error checking lesson completion: %v", err)
						lessonComplete = false
					}

					lessonResponses[j] = map[string]interface{}{
						"id":          lesson.ID,
						"slug":        lesson.Slug,
						"title":       lesson.Title,
						"emoji":       lesson.Emoji,
						"type":        lesson.Type,
						"moduleSlug":  lesson.ModuleSlug,
						"order":       lesson.Order,
						"description": lesson.Description,
						"completed":   lessonComplete,
					}
				}

				moduleResponses[i] = map[string]interface{}{
					"id":          module.ID,
					"slug":        module.Slug,
					"title":       module.Title,
					"description": module.Description,
					"emoji":       module.Emoji,
					"order":       module.Order,
					"category":    module.Category,
					"lessons":     lessonResponses,
					"completed":   moduleComplete,
					"progress":    progress,
				}
			}

			// Calculate overall progress
			var overallProgress float64 = 0
			if totalCount > 0 {
				overallProgress = float64(completedCount) / float64(totalCount) * 100
			}

			// Create course structure response
			courseData := map[string]interface{}{
				"modules":        moduleResponses,
				"completedCount": completedCount,
				"totalCount":     totalCount,
				"progress":       overallProgress,
			}

			// Convert to JSON for template
			courseDataJSON, err := json.Marshal(courseData)
			if err != nil {
				log.Printf("Error marshaling course data: %v", err)
				courseDataJSON = []byte("{}")
			}

			// Create lesson data response
			lessonData := map[string]interface{}{
				"id":          lesson.ID,
				"slug":        lesson.Slug,
				"title":       lesson.Title,
				"emoji":       lesson.Emoji,
				"type":        lesson.Type,
				"moduleSlug":  lesson.ModuleSlug,
				"order":       lesson.Order,
				"description": lesson.Description,
				"completed":   completed,
				"content":     lessonContent,
			}

			// Get lesson rating if completed
			if completed {
				log.Printf("⭐️ Initial page load - Lesson '%s' is completed, checking for rating for user %d", lesson.ID, userID)
				ratingValue, hasRating, err := masterclass.GetLessonRating(db, userID, lesson.ID)
				if err != nil {
					log.Printf("⭐️ Initial page load - Error getting lesson rating: %v", err)
				} else if hasRating {
					log.Printf("⭐️ Initial page load - Found rating %d for lesson '%s', user %d", ratingValue, lesson.ID, userID)
					lessonData["rating"] = ratingValue
				} else {
					log.Printf("⭐️ Initial page load - No rating found for lesson '%s', user %d", lesson.ID, userID)
				}
			}

			// Convert to JSON for template
			lessonDataJSON, err := json.Marshal(lessonData)
			if err != nil {
				log.Printf("Error marshaling lesson data: %v", err)
				lessonDataJSON = []byte("{}")
			}

			// Set default template data
			data := struct {
				User              *models.User
				Modules           []masterclass.Module
				ModuleSlug        string
				LessonSlug        string
				Lesson            *masterclass.Lesson
				Module            *masterclass.Module
				Page              string
				InitialLessonData template.JS
				InitialCourseData template.JS
			}{
				User:              user,
				Modules:           modules,
				ModuleSlug:        moduleSlug,
				LessonSlug:        lessonSlug,
				Lesson:            lesson,
				Module:            module,
				Page:              "masterclass",
				InitialLessonData: template.JS(lessonDataJSON),
				InitialCourseData: template.JS(courseDataJSON),
			}

			data.Lesson = lesson
			data.Module = module

			renderTemplate(w, templates, "lesson-base.html", data)
			return
		}

		// If we get here, redirect to masterclass landing
		http.Redirect(w, r, "/masterclass", http.StatusSeeOther)
	}
}

// MasterclassAPIHandler handles API requests for the masterclass
func MasterclassAPIHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the API endpoint from the URL
		path := strings.TrimPrefix(r.URL.Path, "/masterclass/api/")

		// Route to the appropriate handler
		switch path {
		case "course-structure":
			masterclass.CourseStructureHandler(db).ServeHTTP(w, r)
		case "lesson":
			masterclass.LessonHandler(db).ServeHTTP(w, r)
		case "mark-complete":
			masterclass.MarkLessonCompleteHandler(db).ServeHTTP(w, r)
		case "mark-incomplete":
			masterclass.MarkLessonIncompleteHandler(db).ServeHTTP(w, r)
		case "next-lesson":
			masterclass.NextLessonHandler(db).ServeHTTP(w, r)
		case "previous-lesson":
			masterclass.PreviousLessonHandler(db).ServeHTTP(w, r)
		case "progress":
			masterclass.GetUserProgressHandler(db).ServeHTTP(w, r)
		case "access":
			masterclass.GetUserCourseAccessHandler(db).ServeHTTP(w, r)
		case "rating":
			if r.Method == http.MethodPost {
				masterclass.SetLessonRatingHandler(db).ServeHTTP(w, r)
			} else {
				masterclass.GetLessonRatingHandler(db).ServeHTTP(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}
}

// BlogHandler handles the blog pages
func BlogHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/blog")
		blogService := models.GetBlogService()

		user, _ := getAuthenticatedUser(r, db)

		if path == "" || path == "/" {
			posts := blogService.GetAllPosts()
			data := struct {
				User  *models.User
				Posts []*models.BlogPost
				Page  string
			}{
				User:  user,
				Posts: posts,
				Page:  "blog",
			}
			renderTemplate(w, templates, "blog.html", data)
			return
		}

		slug := strings.TrimPrefix(path, "/")
		post, exists := blogService.GetPost(slug)
		if !exists {
			http.NotFound(w, r)
			return
		}

		data := struct {
			User *models.User
			Post *models.BlogPost
			Page string
		}{
			User: user,
			Post: post,
			Page: "blog",
		}
		renderTemplate(w, templates, "post.html", data)
	}
}

// ChangelogHandler handles the changelog page route
func ChangelogHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "changelog",
		}
		renderTemplate(w, templates, "changelog.html", data)
	}
}

// RoadmapHandler handles the roadmap page route
func RoadmapHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "roadmap",
		}
		renderTemplate(w, templates, "roadmap.html", data)
	}
}

// RegisterHandler handles the registration page and form submission
func RegisterHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Check if signups are allowed
			allowSignups, err := models.GetSignupStatus(db)
			if err != nil {
				log.Printf("Error checking signup status: %v", err)
				// Default to allowing signups if there's an error
			} else if !allowSignups {
				// Redirect to login page with a message
				middleware.SetFlash(r, "Registration is currently disabled ❌")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Generate math problem for human verification
			num1 := rand.Intn(20) + 1 // Random number between 1-20
			num2 := rand.Intn(20) + 1 // Random number between 1-20
			sum := num1 + num2

			// Store in session
			middleware.SetMathProblem(r, num1, num2, sum)

			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			// Pass to template
			data := map[string]interface{}{
				"MathNum1": num1,
				"MathNum2": num2,
				"Quote":    quote,
			}

			renderTemplate(w, templates, "register.html", data)
		case http.MethodPost:
			api.RegisterHandler(db, templates)(w, r)
		default:
			HandleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}
}

// Helper functions for handlers
func renderGuestHome(w http.ResponseWriter, templates *template.Template) {
	if err := templates.ExecuteTemplate(w, "guest-home.html", nil); err != nil {
		handleTemplateError(w, err, "guest-home.html")
	}
}

func renderUserHome(w http.ResponseWriter, r *http.Request, db *sql.DB, templates *template.Template) {
	user, err := getAuthenticatedUser(r, db)
	if err != nil {
		log.Printf("Error getting authenticated user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
	if err != nil {
		log.Printf("Error getting habits: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	habitsJSON, err := json.Marshal(habits)
	if err != nil {
		log.Printf("Error marshaling habits: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	renderTemplate(w, templates, "home.html", data)
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, templates *template.Template) {
	quote, err := models.GetRandomQuote()
	if err != nil {
		log.Printf("Error getting random quote: %v", err)
	}

	templateData := map[string]interface{}{
		"Flash": middleware.GetFlash(r),
		"Quote": quote,
	}

	renderTemplate(w, templates, "login.html", templateData)
}

func handleTemplateError(w http.ResponseWriter, err error, templateName string) {
	if strings.Contains(err.Error(), "write: broken pipe") ||
		strings.Contains(err.Error(), "client disconnected") ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "response timeout exceeded") {
		log.Printf("Client disconnected while rendering %s: %v", templateName, err)
		return
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
