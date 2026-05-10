.PHONY: fmt fmt-go fmt-python lint lint-go lint-python test test-go test-python

fmt: fmt-go fmt-python

fmt-go:
	gofmt -w ./cmd ./internal

fmt-python:
	cd python && uv run ruff format .

lint: lint-go lint-python

lint-go:
	go vet ./...

lint-python:
	cd python && uv run ruff check .

test: test-go test-python

test-go:
	go test ./...

test-python:
	cd python && uv run pytest
