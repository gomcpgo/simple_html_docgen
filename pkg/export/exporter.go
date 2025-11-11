package export

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"simple_html_docgen/pkg/document"
	"time"
)

// Exporter handles document export operations
type Exporter struct {
	pandocTimeout time.Duration
}

// NewExporter creates a new exporter instance
func NewExporter() *Exporter {
	return &Exporter{
		pandocTimeout: 30 * time.Second,
	}
}

// ExportDocument exports a document to the specified format
func (e *Exporter) ExportDocument(documentID, format string, docSvc *document.Service) (string, error) {
	// Get the document
	doc, err := docSvc.GetDocument(documentID)
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	// Generate output filename
	outputPath := filepath.Join(docSvc.GetDocumentPath(documentID), fmt.Sprintf("%s.%s", documentID, format))

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

// exportPDF exports the document as PDF using Pandoc
func (e *Exporter) exportPDF(doc *document.Document, outputPath string, docSvc *document.Service) (string, error) {
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
