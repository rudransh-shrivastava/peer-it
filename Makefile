TRACKER_BINARY_PATH=tracker/bin/tracker
CLIENT_BINARY_PATH=client/bin/client

run:
	@echo "Building the tracker..."
	@mkdir -p tracker/bin
	@go build -o $(TRACKER_BINARY_PATH) ./tracker/
	@echo "Running the tracker..."
	@./$(TRACKER_BINARY_PATH)

build:
	@echo "Building the client..."
	@mkdir -p client/bin
	@go build -o $(CLIENT_BINARY_PATH) ./client/
	@echo "Running the client..."
	@./$(CLIENT_BINARY_PATH)