.PHONY: build test test-integration fuzz ci

FUZZ_TARGETS := FuzzDecodeMethodResponse FuzzXMLScannerToken FuzzReadSCGIResponse FuzzEncodeDecodeString FuzzEncodeDecodeBase64

build:
	go build -o ./dist/rtctl ./cmd/rtctl.go

test:
	go test -v ./...

test-integration:
	go test -tags=integration -v ./...

fuzz:
	for name in $(FUZZ_TARGETS); do \
		echo "FUZZER $$name :"; \
		go test -run=^$$ -fuzz="^$$name\$$" -fuzztime=30s . || exit 1; \
	done

ci:
	unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go build ./...
	go vet ./...
	go test -race ./...
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
