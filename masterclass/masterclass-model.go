package masterclass

import (
	"database/sql"
	"fmt"
	"time"
)

// Custom error types for expected conditions
var (
	ErrModuleNotFound = fmt.Errorf("module not found")
	ErrLessonNotFound = fmt.Errorf("lesson not found")
	ErrNoAccess       = fmt.Errorf("user does not have access to this course")
	ErrNoNextLesson   = fmt.Errorf("no next lesson available")
	ErrNoPrevLesson   = fmt.Errorf("no previous lesson available")
)

// Lesson types
const (
	LessonTypeVideo = "video"
	LessonTypeText  = "text"
	// Future types: quiz, youtube, checklist, etc.
)

// Module represents a section of the masterclass containing multiple lessons
type Module struct {
	ID          string   // Unique identifier
	Slug        string   // URL-friendly identifier (e.g., "introduction")
	Title       string   // Display name
	Description string   // Brief module description
	Emoji       string   // Visual indicator
	Order       int      // Display order in the sidebar
	Lessons     []Lesson // Lessons in this module
}

// Lesson represents an individual learning unit
type Lesson struct {
	ID          string // Unique identifier
	Slug        string // URL-friendly identifier (e.g., "getting-started")
	Title       string // Display name
	Emoji       string // Visual indicator
	Type        string // Lesson type (video, text, etc.)
	ModuleSlug  string // Parent module slug for lookups
	Order       int    // Display order within module
	Description string // Brief description
}

// LessonCompletion represents a user's completion status for a lesson
type LessonCompletion struct {
	ID          int
	UserID      int
	LessonID    string
	ModuleID    string
	Completed   bool
	CompletedAt time.Time
	CreatedAt   time.Time
}

// CourseAccess represents a user's access to a course
type CourseAccess struct {
	ID            int
	UserID        int
	CourseID      string
	PurchasedAt   time.Time
	PurchasePrice float64
	Status        string
}

