VENDOR_DIR = local/public/vendor

# ProseMirror packages — each has dist/index.js
PM_PACKAGES = model transform state view keymap inputrules history \
              commands schema-basic schema-list collab

# Third-party deps with non-standard paths
VENDOR_MAP = orderedmap:dist/index.js \
             w3c-keyname:index.js \
             rope-sequence:dist/index.js \
             htmx.org:dist/htmx.min.js

# Site directory — override to run multiple sites
# Usage: make dev SITE=examples/photoblog
SITE = local

.PHONY: vendor vendor-clean build test lint fmt dev stop stop-all serve reset fixtures fresh press clean

vendor: node_modules
	@mkdir -p $(VENDOR_DIR)
	@for pkg in $(PM_PACKAGES); do \
		cp node_modules/prosemirror-$$pkg/dist/index.js $(VENDOR_DIR)/prosemirror-$$pkg.js; \
	done
	@for entry in $(VENDOR_MAP); do \
		pkg=$${entry%%:*}; \
		path=$${entry##*:}; \
		cp node_modules/$$pkg/$$path $(VENDOR_DIR)/$$pkg.js; \
	done
	@echo "Vendored $$(ls $(VENDOR_DIR)/*.js | wc -l) files to $(VENDOR_DIR)/"

vendor-clean:
	rm -rf $(VENDOR_DIR)

node_modules: package.json
	npm install

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
