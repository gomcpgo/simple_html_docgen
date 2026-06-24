package document

import (
	"fmt"
	"strings"
	"time"
)

// Service provides document operations
type Service struct {
	storage StorageInterface
}

// StorageInterface defines the storage operations needed by the service
type StorageInterface interface {
	DocumentExists(documentID string) bool
	CreateDocument(doc *Document) error
	UpdateDocument(doc *Document) error
	GetDocument(documentID string) (*Document, error)
	ListDocuments() ([]*DocumentInfo, error)
	CopyMediaFile(documentID, sourcePath string) (string, error)
	DeleteDocument(documentID string) error
	GetDocumentPath(documentID string) string
	GetHTMLPath(documentID string) string
}

// NewService creates a new document service
func NewService(storage StorageInterface) *Service {
	return &Service{
		storage: storage,
	}
}

// CreateDocument creates a new HTML document
func (s *Service) CreateDocument(name, htmlContent string) (*Document, error) {
	if name == "" {
		return nil, fmt.Errorf("document name cannot be empty")
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("HTML content cannot be empty")
	}

	// Generate unique document ID
	documentID := GenerateDocumentID(name, s.storage.DocumentExists)

	now := time.Now()
	doc := &Document{
		ID:          documentID,
		Name:        name,
		HTMLContent: htmlContent,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.storage.CreateDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

// UpdateDocument updates an existing document's HTML content
func (s *Service) UpdateDocument(documentID, htmlContent string) (*Document, error) {
	if !ValidateDocumentID(documentID) {
		return nil, fmt.Errorf("invalid document ID: %s", documentID)
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("HTML content cannot be empty")
	}

	// Get existing document to preserve metadata
	doc, err := s.storage.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Update content and timestamp
	doc.HTMLContent = htmlContent
	doc.UpdatedAt = time.Now()

	if err := s.storage.UpdateDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return doc, nil
}

// AppendHTML appends an HTML fragment to an existing document.
// Inserts before the closing </body> tag if present, otherwise appends at the end.
func (s *Service) AppendHTML(documentID, html string) (*Document, error) {
	if !ValidateDocumentID(documentID) {
		return nil, fmt.Errorf("invalid document ID: %s", documentID)
	}

	if html == "" {
		return nil, fmt.Errorf("html fragment cannot be empty")
	}

	doc, err := s.storage.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if idx := strings.LastIndex(doc.HTMLContent, "</body>"); idx != -1 {
		doc.HTMLContent = doc.HTMLContent[:idx] + html + doc.HTMLContent[idx:]
	} else {
		doc.HTMLContent += html
	}
	doc.UpdatedAt = time.Now()

	if err := s.storage.UpdateDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return doc, nil
}

// ReplaceInDocument replaces a single occurrence of oldStr with newStr in the
// document's HTML. Fails if oldStr is not found or occurs more than once, so the
// edit is always unambiguous.
func (s *Service) ReplaceInDocument(documentID, oldStr, newStr string) (*Document, error) {
	if !ValidateDocumentID(documentID) {
		return nil, fmt.Errorf("invalid document ID: %s", documentID)
	}

	if oldStr == "" {
		return nil, fmt.Errorf("old_str cannot be empty")
	}

	doc, err := s.storage.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	count := strings.Count(doc.HTMLContent, oldStr)
	if count == 0 {
		return nil, fmt.Errorf("old_str not found in document")
	}
	if count > 1 {
		return nil, fmt.Errorf("old_str matches %d times; it must match exactly once (add surrounding context to make it unique)", count)
	}

	doc.HTMLContent = strings.Replace(doc.HTMLContent, oldStr, newStr, 1)
	doc.UpdatedAt = time.Now()

	if err := s.storage.UpdateDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return doc, nil
}

// GetDocument retrieves a document by ID
func (s *Service) GetDocument(documentID string) (*Document, error) {
	if !ValidateDocumentID(documentID) {
		return nil, fmt.Errorf("invalid document ID: %s", documentID)
	}

	doc, err := s.storage.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// ListDocuments returns all documents
func (s *Service) ListDocuments() ([]*DocumentInfo, error) {
	docs, err := s.storage.ListDocuments()
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return docs, nil
}

// AddMedia adds a media file (image/video) to a document
// Returns the relative path to use in HTML
func (s *Service) AddMedia(documentID, sourcePath, mediaType string) (string, error) {
	if !ValidateDocumentID(documentID) {
		return "", fmt.Errorf("invalid document ID: %s", documentID)
	}

	if sourcePath == "" {
		return "", fmt.Errorf("source path cannot be empty")
	}

	// Validate media type
	if mediaType != "image" && mediaType != "video" {
		return "", fmt.Errorf("invalid media type: %s (must be 'image' or 'video')", mediaType)
	}

	// Copy file and get relative path
	relativePath, err := s.storage.CopyMediaFile(documentID, sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to add media: %w", err)
	}

	return relativePath, nil
}

// DeleteDocument deletes a document
func (s *Service) DeleteDocument(documentID string) error {
	if !ValidateDocumentID(documentID) {
		return fmt.Errorf("invalid document ID: %s", documentID)
	}

	if err := s.storage.DeleteDocument(documentID); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetDocumentPath returns the absolute path to the document directory
func (s *Service) GetDocumentPath(documentID string) string {
	return s.storage.GetDocumentPath(documentID)
}

// GetHTMLPath returns the absolute path to the HTML file
func (s *Service) GetHTMLPath(documentID string) string {
	return s.storage.GetHTMLPath(documentID)
}
