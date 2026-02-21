VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := manifesto
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install clean

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/manifesto

install:
	go install $(LDFLAGS) ./cmd/manifesto

clean:
	rm -rf bin/

release:
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64  ./cmd/manifesto
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64  ./cmd/manifesto
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64   ./cmd/manifesto
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/manifesto