// GetModuleBySlug returns a module by its slug
func GetModuleBySlug(slug string) (*Module, error) {
	for i := range MasterclassDefinition {
		if MasterclassDefinition[i].Slug == slug {
			return &MasterclassDefinition[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrModuleNotFound, slug)
}

// GetLessonBySlug finds a lesson by its slug within a specified module
func GetLessonBySlug(moduleSlug, lessonSlug string) (*Lesson, error) {
	module, err := GetModuleBySlug(moduleSlug)
	if err != nil {
		return nil, err
	}

	for i := range module.Lessons {
		if module.Lessons[i].Slug == lessonSlug {
			return &module.Lessons[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s in module %s", ErrLessonNotFound, lessonSlug, moduleSlug)
}

// GetNextLesson returns the next lesson after the current one
func GetNextLesson(currentModuleSlug, currentLessonSlug string) (*Lesson, *Module, error) {
	currentModule, err := GetModuleBySlug(currentModuleSlug)
	if err != nil {
		return nil, nil, err
	}

	// Find current lesson index
	currentLessonIndex := -1
	for i, lesson := range currentModule.Lessons {
		if lesson.Slug == currentLessonSlug {
			currentLessonIndex = i
			break
		}
	}

	if currentLessonIndex == -1 {
		return nil, nil, fmt.Errorf("%w: %s in module %s", ErrLessonNotFound, currentLessonSlug, currentModuleSlug)
	}

	// If not the last lesson in the module
	if currentLessonIndex < len(currentModule.Lessons)-1 {
		return &currentModule.Lessons[currentLessonIndex+1], currentModule, nil
	}

	// Find the next module
	nextModuleIndex := -1
	for i, module := range MasterclassDefinition {
		if module.Slug == currentModuleSlug {
			nextModuleIndex = i
			break
		}
	}

	if nextModuleIndex == -1 || nextModuleIndex >= len(MasterclassDefinition)-1 {
		return nil, nil, ErrNoNextLesson
	}

	nextModule := &MasterclassDefinition[nextModuleIndex+1]
	if len(nextModule.Lessons) == 0 {
		return nil, nil, ErrNoNextLesson
	}

	return &nextModule.Lessons[0], nextModule, nil
}

// GetPreviousLesson returns the previous lesson before the current one
func GetPreviousLesson(currentModuleSlug, currentLessonSlug string) (*Lesson, *Module, error) {
	currentModule, err := GetModuleBySlug(currentModuleSlug)
	if err != nil {
		return nil, nil, err
	}

	// Find current lesson index
	currentLessonIndex := -1
	for i, lesson := range currentModule.Lessons {
		if lesson.Slug == currentLessonSlug {
			currentLessonIndex = i
			break
		}
	}

	if currentLessonIndex == -1 {
		return nil, nil, fmt.Errorf("%w: %s in module %s", ErrLessonNotFound, currentLessonSlug, currentModuleSlug)
	}

	// If not the first lesson in the module
	if currentLessonIndex > 0 {
		return &currentModule.Lessons[currentLessonIndex-1], currentModule, nil
	}

	// Find the previous module
	prevModuleIndex := -1
	for i, module := range MasterclassDefinition {
		if module.Slug == currentModuleSlug {
			prevModuleIndex = i
			break
		}
	}

	if prevModuleIndex <= 0 {
		return nil, nil, ErrNoPrevLesson
	}

	prevModule := &MasterclassDefinition[prevModuleIndex-1]
	if len(prevModule.Lessons) == 0 {
		return nil, nil, ErrNoPrevLesson
	}

	// Return last lesson of previous module
	lastLessonIndex := len(prevModule.Lessons) - 1
	return &prevModule.Lessons[lastLessonIndex], prevModule, nil
}

// LessonExists checks if a lesson exists in the masterclass
func LessonExists(moduleSlug, lessonSlug string) bool {
	_, err := GetLessonBySlug(moduleSlug, lessonSlug)
	return err == nil
}

// GetCourseStructure returns the entire course structure
func GetCourseStructure() []Module {
	return MasterclassDefinition
}

// MarkLessonComplete marks a lesson as complete for a user
func MarkLessonComplete(db *sql.DB, userID int, moduleID, lessonID string) error {
	_, err := db.Exec(`
		INSERT INTO user_lesson_completion (user_id, lesson_id, module_id, completed, completed_at)
		VALUES (?, ?, ?, true, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, lesson_id) 
		DO UPDATE SET completed = true, completed_at = CURRENT_TIMESTAMP
	`, userID, lessonID, moduleID)

	return err
}

// MarkLessonIncomplete marks a lesson as incomplete for a user
func MarkLessonIncomplete(db *sql.DB, userID int, lessonID string) error {
	_, err := db.Exec(`
		UPDATE user_lesson_completion 
		SET completed = false, completed_at = NULL
		WHERE user_id = ? AND lesson_id = ?
	`, userID, lessonID)

	return err
}

// GetLessonCompletionStatus gets the completion status of a lesson for a user
func GetLessonCompletionStatus(db *sql.DB, userID int, lessonID string) (bool, error) {
	var completed bool
	err := db.QueryRow(`
		SELECT completed FROM user_lesson_completion
		WHERE user_id = ? AND lesson_id = ?
	`, userID, lessonID).Scan(&completed)

	if err == sql.ErrNoRows {
		return false, nil
	}

	return completed, err
}

// IsModuleComplete checks if all lessons in a module are completed by a user
func IsModuleComplete(db *sql.DB, userID int, moduleSlug string) (bool, error) {
	module, err := GetModuleBySlug(moduleSlug)
	if err != nil {
		return false, err
	}

	if len(module.Lessons) == 0 {
		return true, nil // Empty module is considered complete
	}

	// Get all lesson IDs for this module
	var lessonIDs []interface{}
	for _, lesson := range module.Lessons {
		lessonIDs = append(lessonIDs, lesson.ID)
	}

	// Build the placeholders for the SQL query
	placeholders := "?"
	for i := 1; i < len(lessonIDs); i++ {
		placeholders += ",?"
	}

	// Count completed lessons in this module
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM user_lesson_completion
		WHERE user_id = ? AND lesson_id IN (%s) AND completed = true
	`, placeholders)

	// Create args array with userID at the beginning
	args := make([]interface{}, 0, len(lessonIDs)+1)
	args = append(args, userID)
	args = append(args, lessonIDs...)

	var completedCount int
	err = db.QueryRow(query, args...).Scan(&completedCount)
	if err != nil {
		return false, err
	}

	return completedCount == len(module.Lessons), nil
}

// GetModuleProgress calculates progress percentage for a specific module
func GetModuleProgress(db *sql.DB, userID int, moduleSlug string) (int, int, float64, error) {
	module, err := GetModuleBySlug(moduleSlug)
	if err != nil {
		return 0, 0, 0, err
	}

	totalLessons := len(module.Lessons)
	if totalLessons == 0 {
		return 0, 0, 100, nil // Empty module is 100% complete
	}

	// Get all lesson IDs for this module
	var lessonIDs []interface{}
	for _, lesson := range module.Lessons {
		lessonIDs = append(lessonIDs, lesson.ID)
	}

	// Build the placeholders for the SQL query
	placeholders := "?"
	for i := 1; i < len(lessonIDs); i++ {
		placeholders += ",?"
	}

	// Count completed lessons in this module
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM user_lesson_completion
		WHERE user_id = ? AND lesson_id IN (%s) AND completed = true
	`, placeholders)

	// Create args array with userID at the beginning
	args := make([]interface{}, 0, len(lessonIDs)+1)
	args = append(args, userID)
	args = append(args, lessonIDs...)

	var completedLessons int
	err = db.QueryRow(query, args...).Scan(&completedLessons)
	if err != nil {
		return 0, 0, 0, err
	}

	progressPercentage := float64(completedLessons) / float64(totalLessons) * 100

	return completedLessons, totalLessons, progressPercentage, nil
}

// GetUserProgress calculates a user's progress through the course
func GetUserProgress(db *sql.DB, userID int) (int, int, float64, error) {
	totalLessons := 0
	for _, module := range MasterclassDefinition {
		totalLessons += len(module.Lessons)
	}

	var completedLessons int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM user_lesson_completion
		WHERE user_id = ? AND completed = true
	`, userID).Scan(&completedLessons)

	if err != nil {
		return 0, 0, 0, err
	}

	var progressPercentage float64 = 0
	if totalLessons > 0 {
		progressPercentage = float64(completedLessons) / float64(totalLessons) * 100
	}

	return completedLessons, totalLessons, progressPercentage, nil
}

// HasCourseAccess checks if a user has access to a course
func HasCourseAccess(db *sql.DB, userID int, courseID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM user_course_access
		WHERE user_id = ? AND course_id = ? AND status = 'active'
	`, userID, courseID).Scan(&count)

	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, ErrNoAccess
	}

	return true, nil
}

// GrantCourseAccess grants a user access to a course
func GrantCourseAccess(db *sql.DB, userID int, courseID string, price float64) error {
	_, err := db.Exec(`
		INSERT INTO user_course_access (user_id, course_id, purchase_price, status)
		VALUES (?, ?, ?, 'active')
		ON CONFLICT(user_id, course_id) 
		DO UPDATE SET status = 'active'
	`, userID, courseID, price)

	return err
}

// RevokeCourseAccess revokes a user's access to a course
func RevokeCourseAccess(db *sql.DB, userID int, courseID string, reason string) error {
	_, err := db.Exec(`
		UPDATE user_course_access
		SET status = ?
		WHERE user_id = ? AND course_id = ?
	`, reason, userID, courseID)

	return err
}
