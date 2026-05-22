# op-deployer Usage and Transition Analysis

This note captures where `op-deployer` is used in the monorepo and how hard
each use would be to move to `op-deployerv2`.

Classification:

- **Trivial transition**: the consuming file mostly needs a rename, import move,
  command substitution, or release artifact path change once `op-deployerv2`
  exposes the equivalent surface.
- **Complex transition**: the consumer depends on current `op-deployer`
  internals, state schemas, typed script bindings, embedded artifacts, or
  current pipeline behavior. More functionality is needed in `op-deployerv2`
  before the consumer can move cleanly.

## Executive Summary

The biggest transition risk is not public CLI documentation. The biggest risk
is internal Go tooling using `op-deployer` as a library.

Complex consumers fall into five groups:

1. Genesis/deployment pipelines that call `deployer.ApplyPipeline`.
2. Test builders that construct `state.Intent` and read `state.State`.
3. Script-host consumers that rely on `opcm` typed script bindings.
4. Upgrade/manage consumers that rely on hardcoded OPCMv2 Go structs.
5. Superchain registry tooling that parses `state.json` and regenerates
   genesis/rollup data from it.

Trivial consumers are mostly docs, image names, release plumbing, and constants.

## Required op-deployerv2 Capabilities

The usage scan implies `op-deployerv2` needs more than the new ABI-driven
`upgrade` command if it is intended to replace all current `op-deployer` usage.

Required capabilities:

- ABI-driven `upgrade` workflow with YAML input, inline `--input.*` overrides,
  generated Go input, delegatecall execution, simulation, and plan/broadcast.
- A deterministic Go code generator for ABI/script input packages, with a CI
  check that fails when checked-in generated code is stale.
- Artifact locator support for explicit file and remote artifact locators.
- Contract artifact build/packaging support through a standalone artifact
  packer, without embedding artifacts in the deployer binary.
- A script runtime that can run release scripts from an artifact/source bundle
  and generate input/output bindings from script ABIs.
- Bootstrap support for superchain and implementations scripts from a selected
  release/commit.
- Generated output packages and CLI output documents derived from script/ABI
  return values.
- Generated deployment/script execution for current deployment scripts, if v2
  is expected to cover more than upgrades.
- Contracts-source-local intent and state adapters that live next to generated
  ABI/script packages and are checked by CI. These adapters are loaded from the
  selected contracts commit or release source, not from the core deployer
  binary.
- A stable adapter protocol that lets core v2 invoke those adapters without
  knowing release-specific OPCM or script structs.
- Validator and verifier support if `verify` and `validate auto` are kept.
- A small stable constants package, or a plan to move constants currently in
  `op-deployer/pkg/deployer/standard` into a neutral package.

## Transition Table

