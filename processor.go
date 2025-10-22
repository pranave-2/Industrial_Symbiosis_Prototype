package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ProcessDocument handles the async document processing pipeline
func ProcessDocument(taskID, fileURL, filename string) {
	log.Printf("Starting document processing for task %s", taskID)

	// Update task status
	task, _ := GetTask(taskID)
	task.Status = "processing"
	SaveTask(task)

	// Call Python worker for document parsing
	profile, err := callPythonWorker(fileURL, filename)
	if err != nil {
		log.Printf("Document processing failed: %v", err)
		task.Status = "failed"
		task.Error = err.Error()
		now := time.Now()
		task.CompletedAt = &now
		SaveTask(task)
		return
	}

	// Save profile to database
	if err := SaveProfile(profile); err != nil {
		log.Printf("Failed to save profile: %v", err)
		task.Status = "failed"
		task.Error = "Failed to save profile"
		now := time.Now()
		task.CompletedAt = &now
		SaveTask(task)
		return
	}

	// Generate matches asynchronously
	go GenerateMatches(profile.ID)

	// Update task as completed
	task.Status = "completed"
	task.ProfileID = profile.ID
	task.Result = map[string]interface{}{
		"profile_id": profile.ID,
		"name":       profile.Name,
	}
	now := time.Now()
	task.CompletedAt = &now
	SaveTask(task)

	log.Printf("Document processing completed for task %s, profile %s", taskID, profile.ID)
}

// callPythonWorker sends the file to Python worker for parsing
func callPythonWorker(fileURL, filename string) (*IndustryProfile, error) {
	workerURL := os.Getenv("PYTHON_WORKER_URL")
	if workerURL == "" {
		workerURL = "http://localhost:5000"
	}

	requestBody := map[string]string{
		"file_url":  fileURL,
		"filename":  filename,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(workerURL+"/parse", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Python worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Python worker error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Profile IndustryProfile `json:"profile"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.Profile, nil
}

// GenerateMatches generates match recommendations for a profile
func GenerateMatches(profileID string) {
	log.Printf("Generating matches for profile %s", profileID)

	profile, err := GetProfile(profileID)
	if err != nil {
		log.Printf("Failed to get profile: %v", err)
		return
	}

	// Get all other profiles as potential candidates
	allProfiles, err := ListAllProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		return
	}

	// Filter out the current profile
	var candidates []*IndustryProfile
	for _, p := range allProfiles {
		if p.ID != profileID {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		log.Printf("No candidate profiles found for matching")
		return
	}

	// Process each output/waste stream
	for _, output := range profile.Outputs {
		log.Printf("Processing waste stream: %s", output.Name)

		// Classify waste using MCP
		classification, err := mcpClient.ClassifyWaste(output.Name, output.State)
		if err != nil {
			log.Printf("Failed to classify waste: %v", err)
			continue
		}

		// Find potential matches
		matchingNames, err := mcpClient.FindMatches(output, candidates)
		if err != nil {
			log.Printf("Failed to find matches: %v", err)
			continue
		}

		// Process each matching candidate
		for _, candidate := range candidates {
			// Check if this candidate is in the matching list
			isMatch := false
			for _, name := range matchingNames {
				if name == candidate.Name {
					isMatch = true
					break
				}
			}

			if !isMatch {
				continue
			}

			// Estimate conversion requirements
			conversionInfo, err := mcpClient.EstimateConversion(output, candidate.Name)
			if err != nil {
				log.Printf("Failed to estimate conversion: %v", err)
				continue
			}

			// Generate reasoning
			reasoning, err := mcpClient.ExplainMatch(output, candidate, conversionInfo)
			if err != nil {
				log.Printf("Failed to generate reasoning: %v", err)
				reasoning = "Match identified based on input/output compatibility"
			}

			// Calculate score based on multiple factors
			score := calculateMatchScore(profile, candidate, output, classification, conversionInfo)

			// Create match recommendation
			match := NewMatchRecommendation(output.Name, profileID, candidate.ID)
			match.ConversionNeeded = getBool(conversionInfo, "conversion_needed", false)
			match.ConversionDescription = getString(conversionInfo, "description", "")
			match.RecommendedConverter = getString(conversionInfo, "recommended_converter", "producer")
			match.EstimatedCost = getString(conversionInfo, "estimated_cost", "Unknown")
			match.Score = score
			match.Reasoning = reasoning

			// Save match
			if err := SaveMatch(match); err != nil {
				log.Printf("Failed to save match: %v", err)
			} else {
				log.Printf("Created match: %s -> %s (score: %.2f)", profile.Name, candidate.Name, score)
			}
		}
	}

	log.Printf("Match generation completed for profile %s", profileID)
}

// calculateMatchScore calculates a score for a match based on various factors
func calculateMatchScore(producer, consumer *IndustryProfile, waste Output, classification, conversionInfo map[string]interface{}) float64 {
	score := 0.5 // Base score

	// Bonus for no conversion needed
	if !getBool(conversionInfo, "conversion_needed", false) {
		score += 0.2
	}

	// Bonus for low complexity conversion
	complexity := getString(conversionInfo, "complexity", "unknown")
	switch complexity {
	case "low":
		score += 0.15
	case "medium":
		score += 0.05
	case "high":
		score -= 0.1
	}

	// Bonus for geographic proximity (simplified - within 100km)
	distance := calculateDistance(producer.Location, consumer.Location)
	if distance < 100 {
		score += 0.15
	} else if distance < 500 {
		score += 0.05
	}

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// calculateDistance calculates distance between two locations (simplified)
func calculateDistance(loc1, loc2 Location) float64 {
	// Haversine formula (simplified for demo)
	dlat := loc2.Lat - loc1.Lat
	dlng := loc2.Lng - loc1.Lng
	return (dlat*dlat + dlng*dlng) * 111.0 // Very rough approximation in km
}

// Helper functions to extract values from maps
func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultVal
}

func getString(m map[string]interface{}, key string, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}