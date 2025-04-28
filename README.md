# find-interfaces

A simple Go utility that finds all interface definitions in a given folder without checking subfolders.

Mostly created to use with [gowrap](https://github.com/hexdigest/gowrap).

## Requirements

- Go 1.22 or later

## Installation

```bash
go install github.com/adlandh/find-interfaces@latest
```

## Usage

```bash
# Search in current directory
find-interfaces

# Search in a specific directory
find-interfaces -path /path/to/directory
```

## Features

- Finds all interface definitions in Go files within a specified directory
- Ignores subdirectories
- Case-insensitive file extension matching
- Uses regular expressions to identify interface definitions
- Prevents directory traversal attacks with path validation
- Processes only files with .go extension

## Output

The tool outputs a space-separated list of interface names found in the specified directory, which can be piped to other tools like gowrap.

Example:
```
$ find-interfaces -path ./pkg/models
Reader Writer Processor Handler
```
