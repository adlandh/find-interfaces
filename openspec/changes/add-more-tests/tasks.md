## 1. Expand helper coverage

- [x] 1.1 Add focused tests for helper functions that govern directory skipping, file selection, and parse-error formatting.
- [x] 1.2 Add targeted branch tests for `visitPath`, `extractInterfacesFromDecl`, and related helper paths that currently have limited direct coverage.

## 2. Add behavior-driven regression cases

- [x] 2.1 Add regression tests covering mixed declarations, ignored interface-like text, and top-level-only scanning requirements.
- [x] 2.2 Add failure-mode tests for walk errors, outside-base-path validation, and the distinction between recoverable and fatal parse failures.

## 3. Verify the suite

- [x] 3.1 Run `go test ./...` and fix any test regressions introduced while expanding coverage.
