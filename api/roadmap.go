package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"mad/middleware"
	"net/http"
)

type LikeResponse struct {
	CardID     string `json:"cardId"`
	TotalLikes int    `json:"totalLikes"`
	UserLiked  bool   `json:"userLiked"`
}

// GetRoadmapLikesHandler returns a handler for getting roadmap likes
func GetRoadmapLikesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("GetRoadmapLikes: Received request")

		userID := middleware.GetUserID(r)
		log.Printf("GetRoadmapLikes: User ID: %d", userID)

		rows, err := db.Query(`
			SELECT 
				card_id,
				COUNT(DISTINCT user_id) as total_likes,
				COUNT(CASE WHEN user_id = ? THEN 1 END) > 0 as user_liked
			FROM roadmap_likes
			GROUP BY card_id`, userID)
		if err != nil {
			log.Printf("GetRoadmapLikes: Database query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		likes := make([]LikeResponse, 0)
		for rows.Next() {
			var like LikeResponse
			err := rows.Scan(&like.CardID, &like.TotalLikes, &like.UserLiked)
			if err != nil {
				log.Printf("GetRoadmapLikes: Row scan error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			likes = append(likes, like)
		}

		log.Printf("GetRoadmapLikes: Returning %d likes", len(likes))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(likes)
	}
}

// ToggleRoadmapLikeHandler returns a handler for toggling roadmap likes
func ToggleRoadmapLikeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ToggleRoadmapLike: Received request")

		userID := middleware.GetUserID(r)
		log.Printf("ToggleRoadmapLike: User ID: %d", userID)

		var req struct {
			CardID string `json:"cardId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("ToggleRoadmapLike: Failed to decode request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		log.Printf("ToggleRoadmapLike: Card ID: %s", req.CardID)

		// Ensure user is authenticated
		if userID == 0 {
			http.Error(w, "Must be logged in to like features", http.StatusUnauthorized)
			return
		}

		// Try to delete existing like first
		result, err := db.Exec(`
			DELETE FROM roadmap_likes 
			WHERE user_id = ? AND card_id = ?`,
			userID, req.CardID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// If no row was deleted, insert new like
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			_, err = db.Exec(`
				INSERT INTO roadmap_likes (user_id, card_id)
				VALUES (?, ?)`,
				userID, req.CardID)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		// Get updated like count and status
		var response LikeResponse
		err = db.QueryRow(`
			SELECT 
				?,
				COUNT(DISTINCT user_id) as total_likes,
				COUNT(CASE WHEN user_id = ? THEN 1 END) > 0 as user_liked
			FROM roadmap_likes
			WHERE card_id = ?
			GROUP BY card_id`,
			req.CardID, userID, req.CardID).Scan(
			&response.CardID,
			&response.TotalLikes,
			&response.UserLiked,
		)
		if err == sql.ErrNoRows {
			// If no likes exist, return zero counts
			response = LikeResponse{
				CardID:     req.CardID,
				TotalLikes: 0,
				UserLiked:  false,
			}
		} else if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}