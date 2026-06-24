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
			Description: "Create a new HTML document. Required parameters: 'name' (string) and 'html_content' (string containing the HTML). Returns the document ID and file path.",
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
			Description: "Replace an existing HTML document's entire content. Preserves metadata like name and created_at. For edits, prefer append_html or replace_in_document, which send only the changed fragment instead of the whole document.",
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
			Name:        "append_html",
			Description: "Append an HTML fragment to the end of an existing document's body. Use this to add content (e.g. a new section or an answer key) without resending the whole document. Inserts before the closing </body> tag if present, otherwise appends at the end.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID (e.g., 'my-report-a3f9')"
					},
					"html": {
						"type": "string",
						"description": "The HTML fragment to append (only the new content, not the whole document)"
					}
				},
				"required": ["document_id", "html"]
			}`),
		},
		{
			Name:        "replace_in_document",
			Description: "Make a targeted edit by replacing a single occurrence of old_str with new_str in an existing document. old_str must match exactly once; if it matches zero or multiple times the call fails, so include enough surrounding context to make it unique. Use this for small edits instead of resending the whole document.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID (e.g., 'my-report-a3f9')"
					},
					"old_str": {
						"type": "string",
						"description": "The exact existing HTML substring to replace. Must occur exactly once in the document."
					},
					"new_str": {
						"type": "string",
						"description": "The replacement HTML (may be empty to delete the matched text)"
					}
				},
				"required": ["document_id", "old_str", "new_str"]
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
			Description: "Retrieve a document's content and metadata by ID. For large documents, pass content='outline' first to get just the heading structure and size instead of the full HTML body.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"document_id": {
						"type": "string",
						"description": "The unique document ID"
					},
					"content": {
						"type": "string",
						"enum": ["full", "outline"],
						"description": "What to return. 'full' (default) returns the entire html_content. 'outline' returns only the list of headings (level + text) and the document size in bytes — use this to inspect a large document without pulling the whole body."
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
