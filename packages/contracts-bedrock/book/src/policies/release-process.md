# Tagging and Release Process

## Creating a tagged release

First select a tag string based on the guidance in [Monorepo Contracts Release Versioning](./versioning.md#monorepo-contracts-release-versioning)

1. Checkout the commit
2. Run `git tag <tag-string>`
3. Run `git push origin <tag-string>`
   Repo [rules](https://github.com/ethereum-optimism/optimism/rules/8196346?ref=refs%2Ftags%2Fop-contracts) require this is done by someone who is a [release-manager](https://github.com/orgs/ethereum-optimism/teams/release-managers). Once pushed a tag cannot be deleted, so please be sure it is correct.
1. Create release notes in Github:
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
1. Create a new release branch, named `proposal/op-contracts/vX.Y.Z` off of the rc tag (all subsequent `-rc` tags
   will be made from this branch).
1. Cherry pick the fixes from `develop` into the release branch, and increment the semver as normal. If this increment results in any of the modified contracts' semver being equal to or greater than it is on `develop`, then the semver should immediately be increased on `develop` to be greater than on the release branch. This avoids a situation where a given contract has two different implementations with the same version.
1. After merging the changes into the new release branch, tag the resulting commit on the proposal branch as `op-contracts/vX.Y.Z-rc.n`.
   Create a new release for this tag per the instructions above.

## Finalizing a release

Once a release has passed governance, a new tag should be created without the `-rc.n` suffix. To do this follow the
instructions in "Creating a tagged release" once again. It should not be necessary to redeploy the contracts with `op-deployer`,
but a new entry will be required in the superchain-registry's toml files regardless.
When creating release notes, _uncheck_ the `Set as a pre-release`  option, and _uncheck_ the
   `Set as the latest release` option (latest releases are reserved for non-contract packages).

