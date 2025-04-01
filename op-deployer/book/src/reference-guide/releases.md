# Releases

## Versioning

For all releases after `v0.0.11`, each minor version of OP Deployer will support a single release of the
governance-approved smart contracts. If you want to deploy an earlier version of the contracts (which may be
dangerous!), you should use an earlier version of OP Deployer. This setup allows our smart contract developers to make
breaking changes on `develop`, while still allowing new chains to be deployed and upgraded using production-ready smart
contracts.

If you deploy from an HTTPS or file [locator](./artifacts-locators.md), the deployment behavior will match the
contract's tag. For example, if version `v0.2.0` supports `v2.0.0` then the deployment will work as if you were
deploying `op-contracts/v2.0.0`. Typically, errors like `unknown selector: <some hex>` imply that you're using the wrong
version of OP Deployer for your contract artifacts. If this happens, we recommend trying different versions until you
get one that works. Note that this workflow is **not recommended** for production chains.

[releases]: https://github.com/ethereum-optimism/optimism/releases

## Version Backports

From time to time, we may backport bugfixes from develop onto earlier versions of OP Deployer. The process for this is
as follows:

1. If one doesn't exist already, make a new branch for the version lineage you're patching (e.g. `v0.2.x`). This branch
   should be protected (not deletable) and should be based on the latest release of that lineage. The branch should be named as follows:
   `backports/op-deployer/<lineage, i.e. v0.2.0>`.
2. Open a PR with the backport against that branch. Be sure to reference the original commit in the backport.
3. Make and push a new tag on that lineage.

Example for backporting fix(es) from `develop` and created a new release `op-deployer/v0.2.1`:
```
git checkout -b backports/op-deployer/v0.2.0 op-deployer/v0.2.0
git push origin backports/op-deployer/v0.2.0
git checkout -b fixes/deployer-v0.2.0 backports/op-deployer/v0.2.0
git cherry-pick <commit-hash>
git push origin fixes/deployer-v0.2.0

1. open pr from fixes/deployer-v0.2.0 targeting backports/op-deployer/v0.2.0
2. merge the pr
3. push a new tag for op-deployer/v0.2.1 on backports/op-deployer/v0.2.0 branch (goreleaser will create the release)
```

## Adding Support for New Contract Versions

Adding support for a new contract version is a multi-step process. Here's a high-level overview. For the sake of
simplicity we will assume you are adding support for a new `rc` release.

### Step 1: Add Support on `develop`

**This section is designed for people developing OP Deployer itself.**

First, you need to add support for the new contract version on the `develop` branch. This means ensuring that the
deployment pipeline supports whatever changes are required for the new version. Typically, this means passing in new
deployment variables, and responding to ABI changes in the Solidity scripts/OPCM.

### Step 2: Add the Published Artifacts

Run the following from the root of the monorepo:

```bash
cd packages/contracts-bedrock
just clean
just build
bash scripts/ops/calculate-checksum.sh
# copy the outputted checksum
cd ../../op-deployer
just calculate-artifacts-hash <checksum>
```

This will calculate the checksum of your artifacts as well as the hash of the artifacts tarball. OP Deployer uses
these values to download and verify tagged contract locators.

Now, update `standard/standard.go` with these values so that the new artifacts tarball can be downloaded:

```go
// Add a new const for your release

const ContractsVXTag = "op-contracts/vX.Y.Z"

var taggedReleases = map[string]TaggedRelease{
    // Other releases...
    ContractsVXTag: {
		ArtifactsHash: common.HexToHash("<the artifacts hash>"),
		ContentHash:   common.HexToHash("<the checksum>"),
	},
}

// Update the L1/L2 versions accordingly
func IsSupportedL1Version(tag string) bool {
	return tag == ContractsVXTag
}
```

### Step 3: Update the SR With the New Release

Add the new RC to the [standard versions][std-vers] in the Superchain Registry.

[std-vers]: https://github.com/ethereum-optimism/superchain-registry/tree/main/validation/standard

### Step 4: Update the `validation` Package

The SR is pulled into OP Deployer via the `validation` package. Update it by running the following command from the
root of the monorepo:

```shell
go get -u github.com/ethereum-optimism/superchain-registry/validation@<SR commit SHA>
```

That should be it!
