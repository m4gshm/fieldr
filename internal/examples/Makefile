.PHONY: all
all: baseMake generate test

.PHONY: baseMake
baseMake:
	cd ../../ && make

.PHONY: generate
generate:
	GOEXPERIMENT=rangefunc go generate -v ./...

.PHONY: test
test:
	GOEXPERIMENT=rangefunc go test ./...

.PHONY: bench
bench:
	GOEXPERIMENT=rangefunc go test -bench . -benchtime 1s ./...

.PHONY: integration-test
integration-test:
	GOEXPERIMENT=rangefunc go test -tags postgres ./...