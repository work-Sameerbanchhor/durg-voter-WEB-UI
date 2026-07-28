.PHONY: build test run clean docker-build deploy-cloudrun

APP_NAME = server
PORT ?= 7860
DB_PATH ?= backend/database/durg_voters.duckdb
GCP_PROJECT ?= kalyan-db-backup
GCP_REGION ?= asia-south1
SERVICE_NAME ?= durg-voter-api
GEMINI_API_KEY ?= $(shell grep GEMINI_API_KEY .env 2>/dev/null | cut -d '=' -f2 | tr -d ' "')

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

deploy-cloudrun:
	@echo "Deploying $(SERVICE_NAME) to Google Cloud Run (Project: $(GCP_PROJECT), Region: $(GCP_REGION))..."
	gcloud config set project $(GCP_PROJECT)
	gcloud run deploy $(SERVICE_NAME) \
		--quiet \
		--source . \
		--region $(GCP_REGION) \
		--platform managed \
		--allow-unauthenticated \
		--memory 1Gi \
		--cpu 1 \
		--set-env-vars "ENVIRONMENT=production,DB_PATH=/app/backend/database/durg_voters.duckdb,CORS_ALLOWED_ORIGINS=*,GEMINI_API_KEY=$(GEMINI_API_KEY)"

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(APP_NAME)

