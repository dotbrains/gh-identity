.PHONY: build build-hook test test-integration lint coverage clean install

BIN_DIR := bin
MODULE := github.com/dotbrains/gh-identity

build:
	go build -o $(BIN_DIR)/gh-identity ./cmd/gh-identity
	go build -o $(BIN_DIR)/gh-identity-hook ./cmd/gh-identity-hook

build-hook:
	go build -o $(BIN_DIR)/gh-identity-hook ./cmd/gh-identity-hook

test:
	go test -race -coverprofile=coverage.out ./...

test-integration:
	go test -race -tags integration ./...

lint:
	golangci-lint run

coverage: test
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BIN_DIR) coverage.out coverage.html dist

install: build
	cp $(BIN_DIR)/gh-identity $(shell gh extension list --json path -q '.[0].path' 2>/dev/null || echo "$$HOME/.local/share/gh/extensions/gh-identity")/gh-identity 2>/dev/null || \
		gh extension install .
