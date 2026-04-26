---
name: test
description: Use when the user wants to run, choose, or interpret Press tests and checks across the Go toolchain.
---

# Running Tests and Checks

Press is primarily a Go project. The main checks are Go tests, linting, formatting, and a build.

## Default commands

```bash
make test    # go test ./...
make lint    # golangci-lint run ./...
make fmt     # gofmt on repo Go files
make build   # go build -o press ./cmd/press
```

Use the `make` targets first when they fit. They are the documented entry points for this repo.

## Go tests

Run the full suite:

```bash
go test ./...
```

Run a package:

```bash
go test ./internal/prosemirror
go test ./internal/server
go test ./cmd/press
```

Run a single test by name:

```bash
go test ./internal/prosemirror -run TestPostSchemaRejects
go test ./internal/server -run TestWhatever
```

Use `-count=1` when cached results would hide a fresh failure:

```bash
go test ./internal/prosemirror -count=1
```

## Lint and format

Lint the repo:

```bash
golangci-lint run ./...
```

Format Go files:

```bash
make fmt
```

If you changed Go code, run formatting before finishing unless there is a reason not to.

## Build and local verification

Build the binary:

```bash
make build
```

Run the local app:

```bash
make dev
make serve
```

The default site directory is `local`. Override with `SITE=...` when needed.

## Check order before commit

For most Go changes, the normal order is:

```bash
make fmt
make test
make lint
make build
```

If the change is narrow, prefer the smallest relevant package tests first, then broaden out if needed.
