# Smart Contract Versioning and Release Process

The Smart Contract Versioning and Release Process closely follows a true [semver](https://semver.org) for both individual contracts and monorepo releases.
However, there are some changes to accommodate the unique nature of smart contract development and governance cycles.

There are five parts to the versioning and release process:

- [Semver Rules](#semver-rules): Follows the rules defined in the [style guide](../contributing/style-guide.md#versioning) for when to bump major, minor, and patch versions in individual contracts.
- [Individual Contract Versioning](#individual-contract-versioning): The versioning scheme for individual contracts and includes beta, release candidate, and feature tags.
- [Monorepo Contracts Release Versioning](#monorepo-contracts-release-versioning): The versioning scheme for monorepo smart contract releases.
- [Release Process](#release-process): The process for deploying contracts, creating a governance proposal, and the required associated releases.
  - [Additional Release Candidates](#additional-release-candidates): How to handle additional release candidates after an initial `op-contracts/vX.Y.Z-rc.1` release.
  - [Merging Back to Develop After Governance Approval](#merging-back-to-develop-after-governance-approval): Explains how to choose the resulting contract versions when merging back into `develop`.

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

The [OPCM](https://github.com/ethereum-optimism/optimism/blob/main/packages/contracts-bedrock/src/L1/OPContractsManager.sol) is the contract that manages the deployment of all contracts on L1.

The `OPCM` is the source of truth for the contracts that belong in a release, available as on-chain addresses by querying [the `getImplementations` function](https://github.com/ethereum-optimism/optimism/blob/4c8764f0453e141555846d8c9dd2af9edbc1d014/packages/contracts-bedrock/src/L1/OPContractsManager.sol#L1061).

When developing a new release of the contracts, [the `isRC` flag](https://github.com/ethereum-optimism/optimism/blob/4c8764f0453e141555846d8c9dd2af9edbc1d014/packages/contracts-bedrock/src/L1/OPContractsManager.sol#L181) must be set to `true` to indicate that the OPCM refers to a release candidate. The flag [is automatically set to `false`](https://github.com/ethereum-optimism/optimism/blob/4c8764f0453e141555846d8c9dd2af9edbc1d014/packages/contracts-bedrock/src/L1/OPContractsManager.sol#L453) the first time the OPCM `upgrade` method is invoked from governance's Upgrade Controller Safe. This Safe is a 2/2 held by the Security Council and Optimism Foundation.

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
8. After governance vote passes, edit the relase to uncheck "set as pre-release", and remove the `-rc.1` tag.

Although the tools exist to apply a [code freeze](./code-freezes.md) to specific contracts, this is
discouraged. If a change is required to a release candidate after it has been tagged, the
[Additional Release Candidates](#additional-release-candidates) for more information on this flow.

### Additional Release Candidates

Sometimes additional release candidate versions are needed, in that case, the follow process should be used.
This process is designed to (1) ensures fixes are made on both the release and the trunk branch
 and (2) avoids the need to stop development efforts on the trunk branch.


1. Make the fixes on `develop`. *For whatever the normal semver level increment should be, bump that value by 5.*
2. Create a new release branch, named `proposal/op-contracts/X.Y.Z-rc.n+1` off of the rc tag.
3. Cherry pick the fixes from `develop` into that branch. *Bump the semvers as normal, ensuring that the resulting version is less than the one on `develop`.
4. After merging the changes into the new release branch, tag the resulting commit on the proposal branch as `op-contracts/vX.Y.Z-rc.2`.
   Create a new release for this tag per the instructions above.

Note: The reason for the larger semver increment on `develop` is to prevent a collision, wherein a
contract could have the same semver, but different source/bytecode on the two branches.

For example: if the current version of a contract is `1.1.1` and a minor bump is required (most common for a bug fix),
   then the fixed version should become `1.8.0` on `develop`. Then on the release branch is should become
   `1.2.0`.

### Merging Back to Develop After Governance Approval

A release will change a set of contracts, and those contracts may have changed on `develop` since the release candidate was created.

If there have been no changes to a contract since the release candidate, the version of that contract stays at `X.Y.Z` and just has the `-rc.n` removed.
For example, if the release candidate is `1.2.3-rc.1`, the resulting version on `develop` will be `1.2.3`.

If there have been changes to a contract, the `X.Y.Z` will stay the same as whatever is the latest version on `develop`, with the `-beta.n` qualifier incremented.

For example, given that ContractA is `1.2.3-rc.1` on develop, then the initial sequence of events is:

- We create the release branch, and on that branch remove the `-rc.1`, giving a final ContractA version on that branch of `1.2.3`
- Governance proposal is posted, pointing to the corresponding monorepo tag.
- Governance approves the release.
- Open a PR to merge the final versions of the contracts (ContractA) back into develop.

Now there are two scenarios for the PR that merges the release branch back into develop:

1. On develop, no changes have been made to ContractA. The PR therefore changes ContractA's version on develop from `1.2.3-rc.1` to `1.2.3`, and no other changes to ContractA occur.
2. On develop, breaking changes have been made to ContractA for a new feature, and it's currently versioned as `2.0.0-beta.3`. The PR should bump the version to `2.0.0-beta.4` if it changes the source code of ContractA.
    - In practice, this one unlikely to occur when using inheritance for feature development, as specified in [Smart Contract Feature Development](https://github.com/ethereum-optimism/design-docs/blob/main/smart-contract-feature-development.md) architecture. It's more likely that (1) is the case, and we merge the version change into the base contract.

This flow also provides a dedicated branch for each release, making it easy to deploy a patch or bug fix, regardless of other changes that may have occurred on develop since the release.

