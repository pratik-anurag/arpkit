APP := arpkit
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell git show -s --format=%cI HEAD 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
CGO_ENABLED ?= 0
LDFLAGS := -buildid= -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test fmt run release-snapshot clean

build:
	mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/arpkit

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

run:
	CGO_ENABLED=$(CGO_ENABLED) go run -trimpath ./cmd/arpkit

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish

clean:
	rm -rf bin dist
