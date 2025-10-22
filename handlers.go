package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleUpload handles file upload and initiates processing
func HandleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	if ext != ".pdf" && ext != ".docx" && ext != ".txt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type. Use PDF, DOCX, or TXT"})
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Upload to storage
	fileURL, err := UploadFile(src, filename, file.Header.Get("Content-Type"), file.Size)
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Create task
	task := NewTask("document_parse")
	task.FileURL = fileURL

	if err := SaveTask(task); err != nil {
		log.Printf("Failed to save task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Process asynchronously
	go ProcessDocument(task.ID, fileURL, filename)

	c.JSON(http.StatusOK, gin.H{
		"task_id":  task.ID,
		"file_url": fileURL,
		"status":   "pending",
	})
}

// GetTaskStatus returns the status of a task
func GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetProfile returns an industry profile
func GetProfile(c *gin.Context) {
	profileID := c.Param("profile_id")

	profile, err := GetProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// GetMatches returns all matches for a profile
func GetMatches(c *gin.Context) {
	profileID := c.Param("profile_id")

	matches, err := GetMatchesByProfile(profileID)
	if err != nil {
		log.Printf("Failed to get matches: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve matches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profile_id": profileID,
		"matches":    matches,
	})
}

// ConfirmMatch confirms a match recommendation
func ConfirmMatch(c *gin.Context) {
	matchID := c.Param("match_id")

	if err := UpdateMatchConfirmation(matchID); err != nil {
		log.Printf("Failed to confirm match: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm match"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"match_id":  matchID,
		"confirmed": true,
		"message":   "Match confirmed successfully",
	})
}

// ListProfiles returns all industry profiles
func ListProfiles(c *gin.Context) {
	profiles, err := ListAllProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve profiles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(profiles),
		"profiles": profiles,
	})
}