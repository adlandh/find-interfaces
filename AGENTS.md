# AGENTS.md

## Repo Shape
- Single-package Go CLI. All production code is in `main.go`; all tests are in `main_test.go`.
- There are no subpackages today, so focused verification is usually package-root only.

## Verified Commands
- Build: `go build .`
- Run all tests: `go test ./...`
- Match CI test settings: `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`
- Run one test: `go test -run '^TestName$' .`
- Inspect coverage by function: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`
- Match CI lint flow: `curl -sS https://raw.githubusercontent.com/adlandh/golangci-lint-config/refs/heads/main/.golangci.yml -o .golangci.yml && golangci-lint run`

## Behavior That Is Easy To Break
- `FindInterfaces` is intentionally non-recursive. It skips every subdirectory below the requested base path.
- Output order is not explicitly sorted. Tests sort results before asserting whenever order is irrelevant.
- Parsing is intentionally tolerant: `extractInterfacesFromFile` can return discovered interfaces together with a parse error when the AST is partially recoverable.
- `FindInterfaces` only returns an error for parse failures when no interfaces were found at all and at least one Go file failed to parse. A directory with no Go files is a success with empty output.
- File extension matching is case-insensitive via `isGoFile`, so `.GO` files are expected to be processed.

## Refactor Guardrails
- Keep the base-dir validation behavior in `validatePathWithinBase`; tests cover both obvious outside paths and sibling paths with a shared prefix.
- If you change CLI behavior in `main()`, add dedicated tests. Current coverage leaves `main()` untested even though the library logic is well covered.
