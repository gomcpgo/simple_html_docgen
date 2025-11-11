package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"simple_html_docgen/pkg/config"
	"simple_html_docgen/pkg/export"
	mcpHandler "simple_html_docgen/pkg/handler"

	"github.com/gomcpgo/mcp/pkg/handler"
	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/mcp/pkg/server"
)

func main() {
	// Define terminal mode flags
	var (
		createDoc    string
		updateDoc    string
		htmlContent  string
		listDocs     bool
		getDoc       string
		exportDoc    string
		exportFormat string
		addMedia     string
		mediaPath    string
		mediaType    string
	)

	flag.StringVar(&createDoc, "create", "", "Create a new document with the specified name")
	flag.StringVar(&updateDoc, "update", "", "Update document with the specified ID")
	flag.StringVar(&htmlContent, "html", "", "HTML content for create/update operations")
	flag.BoolVar(&listDocs, "list", false, "List all documents")
	flag.StringVar(&getDoc, "get", "", "Get document by ID")
	flag.StringVar(&exportDoc, "export", "", "Export document by ID")
	flag.StringVar(&exportFormat, "format", "html", "Export format (html, pdf, docx)")
	flag.StringVar(&addMedia, "add-media", "", "Add media to document (specify document ID)")
	flag.StringVar(&mediaPath, "media-path", "", "Path to media file")
	flag.StringVar(&mediaType, "media-type", "image", "Media type (image, video)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create handler
	exportSvc := export.NewExporter()
	h := mcpHandler.NewHandler(cfg, exportSvc)
	ctx := context.Background()

	// Terminal mode operations
	if createDoc != "" {
		if htmlContent == "" {
			log.Fatal("--html is required when creating a document")
		}
		runTerminalCommand(ctx, h, "create_document", map[string]interface{}{
			"name":         createDoc,
			"html_content": htmlContent,
		})
		return
	}

	if updateDoc != "" {
		if htmlContent == "" {
			log.Fatal("--html is required when updating a document")
		}
		runTerminalCommand(ctx, h, "update_document", map[string]interface{}{
			"document_id":  updateDoc,
			"html_content": htmlContent,
		})
		return
	}

	if listDocs {
		runTerminalCommand(ctx, h, "list_documents", map[string]interface{}{})
		return
	}

	if getDoc != "" {
		runTerminalCommand(ctx, h, "get_document", map[string]interface{}{
			"document_id": getDoc,
		})
		return
	}

	if exportDoc != "" {
		runTerminalCommand(ctx, h, "export_document", map[string]interface{}{
			"document_id": exportDoc,
			"format":      exportFormat,
		})
		return
	}

	if addMedia != "" {
		if mediaPath == "" {
			log.Fatal("--media-path is required when adding media")
		}
		runTerminalCommand(ctx, h, "add_media", map[string]interface{}{
			"document_id": addMedia,
			"source_path": mediaPath,
			"media_type":  mediaType,
		})
		return
	}

	// MCP Server mode (default)
	registry := handler.NewHandlerRegistry()
	registry.RegisterToolHandler(h)

	srv := server.New(server.Options{
		Name:     "simple-html-docgen",
		Version:  "1.0.0",
		Registry: registry,
	})

	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runTerminalCommand executes a tool command in terminal mode
func runTerminalCommand(ctx context.Context, h *mcpHandler.Handler, toolName string, args map[string]interface{}) {
	req := &protocol.CallToolRequest{
		Name:      toolName,
		Arguments: args,
	}

	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}

	// Pretty print response
	for _, content := range resp.Content {
		if content.Type == "text" {
			var data interface{}
			if err := json.Unmarshal([]byte(content.Text), &data); err == nil {
				pretty, _ := json.MarshalIndent(data, "", "  ")
				fmt.Println(string(pretty))
			} else {
				fmt.Println(content.Text)
			}
		}
	}
}
