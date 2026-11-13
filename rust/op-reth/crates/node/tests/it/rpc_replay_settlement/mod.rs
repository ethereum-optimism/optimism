//! RPC replay over a sequenced Lagoon block whose transactions carry SDM refunds.
//!
//! The settlement of an SDM refund (canonical gas, sender credit, fee-vault debits) only happens
//! inside the block executor, so every RPC path that replays block transactions must go through it.
//! A replay on the bare EVM reports pre-refund gas and unsettled balances ([#485]). Each file
//! covers one RPC family against the shared [`fixture::SettledBlock`]. Checks that do not compare
//! gas or balances (revert data, hash coverage, error shapes) are smoke tests of the executor path
//! rather than regression guards; they pass on either pin.
//!
//! [#485]: https://github.com/ethereum-optimism/protocol-team/issues/485

mod affordability;
mod debug_tracers;
mod fixture;
mod otterscan;
mod parity_replays;
mod post_exec_target;
mod prefix_replays;
mod trace_chain;
