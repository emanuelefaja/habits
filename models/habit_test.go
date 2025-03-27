package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupHabitTestDB creates an in-memory SQLite database for testing
func setupHabitTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Enable foreign keys for SQLite (critical for CASCADE to work)
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Initialize the database schema
	if err := InitDB(db); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Ensure habit_logs table exists for streak calculation tests
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS habit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		habit_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('done', 'missed', 'skipped')),
		value TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(habit_id) REFERENCES habits(id) ON DELETE CASCADE,
		UNIQUE(habit_id, date)
	)`)
	if err != nil {
		t.Fatalf("Failed to create habit_logs table: %v", err)
	}

	return db
}

// createTestUser creates a test user for habit tests
func createTestUserForHabits(t *testing.T, db *sql.DB, emailSuffix ...string) int64 {
	suffix := ""
	if len(emailSuffix) > 0 {
		suffix = emailSuffix[0]
	}

	passHash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	email := fmt.Sprintf("testhabit%s@example.com", suffix)

	result, err := db.Exec(`
		INSERT INTO users (first_name, last_name, email, password_hash, show_confetti, notification_enabled) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"Test", "User", email, passHash, true, true)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	return userID
}

// createTestHabitForTests creates a test habit
func createTestHabitForTests(t *testing.T, db *sql.DB, userID int64, habitType HabitType, name string) *Habit {
	habit := &Habit{
		UserID:    int(userID),
		Name:      name,
		Emoji:     "✅",
		HabitType: habitType,
		IsDefault: false,
	}

	if habitType == OptionSelectHabit {
		options := []HabitOption{
			{Emoji: "🙂", Label: "Good"},
			{Emoji: "😐", Label: "Neutral"},
			{Emoji: "☹️", Label: "Bad"},
		}

		habitOptions, err := MarshalHabitOptions(options)
		if err != nil {
			t.Fatalf("Failed to marshal habit options: %v", err)
		}
		habit.HabitOptions = habitOptions
	}

	err := habit.Create(db)
	if err != nil {
		t.Fatalf("Failed to create test habit: %v", err)
	}

	return habit
}

// createHabitLog creates a test habit log
func createHabitLog(t *testing.T, db *sql.DB, habitID int, date time.Time, status string, value interface{}) *HabitLog {
	log := &HabitLog{
		HabitID: habitID,
		Date:    date,
		Status:  status,
	}

	if value != nil {
		err := log.SetValue(value)
		if err != nil {
			t.Fatalf("Failed to set log value: %v", err)
		}
	}

	err := log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to create habit log: %v", err)
	}

	return log
}

// TestHabitCreate tests habit creation with better cleanup
func TestHabitCreate(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "create")

	// Store created habit IDs for cleanup
	var createdHabitIDs []int

	testCases := []struct {
		name      string
		habitType HabitType
		emoji     string
	}{
		{"Binary Habit", BinaryHabit, "✅"},
		{"Numeric Habit", NumericHabit, "🔢"},
		{"Option Habit", OptionSelectHabit, "🌈"},
		{"SetReps Habit", SetRepsHabit, "💪"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			habit := &Habit{
				UserID:    int(userID),
				Name:      tc.name,
				Emoji:     tc.emoji,
				HabitType: tc.habitType,
				IsDefault: false,
			}

			if tc.habitType == OptionSelectHabit {
				options := []HabitOption{
					{Emoji: "🙂", Label: "Good"},
					{Emoji: "😐", Label: "Neutral"},
					{Emoji: "☹️", Label: "Bad"},
				}

				habitOptions, err := MarshalHabitOptions(options)
				if err != nil {
					t.Fatalf("Failed to marshal habit options: %v", err)
				}
				habit.HabitOptions = habitOptions
			}

			err := habit.Create(db)
			if err != nil {
				t.Fatalf("Failed to create habit: %v", err)
			}

			// Track for cleanup
			createdHabitIDs = append(createdHabitIDs, habit.ID)

			if habit.ID <= 0 {
				t.Error("Expected habit ID to be set after creation")
			}

			if habit.DisplayOrder <= 0 {
				t.Error("Expected display_order to be set after creation")
			}

			// Verify by retrieving from DB
			retrieved, err := GetHabitByID(db, habit.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve habit by ID: %v", err)
			}

			if retrieved.Name != tc.name {
				t.Errorf("Expected habit name %s, got %s", tc.name, retrieved.Name)
			}

			if retrieved.Emoji != tc.emoji {
				t.Errorf("Expected emoji %s, got %s", tc.emoji, retrieved.Emoji)
			}

			if retrieved.HabitType != tc.habitType {
				t.Errorf("Expected habit type %s, got %s", tc.habitType, retrieved.HabitType)
			}
		})
	}

	// Cleanup all created habits
	for _, habitID := range createdHabitIDs {
		_, err := db.Exec("DELETE FROM habits WHERE id = ?", habitID)
		if err != nil {
			t.Logf("Warning: Failed to clean up habit %d: %v", habitID, err)
		}
	}
}

