package document

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
)

const (
	// MaxSlugLength is the maximum length of the slugified name portion
	MaxSlugLength = 30
	// SuffixLength is the length of the random suffix
	SuffixLength = 4
)

// GenerateDocumentID creates a unique document ID from a name
// Format: slugified-name-abc1
// Example: "My Report" -> "my-report-a3f9"
func GenerateDocumentID(name string, existsFunc func(string) bool) string {
	// Slugify the name
	slugified := slug.Make(name)

	// Trim if too long
	if len(slugified) > MaxSlugLength {
		slugified = slugified[:MaxSlugLength]
	}

	// Handle empty slug
	if slugified == "" {
		slugified = "document"
	}

	// Generate unique ID with random suffix
	for i := 0; i < 100; i++ { // Try up to 100 times
		suffix := generateRandomSuffix()
		id := fmt.Sprintf("%s-%s", slugified, suffix)

		if !existsFunc(id) {
			return id
		}
	}

	// Fallback: use longer random suffix if we couldn't find unique ID
	longSuffix := generateRandomSuffix() + generateRandomSuffix()
	return fmt.Sprintf("%s-%s", slugified, longSuffix)
}

// generateRandomSuffix generates a random hex string of specified length
func generateRandomSuffix() string {
	bytes := make([]byte, SuffixLength/2+1) // +1 to ensure we have enough bytes
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to simple method if crypto/rand fails
		return fmt.Sprintf("%04x", len(bytes))
	}
	return hex.EncodeToString(bytes)[:SuffixLength]
}

// ValidateDocumentID checks if a document ID is valid
func ValidateDocumentID(id string) bool {
	if id == "" {
		return false
	}

	// Should contain at least one hyphen
	if !strings.Contains(id, "-") {
		return false
	}

	// Should not be too long
	if len(id) > MaxSlugLength+SuffixLength+1 {
		return false
	}

	return true
}
