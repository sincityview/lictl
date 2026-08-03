VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build build-dev clean test

build:
	GONOSUMCHECK=* GONOSUMDB=* GOINSECURE=* GOPROXY=direct GOTOOLCHAIN=local \
		go build $(LDFLAGS) -o lictl ./cmd/lictl/

build-dev:
	GONOSUMCHECK=* GONOSUMDB=* GOINSECURE=* GOPROXY=direct GOTOOLCHAIN=local \
		go build -ldflags "-X main.version=dev -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)" \
		-o lictl ./cmd/lictl/

clean:
	rm -f lictl

test:
	GONOSUMCHECK=* GONOSUMDB=* GOINSECURE=* GOPROXY=direct GOTOOLCHAIN=local \
		go test ./...
