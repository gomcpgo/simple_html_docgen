package document

import "time"

// Document represents an HTML document
type Document struct {
	ID          string    `json:"id"`           // Unique identifier (e.g., "my-report-a3f9")
	Name        string    `json:"name"`         // Human-readable name
	HTMLContent string    `json:"html_content"` // Full HTML content
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Metadata represents document metadata stored in metadata.json
type Metadata struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DocumentInfo is a lightweight document summary for listing
type DocumentInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	FilePath  string    `json:"file_path"` // Relative path to index.html
}
