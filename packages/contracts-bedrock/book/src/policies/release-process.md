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
   Deploy and verify contracts on both Sepolia and Mainnet. Verification must run immediately after deployment;
   any verification failure blocks the release.
1. In the superchain-registry edit the following files to add a new `[<tag-string>]` entry, with the addresses from the
   previous step:
   - [standard-versions-mainnet.toml](https://github.com/ethereum-optimism/superchain-registry/blob/main/validation/standard/standard-versions-mainnet.toml)
   - [standard-versions-sepolia.toml](https://github.com/ethereum-optimism/superchain-registry/blob/main/validation/standard/standard-versions-sepolia.toml)
1. Once the changes are merged into the superchain-registry, you can follow the [instructions](https://devdocs.optimism.io/op-deployer/reference-guide/releases.html#step-3-update-the-sr-with-the-new-release)
   for creating a new release of `op-deployer`.

## Implications for audits

Smart contract releases are audited as a whole. A feature-specific audit may be added when the feature's risk warrants
extra review.

The process above should be followed to create an `-rc.1` release prior to audit. This is the target commit for the
audit. If no fixes are required, the final release tag must point to this same commit and no proposal branch is needed.
If fixes are required by the audit results, an Additional Release Candidate is required.

## Feature readiness

Development features and system features have different release lifecycles:

- Development features are shipped by enabling the flag on `develop`, not by removing it. The feature's getter is hard
  coded to `return true`, overriding the feature flag bitmap, which is still asserted to be `0x0000` on mainnet chains.
  The PR that enables the feature is the shipping signoff. Keeping the flag through the audit means a major audit
  finding can be handled by toggling the feature back off rather than delaying the rest of the release; if the flag had
  already been removed, the only escape hatch is a revert, which conflicts with nearby changes very quickly. The flag is
  removed on `develop` once the release is fully deployed, so the cleanup ships with the next release. (`CANNON_KONA`
  and `L2CM` were shipped this way in U19.) This mechanism is awkward and is expected to be replaced by flags with
  proper default-on/default-off semantics that can be overridden in either direction.
- System features are production-supported settings stored in `SystemConfig`. They may remain configurable after the
  release and do not follow the development-feature cleanup lifecycle. For a system feature, the PR that puts `develop`
  in the intended production configuration is the shipping signoff.

Subsequent acceptance tests must exercise the signed-off production configuration. Cut and audit the release candidate
from the signed-off commit. Do not change the release configuration between audit and production.

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

Proposal branches should be created lazily when an additional release candidate is needed. If governance needs a
consistent proposal branch, it may be created earlier from the audited tag but must not diverge from that tag.

## Regarding L2 contract releases

L2 contracts should be tagged and released on the same commit as L1 contracts, however unlike L1 contracts which are released by a new
`op-deployer`, L2 contracts are released by two different components depending whether it is an upgrade or new deployment:

- **Upgrades:** Existing chains are upgraded to the newly released implementations by the consensus client (`op-node` or `kona`) executing a [NUT bundle](../../../../../op-core/nuts/README.md) which define the set of transactions required to perform the upgrade via the `L2ContractsManager` (L2CM).
- **Deployments:** New chains are deployed with the newly released implementations inserted into the genesis state, which is generated by `op-deployer` using the `L2Genesis.s.sol` script.

Since `op-node` and `kona` are typically released from the trunk (`develop`) branch, while `op-deployer` may be released
from a `proposal/op-contracts/vX.0.0` branch when an additional release candidate is required, special effort is
required to keep the L2 contract versions in sync when the branches diverge. In the event that a change is made which modifies the
`current-upgrade-bundle.json` file on the proposal branch (ie. any change to an L2 contract) the following steps would be required:

1. Merge a PR _to the proposal branch_ updating the corresponding `<fork>_nut_bundle.json` file according to the steps outlined [here](../../../../../op-core/nuts/README.md#pr-2--snapshot-the-bundle-for-a-fork). (This would also require adding a special case to the `just check-nut-locks` recipe, similar to what is outlined [here](https://github.com/ethereum-optimism/optimism/pull/20921#discussion_r3282817526)).
2. Cherry pick that PR to the `develop` branch.

## Finalizing a release

Once a release has passed governance, a new tag should be created without the `-rc.n` suffix. To do this follow the
instructions in "Creating a tagged release" once again. It should not be necessary to redeploy the contracts with `op-deployer`,
but a new entry will be required in the superchain-registry's toml files regardless.
When creating release notes, _uncheck_ the `Set as a pre-release`  option, and _uncheck_ the
   `Set as the latest release` option (latest releases are reserved for non-contract packages).
