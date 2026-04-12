# find-interfaces

`find-interfaces` is a small Go CLI that prints interface type names found in Go source files in a single directory.

It is designed for shell usage and works well in pipelines with tools such as [gowrap](https://github.com/hexdigest/gowrap).

## What It Does

- Scans `.go` files in the target directory
- Parses source code using Go's AST
- Extracts declared interface type names
- Ignores subdirectories
- Prints results as a single space-separated line

## Requirements

- Go 1.25 or later

## Installation

Install the latest version:

```bash
go install github.com/adlandh/find-interfaces@latest
```

Build from source:

```bash
go build .
```

## Development

Run all tests:

```bash
go test ./...
```

Match CI test settings:

```bash
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

Inspect coverage by function:

```bash
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Match the CI lint flow:

```bash
curl -sS https://raw.githubusercontent.com/adlandh/golangci-lint-config/refs/heads/main/.golangci.yml -o .golangci.yml && golangci-lint run
```

## Usage

Search the current directory:

```bash
find-interfaces
```

Search a specific directory:

```bash
find-interfaces -path /path/to/package
```

Show help:

```bash
find-interfaces -h
```

## Example

Given this Go source:

```go
package sample

type Reader interface {
	Read([]byte) (int, error)
}

type Writer[T any] interface {
	Write(T) error
}

type handlerFunc func()
```

Running:

```bash
find-interfaces -path ./sample
```

Produces:

```text
Reader Writer
```

## Behavior

- Only the top-level target directory is scanned
- Only files with a `.go` extension are considered
- File extension matching is case-insensitive
- Comments and string literals that merely contain interface-like text are ignored
- Files with parse errors may still contribute interface names when the AST is partially recoverable
- Output order follows file traversal order and is not explicitly sorted

## Exit Behavior

- On success, the tool prints discovered interface names to standard output
- If no interfaces are found, it prints nothing
- On failure, it exits with a non-zero status and prints an error message

## Typical Pipeline Use

```bash
for iface in $(find-interfaces -path ./pkg/service); do
  gowrap gen -p ./pkg/service -i "$iface" -t fallback -o "./pkg/service/${iface,,}_wrapper.go"
done
```

## Limitations

- It does not recurse into nested directories
- It does not filter by package name
- It only reports interface names, not file paths or method details
