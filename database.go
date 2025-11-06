package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// InitDB initializes the database connection
func InitDB() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=postgres dbname=industrial_symbiosis sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	// Create tables
	if err = createTables(); err != nil {
		return err
	}

	return nil
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS industry_profiles (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		location JSONB NOT NULL,
		inputs JSONB NOT NULL,
		outputs JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS match_recommendations (
		id VARCHAR(36) PRIMARY KEY,
		waste_id VARCHAR(255) NOT NULL,
		producer_id VARCHAR(36) NOT NULL,
		candidate_id VARCHAR(36) NOT NULL,
		conversion_needed BOOLEAN NOT NULL,
		conversion_description TEXT,
		recommended_converter VARCHAR(50),
		score FLOAT NOT NULL,
		reasoning TEXT,
		estimated_cost TEXT,
		created_at TIMESTAMP NOT NULL,
		confirmed BOOLEAN DEFAULT FALSE,
		confirmed_at TIMESTAMP,
		FOREIGN KEY (producer_id) REFERENCES industry_profiles(id),
		FOREIGN KEY (candidate_id) REFERENCES industry_profiles(id)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id VARCHAR(36) PRIMARY KEY,
		status VARCHAR(50) NOT NULL,
		type VARCHAR(50) NOT NULL,
		file_url TEXT,
		profile_id VARCHAR(36),
		error TEXT,
		result JSONB,
		created_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES industry_profiles(id)
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_matches_producer ON match_recommendations(producer_id);
	CREATE INDEX IF NOT EXISTS idx_matches_candidate ON match_recommendations(candidate_id);
	`

	_, err := db.Exec(schema)
	return err
}

// SaveProfile saves an industry profile to the database
func SaveProfile(profile *IndustryProfile) error {
	locationJSON, _ := json.Marshal(profile.Location)
	inputsJSON, _ := json.Marshal(profile.Inputs)
	outputsJSON, _ := json.Marshal(profile.Outputs)

	query := `
		INSERT INTO industry_profiles (id, name, location, inputs, outputs, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = $2, location = $3, inputs = $4, outputs = $5, updated_at = $7
	`

	_, err := db.Exec(query, profile.ID, profile.Name, locationJSON, inputsJSON, outputsJSON, profile.CreatedAt, profile.UpdatedAt)
	return err
}

// GetProfile retrieves a profile by ID
func GetProfile(id string) (*IndustryProfile, error) {
	query := `SELECT id, name, location, inputs, outputs, created_at, updated_at FROM industry_profiles WHERE id = $1`

	var profile IndustryProfile
	var locationJSON, inputsJSON, outputsJSON []byte

	err := db.QueryRow(query, id).Scan(&profile.ID, &profile.Name, &locationJSON, &inputsJSON, &outputsJSON, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(locationJSON, &profile.Location)
	json.Unmarshal(inputsJSON, &profile.Inputs)
	json.Unmarshal(outputsJSON, &profile.Outputs)

	return &profile, nil
}

// ListAllProfiles retrieves all profiles
func ListAllProfiles() ([]*IndustryProfile, error) {
	query := `SELECT id, name, location, inputs, outputs, created_at, updated_at FROM industry_profiles ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*IndustryProfile
	for rows.Next() {
		var profile IndustryProfile
		var locationJSON, inputsJSON, outputsJSON []byte

		err := rows.Scan(&profile.ID, &profile.Name, &locationJSON, &inputsJSON, &outputsJSON, &profile.CreatedAt, &profile.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(locationJSON, &profile.Location)
		json.Unmarshal(inputsJSON, &profile.Inputs)
		json.Unmarshal(outputsJSON, &profile.Outputs)

		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

// SaveMatch saves a match recommendation
func SaveMatch(match *MatchRecommendation) error {
	query := `
		INSERT INTO match_recommendations 
		(id, waste_id, producer_id, candidate_id, conversion_needed, conversion_description, 
		 recommended_converter, score, reasoning, estimated_cost, created_at, confirmed, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := db.Exec(query, match.ID, match.WasteID, match.ProducerID, match.CandidateID,
		match.ConversionNeeded, match.ConversionDescription, match.RecommendedConverter,
		match.Score, match.Reasoning, match.EstimatedCost, match.CreatedAt, match.Confirmed, match.ConfirmedAt)
	return err
}

// GetMatchesByProfile retrieves all matches for a profile
func GetMatchesByProfile(profileID string) ([]*MatchRecommendation, error) {
	query := `
		SELECT id, waste_id, producer_id, candidate_id, conversion_needed, conversion_description,
		       recommended_converter, score, reasoning, estimated_cost, created_at, confirmed, confirmed_at
		FROM match_recommendations 
		WHERE producer_id = $1 
		ORDER BY score DESC
	`

	rows, err := db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*MatchRecommendation
	for rows.Next() {
		var match MatchRecommendation
		err := rows.Scan(&match.ID, &match.WasteID, &match.ProducerID, &match.CandidateID,
			&match.ConversionNeeded, &match.ConversionDescription, &match.RecommendedConverter,
			&match.Score, &match.Reasoning, &match.EstimatedCost, &match.CreatedAt,
			&match.Confirmed, &match.ConfirmedAt)
		if err != nil {
			continue
		}
		matches = append(matches, &match)
	}

	return matches, nil
}

// UpdateMatchConfirmation updates the confirmation status of a match
func UpdateMatchConfirmation(matchID string) error {
	now := time.Now()
	query := `UPDATE match_recommendations SET confirmed = TRUE, confirmed_at = $1 WHERE id = $2`
	_, err := db.Exec(query, now, matchID)
	return err
}

// SaveTask saves a task
func SaveTask(task *Task) error {
	resultJSON, _ := json.Marshal(task.Result)

	query := `
		INSERT INTO tasks (id, status, type, file_url, profile_id, error, result, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			status = $2, error = $6, result = $7, completed_at = $9
	`

	_, err := db.Exec(query, task.ID, task.Status, task.Type, task.FileURL, task.ProfileID,
		task.Error, resultJSON, task.CreatedAt, task.CompletedAt)
	return err
}

// GetTask retrieves a task by ID
func GetTask(id string) (*Task, error) {
	query := `SELECT id, status, type, file_url, profile_id, error, result, created_at, completed_at FROM tasks WHERE id = $1`

	var task Task
	var resultJSON []byte
	var fileURL, profileID, errorMsg sql.NullString
	var completedAt sql.NullTime

	err := db.QueryRow(query, id).Scan(&task.ID, &task.Status, &task.Type, &fileURL, &profileID,
		&errorMsg, &resultJSON, &task.CreatedAt, &completedAt)
	if err != nil {
		return nil, err
	}

	if fileURL.Valid {
		task.FileURL = fileURL.String
	}
	if profileID.Valid {
		task.ProfileID = profileID.String
	}
	if errorMsg.Valid {
		task.Error = errorMsg.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if len(resultJSON) > 0 {
		json.Unmarshal(resultJSON, &task.Result)
	}

	return &task, nil
}
