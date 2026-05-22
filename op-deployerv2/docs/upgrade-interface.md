# op-deployerv2 Upgrade Interface Notes

## Problem

The current `op-deployer` upgrade path is tightly coupled to a specific OPCM
input schema. For example, the current implementation has Go structs and custom
encoders for fields like `systemConfig`, `disputeGameConfigs`, and
`extraInstructions`.

That means a new OPCMv2 input shape can require a new `op-deployer` release.
This is the failure mode `op-deployerv2` should avoid.

## Core Boundary

`op-deployerv2` may hardcode the user workflow:

```text
op-deployerv2 upgrade
```

For this workflow, the tool always calls:

```solidity
OPContractsManagerV2.upgrade(...)
```

But `op-deployerv2` must not hardcode:

- the `upgrade` argument names
- the `UpgradeInput` struct fields
- nested tuple fields
- array item shapes
- future OPCM inputs
- dispute-game-specific config structs

The `upgrade` input schema comes from the supplied OPCM artifact or ABI at
runtime.

## CLI Shape

The default CLI should be plan-first:

```bash
op-deployerv2 upgrade \
  --network sepolia \
  --rpc-url $RPC \
  --opcm 0xOPCM \
  --artifact ./OPContractsManagerV2.json \
  --executor 0xExecutor \
  --input ./upgrade.yml \
  --out ./upgrade-plan.json
```

Stable CLI flags are execution and environment concerns:

- `--network`
- `--rpc-url`
- `--opcm`
- `--artifact` or `--abi`
- `--executor`
- `--input`
- `--out`
- `--broadcast`

OPCM function inputs should not become stable first-class CLI flags.

## YAML Input

The input file is ABI-shaped. For current OPCMv2, it would look like:

```yaml
_inp:
  systemConfig: "0xSystemConfig"
  disputeGameConfigs:
    - enabled: true
      initBond: "0"
      gameType: 0
      gameArgs: "0x..."
  extraInstructions: []
```

If a future OPCM adds a field, only the YAML changes:

```yaml
_inp:
  systemConfig: "0xSystemConfig"
  disputeGameConfigs: []
  extraInstructions: []
  newFieldAddedByFutureOPCM: "0x..."
```

The binary should validate this object against the `upgrade` ABI at runtime.

## Inline CLI Input

The CLI should also support direct ABI-typed input flags:

```bash
op-deployerv2 upgrade \
  --opcm 0xOPCM \
  --artifact ./OPContractsManagerV2.json \
  --executor 0xExecutor \
  --input.systemConfig=0xSystemConfig \
  --input.disputeGameConfigs='[
    {
      "enabled": true,
      "initBond": "0",
      "gameType": 0,
      "gameArgs": "0x..."
    }
  ]' \
  --input.extraInstructions='[]'
```

For the current single-top-level-tuple `upgrade` ABI, the CLI may allow:

```bash
--input.systemConfig=0x...
```

as shorthand for:

```bash
--input._inp.systemConfig=0x...
```

The shorthand is only valid when `upgrade` has one top-level tuple argument.

## Typed Dynamic Flags

`--input.*` flags are dynamically typed from the ABI.

Examples:

```bash
--input.systemConfig=0x...
--input.disputeGameConfigs='[...]'
--input.extraInstructions='[]'
```

Parser behavior:

- scalar ABI types parse as scalar values
- tuple and array ABI types parse as JSON or YAML inline objects
- indexed paths remain available as an escape hatch

Example indexed path:

```bash
--input.disputeGameConfigs[0].enabled=true
--input.disputeGameConfigs[0].gameType=0
```

For nested arrays, the JSON-array form should be the documented default.

## Bytes Helpers

Some OPCM fields are typed as `bytes` but contain nested ABI-encoded data. In
current OPCMv2, `disputeGameConfigs[].gameArgs` is one of these fields.

The tool should always accept raw bytes:

```yaml
gameArgs: "0x..."
```

It may also support a generic ABI-encoding helper:

```yaml
gameArgs:
  abiType: "(bytes32 absolutePrestate,address proposer,address challenger)"
  value:
    absolutePrestate: "0x..."
    proposer: "0x..."
    challenger: "0x..."
```

This helper is generic. It does not require `op-deployerv2` to know about
`FaultDisputeGameConfig`, `PermissionedDisputeGameConfig`, or future dispute
game structs.

## Merge Rules

Inputs may come from a file and direct CLI overrides:

```bash
op-deployerv2 upgrade \
  --input ./upgrade.yml \
  --input.extraInstructions='[]'
```

Precedence:

```text
YAML input < inline --input.* overrides
```

All merged values are validated against the runtime ABI before calldata is
produced.

## Programmatic Interface

Programmatic callers should use generated typed packages, not caller-owned
handwritten structs.

The generated package is produced from the OPCM artifact or ABI:

```bash
op-deployerv2 gen go \
  --artifact ./OPContractsManagerV2.json \
  --function upgrade \
  --package opcmupgrade \
  --out ./internal/generated/opcmupgrade
```

Tests and internal tools then use the generated types:

