.DEFAULT_GOAL := build

.PHONY: fmt vet build clean test test-verbose

fmt:
	go fmt ./...

vet:
	go vet ./...

build: vet
	go build ./...

clean:
	go clean
	go clean -testcache

test: build
	go clean -testcache && go test -cover ./...

test-verbose: build
	go clean -testcache && go test -v -cover ./...
