.PHONY: all generate binary test lint

CGO_LDFLAGS="-L$(CURDIR) -lm"

all: binary

libzks-crypto.a:
	rm -f libzks-crypto.so # remove the dynamic library if it exists
	wget https://github.com/zksync-sdk/zksync-crypto-c/releases/download/v0.1.2/zks-crypto-x86_64-unknown-linux-gnu.a -O $(CURDIR)/libzks-crypto.a

generate:
	go generate ./...

binary: libzks-crypto.a
	CGO_LDFLAGS=$(CGO_LDFLAGS) go build ./cmd/crybapy

test: libzks-crypto.a
	CGO_LDFLAGS=$(CGO_LDFLAGS) go test -race ./...

lint:
	docker run --rm -v $(CURDIR):/app -v $(HOME)/.cache/golangci-lint/v1.59.1:/root/.cache -w /app golangci/golangci-lint:v1.59.1 golangci-lint run -v
