package main

import (
	"time"

	"github.com/google/uuid"
)

// Location represents geographical coordinates
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Output represents an output stream from an industry
type Output struct {
	Name     string  `json:"name"`
	State    string  `json:"state"` // solid, liquid, gas
	Quantity string  `json:"quantity"`
	Tags     []string `json:"tags,omitempty"`
}

// IndustryProfile represents a company's I/O profile
type IndustryProfile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  Location  `json:"location"`
	Inputs    []string  `json:"inputs"`
	Outputs   []Output  `json:"outputs"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MatchRecommendation represents a potential symbiotic match
type MatchRecommendation struct {
	ID                     string    `json:"id"`
	WasteID                string    `json:"waste_id"`
	ProducerID             string    `json:"producer_id"`
	CandidateID            string    `json:"candidate_id"`
	ConversionNeeded       bool      `json:"conversion_needed"`
	ConversionDescription  string    `json:"conversion_description,omitempty"`
	RecommendedConverter   string    `json:"recommended_converter"` // producer, consumer, third-party
	Score                  float64   `json:"score"`
	Reasoning              string    `json:"reasoning"`
	EstimatedCost          string    `json:"estimated_cost,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	Confirmed              bool      `json:"confirmed"`
	ConfirmedAt            *time.Time `json:"confirmed_at,omitempty"`
}

// Task represents an asynchronous processing task
type Task struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	Type        string    `json:"type"`   // document_parse, match_generation
	FileURL     string    `json:"file_url,omitempty"`
	ProfileID   string    `json:"profile_id,omitempty"`
	Error       string    `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// MCPToolCall represents a call to an MCP tool
type MCPToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

// MCPToolResponse represents the response from an MCP tool
type MCPToolResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewIndustryProfile creates a new industry profile with generated ID
func NewIndustryProfile(name string, location Location, inputs []string, outputs []Output) *IndustryProfile {
	now := time.Now()
	return &IndustryProfile{
		ID:        uuid.New().String(),
		Name:      name,
		Location:  location,
		Inputs:    inputs,
		Outputs:   outputs,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTask creates a new task
func NewTask(taskType string) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Status:    "pending",
		Type:      taskType,
		CreatedAt: time.Now(),
	}
}

// NewMatchRecommendation creates a new match recommendation
func NewMatchRecommendation(wasteID, producerID, candidateID string) *MatchRecommendation {
	return &MatchRecommendation{
		ID:          uuid.New().String(),
		WasteID:     wasteID,
		ProducerID:  producerID,
		CandidateID: candidateID,
		CreatedAt:   time.Now(),
		Confirmed:   false,
	}
}