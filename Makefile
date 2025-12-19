BINARY_NAME := cleanup-events

.PHONY: install
install: ## Install the binary 
	@CGO_ENABLED=0  go build -o $(shell go env GOPATH)/bin/${BINARY_NAME} -ldflags="-s -w"