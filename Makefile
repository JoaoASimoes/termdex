BINARY    := termdex
PKG       := ./cmd/termdex
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build install run test cover vet fmt fmt-check check dist clean

build: ## build ./termdex
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) $(PKG)

install: ## install into $(go env GOPATH)/bin
	go install -trimpath -ldflags="$(LDFLAGS)" $(PKG)

run: build
	./$(BINARY)

test:
	go test -race ./...

cover: ## test coverage report in coverage.html
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check: ## fail if some file is not gofmt-ed (what CI runs)
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' on:"; gofmt -l .; exit 1)

check: fmt-check vet test ## everything CI checks

dist: ## cross-compile into dist/
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=$$( [ $$os = windows ] && echo .exe ); \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-$$os-$$arch$$ext $(PKG) || exit 1; \
	done

clean:
	rm -rf $(BINARY) dist coverage.out coverage.html
