# macOS only — requires Xcode Command Line Tools
BINARY := shadowplay
PKG := ./cmd/shadowplay

.PHONY: build run-buffer clean

build:
	CGO_ENABLED=1 go build -o $(BINARY) $(PKG)

run-buffer: build
	./$(BINARY) buffer

clean:
	rm -f $(BINARY)
