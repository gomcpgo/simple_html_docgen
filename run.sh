#!/bin/bash
set -e

# Simple HTML Document Generator MCP Server Build/Test Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

command=$1
shift || true

case "$command" in
    build)
        echo "Building simple_html_docgen..."
        mkdir -p bin
        go build -o bin/simple_html_docgen cmd/main.go
        echo "Build complete: bin/simple_html_docgen"
        ;;

    test)
        echo "Running tests..."
        go test ./... -v
        ;;

    install)
        echo "Installing dependencies..."
        go mod download
        go mod tidy
        ;;

    create)
        if [ -z "$1" ] || [ -z "$2" ]; then
            echo "Usage: ./run.sh create <name> <html_content>"
            exit 1
        fi
        bin/simple_html_docgen -create "$1" -html "$2"
        ;;

    list)
        bin/simple_html_docgen -list
        ;;

    get)
        if [ -z "$1" ]; then
            echo "Usage: ./run.sh get <document_id>"
            exit 1
        fi
        bin/simple_html_docgen -get "$1"
        ;;

    update)
        if [ -z "$1" ] || [ -z "$2" ]; then
            echo "Usage: ./run.sh update <document_id> <html_content>"
            exit 1
        fi
        bin/simple_html_docgen -update "$1" -html "$2"
        ;;

    export)
        if [ -z "$1" ] || [ -z "$2" ]; then
            echo "Usage: ./run.sh export <document_id> <format>"
            exit 1
        fi
        bin/simple_html_docgen -export "$1" -format "$2"
        ;;

    add-media)
        if [ -z "$1" ] || [ -z "$2" ]; then
            echo "Usage: ./run.sh add-media <document_id> <media_path> [media_type]"
            exit 1
        fi
        media_type="${3:-image}"
        bin/simple_html_docgen -add-media "$1" -media-path "$2" -media-type "$media_type"
        ;;

    clean)
        echo "Cleaning build artifacts..."
        rm -rf bin
        echo "Clean complete"
        ;;

    *)
        echo "Simple HTML Document Generator MCP Server"
        echo ""
        echo "Usage: ./run.sh <command> [args]"
        echo ""
        echo "Commands:"
        echo "  build                          Build the MCP server"
        echo "  test                           Run tests"
        echo "  install                        Install dependencies"
        echo "  create <name> <html>           Create a new document"
        echo "  list                           List all documents"
        echo "  get <id>                       Get document by ID"
        echo "  update <id> <html>             Update document content"
        echo "  export <id> <format>           Export document (html/pdf/docx)"
        echo "  add-media <id> <path> [type]   Add media file to document"
        echo "  clean                          Remove build artifacts"
        echo ""
        echo "Examples:"
        echo "  ./run.sh build"
        echo "  ./run.sh create 'My Report' '<h1>Hello World</h1>'"
        echo "  ./run.sh list"
        echo "  ./run.sh export my-report-a3f9 pdf"
        ;;
esac
