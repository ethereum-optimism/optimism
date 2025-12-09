<h1 align="center">
<img src="./assets/banner.png" alt="Kona" width="100%" align="center">
</h1>

<h4 align="center">
    The Monorepo for <a href="https://specs.optimism.io/">OP Stack</a> Types, Components, and Services built in Rust.
</h4>

<p align="center">
  <a href="https://github.com/op-rs/kona/releases"><img src="https://img.shields.io/github/v/release/op-rs/kona?style=flat&labelColor=1C2C2E&color=C96329&logo=GitHub&logoColor=white"></a>
  <a href="https://docs.rs/kona-derive/"><img src="https://img.shields.io/docsrs/kona-derive?style=flat&labelColor=1C2C2E&color=C96329&logo=Rust&logoColor=white"></a>
  <a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://img.shields.io/github/actions/workflow/status/op-rs/kona/rust_ci.yaml?style=flat&labelColor=1C2C2E&label=ci&color=BEC5C9&logo=GitHub%20Actions&logoColor=BEC5C9" alt="CI"></a>
  <a href="https://app.codecov.io/gh/op-rs/kona"><img src="https://img.shields.io/codecov/c/gh/op-rs/kona?style=flat&labelColor=1C2C2E&logo=Codecov&color=BEC5C9&logoColor=BEC5C9" alt="Codecov"></a>
  <a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?style=flat&labelColor=1C2C2E&color=BEC5C9&logo=googledocs&label=license&logoColor=BEC5C9" alt="License"></a>
  <a href="https://rollup.yoga"><img src="https://img.shields.io/badge/Docs-854a15?style=flat&labelColor=1C2C2E&color=BEC5C9&logo=mdBook&logoColor=BEC5C9" alt="Docs"></a>
</p>

<p align="center">
  <a href="#whats-kona">What's Kona?</a> •
  <a href="#overview">Overview</a> •
  <a href="#msrv">MSRV</a> •
  <a href="https://rollup.yoga/intro/contributing">Contributing</a> •
  <a href="#credits">Credits</a> •
  <a href="#license">License</a>
</p>


## What's Kona?

Originally a suite of portable implementations of the OP Stack rollup state transition,
Kona has been extended to be _the monorepo_ for <a href="https://specs.optimism.io/">OP Stack</a>
types, components, and services built in Rust. Kona provides an ecosystem of extensible, low-level
crates that compose into components and services required for the OP Stack.

The [docs][site] contains a more in-depth overview of the project, contributor guidelines, tutorials for
getting started with building your own programs, and a reference for the libraries and tools provided by Kona.

## Overview

