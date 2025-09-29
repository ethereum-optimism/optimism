# Diamond Code Review Rules

This file explains the rules that you should use when reviewing a PR.

## Applicability

You are ONLY to review changes to Solidity files (*.sol). Do NOT leave comments on any other file types.

## Rules for Reviewing Solidity Files

This section applies to Solidity files ONLY.

### Style Guide

- Follow the style guide found at `.cursor/rules/solidity-styles.mdc` in the root of this repository.

### Versioning

- Verify that the version bumps being made in source files are appropriate for the type of change that was made. For example, a change that only modifies a contract should be a patch change, not a minor change.

### Interfaces

- Source files are expected to have a corresponding interface file in the `interfaces/` folder
- Do NOT review for missing interface files, CI checks will handle that
- Do NOT review for discrepancies between interface files and the source files, CI will handle that
- We do NOT require natspec comments in interface files, only in the source files
