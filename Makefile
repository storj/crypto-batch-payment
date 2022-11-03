.PHONY: all generate payouts accountant test lint

all: binary

libzks-crypto.so:
	wget https://github.com/zksync-sdk/zksync-crypto-c/releases/download/v0.1.2/zks-crypto-x86_64-unknown-linux-gnu.so -O $(CURDIR)/libzks-crypto.so

generate:
	go generate ./...

binary: libzks-crypto.so
	LD_LIBRARY_PATH=$(CURDIR) CGO_LDFLAGS=-L$(CURDIR) go build ./cmd/crybapy

test: libzks-crypto.so
	LD_LIBRARY_PATH=$(CURDIR) CGO_LDFLAGS=-L$(CURDIR) go test -race ./...

lint: libzks-crypto.so
	LD_LIBRARY_PATH=$(CURDIR) CGO_LDFLAGS=-L$(CURDIR) go run honnef.co/go/tools/cmd/staticcheck@v0.3.0 -checks all ./...
	LD_LIBRARY_PATH=$(CURDIR) CGO_LDFLAGS=-L$(CURDIR) go vet ./...


