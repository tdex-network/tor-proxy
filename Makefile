.PHONY: build clean fmt help

## build: build the binary in ./bin
build: clean
	@echo "Build..."
	env go build -tags netgo -ldflags="-s -w" -o bin/torproxy cmd/*

## clean: remove the compiled binaries in ./bin
clean: 
	@echo "Clean..."
	@rm -rf ./bin

## fmt: Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l ./...)" ]; then echo "Go code is not formatted"; exit 1; fi

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
