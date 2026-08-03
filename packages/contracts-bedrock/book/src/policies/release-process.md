# Tagging and Release Process

This release process applies to all contract release namespaces:
- `op-contracts/vX.Y.Z`: Core protocol contracts (which includes both L1 and L2 contracts)
- `op-safe-contracts/vX.Y.Z`: Safe multisig extensions

## OPCM Versioning

The OPCM contract uses a specific versioning scheme:

- **Major version bump**: New required sequential upgrade (e.g., U16 → U17 → U18).
- **Minor version bump**: Replacement OPCM for the same upgrade (e.g., bug fixes, U16a).
- **Patch version bump**: Development changes on `develop` branch.

See [OPCM Semver Rules](./versioning.md#opcm-semver-rules) for more details.

## Creating a tagged release

First select a tag string based on the guidance in [Monorepo Contracts Release Versioning](./versioning.md#monorepo-contracts-release-versioning)

1. Before creating a [finalized release](#finalizing-a-release) (i.e. not a release candidate), you MUST have the contracts deployed on Sepolia and Mainnet and work with the EVM Safety team to perform the [Contracts Release Checklist](https://www.notion.so/oplabs/Contracts-Release-Checklists-216f153ee16280fda3c2f141c062f974)
1. Checkout the commit
2. Run `git tag <tag-string>`
3. Run `git push origin <tag-string>`
   Repo [rules](https://github.com/ethereum-optimism/optimism/rules/8196346?ref=refs%2Ftags%2Fop-contracts) require this is done by someone who is a [release-manager](https://github.com/orgs/ethereum-optimism/teams/release-managers). Once pushed a tag cannot be deleted, so please be sure it is correct.
1. Create release notes in GitHub:
   - Go to the [Releases page](https://github.com/ethereum-optimism/optimism/releases), enter or select `<tag-string>`
     from the dropdown.
1. Populate the release notes. If the tag is a release candidate, check the `Set as a pre-release`  option, and uncheck the
   `Set as the latest release` option.
1. Deploy the OPCM using the following op-deployer just recipes (which call the `op-deployer bootstrap implementations` [command](https://devdocs.optimism.io/op-deployer/user-guide/bootstrap.html)),
   this will write the addresses of the deployed contracts to `stdout` (or to disk if you provide an `--outfile` argument).
   ```
   cd op-deployer
   just build // compiles contracts, builds go binary
   just deploy-opcm // deploys the implementations contracts bundle
   just verify-opcm // verifies contracts on block-explorer
   ```
   Deploy and verify contracts on both Sepolia and Mainnet.
1. In the superchain-registry edit the following files to add a new `[<tag-string>]` entry, with the addresses from the
   previous step:
   - [standard-versions-mainnet.toml](https://github.com/ethereum-optimism/superchain-registry/blob/main/validation/standard/standard-versions-mainnet.toml)
   - [standard-versions-sepolia.toml](https://github.com/ethereum-optimism/superchain-registry/blob/main/validation/standard/standard-versions-sepolia.toml)
1. Once the changes are merged into the superchain-registry, you can follow the [instructions](https://devdocs.optimism.io/op-deployer/reference-guide/releases.html#step-3-update-the-sr-with-the-new-release)
   for creating a new release of `op-deployer`.

## Implications for audits

The process above should be followed to create an `-rc.1` release prior to audit. This will be the target commit for
the audit. If any fixes are required by the audit results an Additional Release Candidate will be required.

## Additional Release Candidates

Sometimes fixes or additional changes need to be added to a release candidate version. In that case
we want to ensure fixes are made on both the release and the trunk branch, without stopping development
efforts on the trunk branch.

The process is as follows:

1. Make the fixes on `develop`. Increment the contracts semver as normal.
1. Create a new release branch, named `proposal/<namespace>/vX.Y.Z` off of the rc tag (all subsequent `-rc` tags
   will be made from this branch). For example: `proposal/op-contracts/vX.Y.Z` or `proposal/op-safe-contracts/vX.Y.Z`.
1. Cherry pick the fixes from `develop` into the release branch, and increment the semver as normal. If this increment results in any of the modified contracts' semver being equal to or greater than it is on `develop`, then the semver should immediately be increased on `develop` to be greater than on the release branch. This avoids a situation where a given contract has two different implementations with the same version.
1. After merging the changes into the new release branch, tag the resulting commit on the proposal branch as `<namespace>/vX.Y.Z-rc.n` (e.g., `op-contracts/vX.Y.Z-rc.n` or `op-safe-contracts/vX.Y.Z-rc.n`).
   Create a new release for this tag per the instructions above.

## Regarding L2 contract releases

L2 contracts are tagged and released on the same commit as L1 contracts, but they ship through two different components:

- **Upgrades:** existing chains are upgraded by the consensus client (`op-node` or `kona`) executing a [NUT bundle](../../../../../op-core/nuts/README.md), which defines the transactions that perform the upgrade via the `L2ContractsManager` (L2CM).
- **Deployments:** new chains are deployed with the released implementations in their genesis state, generated by `op-deployer` using the `L2Genesis.s.sol` script.

Since `op-node` and `kona` are released from `develop` while `op-deployer` is released from a `proposal/op-contracts/vX.Y.Z` branch, both branches consume the fork's snapshotted bundle (`op-core/nuts/bundles/<fork>_nut_bundle.json`).

### The situation

Normally L2 contract changes merge to `develop` before the fork bundle is snapshotted, and the standard [two-PR flow](../../../../../op-core/nuts/README.md#pr-2--snapshot-the-bundle-for-a-fork) applies. The problem arises when a change modifying `current-upgrade-bundle.json` (i.e. any L2 contract change) lands on the proposal branch after `develop`'s bundle has moved on due to other contract changes. The fork bundle must then be snapshotted from a proposal-branch commit, and no commit on `develop` can reproduce it.

Before proceeding, check whether the proposal-branch change can be avoided: the only occurrence to date was resolved by [reverting it](https://github.com/ethereum-optimism/optimism/pull/20981), making everything below unnecessary.

### Invariants

1. The fork's snapshotted bundle and its `fork_lock.toml` entry are identical on `develop` and the proposal branch.
2. The bundle is reproducible from the source commit recorded in `fork_lock.toml`. CI enforces this via [`checkCommitAncestry()`](../../../../../ops/scripts/check-nut-locks/main.go), which requires that commit to be an ancestor of `origin/develop`, so a bundle generated from a proposal-branch commit needs a fork-scoped special case there. The special case is permanent: it documents that fork's bundle provenance forever.

### Steps

1. Merge the contract change (with regenerated `current-upgrade-bundle.json`) to the proposal branch.
2. In a second PR to the proposal branch, snapshot the bundle per the [README steps](../../../../../op-core/nuts/README.md#pr-2--snapshot-the-bundle-for-a-fork), with two deviations:
   - Set the source commit in `fork_lock.toml` by hand to the proposal-branch commit that produced the bundle (`just nut-snapshot-for` records the merge-base with `develop`, which is wrong here).
   - Add a fork-scoped special case to `checkCommitAncestry()` accepting ancestors of the proposal branch for this fork ([#20921](https://github.com/ethereum-optimism/optimism/pull/20921) shows both changes; [this suggestion](https://github.com/ethereum-optimism/optimism/pull/20921#discussion_r3282817526) is the preferred shape).
3. Cherry-pick the snapshot PR to `develop`, restoring invariant 1.

## Finalizing a release

Once a release has passed governance, a new tag should be created without the `-rc.n` suffix. To do this follow the
instructions in "Creating a tagged release" once again. It should not be necessary to redeploy the contracts with `op-deployer`,
but a new entry will be required in the superchain-registry's toml files regardless.
When creating release notes, _uncheck_ the `Set as a pre-release`  option, and _uncheck_ the
   `Set as the latest release` option (latest releases are reserved for non-contract packages).

