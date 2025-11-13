package handler

import (
	"encoding/json"

	"github.com/gomcpgo/mcp/pkg/protocol"
)

// GetTools returns the list of available MCP tools
func (h *Handler) GetTools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "create_document",
			Description: "Create a new HTML document with a name and HTML content. Returns the document ID and file path.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the document (will be used to generate a unique ID)"
					},
					"html_content": {
						"type": "string",
						"description": "The HTML content of the document. Can include embedded CSS in <style> tags. Please include @media print CSS rules to optimize for PDF export: remove decorative backgrounds (gradients, colors), box-shadow, and text-shadow properties while preserving essential styling like fonts, colors that convey meaning, and layout. Example: @media print { body { background: white !important; } .container { box-shadow: none !important; } }"
					}
				},
				"required": ["name", "html_content"]
			}`),
		},
		{
			Name:        "update_document",
			Description: "Update an existing HTML document's content. Preserves metadata like name and created_at.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID (e.g., 'my-report-a3f9')"
					},
					"html_content": {
						"type": "string",
						"description": "The new HTML content for the document. Please include @media print CSS rules to optimize for PDF export: remove decorative backgrounds (gradients, colors), box-shadow, and text-shadow properties while preserving essential styling like fonts, colors that convey meaning, and layout. Example: @media print { body { background: white !important; } .container { box-shadow: none !important; } }"
					}
				},
				"required": ["document_id", "html_content"]
			}`),
		},
		{
			Name:        "add_media",
			Description: "Add an image or video file to a document. Copies the file to the document's media folder and returns the relative path to use in HTML.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID"
					},
					"source_path": {
						"type": "string",
						"description": "The absolute path to the source media file"
					},
					"media_type": {
						"type": "string",
						"enum": ["image", "video"],
						"description": "The type of media file"
					}
				},
				"required": ["document_id", "source_path", "media_type"]
			}`),
		},
		{
			Name:        "get_document",
			Description: "Retrieve a document's content and metadata by ID.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID"
					}
				},
				"required": ["document_id"]
			}`),
		},
		{
			Name:        "list_documents",
			Description: "List all HTML documents with their metadata.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "export_document",
			Description: "Export an HTML document to a specified format (html, pdf, or docx). Returns the path to the exported file.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID"
					},
					"format": {
						"type": "string",
						"enum": ["html", "pdf", "docx"],
						"description": "The export format"
					},
					"output_path": {
						"type": "string",
						"description": "Optional output file path. If not provided, exports to the document's directory."
					}
				},
				"required": ["document_id", "format"]
			}`),
		},
	}
}
