<h1 align="center">
<img src="./assets/banner.png" alt="Kona" width="100%" align="center">
</h1>

# Kona

Kona is the Rust implementation of OP Stack rollup components in the
Optimism monorepo (`ethereum-optimism/optimism`).

## What is here

- `bin/client`: fault proof program for prover targets.
- `bin/host`: native preimage-oracle host program.
- `bin/node`: Rust rollup-node implementation.
- `crates/protocol/*`: derivation, genesis, hardfork, interop, registry, and
  protocol types.
- `crates/proof/*`: proof SDK, executor, MPT, preimage, and FPVM support.
- `crates/node/*`: rollup-node service, engine, RPC, P2P, and source utilities.
- `crates/providers/*` and `crates/utilities/*`: shared provider and utility
  crates.

OP Stack Alloy extensions live in
[`rust/op-alloy`](../op-alloy).

## Working in this directory

Run Kona commands from `rust/kona`:

- `just b` / `just build-native`: build the workspace.
- `just t` / `just tests`: run tests.
- `just l` / `just lint-native`: run lint checks.
- `just f` / `just fmt-native-fix`: format the workspace.
- `just test-docs`: test documentation examples.

The Rust toolchain and MSRV are pinned in [`../rust-toolchain.toml`](../rust-toolchain.toml).
Release the kona crates from `rust/` with `just release kona <version>` (see the recipe in [`../justfile`](../justfile)).

## Links

- [OP Stack specifications](https://specs.optimism.io/)
- [Optimism monorepo](https://github.com/ethereum-optimism/optimism)
- [Contributing guide](../../CONTRIBUTING.md)
- [License](./LICENSE.md)
