CGO_ENABLED = 1
COMMIT_HASH := $(shell git --no-pager describe --tags --always --dirty)
LDFLAGS = "-X github.com/stumble/v8runner/internal/info.Version=$(COMMIT_HASH)"

.PHONY: build install-v8runner test vet fmt

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags=$(LDFLAGS) -o bin/ ./cmd/...

install-v8runner:
	CGO_ENABLED=$(CGO_ENABLED) go install -ldflags=$(LDFLAGS) ./cmd/v8runner/...

test:
	CGO_ENABLED=$(CGO_ENABLED) go test ./...

vet:
	CGO_ENABLED=$(CGO_ENABLED) go vet ./...

fmt:
	go fmt ./...

.PHONY: lint lint-fix
lint:
	@echo "--> Running linter"
	@golangci-lint run

lint-fix:
	@echo "--> Running linter auto fix"
	@golangci-lint run --fix
