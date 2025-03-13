package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"mad/middleware"
	"mad/models"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CreateHabitRequest struct {
	Name         string               `json:"name"`
	Emoji        string               `json:"emoji"`
	HabitType    models.HabitType     `json:"habit_type"`
	HabitOptions []models.HabitOption `json:"habit_options,omitempty"`
}

// BulkHabitRequest represents a request to create multiple habits
type BulkHabitRequest struct {
	Name      string           `json:"name"`
	Emoji     string           `json:"emoji"`
	HabitType models.HabitType `json:"habit_type"`
}

// SetRepsResponse represents a response for set-reps habits
type SetRepsResponse struct {
	*models.HabitLog
	TotalSets int `json:"total_sets"`
	TotalReps int `json:"total_reps"`
}

// CreateHabitHandler handles the creation of a new habit
func CreateHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Starting CreateHabitHandler")
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from session
		userID := middleware.GetUserID(r)
		log.Printf("User ID: %d", userID)

		// Read and log request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error reading request",
			})
			return
		}
		log.Printf("Raw request body: %s", string(body))

		// Restore body for further processing
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		// Decode request body
		var request CreateHabitRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Printf("Error decoding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}
		log.Printf("Decoded request: %+v", request)

		// Handle option-select habit options
		var habitOptionsSql sql.NullString
		if request.HabitType == models.OptionSelectHabit {
			log.Printf("Processing option-select habit with options: %+v", request.HabitOptions)
			if len(request.HabitOptions) == 0 {
				log.Printf("Error: No habit options provided for option-select habit")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "option-select requires habit_options",
				})
				return
			}
			ho, err := models.MarshalHabitOptions(request.HabitOptions)
			if err != nil {
				log.Printf("Error marshaling habit options: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Error marshaling habit_options",
				})
				return
			}
			log.Printf("Marshaled habit options: %s", ho.String)
			habitOptionsSql = ho
		}

		habit := models.Habit{
			UserID:       userID,
			Name:         request.Name,
			Emoji:        request.Emoji,
			HabitType:    request.HabitType,
			IsDefault:    false,
			HabitOptions: habitOptionsSql,
		}

		// Check if habit already exists
		exists, err := models.HabitExists(db, habit.Name, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error checking for duplicate habit",
			})
			return
		}
		if exists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "A habit with this name already exists",
			})
			return
		}

		// Create the habit
		if err := habit.Create(db); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error creating habit",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit created successfully",
			Data:    habit,
		})
	}
}

// GetHabitHandler retrieves a habit by ID
func GetHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		habit, err := models.GetHabitByID(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(habit)
	}
}

// UpdateHabitHandler updates an existing habit
func UpdateHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var habit models.Habit
		if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := habit.Update(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(habit)
	}
}

// DeleteHabitHandler deletes a habit by ID
func DeleteHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		habit := models.Habit{ID: id}
		if err := habit.Delete(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return a JSON response with success and redirect URL
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"redirect": "/",
			"message":  "Habit deleted successfully",
		})
	}
}

