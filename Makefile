# Site directory — override to run multiple sites
# Usage: make dev SITE=examples/photoblog
SITE = local

.PHONY: assets build test lint fmt dev stop stop-all serve reset fixtures fresh press clean

assets: build
	cd $(SITE) && ../press assets

build:
	go build -o press ./cmd/press

# Start the development server with file watching
dev: build
	@bin/dev $(SITE)

# Stop the development server for a SITE
stop:
	@bin/stop $(SITE)

# Stop all running development servers
stop-all:
	@bin/stop-all

# Start the development server (no file watching)
serve: build
	cd $(SITE) && ../press serve

# Drop database and re-run migrations
reset: build
	cd $(SITE) && ../press db reset

# Load fixture data into the database
fixtures:
	go run ./data/fixtures/ $(SITE)/storage/press.db

# Reset and reload: drop, migrate, load fixtures
fresh: reset fixtures

# Run press CLI commands (usage: make press CMD="user list")
press: build
	cd $(SITE) && ../press $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	find . -name '*.go' -not -path './internal/template/parse/*' | xargs gofmt -w

clean:
	rm -f press
	rm -rf tmp/pids
