.PHONY: all
all: build test

.PHONY: test
test:
	$(info #Running tests...)
	go test

.PHONY: build
build:
	$(info #Building...)
	go install