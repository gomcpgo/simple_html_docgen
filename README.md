# Simple HTML Document Generator MCP Server

An MCP server that enables LLMs to create and edit HTML documents with embedded styles, images, and videos.

## Features

- Create HTML documents with unique, human-readable IDs
- Update existing documents
- Add images and videos (automatically copied to document folder)
- Export to HTML, PDF, or DOCX (requires Pandoc)
- List and retrieve documents
- Terminal mode for testing

## Document Structure

Each document is stored in its own folder:

```
{ROOT_DIR}/my-document-a3f9/
├── index.html        # HTML with embedded <style>
├── metadata.json     # Document metadata
└── media/            # Images and videos
    ├── image1.png
    └── video1.mp4
```

## Document IDs

Document IDs are generated from the document name:
- Input: "My Report"
- Output: "my-report-a3f9" (slugified name + 4-char random suffix)

## Configuration

Set the root directory via environment variable:
```bash
export SIMPLE_HTML_ROOT_DIR="/path/to/documents"
```

Default: `~/.simple_html_docs`

## Building

```bash
./run.sh install  # Install dependencies
./run.sh build    # Build binary to bin/simple_html_docgen
```

## Terminal Mode (Testing)

```bash
# Create a document
./run.sh create "My Report" "<h1>Hello World</h1><p>This is a test.</p>"

# List documents
./run.sh list

# Get document
./run.sh get my-report-a3f9

# Update document
./run.sh update my-report-a3f9 "<h1>Updated Content</h1>"

# Add media
./run.sh add-media my-report-a3f9 /path/to/image.png image

# Export to PDF
./run.sh export my-report-a3f9 pdf
```

## MCP Tools

### create_document
Create a new HTML document.

**Parameters:**
- `name` (string, required): Document name
- `html_content` (string, required): HTML content

**Returns:**
```json
{
  "status": "succeeded",
  "document_id": "my-report-a3f9",
  "name": "My Report",
  "file_path": "/path/to/my-report-a3f9/index.html",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### update_document
Update an existing document's HTML content.

**Parameters:**
- `document_id` (string, required): Document ID
- `html_content` (string, required): New HTML content

### add_media
Add an image or video file to a document.

**Parameters:**
- `document_id` (string, required): Document ID
- `source_path` (string, required): Absolute path to media file
- `media_type` (string, required): "image" or "video"

**Returns:**
```json
{
  "status": "succeeded",
  "document_id": "my-report-a3f9",
  "relative_path": "media/image1.png",
  "media_type": "image"
}
```

Use the `relative_path` in HTML:
```html
<img src="media/image1.png" alt="Image">
```

### get_document
Retrieve a document by ID.

**Parameters:**
- `document_id` (string, required): Document ID

### list_documents
List all documents.

**Returns:**
```json
{
  "status": "succeeded",
  "count": 2,
  "documents": [
    {
      "document_id": "my-report-a3f9",
      "name": "My Report",
      "file_path": "/path/to/my-report-a3f9/index.html",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### export_document
Export a document to HTML, PDF, or DOCX.

**Parameters:**
- `document_id` (string, required): Document ID
- `format` (string, required): "html", "pdf", or "docx"

**Returns:**
```json
{
  "status": "succeeded",
  "document_id": "my-report-a3f9",
  "format": "pdf",
  "output_path": "/path/to/my-report-a3f9/my-report-a3f9.pdf"
}
```

## Export Requirements

For PDF and DOCX export, install Pandoc:

**macOS:**
```bash
brew install pandoc
brew install basictex  # For PDF support
```

**Linux:**
```bash
sudo apt-get install pandoc texlive-xetex
```

**Windows:**
Download from https://pandoc.org/installing.html

## Testing

```bash
./run.sh test
```

## Architecture

- `cmd/main.go` - Entry point with terminal mode
- `pkg/config/` - Configuration from env vars
- `pkg/document/` - Core document logic
- `pkg/storage/` - File operations
- `pkg/export/` - Export functionality
- `pkg/handler/` - MCP protocol implementation

## License

MIT License