package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var uploadDir string

// InitStorage initializes local file storage
func InitStorage() error {
	uploadDir = os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	return nil
}

// UploadFile saves a file to local storage and returns the path
func UploadFile(reader io.Reader, filename string, contentType string, size int64) (string, error) {
	filePath := filepath.Join(uploadDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return absolute path
	absPath, _ := filepath.Abs(filePath)
	return absPath, nil
}

// GetFile retrieves a file from local storage
func GetFile(filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// GeneratePresignedURL returns the file path (not used for local storage)
func GeneratePresignedURL(filePath string) (string, error) {
	return filePath, nil
}

// GetFileExtension returns the file extension from filename
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}