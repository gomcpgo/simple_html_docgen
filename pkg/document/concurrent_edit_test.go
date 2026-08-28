package document

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// copyingStorage models the real file-backed storage: GetDocument returns a
// snapshot (a fresh struct, as reading index.html does) and UpdateDocument
// writes the whole document back. The read delay widens the window between the
// read and the write so a lost update is reproduced deterministically rather
// than depending on goroutine scheduling.
type copyingStorage struct {
	mu        sync.Mutex
	docs      map[string]Document
	readDelay time.Duration
}

func newCopyingStorage(readDelay time.Duration) *copyingStorage {
	return &copyingStorage{docs: map[string]Document{}, readDelay: readDelay}
}

func (c *copyingStorage) DocumentExists(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.docs[id]
	return ok
}

func (c *copyingStorage) CreateDocument(doc *Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.docs[doc.ID] = *doc
	return nil
}

func (c *copyingStorage) UpdateDocument(doc *Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.docs[doc.ID]; !ok {
		return fmt.Errorf("document %s does not exist", doc.ID)
	}
	c.docs[doc.ID] = *doc
	return nil
}

func (c *copyingStorage) GetDocument(id string) (*Document, error) {
	c.mu.Lock()
	doc, ok := c.docs[id]
	c.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("document %s does not exist", id)
	}
	time.Sleep(c.readDelay)
	snapshot := doc
	return &snapshot, nil
}

func (c *copyingStorage) ListDocuments() ([]*DocumentInfo, error)      { return nil, nil }
func (c *copyingStorage) CopyMediaFile(id, src string) (string, error) { return "", nil }
func (c *copyingStorage) DeleteDocument(id string) error               { return nil }
func (c *copyingStorage) GetDocumentPath(id string) string             { return id }
func (c *copyingStorage) GetHTMLPath(id string) string                 { return id + "/index.html" }

// TestReplaceInDocument_ConcurrentEditsAllLand is the regression test for the
// lost-update bug: a model that issues several replace_in_document calls in one
// turn had them run concurrently, each reading the original and writing its own
// whole-file result. Every call returned success and only the last write
// survived. All edits must land.
func TestReplaceInDocument_ConcurrentEditsAllLand(t *testing.T) {
	svc := NewService(newCopyingStorage(20 * time.Millisecond))

	const body = `<html><body><p>alpha</p><p>bravo</p><p>charlie</p><p>delta</p><p>echo</p></body></html>`
	doc, err := svc.CreateDocument("resume", body)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	edits := []struct{ old, new string }{
		{"alpha", "ALPHA"},
		{"bravo", "BRAVO"},
		{"charlie", "CHARLIE"},
		{"delta", "DELTA"},
		{"echo", "ECHO"},
	}

	var wg sync.WaitGroup
	errs := make([]error, len(edits))
	for i, e := range edits {
		wg.Add(1)
		go func(i int, oldStr, newStr string) {
			defer wg.Done()
			_, errs[i] = svc.ReplaceInDocument(doc.ID, oldStr, newStr)
		}(i, e.old, e.new)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("edit %d (%s) returned an error: %v", i, edits[i].old, err)
		}
	}

	final, err := svc.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var lost []string
	for _, e := range edits {
		if !strings.Contains(final.HTMLContent, e.new) {
			lost = append(lost, e.old)
		}
	}
	if len(lost) > 0 {
		t.Errorf("%d of %d edits reported success but are not in the document (lost: %v)\nfinal content: %s",
			len(lost), len(edits), lost, final.HTMLContent)
	}
}

// TestAppendHTML_ConcurrentAppendsAllLand covers the same read-modify-write
// shape in AppendHTML, which a model can also call several times in one turn.
func TestAppendHTML_ConcurrentAppendsAllLand(t *testing.T) {
	svc := NewService(newCopyingStorage(20 * time.Millisecond))

	doc, err := svc.CreateDocument("notes", "<html><body></body></html>")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	fragments := []string{"<p>one</p>", "<p>two</p>", "<p>three</p>", "<p>four</p>"}

	var wg sync.WaitGroup
	for _, f := range fragments {
		wg.Add(1)
		go func(frag string) {
			defer wg.Done()
			if _, err := svc.AppendHTML(doc.ID, frag); err != nil {
				t.Errorf("append %s: %v", frag, err)
			}
		}(f)
	}
	wg.Wait()

	final, err := svc.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, f := range fragments {
		if !strings.Contains(final.HTMLContent, f) {
			t.Errorf("append %s reported success but is not in the document\nfinal content: %s", f, final.HTMLContent)
		}
	}
}
