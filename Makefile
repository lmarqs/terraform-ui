VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := dist/tfui

.PHONY: build test lint clean install

build:
	@mkdir -p dist
	go build $(LDFLAGS) -o $(BINARY) ./cmd/tfui

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf dist

install:
	go install $(LDFLAGS) ./cmd/tfui
