.PHONY: all
all: baseMake generate test bench

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