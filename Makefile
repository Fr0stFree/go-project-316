GOLANGCI_LINT_VERSION := v2.12.2
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
GOLANGCI_LINT_CONFIG := .golangci.yml

BINARY_PATH := ./bin/crawler
ENTRYPOINT_PATH := ./cmd/crawler
COVERAGE_PROFILE := coverage.out

.PHONY: \
	install \
	start \
	build \
	run \
	test \
	test-coverage \
	lint \
	lint-fix \
	fmt \
	fmt-check \
	tidy-check \
	install-lint \
	require-lint 

build:
	@go build -o $(BINARY_PATH) $(ENTRYPOINT_PATH)

start: build
	@$(BINARY_PATH) $(ARGS)

install:
	@go mod download

run:
	@$(BINARY_PATH) $(ARGS)

test:
	@go test ./... -v

test-coverage:
	@go test ./... -coverprofile=$(COVERAGE_PROFILE)
	@go tool cover -func=$(COVERAGE_PROFILE)

install-lint:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

require-lint:
	@test -x $(GOLANGCI_LINT) || (echo "golangci-lint not found. Run 'make install-lint' first."; exit 1)

lint: require-lint
	@$(GOLANGCI_LINT) run $(ARGS) --config $(GOLANGCI_LINT_CONFIG)

fmt: require-lint
	@$(GOLANGCI_LINT) fmt --config $(GOLANGCI_LINT_CONFIG)

fmt-check: require-lint
	@$(GOLANGCI_LINT) fmt --diff --config $(GOLANGCI_LINT_CONFIG)

tidy-check:
	@go mod tidy -diff

lint-fix: require-lint
	@$(MAKE) fmt
	@$(MAKE) lint ARGS="--fix"
