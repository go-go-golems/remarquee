.PHONY: dev-backend dev-frontend build clean install test help

# Development: run backend + frontend in separate terminals
dev-backend:
	go run main.go --dev

dev-frontend:
	cd frontend && npm install && npm run dev

# Production build: bundle frontend + embed in Go binary
build:
	@echo "Building frontend..."
	cd frontend && npm install && npm run build
	@echo "Building Go binary..."
	go build -o remarquee-ui main.go
	@echo "Done: ./remarquee-ui"

# Clean build artifacts
clean:
	rm -rf frontend/dist frontend/node_modules outputs/*.pdf
	rm -f remarquee-ui

# Install dependencies
install:
	cd frontend && npm install
	go mod download

# Run tests
test:
	go test ./api/...
	cd frontend && npm run test

# Help
help:
	@echo "remarquee-ui Makefile targets:"
	@echo "  dev-backend    - Run Go server in dev mode (port 8080)"
	@echo "  dev-frontend   - Run Vite dev server (port 5173)"
	@echo "  build          - Build production binary with embedded assets"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install all dependencies"
	@echo "  test           - Run all tests"

