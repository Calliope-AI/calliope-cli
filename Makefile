.PHONY: build test lint
build:
	go build -o bin/calliope ./cmd/calliope
test:
	go test ./... -race
lint:
	go vet ./...
