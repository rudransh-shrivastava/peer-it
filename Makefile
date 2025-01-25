CLIENT_BINARY_PATH = bin/client
TRACKER_BINARY_PATH = bin/tracker
PROTO_FILE = internal/shared/protocol/protocol.proto
PROTO_GO = internal/shared/protocol/protocol.pb.go

# Protocol buffer generation
.PHONY: protoc
protoc: $(PROTO_GO)

$(PROTO_GO): $(PROTO_FILE)
	@echo "Generating protocol buffers..."
	@protoc --go_out=. --go_opt=paths=source_relative $<

# Client targets
.PHONY: client
client: protoc
		@echo "Building client..."
		@mkdir -p bin
		@go build -o $(CLIENT_BINARY_PATH) ./cmd/client

.PHONY: run-client
run-client: client
		@echo "Running client..."
		@./$(CLIENT_BINARY_PATH)

# Tracker targets
.PHONY: tracker
tracker: protoc
		@echo "Building tracker..."
		@mkdir -p bin
		@go build -o $(TRACKER_BINARY_PATH) ./cmd/tracker

.PHONY: run-tracker
run-tracker: tracker
		@echo "Running tracker..."
		@./$(TRACKER_BINARY_PATH)