.PHONY: help build run test clean deps vet fmt docker-build docker-run docker-push compose-up compose-down compose-logs

# Binary name
BINARY_NAME=kickcraft
VERSION?=1.0
REGISTRY_USER?=jetfuls

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build parameters
BINARY_DIR=.
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64

# Docker parameters
DOCKER_IMAGE=$(REGISTRY_USER)/kickcraft:$(VERSION)
ROCKY_VERSION?=9
COMPOSE_FILE=docker-compose.yml

help: ## Display this help message
	@echo "KickCraft Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the project and place the binary in the current directory
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o $(BINARY_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

run: ## Run the project
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run main.go cmd/cmd.go

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

deps: ## Install project dependencies
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

vet: ## Run static analysis
	@echo "Running vet..."
	$(GOVET) ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

clean: ## Clean up build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BINARY_DIR)/$(BINARY_NAME)
	rm -rf docs/swagger.*

package: ## Archive artifacts (zip for Linux, tar.gz for others)
	@echo "Packaging..."
ifeq ($(GOOS),linux)
	zip -r $(BINARY_NAME)-$(VERSION)-linux-amd64.zip $(BINARY_NAME)
else
	tar -czvf $(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz $(BINARY_NAME)
endif

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	swag init -g main.go -o docs

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build --build-arg ROCKY_VERSION=$(ROCKY_VERSION) -t $(DOCKER_IMAGE) .
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE)-rocky$(ROCKY_VERSION)

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -itd -p 8080:8080 --name kickcraft $(DOCKER_IMAGE)-rocky$(ROCKY_VERSION)

docker-push: ## Push to registry
	@echo "Pushing to registry..."
	docker push $(DOCKER_IMAGE)-rocky$(ROCKY_VERSION)

compose-up: ## Start with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose -f $(COMPOSE_FILE) up -d

compose-down: ## Stop with docker-compose
	@echo "Stopping services..."
	docker-compose -f $(COMPOSE_FILE) down

compose-logs: ## View logs
	docker-compose -f $(COMPOSE_FILE) logs -f
