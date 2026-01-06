#!/bin/bash

# SmartRun startup script
# Builds, starts the server, and opens the browser

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Navigate to the project root (one level up from scripts/)
cd "$SCRIPT_DIR/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed."
    echo "Please install Go (https://go.dev/dl/) or run the binary directly if you have it."
    exit 1
fi

echo "Building SmartRun..."
go build -o smartrund ./cmd/smartrund

echo "Stopping any existing SmartRun servers..."
killall smartrund 2>/dev/null || true

echo "Starting SmartRun server on port 8080..."
./smartrund --port 8080 &

# Wait a moment for server to start
sleep 2

echo "Opening browser..."
open http://localhost:8080

echo ""
echo "SmartRun is running!"
echo "Local access: http://localhost:8080"
echo "Mobile access: http://YOUR_LOCAL_IP:8080"
echo ""
echo "Press Ctrl+C to stop the server"

# Keep script running to show server logs
wait
