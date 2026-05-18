# Contributing

Thanks for contributing to es.

## Setup

1. Fork and clone the repository.
2. `go mod download`
3. `go test ./...`

## Pull requests

- Run `go fmt ./...` and `go vet ./...`.
- Prefer behavioral test names (`TestShould...`).
- Use `NewInMemoryEventStore()` in unit tests unless testing store-specific code.
- Update `docs/` for changes to `Aggregate`, `Repository`, or audit semantics.

## Changelog

Note changes under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md).
