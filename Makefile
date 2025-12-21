BINARY_NAME := cleanup-events
REGISTRY := ghcr.io/martinweindel/cleanup-events
IMAGE_NAME := cleanup-events
TAG ?= latest

.PHONY: install docker-image
install: ## Install the binary
	@CGO_ENABLED=0  go build -o $(shell go env GOPATH)/bin/${BINARY_NAME} -ldflags="-s -w"

docker-image-local: ## Build Docker image for local architecture
	@docker build -t $(REGISTRY)/$(IMAGE_NAME):$(TAG) .

docker-image-push: ## Build Docker image for amd64 and arm64
	@docker buildx build --platform linux/amd64,linux/arm64 -t $(REGISTRY)/$(IMAGE_NAME):$(TAG) --push .


.PHONY: install
install: ## Install the binary 
	@CGO_ENABLED=0  go build -o $(shell go env GOPATH)/bin/${BINARY_NAME} -ldflags="-s -w"