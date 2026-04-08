# op-revm

Optimism variant of [revm](https://github.com/bluealloy/revm) — the OP Stack's
modifications to the Ethereum Virtual Machine, packaged as a custom EVM built
on top of the upstream `revm` framework.

`op-revm` adds:

- Deposit transactions and the deposit transaction type
- L1 cost / blob fee accounting (`L1BlockInfo`)
- Operator fee handling (Isthmus / Jovian)
- OP-specific precompiles (BN254 pairing acceleration, etc.)
- OP-specific halt reasons and transaction errors
- Hardfork-aware spec selection (`OpSpecId`)

## Provenance

This crate is a vendored copy of upstream
[`bluealloy/revm`'s `crates/op-revm`](https://github.com/bluealloy/revm/tree/main/crates/op-revm)
imported into the monorepo so it can evolve in lock-step with the rest of the
OP Stack Rust code. See [`CHANGELOG.md`](./CHANGELOG.md) for the upstream
release history.

## Features

- `default = ["std", "c-kzg", "secp256k1", "portable", "blst"]`
- `std` — enables `std`-dependent features in `revm`, `alloy-primitives`,
  `serde_json`, etc.
- `serde` — derives `serde` impls and forwards to `revm/serde` and
  `alloy-primitives/serde`
- `portable`, `c-kzg`, `secp256k1`, `blst`, `bn` — pass-through feature gates
  to the corresponding `revm` cryptographic backends
- `dev`, `memory_limit`, `optional_balance_check`, `optional_block_gas_limit`,
  `optional_eip3541`, `optional_eip3607`, `optional_no_base_fee`,
  `optional_fee_charge` — debugging / testing knobs forwarded to `revm`

## Building & Testing

From `rust/`:

```bash
# Build
cargo build -p op-revm

# Tests
cargo nextest run -p op-revm --all-features

# no_std check (mirrors upstream's riscv32imac CI)
cargo build -p op-revm --target riscv32imac-unknown-none-elf --no-default-features
```

The no_std build is also exercised by `just check-no-std` and runs in CI on
every PR that touches `rust/**`.

## License

MIT — see [`LICENSE`](./LICENSE).
