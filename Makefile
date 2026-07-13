.PHONY: help build test vet fmt install clean

BIN_DIR := bin
BINARIES := ht

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the ht binary into bin/
	@mkdir -p $(BIN_DIR)
	@for b in $(BINARIES); do go build -o $(BIN_DIR)/$$b ./cmd/$$b; done

test: ## Run the test suite
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Check formatting (gofmt)
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

install: ## Install all binaries to $GOBIN (or $GOPATH/bin)
	go install ./cmd/...

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	go clean