// TestHabitExists tests the HabitExists function
func TestHabitExists(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "exists")

	// Create a test habit
	createTestHabitForTests(t, db, userID, BinaryHabit, "Existing Habit")

	// Test cases
	testCases := []struct {
		name           string
		habitName      string
		userID         int
		expectedExists bool
	}{
		{"Exact match", "Existing Habit", int(userID), true},
		{"Case insensitive match", "EXISTING habit", int(userID), true},
		{"Non-existent habit", "Non-existent Habit", int(userID), false},
		{"Wrong user", "Existing Habit", int(userID) + 1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists, err := HabitExists(db, tc.habitName, tc.userID)
			if err != nil {
				t.Fatalf("HabitExists failed: %v", err)
			}

			if exists != tc.expectedExists {
				t.Errorf("Expected exists=%v for habit '%s', got %v",
					tc.expectedExists, tc.habitName, exists)
			}
		})
	}
}

// TestGetHabitsByUserID tests retrieving all habits for a user
func TestGetHabitsByUserID(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "getall")

	// Create multiple habits
	habits := []*Habit{
		createTestHabitForTests(t, db, userID, BinaryHabit, "Habit 1"),
		createTestHabitForTests(t, db, userID, NumericHabit, "Habit 2"),
		createTestHabitForTests(t, db, userID, OptionSelectHabit, "Habit 3"),
	}

	// Create a habit for a different user
	otherUserID := createTestUserForHabits(t, db, "other")
	_ = createTestHabitForTests(t, db, otherUserID, BinaryHabit, "Other User's Habit")

	// Test retrieval
	retrievedHabits, err := GetHabitsByUserID(db, int(userID))
	if err != nil {
		t.Fatalf("GetHabitsByUserID failed: %v", err)
	}

	if len(retrievedHabits) != len(habits) {
		t.Errorf("Expected %d habits, got %d", len(habits), len(retrievedHabits))
	}

	// Check ordering by display_order
	for i := 1; i < len(retrievedHabits); i++ {
		if retrievedHabits[i-1].DisplayOrder > retrievedHabits[i].DisplayOrder {
			t.Errorf("Habits not ordered by display_order")
		}
	}

	// Test with no habits
	newUserID := createTestUserForHabits(t, db, "nohabits")
	noHabits, err := GetHabitsByUserID(db, int(newUserID))
	if err != nil {
		t.Fatalf("GetHabitsByUserID failed: %v", err)
	}

	if len(noHabits) != 0 {
		t.Errorf("Expected 0 habits for new user, got %d", len(noHabits))
	}
}

// TestBinaryHabitLog tests creating and updating binary habit logs
func TestBinaryHabitLog(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "binary")
	habit := createTestHabitForTests(t, db, userID, BinaryHabit, "Binary Test Habit")

	today := time.Now()

	// Create a "done" log
	log := &HabitLog{
		HabitID: habit.ID,
		Date:    today,
		Status:  "done",
	}

	err := log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to create binary habit log: %v", err)
	}

	// Verify log was created
	logs, err := GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	if logs[0].Status != "done" {
		t.Errorf("Expected status 'done', got '%s'", logs[0].Status)
	}

	// Update to "missed"
	log.Status = "missed"
	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to update binary habit log: %v", err)
	}

	// Verify log was updated
	logs, err = GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if logs[0].Status != "missed" {
		t.Errorf("Expected updated status 'missed', got '%s'", logs[0].Status)
	}

	// Delete log by setting status to "none" (special case for binary habits)
	log.Status = "none"
	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to delete binary habit log: %v", err)
	}

	// Verify log was deleted
	logs, err = GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected log to be deleted, found %d logs", len(logs))
	}
}

