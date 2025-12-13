CLIENT_BINARY = bin/client
TRACKER_BINARY = bin/tracker
PROTO_FILE = internal/shared/protocol/protocol.proto
PROTO_OUT = internal/shared/protocol/protocol.pb.go

build: build-client build-tracker

build-client: protoc
	@mkdir -p bin
	@go build -o $(CLIENT_BINARY) ./cmd/client

build-tracker: protoc
	@mkdir -p bin
	@go build -o $(TRACKER_BINARY) ./cmd/tracker

check:
	@pre-commit run -a

clean:
	@rm -rf bin

protoc: $(PROTO_OUT)

run-client: build-client
	@./$(CLIENT_BINARY)

run-daemon: build-client
	@./$(CLIENT_BINARY) daemon

run-tracker: build-tracker
	@./$(TRACKER_BINARY)

$(PROTO_OUT): $(PROTO_FILE)
	@protoc --go_out=. --go_opt=paths=source_relative $<
