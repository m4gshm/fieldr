.PHONY: all
all: baseMake generate test lint

.PHONY: baseMake
baseMake:
	cd ../ && make

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: test
test:
	go test ./...

.PHONY: bench
bench:
	go test -bench . -benchtime 1s ./...

.PHONY: lint-install
lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: lint-install
	golangci-lint run

.PHONY: test-postgres
test-integration:
	go test -tags postgres ./...