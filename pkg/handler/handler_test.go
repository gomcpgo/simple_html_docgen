package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"simple_html_docgen/pkg/config"
	"simple_html_docgen/pkg/document"
	"simple_html_docgen/pkg/export"

	"github.com/gomcpgo/mcp/pkg/protocol"
)

type fakeExporter struct {
	gotFormat string
	gotOpts   export.Options
	result    *export.Result
	err       error
}

func (f *fakeExporter) ExportDocument(ctx context.Context, documentID, format, outputPath string, opts export.Options, docSvc *document.Service) (*export.Result, error) {
	f.gotFormat, f.gotOpts = format, opts
	return f.result, f.err
}

func callExport(t *testing.T, f *fakeExporter, args map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	h := NewHandler(&config.Config{RootDir: t.TempDir()}, f)
	resp, err := h.CallTool(context.Background(), &protocol.CallToolRequest{Name: "export_document", Arguments: args})
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content[0].Text), &out); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	return out, nil
}

func TestExportDocumentReturnsWarnings(t *testing.T) {
	f := &fakeExporter{result: &export.Result{
		Path:     "/tmp/x.docx",
		PageSize: "letter",
		Warnings: []export.Warning{{Code: "image", Message: "could not load image media/a.png"}},
	}}
	out, err := callExport(t, f, map[string]interface{}{"document_id": "doc-1", "format": "docx", "page_size": "letter"})
	if err != nil {
		t.Fatal(err)
	}
	if f.gotOpts.PageSize != "letter" {
		t.Errorf("page size passed to exporter = %q, want letter", f.gotOpts.PageSize)
	}
	if out["status"] != "succeeded" || out["output_path"] != "/tmp/x.docx" || out["page_size"] != "letter" {
		t.Errorf("unexpected result: %v", out)
	}
	ws, ok := out["warnings"].([]interface{})
	if !ok || len(ws) != 1 {
		t.Fatalf("warnings = %v, want one", out["warnings"])
	}
	w := ws[0].(map[string]interface{})
	if w["code"] != "image" || w["message"] != "could not load image media/a.png" {
		t.Errorf("warning = %v", w)
	}
}

func TestExportDocumentWithoutWarnings(t *testing.T) {
	f := &fakeExporter{result: &export.Result{Path: "/tmp/x.html"}}
	out, err := callExport(t, f, map[string]interface{}{"document_id": "doc-1", "format": "html"})
	if err != nil {
		t.Fatal(err)
	}
	if f.gotOpts.PageSize != "" {
		t.Errorf("page size = %q, want empty (locale default)", f.gotOpts.PageSize)
	}
	if _, ok := out["warnings"]; ok {
		t.Errorf("an export with no warnings should not list any: %v", out)
	}
	if _, ok := out["page_size"]; ok {
		t.Errorf("HTML export has no page size: %v", out)
	}
}

func TestExportDocumentRejectsBadPageSize(t *testing.T) {
	f := &fakeExporter{result: &export.Result{Path: "/tmp/x.docx"}}
	_, err := callExport(t, f, map[string]interface{}{"document_id": "doc-1", "format": "docx", "page_size": 4})
	if err == nil {
		t.Error("a page_size that is not a string should be rejected")
	}
	if f.gotFormat != "" {
		t.Error("exporter should not be called")
	}
}

func TestExportDocumentReportsFailure(t *testing.T) {
	f := &fakeExporter{err: errors.New(`unsupported page size "tabloid"`)}
	out, err := callExport(t, f, map[string]interface{}{"document_id": "doc-1", "format": "docx", "page_size": "tabloid"})
	if err != nil {
		t.Fatal(err)
	}
	if out["status"] != "failed" {
		t.Errorf("status = %v, want failed", out["status"])
	}
}

func TestExportToolListsPageSizes(t *testing.T) {
	h := NewHandler(&config.Config{RootDir: t.TempDir()}, &fakeExporter{})
	for _, tool := range h.GetTools() {
		if tool.Name != "export_document" {
			continue
		}
		var schema struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		got := schema.Properties["page_size"].Enum
		if len(got) != len(export.PageSizes) {
			t.Fatalf("page_size enum = %v, want %v", got, export.PageSizes)
		}
		for i := range got {
			if got[i] != export.PageSizes[i] {
				t.Errorf("page_size enum = %v, want %v", got, export.PageSizes)
			}
		}
		return
	}
	t.Fatal("export_document tool not found")
}
