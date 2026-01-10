CLIENT_BINARY = bin/client

build: build-client

build-client:
	@mkdir -p bin
	@go build -o $(CLIENT_BINARY) ./cmd/client

check:
	@pre-commit run -a

check-test: \
	check \
	test

clean:
	@rm -rf bin

generate: sqlc

run: build-client
	@./$(CLIENT_BINARY)

sqlc:
	@sqlc generate

test:
	@go test ./... -v
