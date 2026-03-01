.PHONY: all build run-server run-client test proto clean

# Default target: build
all: build

# Build server and client binaries
build:
	mkdir -p bin/
	go build -o bin/server ./server
	go build -o bin/client ./client

# Run server (foreground)
run-server: build
	./bin/server

# Run client test
run-client: build
	./bin/client

# Test: start server (background), run client, then clean up
test: build
	./bin/server & \
	SERVER_PID=$$!; \
	sleep 4; \
	./bin/client; \
	kill $$SERVER_PID 2>/dev/null || true

# Regenerate protobuf files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/service.proto

# Clean build artifacts
clean:
	rm -rf bin/
