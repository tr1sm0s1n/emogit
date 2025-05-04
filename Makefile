GOFILES := $(shell find . -name "*.go")
BIN_DIR := $(shell pwd)/bin

vet:
	@go vet ./...

fmt:
	@gofmt -s -w $(GOFILES)

lint:
	@if [ ! -f "$(BIN_DIR)/golangci-lint" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s latest; \
	fi
	@$(BIN_DIR)/golangci-lint run --config .golangci.yaml

release:
	@if [ ! -f "$(BIN_DIR)/goreleaser" ]; then \
		echo "Installing goreleaser..."; \
		GOBIN=$(BIN_DIR) go install github.com/goreleaser/goreleaser/v2@latest; \
	fi
	@$(BIN_DIR)/goreleaser release --clean
