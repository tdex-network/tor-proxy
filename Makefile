.PHONY: build clean fmt help

## build: build the binary in ./bin
build: clean
	@echo "Build..."
	@export GO111MODULE=on; \
	go build -o bin/torproxy-`go env GOOS`-`go env GOARCH` ./cmd/*.go

## clean: remove the compiled binaries in ./bin
clean: 
	@echo "Clean..."
	@rm -rf ./bin

## fmt: Go Format
fmt:
	@echo "Gofmt..."
	if [ -n "$(gofmt -l ./...)" ]; then echo "Go code is not formatted"; exit 1; fi

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
