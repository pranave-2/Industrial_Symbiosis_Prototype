package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type MCPClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

var mcpClient *MCPClient

// InitMCPClient initializes the MCP client for Gemini API
func InitMCPClient() error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY not set")
	}

	mcpClient = &MCPClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return nil
}

// ExtractIO calls the MCP tool to extract inputs/outputs from text
func (m *MCPClient) ExtractIO(text string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Extract the following from this industrial company description:
- Company name
- Location (if mentioned, provide lat/lng or city name)
- Input materials/resources (as array)
- Output products/waste streams (as array with name, state, quantity)

Text: %s

Respond with valid JSON only.`, text)

	response, err := m.callGemini(prompt)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If response is not JSON, try to parse it
		result = map[string]interface{}{
			"raw_response": response,
		}
	}

	return result, nil
}

// ClassifyWaste classifies waste type and adds tags
func (m *MCPClient) ClassifyWaste(wasteName, state string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Classify this waste stream and provide relevant tags:
Waste: %s
State: %s

Provide classification, industry tags, and potential uses. Respond with JSON containing:
{
  "waste_type": "category",
  "tags": ["tag1", "tag2"],
  "potential_uses": ["use1", "use2"]
}`, wasteName, state)

	response, err := m.callGemini(prompt)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		result = map[string]interface{}{
			"waste_type":     "unclassified",
			"tags":           []string{},
			"potential_uses": []string{},
		}
	}

	return result, nil
}

// FindMatches finds potential candidate industries for a waste stream
func (m *MCPClient) FindMatches(waste Output, candidates []*IndustryProfile) ([]string, error) {
	candidateNames := make([]string, len(candidates))
	for i, c := range candidates {
		candidateNames[i] = fmt.Sprintf("%s (inputs: %v)", c.Name, c.Inputs)
	}

	prompt := fmt.Sprintf(`Given this waste stream:
Name: %s
State: %s
Quantity: %s

Find which of these industries could use it as input:
%v

Respond with JSON array of matching industry names: ["industry1", "industry2"]`, 
		waste.Name, waste.State, waste.Quantity, candidateNames)

	response, err := m.callGemini(prompt)
	if err != nil {
		return nil, err
	}

	var matches []string
	if err := json.Unmarshal([]byte(response), &matches); err != nil {
		// Return empty if parsing fails
		return []string{}, nil
	}

	return matches, nil
}

// EstimateConversion estimates the conversion process needed
func (m *MCPClient) EstimateConversion(waste Output, candidateInput string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Determine if conversion is needed to transform this waste into usable input:
Waste: %s (state: %s, quantity: %s)
Target Input: %s

Respond with JSON:
{
  "conversion_needed": true/false,
  "description": "conversion process description",
  "recommended_converter": "producer/consumer/third-party",
  "estimated_cost": "cost estimate",
  "complexity": "low/medium/high"
}`, waste.Name, waste.State, waste.Quantity, candidateInput)

	response, err := m.callGemini(prompt)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		result = map[string]interface{}{
			"conversion_needed":      false,
			"description":            "Unable to determine",
			"recommended_converter":  "unknown",
			"estimated_cost":         "Unknown",
			"complexity":             "unknown",
		}
	}

	return result, nil
}

// ExplainMatch generates reasoning for why a match is good
func (m *MCPClient) ExplainMatch(waste Output, candidate *IndustryProfile, conversionInfo map[string]interface{}) (string, error) {
	prompt := fmt.Sprintf(`Explain why this is a good industrial symbiosis match:
Producer Waste: %s (%s, %s)
Consumer: %s
Consumer Inputs: %v
Conversion: %v

Provide a clear, concise explanation of the symbiotic benefit.`, 
		waste.Name, waste.State, waste.Quantity, 
		candidate.Name, candidate.Inputs, conversionInfo)

	reasoning, err := m.callGemini(prompt)
	if err != nil {
		return "", err
	}

	return reasoning, nil
}

// callGemini makes an API call to Gemini
func (m *MCPClient) callGemini(prompt string) (string, error) {
	url := fmt.Sprintf("%s/models/gemini-pro:generateContent?key=%s", m.baseURL, m.apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
			"topK":        40,
			"topP":        0.95,
			"maxOutputTokens": 2048,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from Gemini response structure
	if candidates, ok := response["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format from Gemini API")
}

// CallWithRetry calls an MCP tool with retry logic
func (m *MCPClient) CallWithRetry(fn func() (interface{}, error), maxRetries int) (interface{}, error) {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}