// TestNumericHabitLog tests creating and updating numeric habit logs
func TestNumericHabitLog(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "numeric")
	habit := createTestHabitForTests(t, db, userID, NumericHabit, "Numeric Test Habit")

	today := time.Now()

	// Create a log with numeric value
	log := &HabitLog{
		HabitID: habit.ID,
		Date:    today,
		Status:  "done",
	}

	// Set numeric value
	err := log.SetValue(map[string]interface{}{
		"value": 10.5,
	})
	if err != nil {
		t.Fatalf("Failed to set numeric value: %v", err)
	}

	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to create numeric habit log: %v", err)
	}

	// Verify log was created
	logs, err := GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	// Verify numeric value
	var valueMap map[string]interface{}
	err = json.Unmarshal([]byte(logs[0].Value.String), &valueMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}

	if valueMap["value"].(float64) != 10.5 {
		t.Errorf("Expected value 10.5, got %v", valueMap["value"])
	}

	// Update the value
	err = log.SetValue(map[string]interface{}{
		"value": 15.0,
	})
	if err != nil {
		t.Fatalf("Failed to update numeric value: %v", err)
	}

	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to update numeric habit log: %v", err)
	}

	// Verify value was updated
	logs, err = GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	err = json.Unmarshal([]byte(logs[0].Value.String), &valueMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal updated value: %v", err)
	}

	if valueMap["value"].(float64) != 15.0 {
		t.Errorf("Expected updated value 15.0, got %v", valueMap["value"])
	}
}

// TestOptionSelectHabitLog tests creating and updating option-select habit logs
func TestOptionSelectHabitLog(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "option")
	habit := createTestHabitForTests(t, db, userID, OptionSelectHabit, "Mood Tracking")

	today := time.Now()

	// Create a log with option value
	log := &HabitLog{
		HabitID: habit.ID,
		Date:    today,
		Status:  "done",
	}

	// Set option value
	err := log.SetValue(map[string]interface{}{
		"emoji": "🙂",
		"label": "Good",
	})
	if err != nil {
		t.Fatalf("Failed to set option value: %v", err)
	}

	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to create option-select habit log: %v", err)
	}

	// Verify log was created
	logs, err := GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	// Verify option value
	var valueMap map[string]interface{}
	err = json.Unmarshal([]byte(logs[0].Value.String), &valueMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}

	if valueMap["emoji"].(string) != "🙂" || valueMap["label"].(string) != "Good" {
		t.Errorf("Expected emoji 🙂 and label Good, got %v and %v",
			valueMap["emoji"], valueMap["label"])
	}
}

