# Mono-repo orchestration Makefile
.PHONY: all clean test docs-serve docs-build docs-deploy build-binary build-all install-deps help

# Default target - build everything
all: build-binary docs-build

# Clean all projects
clean:
	@echo "Cleaning go build artifacts..."
	@$(MAKE) -C src clean
	@echo "Cleaning docs..."
	@$(MAKE) -C docs clean

# Run all tests
test:
	@$(MAKE) -C src test

# Documentation targets
docs-serve:
	@$(MAKE) -C docs serve

docs-serve-dev:
	@$(MAKE) -C docs serve-dev

docs-build:
	@$(MAKE) -C src docs
	@$(MAKE) -C docs build

docs-deploy:
	@$(MAKE) -C docs deploy

docs-validate:
	@$(MAKE) -C docs validate

# Binary targets
build-binary:
	@$(MAKE) -C src build

build-all:
	@$(MAKE) -C src build-all

run-binary:
	@$(MAKE) -C src run

# Install dependencies for all projects
install-deps:
	@$(MAKE) -C src modtidy
	@$(MAKE) -C docs install-deps

# Help target
help:
	@echo "Available targets:"
	@echo "  all              - Build binary and docs"
	@echo "  clean            - Clean all projects"
	@echo "  test             - Run Go tests"
	@echo "  build-binary     - Build Go binary for current platform"
	@echo "  build-all        - Build Go binary for all platforms"
	@echo "  run-binary       - Run Go application"
	@echo "  docs-serve       - Serve docs locally"
	@echo "  docs-serve-dev   - Serve docs on 0.0.0.0:8000"
	@echo "  docs-build       - Build static docs"
	@echo "  docs-deploy      - Deploy docs"
	@echo "  docs-validate    - Validate docs"
	@echo "  install-deps     - Install dependencies for all projects"
	@echo "  help             - Show this help"
