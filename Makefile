.PHONY: build test run clean docker-build

APP_NAME = server
PORT ?= 8080
DB_PATH ?= dataset/durg_voters.duckdb

build:
	@echo "Building production Go binary..."
	go build -ldflags="-w -s" -o $(APP_NAME) main.go

test:
	@echo "Running test suite..."
	go test -v ./...

run: build
	@echo "Starting Durg Voter REST API server..."
	PORT=$(PORT) DB_PATH=$(DB_PATH) ./\$(APP_NAME)

docker-build:
	@echo "Building production Docker image..."
	docker build -t durg-voter-api:latest .

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(APP_NAME)
