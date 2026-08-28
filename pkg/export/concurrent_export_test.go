package export

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"simple_html_docgen/pkg/document"
	"simple_html_docgen/pkg/storage"
)

// TestConcurrentExportsDoNotCollide is the regression test for exports of one
// document clobbering each other. Every export wrote its intermediate HTML to
// the same <docdir>/temp_export.html and removed it on the way out, so two
// exports issued in the same turn (a model asking for PDF and DOCX together)
// raced: one deleted or overwrote the file the other was still converting. The
// PDF route then failed and fell back to pandoc, which reported "xelatex not
// found" — a message describing the fallback, not the real failure.
func TestConcurrentExportsDoNotCollide(t *testing.T) {
	root := t.TempDir()
	svc := document.NewService(storage.NewStorage(root))

	doc, err := svc.CreateDocument("collision doc", "<html><body><h1>Resume</h1><p>content</p></body></html>")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	e := NewExporter()

	type result struct {
		format string
		path   string
		err    error
	}
	results := make([]result, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p, err := e.ExportDocument(doc.ID, "pdf", filepath.Join(root, "out", "doc.pdf"), svc)
		results[0] = result{"pdf", p, err}
	}()
	go func() {
		defer wg.Done()
		p, err := e.ExportDocument(doc.ID, "docx", filepath.Join(root, "out", "doc.docx"), svc)
		results[1] = result{"docx", p, err}
	}()
	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			t.Errorf("%s export failed: %v", r.format, r.err)
			continue
		}
		fi, err := os.Stat(r.path)
		if err != nil {
			t.Errorf("%s export reported success but %s is missing: %v", r.format, r.path, err)
			continue
		}
		if fi.Size() == 0 {
			t.Errorf("%s export wrote an empty file", r.format)
		}
	}
}
