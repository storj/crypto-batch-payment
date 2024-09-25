.PHONY: all generate binary test lint

all: binary

generate:
	go generate ./...

binary:
	go build ./cmd/crybapy

test:
	go test -race ./...

lint:
	docker run --rm -v $(CURDIR):/app -v $(HOME)/.cache/golangci-lint/v1.61.0:/root/.cache -w /app golangci/golangci-lint:v1.61.0 golangci-lint run -v
