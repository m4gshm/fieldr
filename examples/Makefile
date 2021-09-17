.PHONY: all
all: baseMake generate test lint bench

.PHONY: baseMake
baseMake:
	cd ../ && make

.PHONY: generate
generate:
	$(info #Generate...)
	go generate -v

.PHONY: test
test:
	$(info #Test...)
	go test ./...


.PHONY: bench
bench:
	$(info #Benchmark...)
	go test -bench . -benchtime 5s ./...

.PHONY: lint-install
lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: lint-install
	$(info #Lint...)
	golangci-lint run