| Area | Current usage | Transition | Needed in op-deployerv2 |
| --- | --- | --- | --- |
| Public operator docs | Document `op-deployer init`, `apply`, `inspect`, `bootstrap`, `verify`, `upgrade`, and install flows. | **Trivial transition** for text once command parity exists. | Compatible CLI story and updated examples. |
| Create L2 rollup example | Downloads `op-deployer`, runs `init`, edits `intent.toml`, runs `apply`, `inspect genesis`, and `inspect rollup`. | **Complex transition**. | Generated deployment flow plus contracts-source-local intent/state adapters. |
| `op-deployer/scripts/test-sepolia-op-deployer.sh` | Interactive full lifecycle: choose binary/docker/go-run, bootstrap superchain/implementations, init, apply, verify, validate. | **Complex transition**. | Bootstrap, init, apply, verify, validate, release/docker packaging. |
| `op-e2e/config/init.go` | Programmatically builds `state.Intent`, calls `deployer.ApplyPipeline` for genesis deployments, reads `state.State`, `inspect.DeployConfig`, and `inspect.L1`. | **Complex transition**. | Genesis deployment API, state model, inspect APIs, artifact locators. |
| `op-e2e/e2eutils/intentbuilder` | Builder API around `state.Intent`, `state.ChainIntent`, `state.AdditionalDisputeGame`, and `standard` constants. | **Complex transition**. | Contracts-source-local intent adapter or rewrite around generated v2 input packages. |
| `op-devstack/sysgo/deployer.go` | Builds devstack worlds with `ApplyPipeline`, receives `state.State` through `StateWriter`, then builds L1/L2 genesis and rollup configs. | **Complex transition**. | Programmatic deploy pipeline, state writer, genesis/rollup inspection. |
| `op-devstack/sysgo/add_game_type.go` | Builds hardcoded `current.UpgradeInputV2`, then calls OPCMv2 upgrade via script host and SetCode delegatecall. | **Complex transition**. | ABI-driven upgrade API, generated Go input, nested bytes helper, delegatecall executor. |
| `op-devstack/sysgo/superroot_via_upgrade.go` | Builds hardcoded super-root dispute game upgrade inputs and extra instructions. | **Complex transition**. | Same upgrade API plus generic encoding for `extraInstructions` override data. |
| `op-devstack/sysgo/opcm_upgrade.go` | Uses `env.DefaultForkedScriptHost`, `broadcaster.CalldataBroadcaster`, and `current.Upgrade` to produce calldata. | **Complex transition**. | Script execution or direct ABI execution path with calldata capture. |
| `op-devstack/sysgo/pre_genesis_super_game.go` | Mutates `state.State` start block and rebuilds L2 genesis/dep-set after an interop migration. | **Complex transition**. | Mutable state/rollup regeneration API or equivalent devstack hook. |
| `op-chain-ops/interopgen` | Uses `opcm` script bindings for superchain, implementations, OP chain, L2 genesis; uses `manage.Migrate` for interop migration. | **Complex transition**. | Generated script bindings or generic script runner; migrate support. |
| `op-fetcher` | Uses `env.DefaultForkedScriptHost`, `broadcaster.NoopBroadcaster`, and `opcm.RunScriptSingle` to run `FetchChainInfo.s.sol`. | **Complex transition**. | Generic read-only script runner or move script runtime to a neutral package. |
| Superchain registry `create_config` | Reads an `op-deployer` `state.json`, inflates registry config, calls `inspect.GenesisAndRollup`. | **Complex transition**. | State adapter that can emit legacy-compatible state or equivalent generated outputs. |
| Superchain registry L1/L2 reports | Uses artifact locators, standard OPCM address lookup, `state.CombineDeployConfig`, script host, and v170 L2 genesis bindings. | **Complex transition**. | State/deploy-config adapter plus generated L2 genesis script package. |
| `op-validator` | Imports `standard` release tag constants and standard validator addresses mention OP Deployer bootstraps. | **Trivial transition**. | Move or mirror constants; no deployment pipeline dependency. |
| `op-service/github/release` | Tests include `op-deployer` as a release asset/module name. | **Trivial transition**. | Add or rename release asset metadata. |
| `op-up/justfile`, Rust test justfiles | Build contracts and create an external `artifacts.tzst` archive with `mktar`. | **Trivial transition**. | Artifact packaging recipe and stable archive location. |
| `packages/contracts-bedrock/scripts/ops/publish-artifacts.sh` | Enters `op-deployer`, runs `just build-contracts`, creates an upload archive with `mktar`, uploads archive. | **Trivial transition**. | Same standalone artifact packer. |
| Docker build/release | Builds `op-deployer` image without embedding contract artifacts, reads `forge/version.json`, has docker-bake target. | **Trivial transition** for Docker plumbing. | Binary target, image target, forge version file. |
| CircleCI release/build | Checks forge version and releases tags matching `op-deployer/*`; artifacts are external locators. | **Trivial transition** for CI wiring. | Equivalent just recipes and release tag convention. |
| Rollup Boost Kurtosis params | Pins an `op-deployer` docker image for contract deployment. | **Trivial transition** if package accepts new image/name. | Published image with compatible Kurtosis deployment behavior. |
| CODEOWNERS and docker image metadata | Lists `op-deployer` paths/images. | **Trivial transition**. | Path/image rename. |

## Detailed Notes

### Public Docs and Examples

Files under `docs/public-docs/chain-operators/tools/op-deployer`,
`docs/public-docs/chain-operators/tutorials/create-l2-rollup`, and the
`docs/public-docs/create-l2-rollup-example` scripts are operator-facing.

They currently assume:

- a released binary named `op-deployer`
- an `init` command that writes `.deployer/intent.toml`
- an `apply` command that deploys contracts and writes state
- `inspect genesis` and `inspect rollup`
- optional `verify` and `validate auto`

The text migration is easy, but only after those command-level behaviors exist
in `op-deployerv2`. The create-rollup example is therefore complex as a working
system even though the eventual file edits are simple.

### op-e2e

`op-e2e/config/init.go` is a heavy programmatic consumer. It generates in-memory
dev genesis allocs by constructing `state.Intent` and calling:

```go
deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
    DeploymentTarget: deployer.DeploymentTargetGenesis,
    DeployerPrivateKey: pk,
    Intent: intent,
    State: st,
    StateWriter: pipeline.NoopStateWriter(),
})
```

It then uses inspect helpers to extract deploy config, L1 contract addresses,
L2 allocs, and rollup data.

This is a complex transition. The preferred path is an adapter from the selected
contracts source that maps the legacy intent model into generated deployment
script inputs and maps generated outputs back into the state shape current
tests need.

### op-devstack

