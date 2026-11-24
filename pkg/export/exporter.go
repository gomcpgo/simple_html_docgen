package export

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"simple_html_docgen/pkg/document"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gomcpgo/htmltodocx"
	"github.com/ysmood/gson"
)

// Exporter handles document export operations
type Exporter struct {
	chromeTimeout time.Duration
	docxTimeout   time.Duration
	// defaultPageSize gives the page size when the caller names none.
	defaultPageSize func() string
}

// NewExporter creates a new exporter instance
func NewExporter() *Exporter {
	return &Exporter{
		chromeTimeout:   30 * time.Second,
		docxTimeout:     30 * time.Second,
		defaultPageSize: DefaultPageSize,
	}
}

// Options are the caller's choices for an export.
type Options struct {
	// PageSize is one of PageSizes for DOCX and PDF; empty means the size
	// for the user's locale. HTML export ignores it.
	PageSize string
}

// Result describes a finished export.
type Result struct {
	Path string
	// PageSize is the page size used, for DOCX and PDF.
	PageSize string
	// Warnings report what a DOCX export could not carry over exactly, such
	// as an image that could not be loaded. The file was still written.
	Warnings []Warning
}

// Warning is one thing the export approximated or left out.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// maxWarnings bounds the warnings returned for one export; the document's
// HTML is model-written and could produce any number of distinct ones.
const maxWarnings = 50

