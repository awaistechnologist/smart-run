
.PHONY: all build run clean help

# Binary name
BINARY_NAME=smartrund
CLI_BINARY_NAME=smart-run

all: build

build: ## Build the server and CLI binaries
	@echo "Building server..."
	go build -o $(BINARY_NAME) ./cmd/smartrund
	@echo "Building CLI..."
	go build -o $(CLI_BINARY_NAME) ./cmd/smart-run

run: build ## Build and run the server
	@echo "Starting server..."
	./$(BINARY_NAME) --port 8080

init: ## Initialize the database
	go run ./cmd/smart-run init

clean: ## Remove binaries
	rm -f $(BINARY_NAME) $(CLI_BINARY_NAME)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
