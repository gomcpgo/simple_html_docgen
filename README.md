# Simple HTML Document Generator MCP Server

An MCP server that enables LLMs to create and edit HTML documents with embedded styles, images, and videos.

## Features

- Create HTML documents with unique, human-readable IDs
- Update existing documents
- Add images and videos (automatically copied to document folder)
- Export to HTML, PDF (headless Chrome), or DOCX (built in, through [htmltodocx](https://github.com/gomcpgo/htmltodocx))
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
- `output_path` (string, optional): where to write the file; defaults to the document's folder
- `page_size` (string, optional): "a4", "letter" or "legal", for PDF and DOCX. Defaults to the
  size for the user's locale: Letter in the regions that use it (US, Canada, Mexico and a few
  others), A4 elsewhere. The locale comes from `LC_ALL`, `LC_PAPER` or `LANG`, and from the system
  setting when those name no region (macOS `AppleLocale`, the Windows user locale).

**Returns:**
```json
{
  "status": "succeeded",
  "document_id": "my-report-a3f9",
  "format": "docx",
  "output_path": "/path/to/my-report-a3f9/my-report-a3f9.docx",
  "page_size": "a4",
  "warnings": [
    {"code": "image", "message": "image media/chart.png was replaced by its alt text: cannot open file"}
  ]
}
```

`page_size` is present for PDF and DOCX. `warnings` is present only when a DOCX export had to
approximate or leave something out; the file is still written. The caller decides whether to show
them.

## Export Requirements

- DOCX needs nothing installed. The HTML is converted in-process; images and stylesheets are read
  from the document's own folder only, so remote URLs and paths outside the folder are skipped with
  a warning. The file is written under a temporary name and renamed into place, so a failed export
  leaves no partial file.
- PDF is printed by headless Chrome or Chromium. An installed browser is used if there is one;
  otherwise rod downloads Chromium on first use. Nothing else is needed.

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