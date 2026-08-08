.PHONY: build test check

build:
	go build -o aads .

test:
	go test ./...

check:
	go test -race ./...
	go vet ./...
	go build -o aads .
