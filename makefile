APP_NAME   := whatsupbot
CMD_DIR    := ./cmd/bot
MCP_CMD_DIR := ./cmd/mcp
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(APP_NAME)
MCP_BIN    := $(BIN_DIR)/$(APP_NAME)-mcp

GO         ?= go
GOFLAGS    ?=
LDFLAGS    ?= -s -w
# sqlite3 driver (mattn/go-sqlite3) needs cgo
CGO_ENABLED ?= 1

# Cross-compile target for deploy (override: make build-linux GOARCH=arm64)
GOOS_LINUX ?= linux
GOARCH     ?= amd64

# Load local env vars (GEMINI_API_KEY, GEMINI_MODEL, ...) if present
ENV_FILE   ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the bot locally
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(CMD_DIR)

.PHONY: run-mcp
run-mcp: ## Run the MCP server locally
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(MCP_CMD_DIR)

.PHONY: build
build: ## Build bot and MCP server binaries into bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(MCP_BIN) $(MCP_CMD_DIR)

.PHONY: build-linux
build-linux: ## Cross-compile for linux (requires a cgo cross toolchain, e.g. zig cc)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN)-$(GOOS_LINUX)-$(GOARCH) $(CMD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(MCP_BIN)-$(GOOS_LINUX)-$(GOARCH) $(MCP_CMD_DIR)

.PHONY: test
test: ## Run tests with race detector
	CGO_ENABLED=1 $(GO) test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests with coverage report
	@mkdir -p $(BIN_DIR)
	$(GO) test -coverprofile=$(BIN_DIR)/coverage.out ./...
	$(GO) tool cover -func=$(BIN_DIR)/coverage.out

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: tidy
tidy: ## Tidy and verify go modules
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: check
check: fmt vet lint test ## Run fmt, vet, lint and test

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) cmd/bot/bot cmd/mcp/mcp

.PHONY: reset-db
reset-db: ## Delete local app DB (keeps WhatsApp session)
	rm -f whatsup.db whatsup.db-wal whatsup.db-shm

.PHONY: reset-session
reset-session: ## Delete WhatsApp session store (forces QR re-login)
	rm -f whatsmeow.db
