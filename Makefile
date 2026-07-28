.PHONY: build test run clean docker-build

APP_NAME = server
PORT ?= 7860
DB_PATH ?= backend/database/durg_voters.duckdb

build:
	@echo "Building production Go binary..."
	cd backend && go build -ldflags="-w -s" -o ../$(APP_NAME) main.go

test:
	@echo "Running test suite..."
	cd backend && go test -v ./...

run: build
	@echo "Starting Durg Voter REST API server..."
	PORT=$(PORT) DB_PATH=$(DB_PATH) ./$(APP_NAME)

docker-build:
	@echo "Building production Docker image..."
	docker build -t durg-voter-api:latest .

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(APP_NAME)
