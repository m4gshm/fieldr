.PHONY: all
all: build test lint

.PHONY: test
test:
	$(info #Running tests...)
	go clean -testcache
	go test

.PHONY: build
build: gofmt govet
	$(info #Building...)
	go install

.PHONY: lint-install
lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: lint-install
	$(info #Lint...)
	golangci-lint run

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: govet
govet:
	go vet ./...