// ExportDocument exports a document to the specified format
func (e *Exporter) ExportDocument(ctx context.Context, documentID, format, outputPath string, opts Options, docSvc *document.Service) (*Result, error) {
	var ps pageSize
	if format == "pdf" || format == "docx" {
		name := opts.PageSize
		if name == "" {
			name = e.defaultPageSize()
		}
		var err error
		if ps, err = parsePageSize(name); err != nil {
			return nil, err
		}
	}

	// Get the document
	doc, err := docSvc.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Use provided output path or generate default
	if outputPath == "" {
		outputPath = filepath.Join(docSvc.GetDocumentPath(documentID), fmt.Sprintf("%s.%s", documentID, format))
	} else {
		// Ensure parent directory exists
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	switch format {
	case "html":
		path, err := e.exportHTML(doc, outputPath)
		if err != nil {
			return nil, err
		}
		return &Result{Path: path}, nil
	case "pdf":
		path, err := e.exportPDF(ctx, doc, outputPath, ps, docSvc)
		if err != nil {
			return nil, err
		}
		return &Result{Path: path, PageSize: ps.name}, nil
	case "docx":
		warnings, err := e.exportDOCX(ctx, doc, outputPath, ps, docSvc)
		if err != nil {
			return nil, err
		}
		return &Result{Path: outputPath, PageSize: ps.name, Warnings: warnings}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// writeTempHTML writes content to a uniquely-named temp file inside the
// document's own folder and returns its path plus a cleanup func. The name must
// be unique per export: every export used to write the same "temp_export.html"
// and remove it on the way out, so two exports of one document issued in the
// same turn (a model asking for PDF and DOCX together) raced — one deleted the
// file the other was still converting.
func writeTempHTML(dir, content string) (string, func(), error) {
	f, err := os.CreateTemp(dir, "temp_export_*.html")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("failed to write temp HTML file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("failed to write temp HTML file: %w", err)
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// exportHTML exports the document as HTML (simple copy)
func (e *Exporter) exportHTML(doc *document.Document, outputPath string) (string, error) {
	// Write HTML content to output file
	if err := os.WriteFile(outputPath, []byte(doc.HTMLContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML file: %w", err)
	}

	return outputPath, nil
}

// exportPDF prints the document to PDF with headless Chrome or Chromium,
// using an installed browser if there is one and letting rod download
// Chromium otherwise. Like DOCX, the file is written under a temporary name
// and renamed into place.
func (e *Exporter) exportPDF(ctx context.Context, doc *document.Document, outputPath string, ps pageSize, docSvc *document.Service) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, e.chromeTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("PDF export failed: %w", err)
	}

	// Default print styles; the document's own @media print rules win.
	tmpHTMLPath, cleanup, err := writeTempHTML(docSvc.GetDocumentPath(doc.ID), InjectDefaultPrintStyles(doc.HTMLContent))
	if err != nil {
		return "", err
	}
	defer cleanup()

	l := launcher.New().Context(ctx).Headless(true)
	if chromePath, ok := launcher.LookPath(); ok {
		l = l.Bin(chromePath)
	}
	controlURL, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("PDF export needs Google Chrome or Chromium, which could not be started: %w", err)
	}
	defer l.Kill()

	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		return "", fmt.Errorf("PDF export could not connect to Chrome: %w", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{URL: "file://" + tmpHTMLPath})
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("failed to load page: %w", err)
	}

	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground:   true, // Include background colors and images
		PaperWidth:        gson.Num(ps.widthIn),
		PaperHeight:       gson.Num(ps.heightIn),
		MarginTop:         gson.Num(0.4),
		MarginBottom:      gson.Num(0.4),
		MarginLeft:        gson.Num(0.4),
		MarginRight:       gson.Num(0.4),
		PreferCSSPageSize: true, // A CSS @page size wins over PaperWidth/PaperHeight
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate PDF: %w", err)
	}
	err = writeFileAtomic(outputPath, func(w io.Writer) error {
		_, err := io.Copy(w, pdfStream)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to write PDF file: %w", err)
	}
	return outputPath, nil
}

// exportDOCX converts the document with htmltodocx. Images and stylesheets
// are read from the document's own folder and nowhere else. The file is
// written under a temporary name and renamed into place, so a failed export
// leaves no partial file and does not replace an earlier one.
func (e *Exporter) exportDOCX(ctx context.Context, doc *document.Document, outputPath string, ps pageSize, docSvc *document.Service) ([]Warning, error) {
	ctx, cancel := context.WithTimeout(ctx, e.docxTimeout)
	defer cancel()

	var warnings []Warning
	dropped := 0
	opts := &htmltodocx.Options{
		Page:      htmltodocx.Page{Size: ps.docx},
		Resources: htmltodocx.DirLoader(docSvc.GetDocumentPath(doc.ID)),
		Warn: func(w htmltodocx.Warning) {
			if len(warnings) < maxWarnings {
				warnings = append(warnings, Warning{Code: w.Code, Message: w.Message})
			} else {
				dropped++
			}
		},
	}
	err := writeFileAtomic(outputPath, func(w io.Writer) error {
		return htmltodocx.Convert(ctx, strings.NewReader(doc.HTMLContent), w, opts)
	})
	if err != nil {
		return nil, fmt.Errorf("DOCX conversion failed: %w", err)
	}
	if dropped > 0 {
		warnings = append(warnings, Warning{Code: "more", Message: fmt.Sprintf("%d more warnings not shown", dropped)})
	}
	return warnings, nil
}

// writeFileAtomic writes a file through write and moves it to path only if
// write succeeds.
func writeFileAtomic(path string, write func(io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	bw := bufio.NewWriter(f)
	if err := write(bw); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	if err := f.Chmod(0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// InjectDefaultPrintStyles adds conservative default print styles to HTML content
// These styles act as a fallback if the LLM didn't include @media print rules
// The styles use !important only for decorative properties that should be removed in print
func InjectDefaultPrintStyles(htmlContent string) string {
	// Default print styles - conservative approach
	// Only removes obvious decorative elements (backgrounds, shadows)
	// Preserves layout, fonts, and meaningful colors
	defaultPrintStyles := `
<style media="print">
/* Auto-injected print optimization fallback */
/* These rules apply only if document doesn't define its own @media print styles */
@media print {
  /* Remove decorative body backgrounds */
  body {
    background: white !important;
    background-color: white !important;
    background-image: none !important;
  }

  /* Remove decorative shadows that waste ink */
  * {
    box-shadow: none !important;
    text-shadow: none !important;
  }

  /* Preserve page breaks and layout */
  @page {
    margin: 0.5in;
  }
}
</style>`

	// Find the closing </head> tag and inject before it
	// If no </head>, inject at the beginning of <body> or start of document
	if idx := strings.Index(strings.ToLower(htmlContent), "</head>"); idx != -1 {
		return htmlContent[:idx] + defaultPrintStyles + "\n" + htmlContent[idx:]
	} else if idx := strings.Index(strings.ToLower(htmlContent), "<body"); idx != -1 {
		// Find the end of the <body> tag
		if endIdx := strings.Index(htmlContent[idx:], ">"); endIdx != -1 {
			insertPos := idx + endIdx + 1
			return htmlContent[:insertPos] + "\n" + defaultPrintStyles + "\n" + htmlContent[insertPos:]
		}
	}

	// Fallback: prepend to the entire document
	return defaultPrintStyles + "\n" + htmlContent
}