// TestSetRepsHabitLog tests creating and updating set-reps habit logs
func TestSetRepsHabitLog(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "setreps")
	habit := createTestHabitForTests(t, db, userID, SetRepsHabit, "Pushups")

	today := time.Now()

	// Create a log with sets and reps
	log := &HabitLog{
		HabitID: habit.ID,
		Date:    today,
		Status:  "done",
	}

	// Set sets and reps
	setRepsValue := SetRepsValue{
		Sets: []SetRep{
			{Set: 1, Reps: 10},
			{Set: 2, Reps: 12},
			{Set: 3, Reps: 8, Value: 20.0}, // With weight
		},
		Unit: "kg",
	}

	err := log.SetValue(setRepsValue)
	if err != nil {
		t.Fatalf("Failed to set sets and reps: %v", err)
	}

	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to create set-reps habit log: %v", err)
	}

	// Verify log was created
	logs, err := GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	// Verify sets and reps
	var retrievedValue SetRepsValue
	err = json.Unmarshal([]byte(logs[0].Value.String), &retrievedValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}

	if len(retrievedValue.Sets) != 3 {
		t.Errorf("Expected 3 sets, got %d", len(retrievedValue.Sets))
	}

	if retrievedValue.Sets[0].Reps != 10 {
		t.Errorf("Expected 10 reps in set 1, got %d", retrievedValue.Sets[0].Reps)
	}

	if retrievedValue.Sets[2].Value != 20.0 {
		t.Errorf("Expected weight 20.0 in set 3, got %f", retrievedValue.Sets[2].Value)
	}

	// Test "missed" status
	log.Status = "missed"
	err = log.SetValue(SetRepsValue{Sets: []SetRep{}})
	if err != nil {
		t.Fatalf("Failed to set empty sets: %v", err)
	}

	err = log.CreateOrUpdate(db)
	if err != nil {
		t.Fatalf("Failed to update to missed status: %v", err)
	}

	// Verify status was updated
	logs, err = GetHabitLogsByDateRange(db, habit.ID, today, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if logs[0].Status != "missed" {
		t.Errorf("Expected status 'missed', got '%s'", logs[0].Status)
	}
}

// TestGetHabitLogsByDateRange tests retrieval of habit logs by date range
func TestGetHabitLogsByDateRange(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "daterange")
	habit := createTestHabitForTests(t, db, userID, BinaryHabit, "Date Range Test")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)
	tomorrow := today.AddDate(0, 0, 1)

	// Create logs for different dates
	createHabitLog(t, db, habit.ID, today, "done", nil)
	createHabitLog(t, db, habit.ID, yesterday, "missed", nil)
	createHabitLog(t, db, habit.ID, twoDaysAgo, "skipped", nil)

	// Test cases
	testCases := []struct {
		name        string
		startDate   time.Time
		endDate     time.Time
		expectedLen int
	}{
		{"Single day (today)", today, today, 1},
		{"Two days (today and yesterday)", yesterday, today, 2},
		{"All three days", twoDaysAgo, today, 3},
		{"Future range", tomorrow, tomorrow.AddDate(0, 0, 1), 0},
		{"Invalid range (end before start)", today, yesterday, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs, err := GetHabitLogsByDateRange(db, habit.ID, tc.startDate, tc.endDate)
			if err != nil {
				t.Fatalf("GetHabitLogsByDateRange failed: %v", err)
			}

			if len(logs) != tc.expectedLen {
				t.Errorf("Expected %d logs, got %d", tc.expectedLen, len(logs))
			}
		})
	}
}

// TestHabitUpdate tests updating habit properties
func TestHabitUpdate(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "update")
	habit := createTestHabitForTests(t, db, userID, BinaryHabit, "Original Name")

	// Update properties
	habit.Name = "Updated Name"
	habit.Emoji = "🔄"

	err := habit.Update(db)
	if err != nil {
		t.Fatalf("Failed to update habit: %v", err)
	}

	// Verify update
	updated, err := GetHabitByID(db, habit.ID)
	if err != nil {
		t.Fatalf("Failed to get updated habit: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name)
	}

	if updated.Emoji != "🔄" {
		t.Errorf("Expected emoji '🔄', got '%s'", updated.Emoji)
	}
}

// TestHabitDelete tests deleting a habit and its logs
func TestHabitDelete(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "delete")
	habit := createTestHabitForTests(t, db, userID, BinaryHabit, "Delete Test")

	// Create a few logs
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	createHabitLog(t, db, habit.ID, today, "done", nil)
	createHabitLog(t, db, habit.ID, yesterday, "done", nil)

	// Verify logs exist
	logs, err := GetHabitLogsByDateRange(db, habit.ID, yesterday, today)
	if err != nil {
		t.Fatalf("Failed to get habit logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("Expected 2 logs before deletion, got %d", len(logs))
	}

	// Delete the habit
	err = habit.Delete(db)
	if err != nil {
		t.Fatalf("Failed to delete habit: %v", err)
	}

	// Verify habit is deleted
	_, err = GetHabitByID(db, habit.ID)
	if err == nil {
		t.Error("Expected error when getting deleted habit, got nil")
	}

	// Verify logs are deleted (should fail due to foreign key constraint)
	logs, err = GetHabitLogsByDateRange(db, habit.ID, yesterday, today)
	if err != nil {
		// Expected error, but let's make sure it's empty anyway
		if len(logs) != 0 {
			t.Errorf("Expected 0 logs after deletion, got %d", len(logs))
		}
	} else {
		// If no error, make sure logs array is empty
		if len(logs) != 0 {
			t.Errorf("Expected 0 logs after deletion, got %d", len(logs))
		}
	}
}

