package export

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"simple_html_docgen/pkg/document"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/gomcpgo/htmltodocx"
	"github.com/ysmood/gson"
)

// Exporter handles document export operations
type Exporter struct {
	pandocTimeout time.Duration
	chromeTimeout time.Duration
	docxTimeout   time.Duration
	// defaultPageSize gives the page size when the caller names none.
	defaultPageSize func() string
}

// NewExporter creates a new exporter instance
func NewExporter() *Exporter {
	return &Exporter{
		pandocTimeout:   30 * time.Second,
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
// file the other was still converting, and the PDF route's failure was then
// reported as pandoc's "xelatex not found".
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

// exportPDF exports the document as PDF, trying Chrome first, then falling back to Pandoc
func (e *Exporter) exportPDF(ctx context.Context, doc *document.Document, outputPath string, ps pageSize, docSvc *document.Service) (string, error) {
	// Try Chrome/Chromium first (best CSS preservation)
	chromeErr := e.exportPDFWithChrome(ctx, doc, outputPath, ps, docSvc)
	if chromeErr == nil {
		return outputPath, nil
	}

	// Fallback to Pandoc. Chrome is the route that works on a machine with no
	// LaTeX engine, so when the fallback fails too its "install xelatex" message
	// describes the fallback and not the real failure. Report both, Chrome first.
	path, pandocErr := e.exportPDFWithPandoc(doc, outputPath, docSvc)
	if pandocErr != nil {
		return "", fmt.Errorf("PDF export failed: %w (fallback also failed: %v)", chromeErr, pandocErr)
	}
	return path, nil
}

// exportPDFWithChrome exports the document as PDF using headless Chrome
func (e *Exporter) exportPDFWithChrome(ctx context.Context, doc *document.Document, outputPath string, ps pageSize, docSvc *document.Service) error {
	// Inject default print styles as fallback (conservative approach)
	// These will be overridden by any @media print rules the LLM includes
	htmlWithPrintStyles := InjectDefaultPrintStyles(doc.HTMLContent)

	// Create a temporary HTML file
	tmpHTMLPath, cleanup, err := writeTempHTML(docSvc.GetDocumentPath(doc.ID), htmlWithPrintStyles)
	if err != nil {
		return err
	}
	defer cleanup()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, e.chromeTimeout)
	defer cancel()

	// Try to find existing Chrome/Chromium installation first
	chromePath, _ := launcher.LookPath()

	// Launch browser with system Chrome if available, otherwise auto-download
	var controlURL string
	if chromePath != "" {
		// Use system Chrome/Chromium
		l := launcher.New().Bin(chromePath).Headless(true)
		controlURL = l.MustLaunch()
	} else {
		// Let rod auto-download Chromium as fallback
		l := launcher.New().Headless(true)
		controlURL = l.MustLaunch()
	}

	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("chrome not available: %w", err)
	}
	defer browser.MustClose()

	// Load the HTML file
	page, err := browser.Page(proto.TargetCreateTarget{URL: "file://" + tmpHTMLPath})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("failed to load page: %w", err)
	}

	// Generate PDF with custom settings
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
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Write PDF stream to output file
	if err := utils.OutputFile(outputPath, pdfStream); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}

	return nil
}

// exportPDFWithPandoc exports the document as PDF using Pandoc (fallback)
func (e *Exporter) exportPDFWithPandoc(doc *document.Document, outputPath string, docSvc *document.Service) (string, error) {
	if err := e.checkPandoc(); err != nil {
		return "", err
	}

	// Create a temporary HTML file for Pandoc
	tmpHTMLPath, cleanup, err := writeTempHTML(docSvc.GetDocumentPath(doc.ID), doc.HTMLContent)
	if err != nil {
		return "", err
	}
	defer cleanup()

	// Run Pandoc conversion
	args := []string{
		"-f", "html",
		"-o", outputPath,
		"--pdf-engine=xelatex", // For Unicode/emoji support
		tmpHTMLPath,
	}

	if err := e.runPandoc(args, docSvc.GetDocumentPath(doc.ID)); err != nil {
		return "", fmt.Errorf("PDF conversion failed: %w", err)
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

// checkPandoc checks if Pandoc is installed
func (e *Exporter) checkPandoc() error {
	cmd := exec.Command("pandoc", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc not found: please install pandoc to enable PDF/DOCX export")
	}
	return nil
}

// runPandoc executes a Pandoc command with timeout
func (e *Exporter) runPandoc(args []string, workDir string) error {
	cmd := exec.Command("pandoc", args...)
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("pandoc error: %s", stderr.String())
			}
			return fmt.Errorf("pandoc failed: %w", err)
		}
		return nil
	case <-time.After(e.pandocTimeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("pandoc conversion timed out after %v", e.pandocTimeout)
	}
}

// CheckPandocAvailable checks if Pandoc is available (exported for terminal mode)
func CheckPandocAvailable() bool {
	cmd := exec.Command("pandoc", "--version")
	return cmd.Run() == nil
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
