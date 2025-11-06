package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database
	if err := InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize storage
	if err := InitStorage(); err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Initialize MCP client
	if err := InitMCPClient(); err != nil {
		log.Fatal("Failed to initialize MCP client:", err)
	}

	// Setup router
	r := gin.Default()

	// Configure CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Upload document
		api.POST("/upload", HandleUpload)

		// Get task status
		api.GET("/tasks/:task_id", GetTaskStatus)

		// Get industry profile
		api.GET("/profiles/:profile_id", GetProfileHandler)

		// Get matches for a profile
		api.GET("/profiles/:profile_id/matches", GetMatches)

		// Confirm match
		api.POST("/matches/:match_id/confirm", ConfirmMatch)

		// List all profiles
		api.GET("/profiles", ListProfiles)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
