# Diamond Code Review Rules

This file explains the rules that you should use when reviewing a PR.

## Applicability

You are ONLY to review changes to Solidity files (*.sol). Do NOT leave comments on any other file types.

## OPCM Version Bump Warnings

If the PR modifies `OPContractsManagerV2.sol` and changes the `version` constant with a major or minor version bump, you MUST leave a prominent comment on the PR with the following message:

> ⚠️ **OPCM Version Bump Detected**
>
> This PR includes a major or minor version bump to `OPContractsManagerV2.sol`.
>
> **Reminder of OPCM versioning rules:**
> - **Major bump**: Only for a new required sequential upgrade (e.g., U16 → U17)
> - **Minor bump**: Only for replacing an existing OPCM for the same upgrade (e.g., bug fixes, U16a)
> - **Patch bump**: Expected for normal development work
>
> Please confirm this version bump is intentional and follows the versioning policy.

## Rules for Reviewing Solidity Files

This section applies to Solidity files ONLY.

### @dev Comments

- Pay close attention to `@dev` natspec comments in the codebase
- These comments often contain important invariants, requirements, or reminders for developers
- When reviewing changes to a function, check if there are `@dev` comments that specify conditions or actions that must be taken when modifying that code
- Flag violations of instructions in `@dev` comments (e.g., "when updating this function, also update X")

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
