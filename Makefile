.PHONY: all
all: build test lint

.PHONY: test
test:
	$(info #Running tests...)
	go clean -testcache
	go test

.PHONY: build
build:
	$(info #Building...)
	go install

.PHONY: lint
lint:
	$(info #Lints...)
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .
	go vet ./...
	go install github.com/tetafro/godot/cmd/godot@latest
	godot ./:
	go install github.com/kisielk/errcheck@latest
	errcheck ./...
	go install github.com/alexkohler/nakedret@latest
	nakedret ./...
	go install golang.org/x/lint/golint@latest
	golint ./...
