package export

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simple_html_docgen/pkg/document"
	"simple_html_docgen/pkg/storage"
)

func newDocService(t *testing.T) (*document.Service, string) {
	t.Helper()
	root := t.TempDir()
	return document.NewService(storage.NewStorage(root)), root
}

func createDoc(t *testing.T, svc *document.Service, html string) *document.Document {
	t.Helper()
	doc, err := svc.CreateDocument("export test", html)
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	return doc
}

// docxParts opens a .docx file and returns its parts by name.
func docxParts(t *testing.T, path string) map[string][]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("%s is not a zip file: %v", path, err)
	}
	parts := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open part %s: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read part %s: %v", f.Name, err)
		}
		parts[f.Name] = b
	}
	return parts
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range 16 {
		img.Set(i%4, i/4, color.RGBA{200, 30, 30, 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExportDOCXWithoutPandoc(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // nothing to run on PATH
	svc, root := newDocService(t)
	doc := createDoc(t, svc, "<html><body><h1>Quarterly report</h1><p>Revenue grew.</p></body></html>")

	out := filepath.Join(root, "out", "report.docx")
	res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", out, Options{PageSize: "a4"}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if res.Path != out {
		t.Errorf("path = %q, want %q", res.Path, out)
	}
	body := string(docxParts(t, out)["word/document.xml"])
	for _, want := range []string{"Quarterly report", "Revenue grew."} {
		if !strings.Contains(body, want) {
			t.Errorf("document.xml is missing %q", want)
		}
	}
}

func TestExportDOCXDefaultPath(t *testing.T) {
	svc, _ := newDocService(t)
	doc := createDoc(t, svc, "<p>hello</p>")

	res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", "", Options{PageSize: "a4"}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	want := filepath.Join(svc.GetDocumentPath(doc.ID), doc.ID+".docx")
	if res.Path != want {
		t.Errorf("path = %q, want %q", res.Path, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("exported file missing: %v", err)
	}
}

func TestExportDOCXEmbedsMedia(t *testing.T) {
	svc, root := newDocService(t)
	doc := createDoc(t, svc, `<p>Logo:</p><img src="media/logo.png" alt="logo">`)
	writePNG(t, filepath.Join(svc.GetDocumentPath(doc.ID), "media", "logo.png"))

	out := filepath.Join(root, "logo.docx")
	res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", out, Options{PageSize: "a4"}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	var media int
	for name := range docxParts(t, out) {
		if strings.HasPrefix(name, "word/media/") {
			media++
		}
	}
	if media != 1 {
		t.Errorf("found %d images in word/media, want 1", media)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", res.Warnings)
	}
}

// Document HTML is written by a model and is untrusted: the export must not
// read files outside the document's folder or keep script links.
func TestExportDOCXStaysInDocumentFolder(t *testing.T) {
	svc, root := newDocService(t)
	secret := filepath.Join(root, "secret.png")
	writePNG(t, secret)
	html := `<p><a href="javascript:alert(1)">click</a></p>` +
		`<img src="../secret.png"><img src="` + secret + `"><img src="file://` + secret + `">`
	doc := createDoc(t, svc, html)

	out := filepath.Join(root, "escape.docx")
	res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", out, Options{PageSize: "a4"}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	parts := docxParts(t, out)
	for name := range parts {
		if strings.HasPrefix(name, "word/media/") {
			t.Errorf("image from outside the document folder was embedded: %s", name)
		}
	}
	if strings.Contains(string(parts["word/_rels/document.xml.rels"]), "javascript:") {
		t.Error("javascript: link was kept")
	}
	if len(res.Warnings) == 0 {
		t.Error("refused images should be reported as warnings")
	}
}

func TestExportDOCXReturnsWarnings(t *testing.T) {
	svc, root := newDocService(t)
	doc := createDoc(t, svc, `<p>Chart:</p><img src="media/missing.png">`)

	res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", filepath.Join(root, "w.docx"), Options{PageSize: "a4"}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("a missing image should produce a warning")
	}
	for _, w := range res.Warnings {
		if w.Code == "" || w.Message == "" {
			t.Errorf("warning without code or message: %+v", w)
		}
	}
}

func TestExportDOCXPageSize(t *testing.T) {
	tests := []struct {
		size  string
		width string
	}{
		{"a4", `w:w="11906"`},
		{"letter", `w:w="12240"`},
		{"legal", `w:h="20160"`},
		{"LETTER", `w:w="12240"`},
	}
	for _, tt := range tests {
		svc, root := newDocService(t)
		doc := createDoc(t, svc, "<p>page</p>")
		out := filepath.Join(root, "p.docx")
		res, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", out, Options{PageSize: tt.size}, svc)
		if err != nil {
			t.Fatalf("%s: export: %v", tt.size, err)
		}
		if res.PageSize != strings.ToLower(tt.size) {
			t.Errorf("%s: result page size = %q", tt.size, res.PageSize)
		}
		body := string(docxParts(t, out)["word/document.xml"])
		i := strings.Index(body, "<w:pgSz")
		if i < 0 {
			t.Fatalf("%s: no w:pgSz in document.xml", tt.size)
		}
		pgSz := body[i : i+strings.Index(body[i:], ">")]
		if !strings.Contains(pgSz, tt.width) {
			t.Errorf("%s: %s does not contain %s", tt.size, pgSz, tt.width)
		}
	}
}

func TestExportDOCXDefaultPageSizeFromLocale(t *testing.T) {
	svc, root := newDocService(t)
	doc := createDoc(t, svc, "<p>page</p>")
	e := NewExporter()
	e.defaultPageSize = func() string { return "letter" }

	res, err := e.ExportDocument(context.Background(), doc.ID, "docx", filepath.Join(root, "d.docx"), Options{}, svc)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if res.PageSize != "letter" {
		t.Errorf("page size = %q, want letter", res.PageSize)
	}
}

func TestExportRejectsUnknownPageSize(t *testing.T) {
	svc, root := newDocService(t)
	doc := createDoc(t, svc, "<p>page</p>")
	out := filepath.Join(root, "x.docx")
	_, err := NewExporter().ExportDocument(context.Background(), doc.ID, "docx", out, Options{PageSize: "tabloid"}, svc)
	if err == nil || !strings.Contains(err.Error(), "tabloid") {
		t.Fatalf("err = %v, want an error naming the page size", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("a failed export left an output file")
	}
}

// A failed conversion must not leave a half-written file, or replace an
// earlier good export, or leave temp files behind.
func TestExportDOCXFailureLeavesNoFile(t *testing.T) {
	svc, root := newDocService(t)
	doc := createDoc(t, svc, "<p>content</p>")
	outDir := filepath.Join(root, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(outDir, "c.docx")
	if err := os.WriteFile(out, []byte("earlier export"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewExporter().ExportDocument(ctx, doc.ID, "docx", out, Options{PageSize: "a4"}, svc); err == nil {
		t.Fatal("export with a cancelled context succeeded")
	}
	got, err := os.ReadFile(out)
	if err != nil || string(got) != "earlier export" {
		t.Errorf("earlier export was changed: %q, %v", got, err)
	}
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("output folder holds %v, want only c.docx", names)
	}
}

// PDF export uses Chrome only. When Chrome cannot produce the PDF the error
// is Chrome's, and pandoc is never run as a fallback.
func TestExportPDFDoesNotFallBackToPandoc(t *testing.T) {
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "pandoc-ran")
	script := "#!/bin/sh\ntouch " + marker + "\n"
	if err := os.WriteFile(filepath.Join(bin, "pandoc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	svc, root := newDocService(t)
	doc := createDoc(t, svc, "<p>pdf</p>")
	out := filepath.Join(root, "x.pdf")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewExporter().ExportDocument(ctx, doc.ID, "pdf", out, Options{PageSize: "a4"}, svc)
	if err == nil {
		t.Fatal("PDF export with a cancelled context succeeded")
	}
	if strings.Contains(strings.ToLower(err.Error()), "pandoc") || strings.Contains(err.Error(), "fallback") {
		t.Errorf("error mentions a pandoc fallback: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("pandoc was run")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("a failed PDF export left an output file")
	}
}