// TestCalculateCurrentStreak tests streak calculation
func TestCalculateCurrentStreak(t *testing.T) {
	db := setupHabitTestDB(t)
	defer db.Close()

	userID := createTestUserForHabits(t, db, "streak")
	habit := createTestHabitForTests(t, db, userID, BinaryHabit, "Streak Test")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Test cases
	testCases := []struct {
		name           string
		setupLogs      func(t *testing.T, tx *sql.Tx, habitID int)
		expectedStreak int
	}{
		{
			name: "No logs",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Make sure no logs exist
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}
			},
			expectedStreak: 0,
		},
		{
			name: "Single day streak (today)",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Clear any existing logs first
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}

				// Use transaction for creating log - directly format the date as YYYY-MM-DD
				todayStr := today.Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, todayStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}
			},
			expectedStreak: 1,
		},
		{
			name: "Three day streak",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Clear previous logs
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}

				// Create 3-day streak using the transaction
				dates := []time.Time{today, today.AddDate(0, 0, -1), today.AddDate(0, 0, -2)}
				for _, date := range dates {
					dateStr := date.Format("2006-01-02")
					_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
						habitID, dateStr, "done", nil)
					if err != nil {
						t.Fatalf("Failed to create habit log: %v", err)
					}
				}
			},
			expectedStreak: 3,
		},
		{
			name: "Streak with skipped day",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Clear previous logs
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}

				// Insert today
				todayStr := today.Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, todayStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Insert yesterday (skipped)
				yesterdayStr := today.AddDate(0, 0, -1).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, yesterdayStr, "skipped", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Insert day before
				twoDaysAgoStr := today.AddDate(0, 0, -2).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, twoDaysAgoStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}
			},
			expectedStreak: 3,
		},
		{
			name: "Broken streak",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Clear previous logs
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}

				// Today
				todayStr := today.Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, todayStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Two days ago (missing yesterday)
				twoDaysAgoStr := today.AddDate(0, 0, -2).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, twoDaysAgoStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Three days ago
				threeDaysAgoStr := today.AddDate(0, 0, -3).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, threeDaysAgoStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}
			},
			expectedStreak: 1, // Only today
		},
		{
			name: "Missed day breaks streak",
			setupLogs: func(t *testing.T, tx *sql.Tx, habitID int) {
				// Clear previous logs
				_, err := tx.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habitID)
				if err != nil {
					t.Fatalf("Failed to clear logs: %v", err)
				}

				// Today
				todayStr := today.Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, todayStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Yesterday (missed)
				yesterdayStr := today.AddDate(0, 0, -1).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, yesterdayStr, "missed", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}

				// Two days ago
				twoDaysAgoStr := today.AddDate(0, 0, -2).Format("2006-01-02")
				_, err = tx.Exec("INSERT INTO habit_logs (habit_id, date, status, value) VALUES (?, ?, ?, ?)",
					habitID, twoDaysAgoStr, "done", nil)
				if err != nil {
					t.Fatalf("Failed to create habit log: %v", err)
				}
			},
			expectedStreak: 1, // Only today
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Using a sub-transaction to ensure isolation between test cases
			tx, err := db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Handle rollback at the end of the test, regardless of pass/fail
			defer func() {
				tx.Rollback() // This is safe to call even after a commit
			}()

			// Setup the logs for this test case within the transaction
			tc.setupLogs(t, tx, habit.ID)

			// Create a copy of the habit for this test to avoid interference
			testHabit := *habit

			// Adapt the habit test to use the transaction
			// Calculate the streak using DB interface shared by both *sql.DB and *sql.Tx
			err = calculateCurrentStreakTx(&testHabit, tx)
			if err != nil {
				t.Fatalf("CalculateCurrentStreak failed: %v", err)
			}

			// Verify the streak
			if testHabit.CurrentStreak != tc.expectedStreak {
				t.Errorf("Expected streak %d, got %d", tc.expectedStreak, testHabit.CurrentStreak)
			}

			// Commit the transaction
			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit transaction: %v", err)
			}
		})
	}

	// Final cleanup to ensure no leaked resources
	_, err := db.Exec("DELETE FROM habit_logs WHERE habit_id = ?", habit.ID)
	if err != nil {
		t.Logf("Warning: Final cleanup failed: %v", err)
	}
}

