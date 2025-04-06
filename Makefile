
lint: .golangci.yml
	golangci-lint run


local-test:
	go test -race ./...

OS:=$(shell go env GOOS)

test:
ifeq ($(OS),darwin)
	@echo "Skipping some tests on Mac CI runner as something is off with timing, threads or the filesystem there."
	go test -race . ./dynloglevel ./endpoint ./examples/...
else
	go test -race ./...
endif

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: lint
