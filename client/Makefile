CLIENT_BINARY_PATH=bin/client

build:
	@echo "Building the client..."
	@mkdir -p bin
	@go build -o $(CLIENT_BINARY_PATH)
run: build
	@echo "Running the client..."
	@./$(CLIENT_BINARY_PATH)
