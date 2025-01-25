TRACKER_BINARY_PATH=bin/tracker

build:
	@echo "Building the tracker..."
	@mkdir -p bin
	@go build -o $(TRACKER_BINARY_PATH)
run: build
	@echo "Running the tracker..."
	@./$(TRACKER_BINARY_PATH)