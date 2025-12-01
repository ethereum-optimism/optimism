# Diamond Code Review Rules

This file explains the rules that you should use when reviewing a PR.

## Applicability

You are ONLY to review changes to Solidity files (*.sol). Do NOT leave comments on any other file types.

## Rules for Reviewing Solidity Files

This section applies to Solidity files ONLY.

### Style Guide

- Follow the style guide found at `.cursor/rules/solidity-styles.mdc` in the root of this repository.

### Versioning

- Do NOT comment on the choice of version increment for a given Solidity file.

### Interfaces

- Source files are expected to have a corresponding interface file in the `interfaces/` folder
- Do NOT review for missing interface files, CI checks will handle that
- Do NOT review for discrepancies between interface files and the source files, CI will handle that
- We do NOT require natspec comments in interface files, only in the source files

### Testing with `vm.expectRevert`

- When `vm.expectRevert` is used with low-level calls (`.call{}`), Foundry inverts the return boolean semantics
- The boolean indicates whether the expectRevert succeeded (NOT whether the call succeeded)
- Code that captures and asserts this boolean is CORRECT and should NOT be flagged:
  ```solidity
  vm.expectRevert(ExpectedError.selector);
  (bool revertsAsExpected,) = address(target).call(data);
  assertTrue(revertsAsExpected, "expectRevert: call did not revert");
  ```
- Do NOT suggest removing the return value checking on low-level calls following `vm.expectRevert`
- DO flag if `vm.expectRevert` is used with low-level calls but the return value is not captured and asserted
