# Contributing

## Development Setup

1. Clone the repo:
   ```sh
   git clone https://github.com/dotbrains/gh-identity.git
   cd gh-identity
   ```

2. Install dependencies:
   ```sh
   go mod tidy
   ```

3. Build:
   ```sh
   make build
   ```

4. Install locally for testing:
   ```sh
   gh extension install .
   ```

## Running Tests

```sh
# Unit tests with coverage
make test

# Integration tests (requires gh to be authenticated)
make test-integration

# View coverage report
make coverage
```

## Code Structure

- `internal/` packages contain all business logic
- `cmd/` packages are thin entry points
- Tests are co-located with source files (`_test.go`)
- Integration tests use `//go:build integration` build tag

## Linting

```sh
make lint
```

Uses `golangci-lint` with the config in `.golangci.yml`.

## Release Process

1. Ensure all tests pass on `main`
2. Tag a new version: `git tag v1.0.0`
3. Push the tag: `git push origin v1.0.0`
4. GoReleaser builds binaries and creates a GitHub Release automatically
5. The Homebrew formula is updated via the release workflow
