# Contributing to arpkit

It's a small project, but suggestions are welcome.

## Prerequisites

- Go 1.22+

## Local development

Run these commands from the repository root.

- Format source code:

```bash
make fmt
```

- Run the test suite:

```bash
make test
# or to run with the race detector
go test -race ./...
```

- Run linters (if `golangci-lint` is available):

```bash
make lint
# or
golangci-lint run ./...
```

- Tidy module files:

```bash
go mod tidy
```

- Build the binary locally:

```bash
make build
# or
CGO_ENABLED=0 go build -trimpath -o bin/arpkit ./cmd/arpkit
```

## Quick commands reference

- `make fmt` — run `gofmt` and apply formatting.
- `make test` — run tests for repository.
- `make lint` — run `golangci-lint` (if configured).
- `go mod tidy` — clean up `go.mod` and `go.sum`.

## Running a single package tests

```bash
go test ./internal/render -run TestSomething -v
```

## Pull request checklist

Before opening a PR, please ensure the following:

- [ ] Fork and branch from `main` using a descriptive branch name.
- [ ] Run `make fmt` and commit formatting changes.
- [ ] Run `make lint` and fix any issues reported (or document why exceptions are required).
- [ ] Run `make test` and ensure tests pass locally.
- [ ] Add or update tests for new behavior or bug fixes.
- [ ] Update documentation or README examples if behavior or flags change.
- [ ] Keep commits small and focused; squash WIP commits when appropriate.
