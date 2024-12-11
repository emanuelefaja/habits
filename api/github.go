package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mad/models"
	"net/http"
	"os"
	"strings"
	"time"
)

type GitHubSyncer struct {
	db    *sql.DB
	token string
	owner string
	repo  string
}

type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Stats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
	Files []struct {
		Status string `json:"status"`
	} `json:"files"`
}

func NewGitHubSyncer(db *sql.DB) *GitHubSyncer {
	return &GitHubSyncer{
		db:    db,
		token: os.Getenv("GITHUB_TOKEN"),
		owner: os.Getenv("GITHUB_OWNER"),
		repo:  os.Getenv("GITHUB_REPO"),
	}
}

func (g *GitHubSyncer) fetchFromGitHub() ([]*models.Commit, error) {
	// Get the most recent commit from our database
	lastCommit, err := models.GetMostRecentCommit(g.db)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var allCommits []*models.Commit
	page := 1
	perPage := 100 // GitHub's max is 100 per page

	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?page=%d&per_page=%d",
			g.owner, g.repo, page, perPage)

		if lastCommit != nil {
			// If we have a last commit, only get commits after that date
			since := lastCommit.Date.Add(time.Second).Format(time.RFC3339)
			url = fmt.Sprintf("%s&since=%s", url, since)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "token "+g.token)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
		}

		var githubCommits []GitHubCommit
		if err := json.NewDecoder(resp.Body).Decode(&githubCommits); err != nil {
			return nil, err
		}

		// If no commits returned, we've reached the end
		if len(githubCommits) == 0 {
			break
		}

		// Process commits for this page
		for _, gc := range githubCommits {
			message := gc.Commit.Message
			title := message
			description := ""
			if idx := len(message); idx > 0 {
				parts := strings.SplitN(message, "\n\n", 2)
				title = parts[0]
				if len(parts) > 1 {
					description = parts[1]
				}
			}

			date, err := time.Parse(time.RFC3339, gc.Commit.Author.Date)
			if err != nil {
				date = time.Now()
			}

			commit := &models.Commit{
				ID:           gc.SHA[:7],
				Title:        title,
				Description:  description,
				Date:         date,
				Additions:    gc.Stats.Additions,
				Deletions:    gc.Stats.Deletions,
				FilesAdded:   len(gc.Files),
				FilesRemoved: 0,
			}
			allCommits = append(allCommits, commit)
		}

		// Check if we got less than perPage results
		if len(githubCommits) < perPage {
			break
		}

		page++
	}

	return allCommits, nil
}

func (g *GitHubSyncer) SyncCommits() error {
	commits, err := g.fetchFromGitHub()
	if err != nil {
		return err
	}

	for _, commit := range commits {
		err := models.SaveCommit(g.db, commit)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add this function to your main.go
func StartGitHubSync(db *sql.DB) {
	syncer := NewGitHubSyncer(db)

	// Initial sync
	syncer.SyncCommits()

	// Sync every hour
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			syncer.SyncCommits()
		}
	}()
}
