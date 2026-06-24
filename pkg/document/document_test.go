package document

import (
	"fmt"
	"strings"
	"testing"
)

// fakeStorage is a minimal in-memory StorageInterface for service tests.
type fakeStorage struct {
	docs map[string]*Document
}

func newFakeStorage() *fakeStorage { return &fakeStorage{docs: map[string]*Document{}} }

func (f *fakeStorage) DocumentExists(id string) bool { _, ok := f.docs[id]; return ok }
func (f *fakeStorage) CreateDocument(doc *Document) error {
	f.docs[doc.ID] = doc
	return nil
}
func (f *fakeStorage) UpdateDocument(doc *Document) error {
	if _, ok := f.docs[doc.ID]; !ok {
		return fmt.Errorf("document %s does not exist", doc.ID)
	}
	f.docs[doc.ID] = doc
	return nil
}
func (f *fakeStorage) GetDocument(id string) (*Document, error) {
	doc, ok := f.docs[id]
	if !ok {
		return nil, fmt.Errorf("document %s does not exist", id)
	}
	return doc, nil
}
func (f *fakeStorage) ListDocuments() ([]*DocumentInfo, error)        { return nil, nil }
func (f *fakeStorage) CopyMediaFile(id, src string) (string, error)   { return "", nil }
func (f *fakeStorage) DeleteDocument(id string) error                 { return nil }
func (f *fakeStorage) GetDocumentPath(id string) string              { return id }
func (f *fakeStorage) GetHTMLPath(id string) string                  { return id + "/index.html" }

func newTestService() *Service { return NewService(newFakeStorage()) }

func TestAppendHTML_InsertsBeforeBody(t *testing.T) {
	svc := newTestService()
	doc, err := svc.CreateDocument("report", "<html><body><p>one</p></body></html>")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.AppendHTML(doc.ID, "<p>two</p>")
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	want := "<html><body><p>one</p><p>two</p></body></html>"
	if updated.HTMLContent != want {
		t.Errorf("got %q, want %q", updated.HTMLContent, want)
	}
}

func TestAppendHTML_NoBodyAppendsAtEnd(t *testing.T) {
	svc := newTestService()
	doc, err := svc.CreateDocument("frag", "<p>one</p>")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.AppendHTML(doc.ID, "<p>two</p>")
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	want := "<p>one</p><p>two</p>"
	if updated.HTMLContent != want {
		t.Errorf("got %q, want %q", updated.HTMLContent, want)
	}
}

func TestAppendHTML_EmptyFragmentErrors(t *testing.T) {
	svc := newTestService()
	doc, _ := svc.CreateDocument("frag", "<p>one</p>")
	if _, err := svc.AppendHTML(doc.ID, ""); err == nil {
		t.Error("expected error for empty html fragment")
	}
}

func TestReplaceInDocument_SingleOccurrence(t *testing.T) {
	svc := newTestService()
	doc, err := svc.CreateDocument("doc", "<p>hello world</p>")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.ReplaceInDocument(doc.ID, "world", "there")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}

	want := "<p>hello there</p>"
	if updated.HTMLContent != want {
		t.Errorf("got %q, want %q", updated.HTMLContent, want)
	}
}

func TestReplaceInDocument_NoMatchErrors(t *testing.T) {
	svc := newTestService()
	doc, _ := svc.CreateDocument("doc", "<p>hello</p>")
	if _, err := svc.ReplaceInDocument(doc.ID, "missing", "x"); err == nil {
		t.Error("expected error when old_str not found")
	}
}

func TestReplaceInDocument_MultipleMatchesErrors(t *testing.T) {
	svc := newTestService()
	doc, _ := svc.CreateDocument("doc", "<p>a</p><p>a</p>")
	_, err := svc.ReplaceInDocument(doc.ID, "<p>a</p>", "<p>b</p>")
	if err == nil {
		t.Error("expected error when old_str matches more than once")
	}
	if err != nil && !strings.Contains(err.Error(), "2") {
		t.Errorf("error should report match count, got: %v", err)
	}
}