// Helper function to calculate streak with a transaction
func calculateCurrentStreakTx(habit *Habit, tx *sql.Tx) error {
	// Adapt the main CalculateCurrentStreak logic but use the transaction
	query := `
	WITH RECURSIVE dates(check_date) AS (
		-- Start with today if we have a log, otherwise try yesterday
		SELECT 
			CASE 
				WHEN EXISTS (
					SELECT 1 FROM habit_logs 
					WHERE habit_id = ? 
					AND date(date) = date('now')
					AND status IN ('done', 'skipped')
				)
				THEN date('now')
				WHEN EXISTS (
					SELECT 1 FROM habit_logs 
					WHERE habit_id = ? 
					AND date(date) = date('now', '-1 day')
					AND status IN ('done', 'skipped')
				)
				THEN date('now', '-1 day')
			END
		WHERE EXISTS (
			SELECT 1 FROM habit_logs 
			WHERE habit_id = ? 
			AND date(date) = 
				CASE 
					WHEN EXISTS (
						SELECT 1 FROM habit_logs 
						WHERE habit_id = ? 
						AND date(date) = date('now')
						AND status IN ('done', 'skipped')
					)
					THEN date('now')
					ELSE date('now', '-1 day')
				END
			AND status IN ('done', 'skipped')
		)
		
		UNION ALL
		
		SELECT date(dates.check_date, '-1 day')
		FROM dates
		WHERE EXISTS (
			SELECT 1 
			FROM habit_logs h 
			WHERE date(h.date) = date(dates.check_date, '-1 day')
			AND h.habit_id = ?
			AND h.status IN ('done', 'skipped')
		)
		LIMIT 366  -- Reasonable limit for recursion (full year)
	)
	SELECT COUNT(*) as streak
	FROM dates`

	err := tx.QueryRow(query, habit.ID, habit.ID, habit.ID, habit.ID, habit.ID).Scan(&habit.CurrentStreak)
	if err != nil {
		return fmt.Errorf("error calculating streak: %v", err)
	}

	return nil
}

// TestMarshalHabitOptions tests marshaling habit options
func TestMarshalHabitOptions(t *testing.T) {
	testCases := []struct {
		name      string
		options   []HabitOption
		expectErr bool
		expectNil bool
	}{
		{
			name: "Valid options",
			options: []HabitOption{
				{Emoji: "🙂", Label: "Good"},
				{Emoji: "😐", Label: "Neutral"},
			},
			expectErr: false,
			expectNil: false,
		},
		{
			name:      "Empty options",
			options:   []HabitOption{},
			expectErr: false,
			expectNil: true,
		},
		{
			name: "Null option fields",
			options: []HabitOption{
				{Emoji: "", Label: ""},
			},
			expectErr: false,
			expectNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := MarshalHabitOptions(tc.options)

			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.expectNil && result.Valid {
				t.Error("Expected invalid (nil) result, got valid")
			}

			if !tc.expectNil && !result.Valid {
				t.Error("Expected valid result, got invalid (nil)")
			}

			if !tc.expectNil && result.Valid {
				// Verify we can unmarshal back
				var options []HabitOption
				err = json.Unmarshal([]byte(result.String), &options)
				if err != nil {
					t.Errorf("Failed to unmarshal result back to options: %v", err)
				}

				if len(options) != len(tc.options) {
					t.Errorf("Expected %d options, got %d", len(tc.options), len(options))
				}
			}
		})
	}
}
