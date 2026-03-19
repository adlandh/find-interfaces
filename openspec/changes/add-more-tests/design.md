## Context

`find-interfaces` is a small CLI with most behavior concentrated in `main.go` and a single test file, `main_test.go`. The current suite already proves the primary happy path, but several helper-level branches remain lightly specified, including walk-time error propagation, parse-error classification, and the exact boundaries of top-level file scanning.

This change is intentionally test-focused: it does not alter runtime behavior, flags, or dependencies. The design therefore centers on expressing the current contract clearly enough that implementation work can add targeted tests without rewriting production code.

## Goals / Non-Goals

**Goals:**
- Increase regression coverage for existing interface-discovery behavior.
- Add tests around helper functions whose branches currently have limited direct coverage.
- Align new tests with an explicit OpenSpec contract so future refactors preserve intended behavior.

**Non-Goals:**
- Changing CLI output, traversal rules, or parse-error semantics.
- Introducing new production abstractions solely to satisfy tests.
- Adding recursive scanning, package filtering, or output sorting.

## Decisions

### Keep production behavior unchanged and expand only tests
The most valuable outcome is higher confidence in the current implementation, not a refactor. New tests should exercise existing helpers such as `visitPath`, `shouldSkipDir`, `shouldProcessFile`, `extractInterfacesFromDecl`, and `fileParseError.Error` through focused unit tests and narrow integration-style cases.

Alternative considered: refactor `main.go` first to create more injectable seams. Rejected because it would broaden the change, risk incidental behavior drift, and make it harder to tell whether failures come from new logic or better coverage.

### Encode behavior as capability requirements before adding tests
Because the repository has no prior OpenSpec specs, this change introduces an `interface-discovery` capability that captures the current contract: only top-level Go files are scanned, valid interfaces are returned from parsable content, and fatal errors are reserved for directory walk failures or cases where no Go file parses successfully.

Alternative considered: treat this as a pure implementation-quality change with no spec. Rejected because the schema requires specs before tasks, and the repository benefits from documenting the CLI's externally visible behavior.

### Prefer table-driven tests only where they improve clarity
Some helper functions naturally fit table-driven tests (`shouldProcessFile`, `shouldSkipDir`, `extractInterfaceName`), while others are easier to understand as dedicated scenarios (`visitPath` propagating walk errors, parse-error formatting). The implementation should use the simplest structure that keeps each behavioral edge case obvious.

Alternative considered: convert the whole suite to a single table-driven style. Rejected because mixed helper and integration scenarios become harder to read and maintain when forced into one pattern.

## Risks / Trade-offs

- [Over-specifying internal details] -> Write tests against observable behavior and returned values rather than fragile AST internals or incidental traversal order beyond what the CLI already guarantees.
- [Duplicating existing coverage] -> Focus new cases on uncovered branches and boundary conditions rather than rewriting happy-path assertions.
- [Test brittleness from filesystem assumptions] -> Use `t.TempDir()` and purpose-built fixture files for each scenario so tests remain isolated and deterministic.

## Migration Plan

No runtime migration is required. The implementation will add tests and then run the Go test suite to confirm that current behavior remains intact.

## Open Questions

No open questions at proposal time; the repository behavior and scope are sufficiently clear for implementation.
