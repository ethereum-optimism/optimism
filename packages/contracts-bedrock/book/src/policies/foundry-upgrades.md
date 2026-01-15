# Foundry Versioning Policy

This document outlines the process for proposing and implementing Foundry version updates in the
OP Stack codebase.

## Unified Foundry Version

The OP Stack codebase maintains a single, unified Foundry version across all components. This ensures
consistency, simplifies maintenance, and reduces the risk of version-related issues. Foundry is a
critical dependency in our supply chain, and if compromised, it could result in draining the bridge.

**Important**: New Foundry versions must not be introduced to any part of the codebase without
going through the formal version update proposal process outlined in this document.

## Update Process

1. **Minimum Delay Period**: A new Foundry version must be at least 1 month old before it can be
   considered for adoption.
2. Only stable releases are recommended to be proposed for adoption. For upgrades to nightly builds, there must be a feature addition that improves the quality of our codebase considerably or a bug fix that addresses a non-trivial security vulnerability.
3. **Proposal Submission**: Before any Foundry version upgrades are made, a detailed proposal must
   be submitted as a pull request to the [`ethereum-optimism/design-docs`][1] repository in a
   `foundry/` subfolder, following the standardized format outlined below. This applies to the monorepo, superchain-ops, and any other repositories that use Foundry.
4. **Review and Approval**: A dedicated review panel will assess the proposal based on the
   following criteria:
   - Is the Foundry version at least 1 month old?
   - Does the proposed upgrade provide clear value to the codebase?
   - Do any new features or bug fixes pose an unnecessary risk to the codebase?
   - Are there any security vulnerabilities addressed by this version?
5. **Implementation**: If the proposal receives unanimous approval from the review panel, the
   Foundry version upgrade will be implemented across the entire OP Stack codebase, including:
   - The monorepo (`mise.toml` and `op-deployer/pkg/deployer/forge/version.json`)
   - The `superchain-ops` repository

## Proposal Submission Guidelines

To submit a Foundry version upgrade proposal, create a new pull request to the
[`ethereum-optimism/design-docs`][1] repository, adding a new file in the `foundry/` subfolder.
Please use a format similar to the [Foundry update proposal format][2], adapted for Foundry.
Ensure that all sections are filled out comprehensively. Incomplete proposals may be delayed or
rejected.

## Review Process

The review panel will evaluate each proposal based on the criteria mentioned in the "Review and
Approval" section above. They may request additional information or clarifications if needed.

## Implementation

If approved, the Foundry version upgrade will be implemented across the entire OP Stack codebase.
This process will be managed by the development team to ensure consistency and minimize potential
issues. The upgrade will apply to all components simultaneously, including:

- `mise.toml`: Update `forge`, `cast`, and `anvil` version entries
- `op-deployer/pkg/deployer/forge/version.json`: Update the `forge` version and corresponding
  checksums for all supported platforms
- `superchain-ops` repository: Update Foundry version configuration

All version updates must be synchronized across these locations to maintain consistency.

<!-- References -->

[1]: https://github.com/ethereum-optimism/design-docs
[2]: https://github.com/ethereum-optimism/design-docs/tree/main/assets/foundry-update-template.md