`op-devstack/sysgo/deployer.go` is similar but more integrated with devstack. It
uses `ApplyPipeline` to produce world state, implements a `StateWriter`, then
derives:

- L1 genesis
- L2 genesis per chain
- rollup configs
- deployment address objects
- interop dependency sets

This is complex because it depends on both execution behavior and state shape.

The upgrade-related files are closer to the planned v2 upgrade design:

- `add_game_type.go`
- `superroot_via_upgrade.go`
- `opcm_upgrade.go`

They currently construct hardcoded `current.UpgradeInputV2` structs and route
through `UpgradeOPChain.s.sol` to satisfy OPCMv2 delegatecall requirements.

These should become cleaner with v2, but they still require real v2 work:

- Generated Go input packages checked into the repo and verified in CI.
- Generic encoding for `bytes` fields like `gameArgs`.
- Generic encoding for `extraInstructions` data.
- Calldata capture and execution through a delegatecall executor.

### op-chain-ops interopgen

`op-chain-ops/interopgen` imports:

- `op-deployer/pkg/deployer/opcm`
- `op-deployer/pkg/deployer/manage`

It uses typed script bindings for:

- `DeploySuperchain`
- `DeployImplementations`
- `DeployOPChain`
- `L2Genesis`
- `migrate`

This is complex because it is effectively using `op-deployer` as the generated
Go SDK for Foundry deployment scripts. A clean v2 transition needs a generic
script runner or generated bindings that are produced from the selected
contracts commit.

### op-fetcher

`op-fetcher` is not deploying contracts, but it still depends on
`op-deployer` internals:

- `env.DefaultForkedScriptHost`
- `broadcaster.NoopBroadcaster`
- `opcm.RunScriptSingle`

It runs `FetchChainInfo.s.sol` against a forked L1 RPC and turns the output into
chain config JSON.

This is complex unless the script runtime is moved out of `op-deployer` into a
neutral package or re-exposed by `op-deployerv2`.

### Superchain Registry

The superchain registry submodule has several dependencies on `op-deployer`.

`ops/cmd/create_config` and `ops/internal/manage/staging.go` read an
`op-deployer` `state.json`, call `inspect.DeployConfig` and
`inspect.GenesisAndRollup`, then inflate a staged registry chain config.

`ops/internal/report` also uses:

- artifact locators and downloads
- standard release tag/address lookup
- `state.CombineDeployConfig`
- script host construction
- v170 L2 genesis script bindings

This is complex. Registry integration should be handled through the same
contracts-source-local state adapter model rather than hardcoding
registry-specific output into core v2.

### op-validator

`op-validator` imports `op-deployer/pkg/deployer/standard` for release tag
constants like `ContractsV500Tag` and `ContractsV600Tag`.

This is a trivial transition. The constants should move to a neutral package or
be mirrored in `op-deployerv2`. No deployment engine behavior is required.

### Artifact Packaging Consumers

These consumers mainly need an external contract artifact archive:

- `op-up/justfile`
- `rust/kona/tests/justfile`
- `rust/op-reth/crates/tests/justfile`
- `packages/contracts-bedrock/scripts/ops/publish-artifacts.sh`
- CircleCI contract build jobs

The transition is trivial if `op-deployerv2` preserves:

- a `build-contracts` recipe
- an `mktar`-style artifact packer recipe
- a stable archive path or documented replacement

If v2 changes the archive format or path, these become complex because multiple
independent build systems consume the archive directly.

### Release, Docker, and CI Plumbing

Release/image plumbing references include:

- `docker-bake.hcl`
- `ops/docker/op-stack-go/Dockerfile`
- `.circleci/continue/main.yml`
- `.github/docker-images.json`
- `.github/CODEOWNERS`
- `op-service/github/release`

These are trivial once the desired release shape is known. They are mostly
string/path/tag updates.

## Recommended Migration Order

1. Define stable v2 library packages before porting call sites:
   - artifacts
   - script runtime
   - ABI input/encoding
   - plan/broadcast
   - generated output
   - contracts-source-local intent/state adapters
2. Port the upgrade-specific devstack consumers first. They match the new v2
   direction and will validate the ABI-driven input design.
3. Port artifact packaging recipes next, because they unblock Docker, CI, Rust,
   and op-up consumers.
4. Add contracts-source-local intent/state adapters for the current deployment
   flows.
5. Move simple constants out of `op-deployer/pkg/deployer/standard` so
   `op-validator` and similar packages are no longer tied to deployment tooling.

## Bottom Line

If `op-deployerv2` is only the new ABI-driven upgrade tool, then the upgrade
call sites are the realistic first migration target.

If `op-deployerv2` is intended to replace all current `op-deployer` usage, it
also needs generated-input/generated-output equivalents for the current
deployment and inspection workflows plus contracts-source-local intent/state
adapters.