```go
input := opcmupgrade.NewInput()

input.Inp.SystemConfig.Set(systemConfig)
input.Inp.DisputeGameConfigs.Append(opcmupgrade.DisputeGameConfig{
    Enabled:  true,
    InitBond: big.NewInt(0),
    GameType: 0,
    GameArgs: gameArgs,
})
input.Inp.ExtraInstructions.SetEmpty()

plan, err := runtime.Upgrade(ctx, opdeployerv2.UpgradeRequest{
    Executor: executor,
    Input:    input,
})
```

This is typed, but the type is generated mechanically from the ABI. Neither
`op-deployerv2` nor the caller handwrites release-specific OPCM input structs.

The generated package should be checked into the repository when used by tests
or other compiled Go packages. CI should verify that generated files are up to
date.

Expected workflow:

```bash
op-deployerv2 gen go \
  --artifact ./OPContractsManagerV2.json \
  --function upgrade \
  --package opcmupgrade \
  --out ./internal/generated/opcmupgrade

git diff --exit-code ./internal/generated/opcmupgrade
```

If the ABI changes but generated code is not refreshed, CI should fail.

There is no fallback runtime schema API in the programmatic surface. Programmatic
callers are expected to have the ABI at compile time and to use checked-in,
generated Go packages.

## Internal Pipeline

Every input source should converge on the same pipeline:

```text
YAML / JSON / generated Go input
  -> ABI-shaped value
  -> runtime ABI validation and coercion
  -> encode OPCM.upgrade calldata
  -> simulate
  -> emit plan
  -> optionally broadcast
```

## Outputs

The primary output should be generated from the ABI or script artifact, just
like the input.

If a script changes its return values, the generated output type and CLI output
shape should change with it. `op-deployerv2` should not hide that behind a
handwritten normalized deployment object.

The stable part of an output is only the execution envelope:

- operation name
- artifact or ABI digest
- contract or script address/source
- execution mode
- transaction plan
- receipts or simulation result
- raw/generated decoded return value

Example shape:

```yaml
operation: upgrade
artifactDigest: "0x..."
execution:
  mode: delegatecall
  executor: "0x..."
  transactions:
    - to: "0x..."
      data: "0x..."
return:
  _generatedFrom: "OPContractsManagerV2.upgrade"
  value:
    systemConfig: "0x..."
    proxyAdmin: "0x..."
```

## Intent And State Adapters

The core ABI/script runner should stay generated-input/generated-output first,
but `op-deployerv2` should support legacy `intent` and `state` through adapter
packages.

Adapters should live in the selected contracts commit/release source, next to
the generated Go packages for that contracts version. They are not built into
the core `op-deployerv2` release.

Example layout:

```text
packages/contracts-bedrock/op-deployerv2/
  generated/
    opcm/
      upgrade.gen.go
    scripts/
      deploy_op_chain.gen.go
      deploy_implementations.gen.go
  adapters/
    intent.go
    state.go
  cmd/
    adapter/
```

The generated files are mechanically produced from the ABI/script artifacts.
The adapter files map between legacy deployer concepts and those generated
types:

```text
legacy intent/state
  -> adapter
  -> generated script inputs
  -> v2 execution
  -> generated script outputs
  -> adapter
  -> legacy-compatible state, if needed
```

Adapters are allowed to know release-specific semantics because they sit next to
the release-specific generated package. If a contracts change alters an input or
output and the adapter no longer compiles or tests fail, CI should catch it.

When the CLI is pointed at a contracts commit, core `op-deployerv2` should fetch
or use that source, load its ABIs/artifacts, and invoke the adapter from that
source when legacy intent/state behavior is requested.

Example:

```bash
op-deployerv2 apply \
  --contracts-source github.com/ethereum-optimism/optimism@<commit> \
  --intent ./intent.toml \
  --out ./state.json
```

The `apply` workflow is stable, but the adapter that translates intent/state for
that workflow is selected from `<commit>`, not from the installed deployer
binary.

Core v2 should communicate with adapters through a small stable adapter
protocol. The adapter implementation can be Go code in the contracts source,
for example under `cmd/adapter`, but it is built or run from that selected
source. It is not linked into the core deployer binary.

The adapter protocol should be stable at the level of deployer concepts:

- read legacy intent
- produce generated script inputs
- consume generated script outputs and execution results
- produce legacy-compatible state when requested

The protocol should not include OPCM-version-specific structs. Those stay in
the generated package in the contracts source.

This keeps the core rule intact:

- no handwritten OPCM input/output structs in core `op-deployerv2`
- generated ABI/script packages are the typed interface
- adapter code is local to the contracts commit/release source and CI-enforced

## Execution Concerns

Current OPCMv2 `upgrade` requires delegatecall. The CLI should expose this as an
execution concern, not as an OPCM input field.

Preferred wording:

```bash
--executor 0xExecutor
```

or, if the distinction must be explicit:

```bash
--delegate-caller 0xDelegateCaller
```

Avoid exposing Foundry-specific terminology like `prank` in the public UX.

## Open Questions

- Should `--input` accept YAML only, or YAML and JSON based on file extension?
- Should inline `--input.*` values parse JSON first for tuple/array ABI types?
- Should the shorthand that drops the top-level `_inp` be enabled by default?
- What should the canonical plan file schema look like?
- How much of the delegatecall execution route should be represented in the
  plan file?
