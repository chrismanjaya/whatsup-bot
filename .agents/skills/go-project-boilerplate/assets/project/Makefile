APP_NAME    := app
CMD_DIR     := ./cmd
BIN_DIR     := bin
BIN         := $(BIN_DIR)/$(APP_NAME)

GO          ?= go
GOFLAGS     ?=
LDFLAGS     ?= -s -w
CGO_ENABLED ?= 1   # set to 0 if no dependency needs cgo

GOLANGCI_LINT_VERSION ?= v2.12.2
GOVULNCHECK_VERSION   ?= latest

# Load local env vars if present (never commit .env)
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the app locally
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(CMD_DIR)

.PHONY: build
build: ## Build binary into bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD_DIR)

.PHONY: test
test: ## Run tests with race detector
	CGO_ENABLED=1 $(GO) test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests with coverage report
	@mkdir -p $(BIN_DIR)
	$(GO) test -coverprofile=$(BIN_DIR)/coverage.out ./...
	$(GO) tool cover -func=$(BIN_DIR)/coverage.out

.PHONY: fmt
fmt: ## Format code (gofmt + goimports via golangci-lint)
	golangci-lint fmt ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: vuln
vuln: ## Scan dependencies for known vulnerabilities
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

.PHONY: tools
tools: ## Install pinned dev tools
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: tidy
tidy: ## Tidy and verify go modules
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: check
check: fmt vet lint test vuln ## Everything CI runs

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
