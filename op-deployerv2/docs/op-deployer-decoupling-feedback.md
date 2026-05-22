# Making op-deployer easier to upgrade

## Summary

`op-deployer` currently knows too much about the exact contracts version it was
built with. That makes every contracts release harder than it should be: when a
deployment script changes, `op-deployer` often needs a matching code change,
release, and rollout.

The direction we are moving toward is simpler:

- `op-deployer` should be a stable tool.
- The contracts/artifacts selected by the user should define the deployment
  inputs, outputs, and compatibility behavior.
- The deployer binary should not secretly carry one specific contracts release.

The first concrete step is removing embedded contract artifacts and making
contract sources/artifacts explicit.

## Why change it?

Today, `op-deployer` is tightly coupled to one contracts release. The coupling
shows up in a few ways:

- The binary has historically included a bundled copy of contract artifacts.
- Some commands assumed those bundled artifacts by default.
- Go code in `op-deployer` mirrors Solidity deployment script inputs and
  outputs.
- When a Solidity deployment script changes, `op-deployer` often has to be
  updated even if the operator workflow did not change.

This creates a few practical problems:

- Contracts and deployer releases become coordinated by accident.
- Testing older or newer contracts with the same deployer is harder.
- It is not obvious which contract artifacts a deployment used.
- Adding a new script input can require deployer code changes, even when the
  input could have been described by the artifact ABI.

The goal is not to throw away what `op-deployer` does today. The goal is to keep
the same useful workflows while removing the hidden dependency on the deployer's
built-in contracts version.

## What changes?

The deployer should treat contracts as an input, not as something baked into the
binary.

Instead of this model:

```text
op-deployer binary
  -> includes one contracts artifact bundle
  -> includes Go structs matching that contracts version
  -> deployment behavior is tied to that release
```

We want this model:

```text
op-deployer binary
  -> user provides contract artifacts/source
  -> deployer reads the script ABI from those artifacts
  -> version-specific mapping lives with the contracts release
```

In practice, this means:

- No embedded artifacts in the `op-deployer` binary.
- Commands that need contracts receive an explicit `file://`, `http://`, or
  `https://` artifacts locator.
- CLI input loading is driven by artifact ABIs instead of hardcoded deployer
  structs where possible.
- Legacy `intent.toml` and `state.json` support remains, but compatibility
  mapping should live next to the generated contract-specific files, not inside
  the stable deployer core.
- Programmatic users can use generated Go types that match the contracts commit
  they are testing against.

## What did it look like before?

The old interaction hid the contracts version in the deployer release.

```bash
op-deployer init \
  --workdir .deployer \
  --l1-chain-id 11155111

# Edit .deployer/intent.toml

op-deployer apply \
  --workdir .deployer \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY"
```

For workflows that needed artifacts directly, users could rely on the embedded
bundle or commands that implied it:

```bash
op-deployer verify \
  --workdir .deployer \
  --l1-rpc-url "$L1_RPC_URL" \
  --artifacts-locator embedded

op-deployer upgrade embedded \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY"
```

That was convenient, but it made the deployed contracts version implicit. The
operator might be using a newer deployer binary than intended, or a contracts
change might require a deployer change for reasons that are not visible in the
CLI.

## What does it look like now?

The contracts source is explicit.

For local contracts:

```bash
ARTIFACTS_LOCATOR="file://$PWD/packages/contracts-bedrock"

op-deployer init \
  --workdir .deployer \
  --l1-chain-id 11155111

# Edit .deployer/intent.toml

op-deployer apply \
  --workdir .deployer \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY" \
  --artifacts-locator "$ARTIFACTS_LOCATOR"
```

For a published artifact bundle:

```bash
ARTIFACTS_LOCATOR="https://example.com/contracts-artifacts.tar.zst"

op-deployer verify \
  --workdir .deployer \
  --l1-rpc-url "$L1_RPC_URL" \
  --artifacts-locator "$ARTIFACTS_LOCATOR"
```

For upgrade flows, the same principle applies: the deployer should not choose a
hidden artifact bundle. The user points the command at the intended contracts
artifact/source.

```bash
op-deployer upgrade v6.0.0-rc.2 \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY" \
  --override-artifacts-url "$ARTIFACTS_LOCATOR"
```

The exact command names may continue to evolve, but the important behavior is
that the contracts input is visible and controlled by the caller.

## How do we keep compatibility?

We still need to support existing users and tests that rely on `intent.toml` and
`state.json`.

The proposed compatibility model is:

- The stable deployer core handles generic concerns: RPC, signing, broadcasting,
  artifact loading, logging, state persistence, and command orchestration.
- The contracts-specific layer handles version-specific concerns: script input
  shapes, output shapes, and legacy intent/state mappings.
- Static mapping files live with the generated contract-specific files, so a
  contracts change and its compatibility update are reviewed together.

This keeps old workflows possible without putting old contracts assumptions back
into the deployer binary.

## Programmatic usage

Some tests and tools import `op-deployer` directly from Go. Those users need
types.

The plan is to generate a typed package from the contracts artifacts. A test or
tool can pin the generated package that matches the contracts commit it wants to
use.

Before:

```go
// Programmatic code imports deployer structs that are maintained manually
// inside op-deployer.
input := opcm.UpgradeInputV2{
    // fields tied to the deployer version
}
```

After:

```go
// Programmatic code imports generated types matching the selected contracts ref.
input := generated.UpgradeInput{
    // fields generated from the selected contracts artifacts
}
```

The CLI does not need to depend on those generated Go types. The CLI can load
the ABI from artifacts and accept YAML input. The generated types are mainly for
Go tests and tools that want compile-time safety.

## What feedback is useful?

The most useful feedback right now is on the product shape:

- Is making the contracts source explicit acceptable for operators?
- Should `--artifacts-locator` be the long-term name, or should we introduce a
  clearer `--contracts-source` style flag?
- Which old workflows must remain byte-for-byte compatible?
- Which workflows can accept a small CLI change if the contracts version becomes
  clearer?
- For programmatic users, is pinning generated types to a contracts commit a
  reasonable workflow?

## Current direction

The direction is to make `op-deployer` less release-coupled one piece at a
time:

1. Stop embedding contract artifacts in the deployer binary.
2. Require explicit artifact/source locators for commands that need contracts.
3. Load CLI input shapes from artifact ABIs where possible.
4. Generate Go types for programmatic consumers.
5. Keep legacy `intent.toml` and `state.json` support through static mappings
   that live with the contracts-specific generated files.

The end state should feel familiar to users, but the contracts version should
always be explicit and inspectable.
