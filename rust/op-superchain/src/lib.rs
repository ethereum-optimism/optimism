//! Type-free producer of compile-time superchain-registry artifacts.
//!
//! `op-superchain` is the single source of truth for which chains are embedded
//! in the OP Stack's Rust binaries. Both `kona-registry` and `reth-optimism-chainspec`
//! depend on this crate; each interprets the same bytes its own way.
//!
//! The runtime API ships:
//! - [`CHAIN_LIST_JSON`] — verbatim `chainList.json` from the superchain-registry.
//! - [`DEPSETS_JSON`] — interop dependency clusters, aggregated from each chain's `[interop]`
//!   block. Kona's `DependencySet` schema.
//! - [`SUPERCHAINS_JSON`] — aggregate `Superchains` JSON used by kona-registry's eager-parse path.
//! - [`config_str`] / [`genesis_bytes`] — per-chain `(name, environment)` lookups used by
//!   reth-optimism-chainspec.
//! - [`supported_chains`] — the canonical `[(name, environment)]` table.
//!
//! All payload bytes live under `gen/` in this crate's source tree (committed,
//! crates.io fallback). When the `packages/contracts-bedrock/lib/superchain-registry`
//! submodule is present, `build.rs` regenerates `gen/` from the submodule on
//! every build; CI's drift gate catches stale snapshots.

#![no_std]
#![doc(issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/")]

/// Verbatim `chainList.json` from the superchain-registry submodule.
pub const CHAIN_LIST_JSON: &str = include_str!("../gen/chainList.json");

/// Interop dependency clusters aggregated from each chain's `[interop]` block.
/// JSON shape matches kona's `Vec<DependencySet>`.
pub const DEPSETS_JSON: &str = include_str!("../gen/depsets.json");

/// Aggregate `Superchains` JSON. Each per-superchain group contains its
/// `SuperchainConfig` (from `superchain.toml`) and the array of `ChainConfig`s.
/// Field names match the raw TOML schema (`snake_case`); consumers using kona's
/// `ChainConfig` deserialize via `serde(alias = ...)` annotations.
pub const SUPERCHAINS_JSON: &str = include_str!("../gen/superchains.json");

include!("../gen/index.rs");
