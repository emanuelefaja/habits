package models

import (
	"database/sql"
	"time"
)

type Commit struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Date         time.Time `json:"date"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	FilesAdded   int       `json:"filesAdded"`
	FilesRemoved int       `json:"filesRemoved"`
	CreatedAt    time.Time `json:"createdAt"`
}

func CreateCommitTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS commits (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT,
        date TIMESTAMP NOT NULL,
        additions INTEGER NOT NULL,
        deletions INTEGER NOT NULL,
        files_added INTEGER NOT NULL,
        files_removed INTEGER NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`
	_, err := db.Exec(query)
	return err
}

func SaveCommit(db *sql.DB, commit *Commit) error {
	query := `
    INSERT INTO commits (id, title, description, date, additions, deletions, files_added, files_removed)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
        title = excluded.title,
        description = excluded.description,
        additions = excluded.additions,
        deletions = excluded.deletions,
        files_added = excluded.files_added,
        files_removed = excluded.files_removed`

	_, err := db.Exec(query, commit.ID, commit.Title, commit.Description,
		commit.Date, commit.Additions, commit.Deletions,
		commit.FilesAdded, commit.FilesRemoved)
	return err
}

func GetCommits(db *sql.DB) ([]Commit, error) {
	query := `SELECT * FROM commits ORDER BY date DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.Date,
			&c.Additions, &c.Deletions, &c.FilesAdded, &c.FilesRemoved,
			&c.CreatedAt)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func GetMostRecentCommit(db *sql.DB) (*Commit, error) {
	query := `SELECT id, title, description, date, additions, deletions, 
			  files_added, files_removed, created_at 
			  FROM commits 
			  ORDER BY date DESC 
			  LIMIT 1`

	var commit Commit
	err := db.QueryRow(query).Scan(
		&commit.ID, &commit.Title, &commit.Description, &commit.Date,
		&commit.Additions, &commit.Deletions, &commit.FilesAdded,
		&commit.FilesRemoved, &commit.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &commit, nil
}
