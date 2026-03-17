VENDOR_DIR = local/public/vendor

# ProseMirror packages — each has dist/index.js
PM_PACKAGES = model transform state view keymap inputrules history \
              commands schema-basic schema-list collab

# Third-party deps with non-standard paths
VENDOR_MAP = orderedmap:dist/index.js \
             w3c-keyname:index.js \
             rope-sequence:dist/index.js \
             htmx.org:dist/htmx.min.js

.PHONY: vendor vendor-clean build test

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
	go build ./...

test:
	go test ./...
