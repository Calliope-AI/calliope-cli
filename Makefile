.PHONY: build test lint snapshot
build:
	go build -o bin/calliope ./cmd/calliope
test:
	go test ./... -race
lint:
	go vet ./...
snapshot:
	goreleaser build --snapshot --clean
