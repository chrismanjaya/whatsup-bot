BINARY_NAME := whatsup-bot
CMD_PATH    := ./cmd/bot
BUILD_DIR   := ./bin

# GCP VM connection details (adjust to your actual values)
VM_USER := your_ssh_user
VM_HOST := your_vm_external_ip
VM_PATH := ~/whatsup-bot

.PHONY: all build run test tidy fmt vet clean build-linux deploy restart logs

all: build

## Build binary for local development (matches your current OS/arch)
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

## Run the bot locally (foreground, for QR pairing / local testing)
run:
	go run $(CMD_PATH)

## Run all tests
test:
	go test ./... -v

## Tidy go.mod / go.sum
tidy:
	go mod tidy

## Format all Go files
fmt:
	gofmt -l -w .

## Run go vet for static checks
vet:
	go vet ./...

## Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## Cross-compile for the GCP VM (Linux amd64, matches e2-micro)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(CMD_PATH)

## Copy the compiled Linux binary + restart the systemd service on the VM
deploy: build-linux
	scp $(BUILD_DIR)/$(BINARY_NAME)-linux $(VM_USER)@$(VM_HOST):$(VM_PATH)/$(BINARY_NAME)
	ssh $(VM_USER)@$(VM_HOST) "sudo systemctl restart $(BINARY_NAME)"

## Restart the service on the VM without redeploying (e.g. after config change)
restart:
	ssh $(VM_USER)@$(VM_HOST) "sudo systemctl restart $(BINARY_NAME)"

## Tail the service logs on the VM
logs:
	ssh $(VM_USER)@$(VM_HOST) "sudo journalctl -u $(BINARY_NAME) -f"