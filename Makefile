BINARY := recipe-lint

.PHONY: build install test ci clean

build:
	go build -o $(BINARY) ./cmd/recipe-lint

install: build
	@echo "Install with: wk plugins install ."

test:
	go test ./...

# Same checks as .github/workflows/ci.yml
ci:
	go build ./...
	go vet ./...
	@out=$$(gofmt -l ./cmd ./pkg); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	go test -count=1 ./...

clean:
	rm -f $(BINARY)
