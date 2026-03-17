# Documentation Makefile for MkDocs
DOCS_DIR = .
SITE_DIR = site

.PHONY: sync serve serve-dev build clean update-deps config get-deps validate new-page help

# Sync and install Python dependencies
sync:
	@echo "Syncing documentation dependencies..."
	@cd $(DOCS_DIR) && uv sync

# Update Python dependencies to their latest versions
update-deps:
	@echo "Updating documentation dependencies..."
	@cd $(DOCS_DIR) && uv sync --upgrade

# Serve documentation locally
serve: sync
	@echo "Starting documentation server..."
	@cd $(DOCS_DIR) && uv run mkdocs serve

# Serve documentation on all interfaces (useful for Docker/VMs)
serve-dev: sync
	@echo "Starting documentation server on 0.0.0.0:8000..."
	@cd $(DOCS_DIR) && uv run mkdocs serve --dev-addr=0.0.0.0:8000

# Build static documentation
build: sync
	@echo "Building documentation..."
	@cd $(DOCS_DIR) && uv run mkdocs build

# Clean generated files
clean:
	@echo "Cleaning documentation build..."
	@cd $(DOCS_DIR) && rm -rf $(SITE_DIR)

# Show configuration
config: sync
	@cd $(DOCS_DIR) && uv run mkdocs config --quiet

# Show required dependencies from plugins
get-deps: sync
	@cd $(DOCS_DIR) && uv run mkdocs get-deps

# Validate documentation (strict mode)
validate: sync
	@echo "Validating documentation with strict mode..."
	@cd $(DOCS_DIR) && uv run mkdocs build --strict --quiet

# Create new page (usage: make new-page PAGE=path/to/page.md)
new-page:
	@if [ -z "$(PAGE)" ]; then \
		echo "Usage: make new-page PAGE=path/to/page.md"; \
		exit 1; \
	fi
	@mkdir -p $(DOCS_DIR)/content/$$(dirname $(PAGE))
	@touch $(DOCS_DIR)/content/$(PAGE)
	@echo "# $$(basename $$(basename $(PAGE)))" > $(DOCS_DIR)/content/$(PAGE)
	@echo "Created new page: $(DOCS_DIR)/content/$(PAGE)"

help:
	@echo "Documentation targets:"
	@echo "  sync          - Sync and install Python dependencies"
	@echo "  update-deps   - Update Python dependencies (--upgrade)"
	@echo "  serve         - Serve docs locally (auto-syncs first)"
	@echo "  serve-dev     - Serve docs on 0.0.0.0:8000 (auto-syncs first)"
	@echo "  build         - Build static docs (auto-syncs first)"
	@echo "  clean         - Clean build files"
	@echo "  config        - Show MkDocs config"
	@echo "  get-deps      - Show required PyPI packages from plugins"
	@echo "  validate      - Validate docs with strict mode"
	@echo "  new-page      - Create new page (PAGE=path/to/page.md)"
	@echo "  help          - Show this help"