// CreateOrUpdateHabitLogHandler handles creating or updating a habit log
func CreateOrUpdateHabitLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse request body
		var request struct {
			HabitID int         `json:"habit_id"`
			Date    string      `json:"date"`
			Status  string      `json:"status,omitempty"` // For binary/option-select, optional
			Value   interface{} `json:"value,omitempty"`  // For numeric/option-select, required
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		// Parse date
		date, err := time.Parse("2006-01-02", request.Date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid date format. Use YYYY-MM-DD",
			})
			return
		}

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		err = db.QueryRow("SELECT user_id FROM habits WHERE id = ?", request.HabitID).Scan(&habitUserID)
		if err != nil || habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Get habit type
		var habitType models.HabitType
		err = db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", request.HabitID).Scan(&habitType)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habit type",
			})
			return
		}

		habitLog := &models.HabitLog{
			HabitID: request.HabitID,
			Date:    date,
		}

		// Handle based on habit type
		switch habitType {
		case models.BinaryHabit:
			if request.Status == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Status is required for binary habits",
				})
				return
			}
			habitLog.Status = request.Status

		case models.NumericHabit:
			if request.Value == nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Value is required for numeric habits",
				})
				return
			}
			habitLog.Status = request.Status
			if habitLog.Status == "" {
				habitLog.Status = "done"
			}
			if err := habitLog.SetValue(request.Value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid numeric value",
				})
				return
			}

		case models.OptionSelectHabit:
			log.Printf("Processing option-select habit log: %+v", request.Value)
			if request.Value == nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "value is required for option-select habits",
				})
				return
			}
			if request.Status == "" {
				habitLog.Status = "done"
			} else {
				habitLog.Status = request.Status
			}

			// Retrieve habit_options
			var habitOptionsStr sql.NullString
			if err := db.QueryRow("SELECT habit_options FROM habits WHERE id = ?", request.HabitID).Scan(&habitOptionsStr); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Error retrieving habit options",
				})
				return
			}
			if !habitOptionsStr.Valid {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "This option-select habit has no options set",
				})
				return
			}

			var habitOptions []models.HabitOption
			if err := json.Unmarshal([]byte(habitOptionsStr.String), &habitOptions); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid habit_options format",
				})
				return
			}

			valMap, ok := request.Value.(map[string]interface{})
			if !ok || valMap["emoji"] == nil || valMap["label"] == nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "value must contain 'emoji' and 'label' for option-select habits",
				})
				return
			}

			chosenEmoji, okEmoji := valMap["emoji"].(string)
			chosenLabel, okLabel := valMap["label"].(string)
			if !okEmoji || !okLabel {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "emoji and label must be strings",
				})
				return
			}

			// Validate chosen option
			validOption := false
			for _, opt := range habitOptions {
				if opt.Emoji == chosenEmoji && opt.Label == chosenLabel {
					validOption = true
					break
				}
			}
			if !validOption {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Chosen option not in habit_options",
				})
				return
			}

			if err := habitLog.SetValue(request.Value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid option-select value",
				})
				return
			}

		case models.SetRepsHabit:
			log.Printf("Processing set-reps habit log. Value: %+v", request.Value)

			// For missed/skipped status, set an empty sets array
			if request.Status == "missed" || request.Status == "skipped" {
				emptyValue := models.SetRepsValue{Sets: []models.SetRep{}}
				valueBytes, err := json.Marshal(emptyValue)
				if err != nil {
					log.Printf("Error marshaling empty set-reps value: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(APIResponse{
						Success: false,
						Message: "Error processing set-reps value",
					})
					return
				}
				habitLog.Value = sql.NullString{
					String: string(valueBytes),
					Valid:  true,
				}
				habitLog.Status = request.Status

				if err := habitLog.CreateOrUpdate(db); err != nil {
					log.Printf("Error saving habit log: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(APIResponse{
						Success: false,
						Message: "Error saving habit log",
					})
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(APIResponse{
					Success: true,
					Message: "Habit log saved successfully",
					Data: SetRepsResponse{
						HabitLog:  habitLog,
						TotalSets: 0,
						TotalReps: 0,
					},
				})
				return
			}

			// Parse the value for set-reps
			var setRepsValue models.SetRepsValue
			switch v := request.Value.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &setRepsValue); err != nil {
					log.Printf("Error unmarshaling string value: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(APIResponse{
						Success: false,
						Message: "Invalid set-reps format",
					})
					return
				}
			case map[string]interface{}:
				// Convert map to JSON string first
				valueBytes, err := json.Marshal(v)
				if err != nil {
					log.Printf("Error marshaling map value: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(APIResponse{
						Success: false,
						Message: "Invalid set-reps format",
					})
					return
				}
				if err := json.Unmarshal(valueBytes, &setRepsValue); err != nil {
					log.Printf("Error unmarshaling map value: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(APIResponse{
						Success: false,
						Message: "Invalid set-reps format",
					})
					return
				}
			default:
				log.Printf("Invalid value type for set-reps: %T", v)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid value type for set-reps",
				})
				return
			}

			// Set the value
			valueBytes, err := json.Marshal(setRepsValue)
			if err != nil {
				log.Printf("Error marshaling set-reps value: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Error processing set-reps value",
				})
				return
			}

			habitLog.Value = sql.NullString{
				String: string(valueBytes),
				Valid:  true,
			}
			habitLog.Status = request.Status

		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unsupported habit type",
			})
			return
		}

		// Validate value format
		if err := habitLog.ValidateValue(db); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Create or update the log
		if err := habitLog.CreateOrUpdate(db); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error saving habit log",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit log saved successfully",
			Data:    habitLog,
		})
	}
}

