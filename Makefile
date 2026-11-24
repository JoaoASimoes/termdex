BINARY := type_calc
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build run test vet fmt clean dist

build:
	go build -trimpath -ldflags="-s -w" -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

dist:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=$$( [ $$os = windows ] && echo .exe ); \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="-s -w" -o dist/$(BINARY)-$$os-$$arch$$ext . || exit 1; \
	done

clean:
	rm -rf $(BINARY) dist
