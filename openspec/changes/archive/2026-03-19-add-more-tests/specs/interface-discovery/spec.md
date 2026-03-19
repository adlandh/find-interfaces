## ADDED Requirements

### Requirement: Discover interfaces from top-level Go files
The CLI SHALL scan only the requested directory's top-level `.go` files and return the names of declared interface types found in those files.

#### Scenario: Interface declarations appear in multiple top-level files
- **WHEN** the target directory contains multiple top-level Go files with interface type declarations
- **THEN** the CLI returns the discovered interface names from those files

#### Scenario: Nested directories contain Go files
- **WHEN** a nested directory under the target path contains additional Go files
- **THEN** the CLI ignores those nested files

### Requirement: Ignore declarations that are not interface types
The CLI SHALL ignore comments, string literals, non-interface type declarations, and other declarations that do not define an interface type.

#### Scenario: Source contains interface-like text outside a type declaration
- **WHEN** a Go file contains comments or string literals that mention interface syntax
- **THEN** the CLI does not report names from that text

#### Scenario: Source contains mixed type declarations
- **WHEN** a Go file contains interface declarations alongside aliases, structs, functions, or other non-interface declarations
- **THEN** the CLI reports only the interface type names

### Requirement: Tolerate recoverable parse failures while preserving fatal errors
The CLI SHALL continue collecting interface names from parsable Go files when other Go files contain recoverable parse errors, and it SHALL return an error when directory walking fails, when a file path is outside the requested directory, or when no Go file parses successfully.

#### Scenario: One Go file parses and another does not
- **WHEN** the target directory contains at least one parsable Go file with interface declarations and another Go file with parse errors
- **THEN** the CLI returns the interface names from the parsable file without failing the overall command

#### Scenario: Every Go file fails to parse
- **WHEN** all discovered Go files fail to parse successfully
- **THEN** the CLI returns an error instead of an empty success result

#### Scenario: A walked file path escapes the base directory
- **WHEN** interface extraction is attempted on a file path outside the requested directory
- **THEN** the CLI returns an error describing that the file path is outside the base directory

#### Scenario: Directory walking returns an error
- **WHEN** directory traversal encounters a walk error for a path being visited
- **THEN** the CLI returns that error to the caller