> [!NOTE]
>
> Ethereum (Alloy) types modified for the OP Stack live in [op-alloy](https://github.com/alloy-rs/op-alloy).

**Binaries**

- [`client`](./bin/client): The bare-metal program that executes the state transition, to be run on a prover.
- [`host`](./bin/host): The host program that runs natively alongside the prover, serving as the [Preimage Oracle][g-preimage-oracle] server.
- [`node`](./bin/node): [WIP] A [Rollup Node][rollup-node-spec] implementation, backed by [`kona-derive`](./crates/protocol/derive). Supports flexible chain ID specification via `--l2-chain-id` using either numeric IDs (`10`) or chain names (`optimism`).
- [`supervisor`](./bin/supervisor): [WIP] A [Supervisor][supervisor-spec] implementation.

**Protocol**

- [`genesis`](./crates/protocol/genesis): Genesis types for OP Stack chains.
- [`protocol`](./crates/protocol/protocol): Core protocol types used across OP Stack rust crates.
- [`derive`](./crates/protocol/derive): `no_std` compatible implementation of the [derivation pipeline][g-derivation-pipeline].
- [`driver`](./crates/proof/driver): Stateful derivation pipeline driver.
- [`interop`](./crates/protocol/interop): Core functionality and primitives for the [Interop feature](https://specs.optimism.io/interop/overview.html) of the OP Stack.
- [`registry`](./crates/protocol/registry): Rust bindings for the [superchain-registry][superchain-registry].
- [`comp`](./crates/batcher/comp): Compression types for the OP Stack.
- [`hardforks`](./crates/protocol/hardforks): Consensus layer hardfork types for the OP Stack including network upgrade transactions.

**Proof**

- [`mpt`](./crates/proof/mpt): Utilities for interacting with the Merkle Patricia Trie in the client program.
- [`executor`](./crates/proof/executor): `no_std` stateless block executor for the [OP Stack][op-stack].
- [`proof`](./crates/proof/proof): High level OP Stack state transition proof SDK.
- [`proof-interop`](./crates/proof/proof-interop): Extension of `kona-proof` with interop support.
- [`preimage`](./crates/proof/preimage): High level interfaces to the [`PreimageOracle`][fpp-specs] ABI.
- [`std-fpvm`](./crates/proof/std-fpvm): Platform specific [Fault Proof VM][g-fault-proof-vm] kernel APIs.
- [`std-fpvm-proc`](./crates/proof/std-fpvm-proc): Proc macro for [Fault Proof Program][fpp-specs] entrypoints.

**Node**

- [`service`](./crates/node/service): The OP Stack rollup node service.
- [`engine`](./crates/node/engine): An extensible implementation of the [OP Stack][op-stack] rollup node engine client
- [`rpc`](./crates/node/rpc): OP Stack RPC types and extensions.
- [`gossip`](./crates/node/gossip): OP Stack P2P Networking - Gossip.
- [`disc`](./crates/node/disc): OP Stack P2P Networking - Discovery.
- [`peers`](./crates/node/peers): Networking Utilities ported from reth.
- [`sources`](./crates/node/sources): Data source types and utilities for the kona-node.

**Providers**

- [`providers-alloy`](./crates/providers/providers-alloy): Provider implementations for `kona-derive` backed by [Alloy][alloy].

**Utilities**

- [`serde`](./crates/utilities/serde): Serialization helpers.
- [`cli`](./crates/utilities/cli): Standard CLI utilities, used across `kona`'s binaries.
- [`macros`](./crates/utilities/macros): Utility macros.

### Proof

Built on top of these libraries, this repository also features a [proof program][fpp-specs]
designed to deterministically execute the rollup state transition in order to verify an
[L2 output root][g-output-root] from the L1 inputs it was [derived from][g-derivation-pipeline].

Kona's libraries were built with alternative backend support and extensibility in mind - the repository features
a fault proof virtual machine backend for use in the governance-approved OP Stack, but it's portable across
provers! Kona is also used by:

- [`op-succinct`][op-succinct]
- [`kailua`][kailua]

To build your own backend for kona, or build a new application on top of its libraries,
see the [SDK section of the docs](https://rollup.yoga/node/design/intro).

## MSRV

The current MSRV (minimum supported rust version) is `1.88`.

The MSRV is not increased automatically, and will be updated
only as part of a patch (pre-1.0) or minor (post-1.0) release.


## Crate Releases

`kona` releases are done using the [`cargo-release`](https://crates.io/crates/cargo-release) crate.
A detailed guide is available in [./RELEASES.md](./RELEASES.md).


## Contributing

`kona` is built by open source contributors like you, thank you for improving the project!

A [contributing guide][contributing] is available that sets guidelines for contributing.

Pull requests will not be merged unless CI passes, so please ensure that your contribution
follows the linting rules and passes clippy.


## Credits

`kona` is inspired by the work of several teams, namely [OP Labs][op-labs] and other contributors' work on the
[Optimism monorepo][op-go-monorepo] and [BadBoiLabs][bad-boi-labs]'s work on [Cannon-rs][badboi-cannon-rs].

`kona` is also built on rust types in [alloy][alloy], [op-alloy][op-alloy], and [maili][maili].

## License

Licensed under the [MIT license.](https://github.com/op-rs/kona/blob/main/LICENSE.md)

> [!NOTE]
>
> Contributions intentionally submitted for inclusion in these crates by you
> shall be licensed as above, without any additional terms or conditions.


<!-- Links -->

[alloy]: https://github.com/alloy-rs/alloy
[maili]: https://github.com/op-rs/maili
[op-alloy]: https://github.com/alloy-rs/op-alloy
[contributing]: https://rollup.yoga/intro/contributing
[op-stack]: https://github.com/ethereum-optimism/optimism
[superchain-registry]: https://github.com/ethereum-optimism/superchain-registry
[op-go-monorepo]: https://github.com/ethereum-optimism/optimism/tree/develop
[cannon]: https://github.com/ethereum-optimism/optimism/tree/develop/cannon
[cannon-rs]: https://github.com/op-rs/cannon-rs
[rollup-node-spec]: https://specs.optimism.io/protocol/rollup-node.html
[supervisor-spec]: https://specs.optimism.io/interop/supervisor.html
[badboi-cannon-rs]: https://github.com/BadBoiLabs/cannon-rs
[asterisc]: https://github.com/ethereum-optimism/asterisc
[fpp-specs]: https://specs.optimism.io/fault-proof/index.html
[site]: https://rollup.yoga
[op-succinct]: https://github.com/succinctlabs/op-succinct
[kailua]: https://github.com/risc0/kailua
[op-labs]: https://github.com/ethereum-optimism
[bad-boi-labs]: https://github.com/BadBoiLabs
[g-output-root]: https://specs.optimism.io/glossary.html#l2-output-root
[g-derivation-pipeline]: https://specs.optimism.io/protocol/derivation.html#l2-chain-derivation-pipeline
[g-fault-proof-vm]: https://specs.optimism.io/experimental/fault-proof/index.html#fault-proof-vm
[g-preimage-oracle]: https://specs.optimism.io/fault-proof/index.html#pre-image-oracle
