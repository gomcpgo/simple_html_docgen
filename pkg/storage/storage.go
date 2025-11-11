package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"simple_html_docgen/pkg/document"
)

// Storage handles file operations for HTML documents
type Storage struct {
	rootDir string
}

// NewStorage creates a new Storage instance
func NewStorage(rootDir string) *Storage {
	return &Storage{
		rootDir: rootDir,
	}
}

// GetDocumentPath returns the directory path for a document
func (s *Storage) GetDocumentPath(documentID string) string {
	return filepath.Join(s.rootDir, documentID)
}

// GetHTMLPath returns the path to the index.html file
func (s *Storage) GetHTMLPath(documentID string) string {
	return filepath.Join(s.GetDocumentPath(documentID), "index.html")
}

// GetMetadataPath returns the path to the metadata.json file
func (s *Storage) GetMetadataPath(documentID string) string {
	return filepath.Join(s.GetDocumentPath(documentID), "metadata.json")
}

// GetMediaDir returns the path to the media directory
func (s *Storage) GetMediaDir(documentID string) string {
	return filepath.Join(s.GetDocumentPath(documentID), "media")
}

// DocumentExists checks if a document exists
func (s *Storage) DocumentExists(documentID string) bool {
	htmlPath := s.GetHTMLPath(documentID)
	_, err := os.Stat(htmlPath)
	return err == nil
}

// CreateDocument creates a new document on disk
func (s *Storage) CreateDocument(doc *document.Document) error {
	// Create document directory
	docPath := s.GetDocumentPath(doc.ID)
	if err := os.MkdirAll(docPath, 0755); err != nil {
		return fmt.Errorf("failed to create document directory: %w", err)
	}

	// Create media directory
	mediaDir := s.GetMediaDir(doc.ID)
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return fmt.Errorf("failed to create media directory: %w", err)
	}

	// Write HTML content
	htmlPath := s.GetHTMLPath(doc.ID)
	if err := os.WriteFile(htmlPath, []byte(doc.HTMLContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Write metadata
	metadata := document.Metadata{
		Name:      doc.Name,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
	if err := s.WriteMetadata(doc.ID, &metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// UpdateDocument updates an existing document's HTML content
func (s *Storage) UpdateDocument(doc *document.Document) error {
	if !s.DocumentExists(doc.ID) {
		return fmt.Errorf("document %s does not exist", doc.ID)
	}

	// Write HTML content
	htmlPath := s.GetHTMLPath(doc.ID)
	if err := os.WriteFile(htmlPath, []byte(doc.HTMLContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Update metadata
	metadata := document.Metadata{
		Name:      doc.Name,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
	if err := s.WriteMetadata(doc.ID, &metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// GetDocument retrieves a document from disk
func (s *Storage) GetDocument(documentID string) (*document.Document, error) {
	if !s.DocumentExists(documentID) {
		return nil, fmt.Errorf("document %s does not exist", documentID)
	}

	// Read HTML content
	htmlPath := s.GetHTMLPath(documentID)
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Read metadata
	metadata, err := s.ReadMetadata(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	return &document.Document{
		ID:          documentID,
		Name:        metadata.Name,
		HTMLContent: string(htmlBytes),
		CreatedAt:   metadata.CreatedAt,
		UpdatedAt:   metadata.UpdatedAt,
	}, nil
}

// WriteMetadata writes metadata to disk
func (s *Storage) WriteMetadata(documentID string, metadata *document.Metadata) error {
	metadataPath := s.GetMetadataPath(documentID)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// ReadMetadata reads metadata from disk
func (s *Storage) ReadMetadata(documentID string) (*document.Metadata, error) {
	metadataPath := s.GetMetadataPath(documentID)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata document.Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// ListDocuments returns all documents
func (s *Storage) ListDocuments() ([]*document.DocumentInfo, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory: %w", err)
	}

	var docs []*document.DocumentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		documentID := entry.Name()
		if !s.DocumentExists(documentID) {
			continue
		}

		metadata, err := s.ReadMetadata(documentID)
		if err != nil {
			// Skip documents with invalid metadata
			continue
		}

		docs = append(docs, &document.DocumentInfo{
			ID:        documentID,
			Name:      metadata.Name,
			CreatedAt: metadata.CreatedAt,
			UpdatedAt: metadata.UpdatedAt,
			FilePath:  filepath.Join(documentID, "index.html"),
		})
	}

	return docs, nil
}

// CopyMediaFile copies a media file to the document's media directory
// Returns the relative path to the media file
func (s *Storage) CopyMediaFile(documentID, sourcePath string) (string, error) {
	if !s.DocumentExists(documentID) {
		return "", fmt.Errorf("document %s does not exist", documentID)
	}

	// Open source file
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get filename
	filename := filepath.Base(sourcePath)

	// Create destination path
	mediaDir := s.GetMediaDir(documentID)
	destPath := filepath.Join(mediaDir, filename)

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Return relative path from document root
	relativePath := filepath.Join("media", filename)
	return relativePath, nil
}

// DeleteDocument deletes a document and all its files
func (s *Storage) DeleteDocument(documentID string) error {
	if !s.DocumentExists(documentID) {
		return fmt.Errorf("document %s does not exist", documentID)
	}

	docPath := s.GetDocumentPath(documentID)
	if err := os.RemoveAll(docPath); err != nil {
		return fmt.Errorf("failed to delete document directory: %w", err)
	}

	return nil
}
