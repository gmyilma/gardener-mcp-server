# SPDX-FileCopyrightText: 2026 Girma Yilma
#
# SPDX-License-Identifier: Apache-2.0

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

MODULE  := github.com/gmyilma/gardener-mcp-server
BINARY  := gardener-mcp-server
BIN_DIR := $(CURDIR)/bin
DIST    := $(CURDIR)/dist

# Dev tools install here rather than ~/go/bin, so every checkout gets the
# versions pinned in go.mod's `tool` directives. Same idea as a virtualenv's
# bin/ — see .mise.toml.
export GOBIN := $(BIN_DIR)

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

##@ General

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1m%s\033[0m\n", "gardener-mcp-server"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""

##@ Setup

# Binary dev tools (golangci-lint, goreleaser, syft, cosign) are pinned in
# .mise.toml with checksums in mise.lock. govulncheck is a Go program installed
# with `go install pkg@version`, which deliberately does not touch go.mod.
GOVULNCHECK_VERSION ?= v1.8.0

.PHONY: tools
tools: ## Install pinned dev tools into ./bin
	mise install
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: deps
deps: ## Download and verify module dependencies
	go mod download
	go mod verify

##@ Development

.PHONY: build
build: ## Build the server binary into ./bin
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) ./cmd/$(BINARY)

.PHONY: run
run: ## Run the server in local/stdio mode
	go run ./cmd/$(BINARY) --mode=local

.PHONY: fmt
fmt: ## Format code
	golangci-lint fmt

.PHONY: lint
lint: ## Run linters
	golangci-lint run

.PHONY: generate
generate: ## Run code and schema generation
	go generate ./...

##@ Testing

.PHONY: test
test: ## Run unit tests with race detector and coverage
	go test -race -covermode=atomic -coverprofile=cover.out ./...

.PHONY: cover
cover: test ## Render the coverage report as HTML
	go tool cover -html=cover.out -o cover.html
	@echo "coverage report: cover.html"

.PHONY: test-integration
test-integration: ## Run envtest integration tests
	go test -tags=integration ./test/integration/...

.PHONY: test-e2e
test-e2e: ## Run e2e tests against a local Gardener setup
	go test -tags=e2e -timeout=30m ./test/e2e/...

##@ Verification

.PHONY: vulncheck
vulncheck: ## Scan dependencies for known vulnerabilities
	govulncheck ./...

.PHONY: tidy-check
tidy-check: ## Fail if go.mod/go.sum are not tidy
	@cp go.mod go.mod.bak && cp go.sum go.sum.bak 2>/dev/null || true
	@go mod tidy
	@if ! diff -q go.mod go.mod.bak >/dev/null 2>&1; then \
		echo "go.mod is not tidy — run 'go mod tidy'"; \
		mv go.mod.bak go.mod; mv go.sum.bak go.sum 2>/dev/null || true; exit 1; \
	fi
	@rm -f go.mod.bak go.sum.bak

.PHONY: generate-check
generate-check: generate ## Fail if generated files are out of date
	@if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "generated files are out of date — run 'make generate' and commit:"; \
		git status --porcelain; exit 1; \
	fi

.PHONY: reuse-lint
reuse-lint: ## Check SPDX/REUSE licensing compliance
	@if command -v uvx >/dev/null 2>&1; then uvx reuse lint; \
	elif command -v reuse >/dev/null 2>&1; then reuse lint; \
	else echo "skipped: install with 'uv tool install reuse' or 'pipx install reuse'"; fi

.PHONY: check
check: lint tidy-check vulncheck reuse-lint ## Run every static check
	@echo "all checks passed"

.PHONY: verify
verify: check test ## Everything CI runs on a pull request

##@ Release

.PHONY: snapshot
snapshot: ## Build a local release snapshot without publishing
	goreleaser release --snapshot --clean

##@ Housekeeping

.PHONY: clean
clean: ## Remove build and test artifacts
	rm -rf $(BIN_DIR) $(DIST) cover.out cover.html
