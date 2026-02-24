# Shadowplay for Mac Makefile

BINARY_NAME=shadowplay
SWIFT_BIN=screencapturekit-go/build/screencapturekit

.PHONY: all build build-swift install clean run

all: build

build-swift:
	@echo "🔨 Building Swift components..."
	@cd screencapturekit-go && make build
	@echo "✅ Swift components ready."

build: build-swift
	@echo "🔨 Building Go application..."
	@go build -o $(BINARY_NAME) cmd/shadowplay/main.go
	@echo "✅ Shadowplay built."

install: build
	@echo "🚀 Installing binary to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@sudo cp $(SWIFT_BIN) /usr/local/bin/
	@echo "✅ Done. You can now run 'shadowplay' from anywhere."

run: build
	@echo "🚀 Starting Shadowplay..."
	@./$(BINARY_NAME)

clean:
	@rm -f $(BINARY_NAME)
	@cd screencapturekit-go && make clean-all
	@echo "🧹 Cleaned."
