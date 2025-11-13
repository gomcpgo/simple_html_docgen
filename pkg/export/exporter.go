package export

import (
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
	"github.com/ysmood/gson"
)

// Exporter handles document export operations
type Exporter struct {
	pandocTimeout time.Duration
	chromeTimeout time.Duration
}

// NewExporter creates a new exporter instance
func NewExporter() *Exporter {
	return &Exporter{
		pandocTimeout: 30 * time.Second,
		chromeTimeout: 30 * time.Second,
	}
}

// ExportDocument exports a document to the specified format
func (e *Exporter) ExportDocument(documentID, format, outputPath string, docSvc *document.Service) (string, error) {
	// Get the document
	doc, err := docSvc.GetDocument(documentID)
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	// Use provided output path or generate default
	if outputPath == "" {
		outputPath = filepath.Join(docSvc.GetDocumentPath(documentID), fmt.Sprintf("%s.%s", documentID, format))
	} else {
		// Ensure parent directory exists
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	switch format {
	case "html":
		return e.exportHTML(doc, outputPath)
	case "pdf":
		return e.exportPDF(doc, outputPath, docSvc)
	case "docx":
		return e.exportDOCX(doc, outputPath, docSvc)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
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
func (e *Exporter) exportPDF(doc *document.Document, outputPath string, docSvc *document.Service) (string, error) {
	// Try Chrome/Chromium first (best CSS preservation)
	if err := e.exportPDFWithChrome(doc, outputPath, docSvc); err == nil {
		return outputPath, nil
	}

	// Fallback to Pandoc if Chrome is not available
	return e.exportPDFWithPandoc(doc, outputPath, docSvc)
}

// exportPDFWithChrome exports the document as PDF using headless Chrome
func (e *Exporter) exportPDFWithChrome(doc *document.Document, outputPath string, docSvc *document.Service) error {
	// Inject default print styles as fallback (conservative approach)
	// These will be overridden by any @media print rules the LLM includes
	htmlWithPrintStyles := InjectDefaultPrintStyles(doc.HTMLContent)

	// Create a temporary HTML file
	tmpHTMLPath := filepath.Join(docSvc.GetDocumentPath(doc.ID), "temp_export.html")
	if err := os.WriteFile(tmpHTMLPath, []byte(htmlWithPrintStyles), 0644); err != nil {
		return fmt.Errorf("failed to write temp HTML file: %w", err)
	}
	defer os.Remove(tmpHTMLPath)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.chromeTimeout)
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
		MarginTop:         gson.Num(0.4),
		MarginBottom:      gson.Num(0.4),
		MarginLeft:        gson.Num(0.4),
		MarginRight:       gson.Num(0.4),
		PreferCSSPageSize: true, // Use CSS @page size if specified
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
	tmpHTMLPath := filepath.Join(docSvc.GetDocumentPath(doc.ID), "temp_export.html")
	if err := os.WriteFile(tmpHTMLPath, []byte(doc.HTMLContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp HTML file: %w", err)
	}
	defer os.Remove(tmpHTMLPath)

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

// exportDOCX exports the document as DOCX using Pandoc
func (e *Exporter) exportDOCX(doc *document.Document, outputPath string, docSvc *document.Service) (string, error) {
	if err := e.checkPandoc(); err != nil {
		return "", err
	}

	// Create a temporary HTML file for Pandoc
	tmpHTMLPath := filepath.Join(docSvc.GetDocumentPath(doc.ID), "temp_export.html")
	if err := os.WriteFile(tmpHTMLPath, []byte(doc.HTMLContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp HTML file: %w", err)
	}
	defer os.Remove(tmpHTMLPath)

	// Run Pandoc conversion
	args := []string{
		"-f", "html",
		"-o", outputPath,
		tmpHTMLPath,
	}

	if err := e.runPandoc(args, docSvc.GetDocumentPath(doc.ID)); err != nil {
		return "", fmt.Errorf("DOCX conversion failed: %w", err)
	}

	return outputPath, nil
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
