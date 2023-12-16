.DEFAULT_GOAL := help

ROOT_DIR      := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

default: help

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Test
	GO111MODULE=on go test -count=1 -mod=vendor -v ./...

.PHONY: fmt
fmt: ## Go format
	go fmt ./...

.PHONY: vet
vet: ## Go vet
	go vet ./...

.PHONY: lint
lint: ## Lint
	@golangci-lint run

.PHONY: deps
deps: ## Get dependencies
	GO111MODULE=on go get ./...

.PHONY: vendor
vendor: ## Go vendor
	GO111MODULE=on go mod vendor

.PHONY: tidy
tidy: ## Go tidy
	GO111MODULE=on go mod tidy

.PHONY: protoc-gen
protoc-gen: ## Generate protobuf files
	earthly +generate
