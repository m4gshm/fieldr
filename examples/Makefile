.PHONY: all
all: baseMake generate test

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

.PHONY: integration-test
integration-test:
	go test -tags postgres ./...