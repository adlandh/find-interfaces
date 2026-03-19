## Why

The CLI already covers the main interface-discovery path, but several helper functions and failure modes are only partially exercised. Adding targeted tests now will lock in the current behavior before future refactors make regressions harder to spot.

## What Changes

- Expand automated test coverage for directory walking, file filtering, AST extraction, and error classification helpers.
- Add focused edge-case tests for mixed valid and invalid declarations, walk-time failures, and parse-error formatting.
- Document the expected interface-discovery behavior in OpenSpec so the new tests validate an explicit contract rather than incidental implementation details.

## Capabilities

### New Capabilities
- `interface-discovery`: Defines the expected CLI behavior for finding top-level Go interface declarations, tolerating partial parse failures, and rejecting files outside the target directory.

### Modified Capabilities

## Impact

- Affected code: `main_test.go`, with requirements captured under `openspec/changes/add-more-tests/specs/interface-discovery/spec.md`.
- No CLI flags, public APIs, or runtime dependencies change.
- Improves confidence for future refactors in `main.go` by broadening regression coverage around existing behavior.
