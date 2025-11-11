package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the configuration for the Simple HTML Document Generator
type Config struct {
	RootDir string // Root directory for storing HTML documents
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	rootDir := os.Getenv("SIMPLE_HTML_ROOT_DIR")
	if rootDir == "" {
		// Default to a reasonable location if not set
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		rootDir = filepath.Join(homeDir, ".simple_html_docs")
	}

	// Ensure root directory exists
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory %s: %w", rootDir, err)
	}

	return &Config{
		RootDir: rootDir,
	}, nil
}
