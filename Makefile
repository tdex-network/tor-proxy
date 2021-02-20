.PHONY: fmt

## fmt: Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l ./...)" ]; then echo "Go code is not formatted"; exit 1; fi

