package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"simple_html_docgen/pkg/config"
	"simple_html_docgen/pkg/document"
	"simple_html_docgen/pkg/storage"

	"github.com/gomcpgo/mcp/pkg/protocol"
)

// Handler implements the MCP protocol for Simple HTML Document Generator
type Handler struct {
	config    *config.Config
	docSvc    *document.Service
	exportSvc ExportService
}

// ExportService defines the interface for export functionality
type ExportService interface {
	ExportDocument(documentID, format string, docSvc *document.Service) (string, error)
}

// NewHandler creates a new handler instance
func NewHandler(cfg *config.Config, exportSvc ExportService) *Handler {
	storage := storage.NewStorage(cfg.RootDir)
	docSvc := document.NewService(storage)

	return &Handler{
		config:    cfg,
		docSvc:    docSvc,
		exportSvc: exportSvc,
	}
}

// ListTools returns the list of available tools
func (h *Handler) ListTools(ctx context.Context) (*protocol.ListToolsResponse, error) {
	tools := h.GetTools()
	return &protocol.ListToolsResponse{
		Tools: tools,
	}, nil
}

// CallTool handles tool invocations
func (h *Handler) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	switch req.Name {
	case "create_document":
		return h.handleCreateDocument(ctx, req.Arguments)
	case "update_document":
		return h.handleUpdateDocument(ctx, req.Arguments)
	case "add_media":
		return h.handleAddMedia(ctx, req.Arguments)
	case "get_document":
		return h.handleGetDocument(ctx, req.Arguments)
	case "list_documents":
		return h.handleListDocuments(ctx, req.Arguments)
	case "export_document":
		return h.handleExportDocument(ctx, req.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Name)
	}
}

func (h *Handler) handleCreateDocument(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required and must be a string")
	}

	htmlContent, ok := args["html_content"].(string)
	if !ok || htmlContent == "" {
		return nil, fmt.Errorf("html_content is required and must be a string")
	}

	doc, err := h.docSvc.CreateDocument(name, htmlContent)
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to create document: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "succeeded",
		"document_id": doc.ID,
		"name":        doc.Name,
		"file_path":   h.docSvc.GetHTMLPath(doc.ID),
		"created_at":  doc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":  doc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return h.successResponse(result), nil
}

func (h *Handler) handleUpdateDocument(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	htmlContent, ok := args["html_content"].(string)
	if !ok || htmlContent == "" {
		return nil, fmt.Errorf("html_content is required and must be a string")
	}

	doc, err := h.docSvc.UpdateDocument(documentID, htmlContent)
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to update document: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "succeeded",
		"document_id": doc.ID,
		"name":        doc.Name,
		"file_path":   h.docSvc.GetHTMLPath(doc.ID),
		"updated_at":  doc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return h.successResponse(result), nil
}

func (h *Handler) handleAddMedia(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	sourcePath, ok := args["source_path"].(string)
	if !ok || sourcePath == "" {
		return nil, fmt.Errorf("source_path is required and must be a string")
	}

	mediaType, ok := args["media_type"].(string)
	if !ok || mediaType == "" {
		return nil, fmt.Errorf("media_type is required and must be a string")
	}

	relativePath, err := h.docSvc.AddMedia(documentID, sourcePath, mediaType)
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to add media: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":        "succeeded",
		"document_id":   documentID,
		"relative_path": relativePath,
		"media_type":    mediaType,
	}

	return h.successResponse(result), nil
}

func (h *Handler) handleGetDocument(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	doc, err := h.docSvc.GetDocument(documentID)
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to get document: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":       "succeeded",
		"document_id":  doc.ID,
		"name":         doc.Name,
		"html_content": doc.HTMLContent,
		"file_path":    h.docSvc.GetHTMLPath(doc.ID),
		"created_at":   doc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":   doc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return h.successResponse(result), nil
}

func (h *Handler) handleListDocuments(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	docs, err := h.docSvc.ListDocuments()
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to list documents: %v", err)), nil
	}

	documents := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		documents[i] = map[string]interface{}{
			"document_id": doc.ID,
			"name":        doc.Name,
			"file_path":   h.docSvc.GetHTMLPath(doc.ID),
			"created_at":  doc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":  doc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	result := map[string]interface{}{
		"status":    "succeeded",
		"count":     len(documents),
		"documents": documents,
	}

	return h.successResponse(result), nil
}

func (h *Handler) handleExportDocument(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	documentID, ok := args["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	format, ok := args["format"].(string)
	if !ok || format == "" {
		return nil, fmt.Errorf("format is required and must be a string")
	}

	// Validate format
	if format != "html" && format != "pdf" && format != "docx" {
		return nil, fmt.Errorf("invalid format: %s (must be html, pdf, or docx)", format)
	}

	outputPath, err := h.exportSvc.ExportDocument(documentID, format, h.docSvc)
	if err != nil {
		return h.errorResponse(fmt.Sprintf("Failed to export document: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "succeeded",
		"document_id": documentID,
		"format":      format,
		"output_path": outputPath,
	}

	return h.successResponse(result), nil
}

// Helper methods

func (h *Handler) successResponse(data map[string]interface{}) *protocol.CallToolResponse {
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}
}

func (h *Handler) errorResponse(errorMsg string) *protocol.CallToolResponse {
	data := map[string]interface{}{
		"status": "failed",
		"error":  errorMsg,
	}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}
}