// GetHabitLogsHandler retrieves habit logs for a date range
func GetHabitLogsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get query parameters
		habitID, err := strconv.Atoi(r.URL.Query().Get("habit_id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid habit ID",
			})
			return
		}

		startDate, err := time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid start date format. Use YYYY-MM-DD",
			})
			return
		}

		endDate, err := time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid end date format. Use YYYY-MM-DD",
			})
			return
		}

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		var habitType models.HabitType
		err = db.QueryRow("SELECT user_id, habit_type FROM habits WHERE id = ?", habitID).Scan(&habitUserID, &habitType)
		if err != nil || habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Get logs
		logs, err := models.GetHabitLogsByDateRange(db, habitID, startDate, endDate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habit logs",
			})
			return
		}

		// If this is a set-reps habit, enhance the response with totals
		if habitType == models.SetRepsHabit {
			var enhancedLogs []SetRepsResponse
			for _, log := range logs {
				var setRepsValue models.SetRepsValue
				totalReps := 0
				totalSets := 0

				if log.Value.Valid {
					if err := json.Unmarshal([]byte(log.Value.String), &setRepsValue); err == nil {
						totalSets = len(setRepsValue.Sets)
						for _, set := range setRepsValue.Sets {
							totalReps += set.Reps
						}
					}
				}

				// Create a copy of the log to get a pointer to it
				logCopy := log
				enhancedLogs = append(enhancedLogs, SetRepsResponse{
					HabitLog:  &logCopy,
					TotalSets: totalSets,
					TotalReps: totalReps,
				})
			}

			// Return enhanced response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    enhancedLogs,
			})
			return
		}

		// Return standard response for other habit types
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    logs,
		})
	}
}

// BulkCreateHabitsHandler handles creating multiple habits at once
func BulkCreateHabitsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("BulkCreateHabitsHandler: Method %s", r.Method)
		userID := middleware.GetUserID(r)
		log.Printf("BulkCreateHabitsHandler: UserID %d", userID)

		// Log raw request body
		var rawBody []byte
		rawBody, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		log.Printf("BulkCreateHabitsHandler: Raw request body: %s", string(rawBody))

		var habits []BulkHabitRequest
		if err := json.NewDecoder(r.Body).Decode(&habits); err != nil {
			log.Printf("BulkCreateHabitsHandler: Error decoding request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Log each habit request
		for i, habit := range habits {
			log.Printf("BulkCreateHabitsHandler: Habit %d - Name: %s, Emoji: %s, Type: %s",
				i, habit.Name, habit.Emoji, habit.HabitType)
		}

		for _, habit := range habits {
			exists, err := models.HabitExists(db, habit.Name, userID)
			if err != nil {
				log.Printf("BulkCreateHabitsHandler: Error checking habit existence: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if exists {
				log.Printf("BulkCreateHabitsHandler: Habit already exists: %s", habit.Name)
				continue
			}

			newHabit := &models.Habit{
				UserID:    userID,
				Name:      habit.Name,
				Emoji:     habit.Emoji,
				HabitType: habit.HabitType,
				IsDefault: false,
			}

			// Log the habit being created
			log.Printf("BulkCreateHabitsHandler: Creating habit - Name: %s, Emoji: %s, Type: %s",
				newHabit.Name, newHabit.Emoji, newHabit.HabitType)

			if err := newHabit.Create(db); err != nil {
				log.Printf("BulkCreateHabitsHandler: Error creating habit: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("BulkCreateHabitsHandler: Successfully created habit: %s", habit.Name)
		}

		log.Printf("BulkCreateHabitsHandler: Successfully created all habits")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

// GetHabitsHandler retrieves all habits for a user
func GetHabitsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID := middleware.GetUserID(r)
		if userID == 0 {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "User not authenticated",
			})
			return
		}

		habits, err := models.GetHabitsByUserID(db, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habits",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    habits,
		})
	}
}

func UpdateHabitOrderHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID := middleware.GetUserID(r)
		if userID == 0 {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "User not authenticated",
			})
			return
		}

		var habitIDs []int
		if err := json.NewDecoder(r.Body).Decode(&habitIDs); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		log.Printf("Reorder request received: %v", habitIDs)

		// Update each habit's display_order
		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error starting transaction",
			})
			return
		}

		for i, id := range habitIDs {
			// Ensure habit belongs to user
			var belongsToUser bool
			err = tx.QueryRow("SELECT EXISTS (SELECT 1 FROM habits WHERE id = ? AND user_id = ?)", id, userID).Scan(&belongsToUser)
			if err != nil || !belongsToUser {
				tx.Rollback()
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Unauthorized access to habit",
				})
				return
			}

			_, err := tx.Exec("UPDATE habits SET display_order = ? WHERE id = ?", i, id)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Error updating order",
				})
				return
			}
		}

		err = tx.Commit()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error committing transaction",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit order updated successfully",
		})
	}
}

// DeleteHabitLogHandler handles the deletion of a habit log
func DeleteHabitLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get log ID from URL parameter
		logID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid log ID",
			})
			return
		}

		// Get current user ID from session
		userID := middleware.GetUserID(r)

		// Verify the habit log belongs to the user and get habit_id
		var habitID int
		var habitUserID int
		err = db.QueryRow(`
			SELECT h.user_id, h.id
			FROM habit_logs hl 
			JOIN habits h ON hl.habit_id = h.id 
			WHERE hl.id = ?`, logID).Scan(&habitUserID, &habitID)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Habit log not found",
			})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error verifying habit log ownership",
			})
			return
		}

		if habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit log",
			})
			return
		}

		// Delete the log
		_, err = db.Exec("DELETE FROM habit_logs WHERE id = ?", logID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error deleting habit log",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit log deleted successfully",
		})
	}
}

// Add this struct for the request
type UpdateHabitNameRequest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Add this handler function
func UpdateHabitNameHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("UpdateHabitNameHandler: Received request")
		w.Header().Set("Content-Type", "application/json")

		// Parse request
		var req UpdateHabitNameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("UpdateHabitNameHandler: Error decoding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}
		log.Printf("UpdateHabitNameHandler: Request data - ID: %d, Name: %s", req.ID, req.Name)

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		log.Printf("UpdateHabitNameHandler: UserID from session: %d", userID)

		var habitUserID int
		err := db.QueryRow("SELECT user_id FROM habits WHERE id = ?", req.ID).Scan(&habitUserID)
		if err != nil {
			log.Printf("UpdateHabitNameHandler: Error getting habit user ID: %v", err)
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}
		log.Printf("UpdateHabitNameHandler: Habit user ID: %d", habitUserID)

		if habitUserID != userID {
			log.Printf("UpdateHabitNameHandler: User ID mismatch - Session: %d, Habit: %d", userID, habitUserID)
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Check if name already exists for this user
		exists, err := models.HabitExists(db, req.Name, userID)
		if err != nil {
			log.Printf("UpdateHabitNameHandler: Error checking habit existence: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error checking for duplicate habit",
			})
			return
		}
		if exists {
			log.Printf("UpdateHabitNameHandler: Habit name already exists for user")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "A habit with this name already exists",
			})
			return
		}

		// Update the habit name
		result, err := db.Exec("UPDATE habits SET name = ? WHERE id = ?", req.Name, req.ID)
		if err != nil {
			log.Printf("UpdateHabitNameHandler: Error updating habit name: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error updating habit name",
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("UpdateHabitNameHandler: Update successful, rows affected: %d", rowsAffected)

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit name updated successfully",
		})
	}
}
