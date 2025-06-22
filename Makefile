# Variables
BINARY_NAME=terraform-file-organize

.PHONY: dev-build
dev-build:
	go build -o $(BINARY_NAME) .

.PHONY: dev-install
dev-install:
	go install .

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: lint
lint:
	golangci-lint run

.PHONY: check
check: lint test

# GoReleaser targets
.PHONY: release-check
release-check:
	goreleaser check

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

.PHONY: release-local
release-local: release-snapshot

.PHONY: release
release:
	@echo "Releases are automated via GitHub Actions when tags are pushed"
	@echo "To create a release:"
	@echo "  git tag v1.0.0"
	@echo "  git push origin v1.0.0"

.PHONY: help
help:
	@echo "Development targets:"
	@echo "  dev-build         - Build binary for development (no version info)"
	@echo "  dev-install       - Install binary for development"
	@echo "  clean             - Remove built binaries and dist directory"
	@echo ""
	@echo "Testing targets:"
	@echo "  test              - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  lint              - Run linter"
	@echo "  check             - Run lint and test"
	@echo ""
	@echo "Release targets (GoReleaser):"
	@echo "  release-check     - Check GoReleaser configuration"
	@echo "  release-snapshot  - Build release binaries locally (snapshot)"
	@echo "  release-local     - Alias for release-snapshot"
	@echo "  release           - Show instructions for creating releases"
	@echo ""
	@echo "  help              - Show this help message"

.DEFAULT_GOAL := dev-build