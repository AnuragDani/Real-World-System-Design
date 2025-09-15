#!/bin/bash

# Connection Pool Benchmark Setup Script

set -e

echo "Setting up Connection Pool Benchmark..."

# Create project directory
PROJECT_DIR="connection-pool-benchmark"
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

echo "✓ Created project directory: $PROJECT_DIR"

# Initialize Go module if go.mod doesn't exist
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init connection-pool-benchmark
    echo "✓ Go module initialized"
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod tidy
echo "✓ Dependencies downloaded"

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "✓ Docker found"
    
    # Check if docker-compose is available
    if command -v docker-compose &> /dev/null; then
        echo "✓ Docker Compose found"
        echo ""
        echo "=== QUICK START ==="
        echo ""
        echo "Option 1: Run with Docker Compose (includes databases):"
        echo "  docker-compose up --build"
        echo ""
        echo "Option 2: Run locally (requires database setup):"
        echo "  go run main.go -dsn 'postgres://user:pass@localhost:5432/db?sslmode=disable'"
        echo ""
    else
        echo "! Docker Compose not found. You can still use Docker manually."
        echo ""
        echo "Manual Docker setup:"
        echo "1. docker run --name postgres-db -e POSTGRES_DB=testdb -e POSTGRES_USER=testuser -e POSTGRES_PASSWORD=testpass -p 5432:5432 -d postgres:15-alpine"
        echo "2. docker build -t connection-benchmark ."
        echo "3. docker run --rm --link postgres-db connection-benchmark ./main"
    fi
else
    echo "! Docker not found. Local setup required."
    echo ""
    echo "Local setup:"
    echo "1. Install and start PostgreSQL or MySQL"
    echo "2. Create database and user"
    echo "3. Run: go run main.go -dsn 'your-connection-string'"
fi

echo ""
echo "=== FILES CREATED ==="
echo "✓ main.go - Main application code"
echo "✓ go.mod - Go module definition"
echo "✓ Dockerfile - Docker container definition"
echo "✓ docker-compose.yml - Multi-service Docker setup"
echo "✓ README.md - Documentation"
echo ""
echo "Setup complete! Check README.md for detailed usage instructions."