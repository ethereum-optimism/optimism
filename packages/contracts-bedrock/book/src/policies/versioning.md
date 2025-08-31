# Smart Contract Versioning and Release Process

The Smart Contract Versioning and Release Process closely follows a true [semver](https://semver.org) for both individual contracts and monorepo releases.
However, there are some changes to accommodate the unique nature of smart contract development and governance cycles.

There are five parts to the versioning and release process:

- [Semver Rules](#semver-rules): Follows the rules defined in the [style guide](../contributing/style-guide.md#versioning) for when to bump major, minor, and patch versions in individual contracts.
- [Individual Contract Versioning](#individual-contract-versioning): The versioning scheme for individual contracts and includes beta, release candidate, and feature tags.
- [Monorepo Contracts Release Versioning](#monorepo-contracts-release-versioning): The versioning scheme for monorepo smart contract releases.
- [Release Process](#release-process): The process for deploying contracts, creating a governance proposal, and the required associated releases.
  - [Additional Release Candidates](#additional-release-candidates): How to handle additional release candidates after an initial `op-contracts/vX.Y.Z-rc.1` release.

> [!NOTE]
> The rules described in this document must be enforced manually.
> Ideally, a check can be added to CI to enforce the conventions defined here, but this is not currently implemented.

## Semver Rules

Version increments follow the [style guide rules](../contributing/style-guide.md#versioning) for when to bump major, minor, and patch versions in individual contracts:

> - `patch` releases are to be used only for changes that do NOT modify contract bytecode (such as updating comments).
> - `minor` releases are to be used for changes that modify bytecode OR changes that expand the contract ABI provided that these changes do NOT break the existing interface.
> - `major` releases are to be used for changes that break the existing contract interface OR changes that modify the security model of a contract.
>
> Bumping the patch version does change the bytecode, so another exception is carved out for this.
> In other words, changing comments increments the patch version, which changes bytecode. This bytecode
> change implies a minor version increment is needed, but because it's just a version change, only a
> patch increment should be used.

## Individual Contract Versioning

Individual contract versioning allows us to uniquely identify which version of a contract from the develop branch corresponds to each deployed contract instance.

Versioning for individual contracts works as follows:

- A contract on develop always has a version of X.Y.Z, regardless of whether is has been governance approved and meets our security bar. This DOES NOT indicate these contracts are always safe for production use. More on this below.
- For contracts with feature-specific changes, a `+feature-name` identifier must be appended to the version number. See the [Smart Contract Feature Development](https://github.com/ethereum-optimism/design-docs/blob/main/smart-contract-feature-development.md) design document to learn more.
- When making changes to a contract, always bump to the lowest possible version based on the specific change you are making. We do not want to e.g. optimistically bump to a major version, because protocol development sequencing may change unexpectedly. Use these examples to know how to bump the version:
  - Example 1: A contract is currently on `1.2.3` on `develop` and you are working on a new feature on your `feature` branch off `develop`.
    - We don't yet know when the next release of this contract will be. However, you are simply fixing typos in comments so you bump the version to `1.2.4`.
    - The next commit to the `feature` branch clarifies some comments. We only consider the aggregated `feature` changes with regards to `develop` when determining the version, so we stay at `1.2.4`.
    - The next commit to the `feature` branch introduces a breaking change, which bumps the version from `1.2.4` to `2.0.0`.
  - Example 2: A contract is currently on `2.4.7`.
    - We know the next release of this contract will be a breaking change. Regardless, as you start development by fixing typos in comments, bump the version to `2.4.8`. This is because we may end up putting out a release before the breaking change is added.
    - Once you start working on the breaking change, bump the version to `3.0.0`.
- New contracts start at `1.0.0`.

Versioning is enforced by CI checks:
  - Any contract that differs from its version in the `develop` branch must be bumped to a new semver value, or the build will fail.
  - Any branch with at least one modified contract must have its `semver-lock.json` file updated, or the build will fail. You can use the `semver-lock` or `pre-commit` just commands to do so.

Note: Previously, the versioning scheme included `-beta.n` and `-rc.n` qualifiers. These are no longer used to reduce the amount of work required to execute this versioning system.

## Deprecating Individual Contract Versioning

Individual contract versioning could be deprecated when the following conditions are met:

1. Every OPCM instance is registered in the superchain registry
2. All contracts are implemented as either proxies or concrete singletons, allowing verification of governance approval through the `OPCM.Implementations` struct
3. We have validated with engineering teams (such as the fault proofs team) and ecosystem partners (such as L2Beat) that removing `version()` functions would not negatively impact their workflows

## Monorepo Contracts Release Versioning

Versioning for monorepo releases works as follows:

- Monorepo releases continue to follow the `op-contracts/vX.Y.Z` naming convention.
- The version used for the next release is determined by the highest version bump of any individual contract in the release.
  - Example 1: The monorepo is at `op-contracts/v1.5.0`. Clarifying comments are made in contracts, so all contracts only bump the patch version. The next monorepo release will be `op-contracts/v1.5.1`.
  - Example 2: The monorepo is at `op-contracts/v1.5.1`. Various tech debt and code is cleaned up in contracts, but no features are added, so at most, contracts bumped the minor version. The next monorepo release will be `op-contracts/v1.6.0`.
  - Example 3: The monorepo is at `op-contracts/v1.5.1`. Legacy `ALL_CAPS()` getter methods are removed from a contract, causing that contract to bump the major version. The next monorepo release will be `op-contracts/v2.0.0`.
- Feature specific monorepo releases (such as a release of the custom gas token feature) are supported, and should follow the guidelines in the [Smart Contract Feature Development](https://github.com/ethereum-optimism/design-docs/blob/main/smart-contract-feature-development.md) design doc. Bump the overall monorepo semver as required by the above rules. For example, if the last release before the custom gas token feature was `op-contracts/v1.5.1`, because the custom gas token introduces breaking changes, its release will be `op-contracts/v2.0.0`.
  - A subsequent release of the custom gas token feature that fixes bugs and introduces an additional breaking change would be `op-contracts/v3.0.0`.
  - This means `+feature-name` naming is not used for monorepo releases, only for individual contracts as described below.
- A monorepo contracts release must map to an exact set of contract semvers, and this mapping must be defined in the contract release notes which are the source of truth. See [`op-contracts/v1.4.0-rc.4`](https://github.com/ethereum-optimism/optimism/releases/tag/op-contracts%2Fv1.4.0-rc.4) for an example of what release notes should look like.

## Optimism Contracts Manager (OPCM) Versioning

The [OPCM](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L1/OPContractsManager.sol) is the contract that manages the deployment of all contracts on L1.

The `OPCM` is the source of truth for the contracts that belong in a release, available as on-chain addresses by querying [the `getImplementations` function](https://github.com/ethereum-optimism/optimism/blob/4c8764f0453e141555846d8c9dd2af9edbc1d014/packages/contracts-bedrock/src/L1/OPContractsManager.sol#L1061).

## Release Process

When a release is proposed to governance, the proposal includes a commit hash, and often the
contracts from that commit hash are already deployed to mainnet with their addresses included
in the proposal.
For example, the [Fault Proofs governance proposal](https://gov.optimism.io/t/upgrade-proposal-fault-proofs/8161) provides specific addresses that will be used.

To accommodate this, once contract changes are ready for governance approval, the release flow is:

1. Go to https://github.com/ethereum-optimism/optimism/releases/new
2. Enter the release title as `op-contracts/vX.Y.Z-rc.1`
3. In the "choose a tag" dropdown, enter the same `op-contracts/vX.Y.Z-rc.1` and click the "Create new tag" option that shows up
4. Populate the release notes.
5. Check "set as pre-release" since it's not yet governance approved
6. Uncheck "Set as the latest release" and "Create a discussion for this release".
7. Click publish release.
8. After governance vote passes, edit the release to uncheck "set as pre-release", and remove the `-rc.1` tag.

Although the tools exist to apply a [code freeze](./code-freezes.md) to specific contracts, this is
discouraged. If a change is required to a release candidate after it has been tagged, the
[Additional Release Candidates](#additional-release-candidates) for more information on this flow.

### Additional Release Candidates

Sometimes fixes or additional changes need to be added to a release candidate version. In that case
we want to ensure fixes are made on both the release and the trunk branch, without stopping development
efforts on the trunk branch.

The process is as follows:

1. Make the fixes on `develop`. Increment the contracts semver as normal.
2. Create a new release branch, named `proposal/op-contracts/X.Y.Z` off of the rc tag.
3. Cherry pick the fixes from `develop` into that branch. Instead of incrementing the semver as normal,
   append `-patch.n` to the end of the version number. The value of `n` should start at 1 and be
   incremented for each additional patch.
4. After merging the changes into the new release branch, tag the resulting commit on the proposal branch as `op-contracts/vX.Y.Z-rc.n`.
   Create a new release for this tag per the instructions above.
