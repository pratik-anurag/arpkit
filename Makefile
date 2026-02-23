APP := arpkit
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test fmt run release-snapshot clean

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/arpkit

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

run:
	go run ./cmd/arpkit

release-snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
