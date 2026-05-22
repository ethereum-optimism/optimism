# Generated Binding Pilot

This document proposes the first concrete coupling burn-down: replace one
manually maintained script binding with generated Go from the corresponding
contracts-bedrock artifact.

## Target

Start with `DeploySuperchain`.

Current manual binding:

- `op-deployer/pkg/deployer/opcm/superchain.go`

Current callers:

- `op-deployer/pkg/deployer/bootstrap/superchain.go`
- `op-deployer/pkg/deployer/pipeline/superchain.go`

Corresponding contracts source:

- `packages/contracts-bedrock/scripts/deploy/DeploySuperchain.s.sol`
- compiled artifact:
  `packages/contracts-bedrock/forge-artifacts/DeploySuperchain.s.sol/DeploySuperchain.json`

## Why this one first

`DeploySuperchain` is the right first target because it is small but still
exercises the important architecture:

- it has a Solidity `Input` struct
- it has a Solidity `Output` struct
- it is currently manually mirrored in Go
- it has both script-host and Forge execution paths
- it is used by a standalone CLI command and by the `apply` pipeline
- replacing it should not require changing public CLI behavior

This lets us prove the generator, CI freshness check, and migration pattern
before touching larger surfaces like `DeployImplementations` or
`DeployOPChain`.

## Current coupling

`opcm/superchain.go` currently hardcodes:

- `DeploySuperchainInput`
- `DeploySuperchainOutput`
- `DeploySuperchain.s.sol`
- `DeploySuperchain`
- Forge path `scripts/deploy/DeploySuperchain.s.sol:DeploySuperchain`
- `runWithBytes(bytes)`
- `DeploySuperchainInput` / `DeploySuperchainOutput` type names for Forge
  byte encoding

The Go structs must manually stay in sync with the Solidity structs. If the
contract script adds an input or output field, `op-deployer` must be changed by
hand.

## Generator source of truth

The generator should read the compiled Foundry artifact for the selected
contracts source:

```text
forge-artifacts/DeploySuperchain.s.sol/DeploySuperchain.json
```

For type generation, use the ABI for:

```solidity
function run(Input memory _input) public returns (Output memory output_)
```

Do not derive the schema from `runWithBytes(bytes)`, because that ABI only says
`bytes -> bytes`. `runWithBytes` is useful for the Forge transport convention,
but the typed input and output schema comes from `run`.

## Generated output

The first generated package can live under `op-deployer` while we are burning
down v1:

```text
op-deployer/pkg/deployer/opcm/generated/deploysuperchain/deploy_superchain.gen.go
```

It should contain mechanically generated equivalents of:

```go
type Input struct {
    // generated from DeploySuperchain.Input
}

type Output struct {
    // generated from DeploySuperchain.Output
}

const ScriptFile = "DeploySuperchain.s.sol"
const ContractName = "DeploySuperchain"
const ForgeScriptPath = "scripts/deploy/DeploySuperchain.s.sol:DeploySuperchain"
const RunWithBytesSignature = "runWithBytes(bytes)"
const InputTypeName = "DeploySuperchainInput"
const OutputTypeName = "DeploySuperchainOutput"
```

The manual package can keep compatibility aliases at first:

```go
type DeploySuperchainInput = deploysuperchain.Input
type DeploySuperchainOutput = deploysuperchain.Output
```

That keeps downstream callers stable while removing handwritten struct
ownership from the core package.

## First implementation steps

1. Add a small generator command that reads a Foundry artifact and a function
   name:

   ```bash
   op-deployer gen script-binding \
     --artifact packages/contracts-bedrock/forge-artifacts/DeploySuperchain.s.sol/DeploySuperchain.json \
     --function run \
     --package deploysuperchain \
     --out op-deployer/pkg/deployer/opcm/generated/deploysuperchain
   ```

2. Generate `Input`, `Output`, and script metadata from the ABI.

3. Replace the handwritten structs in `opcm/superchain.go` with aliases to the
   generated types.

4. Update `NewDeploySuperchainScript` and `NewDeploySuperchainForgeCaller` to
   use generated constants and generated types.

5. Keep `bootstrap/superchain.go` and `pipeline/superchain.go` behavior
   unchanged.

6. Add a freshness check:

   ```bash
   op-deployer gen script-binding \
     --artifact packages/contracts-bedrock/forge-artifacts/DeploySuperchain.s.sol/DeploySuperchain.json \
     --function run \
     --package deploysuperchain \
     --out op-deployer/pkg/deployer/opcm/generated/deploysuperchain

   git diff --exit-code op-deployer/pkg/deployer/opcm/generated/deploysuperchain
   ```

## Acceptance criteria

- No public CLI changes.
- `bootstrap superchain` still works.
- `apply` still deploys superchain through the same pipeline stage.
- `opcm/superchain.go` no longer owns handwritten `DeploySuperchainInput` and
  `DeploySuperchainOutput` struct definitions.
- Generated files are checked in.
- CI fails if the Solidity `DeploySuperchain.Input` or
  `DeploySuperchain.Output` ABI changes and the generated Go is stale.

## Next targets

After this is proven, burn down the larger bindings in increasing blast radius:

1. `DeployImplementations`
2. `ReadSuperchainDeployment`
3. `DeployOPChain`
4. `ReadImplementationAddresses`
5. `L2Genesis`
6. upgrade and migrate bindings

`DeployImplementations` is the likely second target because it has a larger
input/output surface but follows the same pattern.
