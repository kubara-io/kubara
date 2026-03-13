# Documentation Makefile for MkDocs
DOCS_DIR = .
SITE_DIR = site

.PHONY: serve serve-dev build deploy clean install-deps update-deps config validate get-deps new-page help

# Serve documentation locally
serve:
	@echo "Starting documentation server..."
	@cd $(DOCS_DIR) && pipenv run mkdocs serve

# Serve documentation on all interfaces (useful for Docker/VMs)
serve-dev:
	@echo "Starting documentation server on 0.0.0.0:8000..."
	@cd $(DOCS_DIR) && pipenv run mkdocs serve --dev-addr=0.0.0.0:8000

# Build static documentation
build:
	@echo "Building documentation..."
	@cd $(DOCS_DIR) && pipenv run mkdocs build

# Clean generated files
clean:
	@echo "Cleaning documentation build..."
	@cd $(DOCS_DIR) && rm -rf $(SITE_DIR)

# Install Python dependencies
install-deps:
	@echo "Installing documentation dependencies..."
	@cd $(DOCS_DIR) && pipenv install

# Update Python dependencies
update-deps:
	@echo "Updating documentation dependencies..."
	@cd $(DOCS_DIR) && pipenv update

# Show configuration
config:
	@cd $(DOCS_DIR) && pipenv run mkdocs config --quiet

# Show required dependencies from plugins
get-deps:
	@cd $(DOCS_DIR) && pipenv run mkdocs get-deps

# Validate documentation (strict mode)
validate:
	@echo "Validating documentation with strict mode..."
	@cd $(DOCS_DIR) && pipenv run mkdocs build --strict --quiet

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
	@echo "  serve          - Serve docs locally"
	@echo "  serve-dev      - Serve docs on 0.0.0.0:8000"
	@echo "  build          - Build static docs"
	@echo "  clean          - Clean build files"
	@echo "  install-deps   - Install Python dependencies"
	@echo "  update-deps    - Update Python dependencies"
	@echo "  config         - Show MkDocs config"
	@echo "  get-deps       - Show required PyPI packages from plugins"
	@echo "  validate       - Validate docs with strict mode"
	@echo "  new-page       - Create new page (PAGE=path/to/page.md)"
	@echo "  help           - Show this help"
