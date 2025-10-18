#!/bin/bash
# Script to run the MCP server in stdio mode
# Usage: ./run-mcp.sh

cd "$(dirname "$0")"

# Build the server if needed
if [ ! -f bin/server ] || [ cmd/server/main.go -nt bin/server ]; then
    echo "Building server..." >&2
    go build -o bin/server ./cmd/server || exit 1
fi

# Run in stdio mode
exec ./bin/server --stdio
