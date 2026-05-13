//! Helper to produce an [`L1Input`] from raw L1 block data.
//!
//! Filter-only: addresses are checked by static rollup-config fields and
//! topics are checked by the well-known event-hash constants. Nothing in
//! this module parses a fallible structure — no `decode_deposit`, no
//! `process_config_update_log`, no `parse_frames`. The deriver owns those
//! decodes and emits trace events on failure (see [`crate::TraceEntry`]).
//!
//! Why the strict split:
//!
//! 1. **Single-sourced trace.** Every dropped item shows up in exactly one place — the deriver's
//!    [`crate::DeriveTrace`] — so tests assert against one source, not "extraction trace + deriver
//!    trace".
//! 2. **Sysconfig-blind helper.** Filtering by `tx.to == batch_inbox_addr` is static. Filtering by
//!    `tx.from == batcher_addr` is dynamic and lives in the deriver, which already owns the rolling
//!    system config. Pulling that dynamic filter out into the helper would require duplicating the
//!    rolling sysconfig and re-introducing drift risk.
//! 3. **Type-level enforcement.** [`L1Input`] carries `Vec<Log>` and `Vec<(Address, Bytes)>`, not
//!    `Vec<DepositTx>` or `Vec<ParsedConfigUpdate>`. The compiler enforces "no fallible parsing in
//!    extract" — the function signature couldn't accommodate it.

use crate::pure::L1Input;
use alloc::vec::Vec;
use alloy_consensus::{Eip658Value, Header, Receipt};
use alloy_primitives::{Address, Bytes};
use kona_genesis::{CONFIG_UPDATE_TOPIC, RollupConfig};
use kona_protocol::DEPOSIT_EVENT_ABI_HASH;

/// One unpacked L1 transaction view used by [`extract_l1_input`].
///
/// `to` is `None` for contract-creation txs; `input` is the calldata or, in
/// the EIP-4844 case, the caller-decoded blob bytes (KZG verified, version
/// byte stripped). `from` is the EIP-155 recovered sender.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct L1TxView {
    /// The transaction's sender.
    pub from: Address,
    /// The transaction's `to` field. `None` for contract-creation txs.
    pub to: Option<Address>,
    /// The transaction's calldata or decoded blob payload.
    pub input: Bytes,
}

/// Extract derivation inputs from a fully-fetched L1 block.
///
/// `txs` iterates the L1 block's transactions in block order, with each one
/// flattened to `(from, to, input_or_decoded_blob)` via [`L1TxView`]. The
/// caller is expected to have already KZG-verified and version-byte-stripped
/// any blob bodies (so calldata-or-decoded-blob bytes arrive here ready to
/// feed into the frame queue). `receipts` must be the L1 receipts for the
/// same block, in block order; status-failed receipts and their logs are
/// skipped.
///
/// The result's [`L1Input::deposit_logs`] / [`L1Input::config_logs`] are
/// raw `Log`s — the deriver decodes them later and emits trace events for
/// individual failures.
///
/// # Panics
///
/// Never. This function is total over any input shape: invalid logs end up
/// in the deriver's trace, not as a panic here.
pub fn extract_l1_input(
    header: Header,
    txs: impl IntoIterator<Item = L1TxView>,
    receipts: &[Receipt],
    rollup_cfg: &RollupConfig,
) -> L1Input {
    let mut batch_inbox_data: Vec<(Address, Bytes)> = Vec::new();
    for L1TxView { from, to, input } in txs {
        let Some(to) = to else { continue };
        if to == rollup_cfg.batch_inbox_address {
            batch_inbox_data.push((from, input));
        }
    }

    let mut deposit_logs = Vec::new();
    let mut config_logs = Vec::new();
    for receipt in receipts {
        if receipt.status == Eip658Value::Eip658(false) {
            continue;
        }
        for log in &receipt.logs {
            if log.address == rollup_cfg.deposit_contract_address &&
                log.topics().first() == Some(&DEPOSIT_EVENT_ABI_HASH)
            {
                deposit_logs.push(log.clone());
                continue;
            }
            if log.address == rollup_cfg.l1_system_config_address &&
                log.topics().first() == Some(&CONFIG_UPDATE_TOPIC)
            {
                config_logs.push(log.clone());
            }
        }
    }

    L1Input { header, batch_inbox_data, deposit_logs, config_logs }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_consensus::{Header, Receipt};
    use alloy_primitives::{B256, Bytes, Log, LogData, address};
    use kona_genesis::{CONFIG_UPDATE_TOPIC, RollupConfig};
    use kona_protocol::DEPOSIT_EVENT_ABI_HASH;

    fn rollup_cfg() -> RollupConfig {
        RollupConfig {
            batch_inbox_address: address!("ff00000000000000000000000000000000042069"),
            deposit_contract_address: address!("0808080808080808080808080808080808080808"),
            l1_system_config_address: address!("0909090909090909090909090909090909090909"),
            ..Default::default()
        }
    }

    #[test]
    fn batch_inbox_filter_keeps_matching_to_only() {
        let cfg = rollup_cfg();
        let from1 = address!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
        let from2 = address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
        let txs = vec![
            L1TxView {
                from: from1,
                to: Some(cfg.batch_inbox_address),
                input: Bytes::from_static(b"hello"),
            },
            L1TxView {
                from: from2,
                to: Some(address!("1111111111111111111111111111111111111111")),
                input: Bytes::from_static(b"ignore"),
            },
            L1TxView { from: from2, to: None, input: Bytes::from_static(b"create") },
        ];
        let input = extract_l1_input(Header::default(), txs, &[], &cfg);
        assert_eq!(input.batch_inbox_data.len(), 1);
        assert_eq!(input.batch_inbox_data[0].0, from1);
        assert_eq!(input.batch_inbox_data[0].1.as_ref(), b"hello");
    }

    #[test]
    fn deposit_logs_filter_topic_and_addr() {
        let cfg = rollup_cfg();
        let good = Log {
            address: cfg.deposit_contract_address,
            data: LogData::new_unchecked(vec![DEPOSIT_EVENT_ABI_HASH], Bytes::from_static(b"x")),
        };
        let wrong_addr = Log {
            address: address!("1111111111111111111111111111111111111111"),
            data: LogData::new_unchecked(vec![DEPOSIT_EVENT_ABI_HASH], Bytes::default()),
        };
        let wrong_topic = Log {
            address: cfg.deposit_contract_address,
            data: LogData::new_unchecked(vec![B256::ZERO], Bytes::default()),
        };
        let receipts = vec![Receipt {
            status: Eip658Value::Eip658(true),
            logs: vec![good.clone(), wrong_addr, wrong_topic],
            ..Default::default()
        }];
        let input = extract_l1_input(Header::default(), Vec::<L1TxView>::new(), &receipts, &cfg);
        assert_eq!(input.deposit_logs, vec![good]);
    }

    #[test]
    fn config_logs_filter_topic_and_addr() {
        let cfg = rollup_cfg();
        let good = Log {
            address: cfg.l1_system_config_address,
            data: LogData::new_unchecked(vec![CONFIG_UPDATE_TOPIC], Bytes::from_static(b"x")),
        };
        let wrong_addr = Log {
            address: address!("1111111111111111111111111111111111111111"),
            data: LogData::new_unchecked(vec![CONFIG_UPDATE_TOPIC], Bytes::default()),
        };
        let wrong_topic = Log {
            address: cfg.l1_system_config_address,
            data: LogData::new_unchecked(vec![B256::ZERO], Bytes::default()),
        };
        let receipts = vec![Receipt {
            status: Eip658Value::Eip658(true),
            logs: vec![good.clone(), wrong_addr, wrong_topic],
            ..Default::default()
        }];
        let input = extract_l1_input(Header::default(), Vec::<L1TxView>::new(), &receipts, &cfg);
        assert_eq!(input.config_logs, vec![good]);
    }

    #[test]
    fn failed_receipts_logs_ignored() {
        let cfg = rollup_cfg();
        let log = Log {
            address: cfg.deposit_contract_address,
            data: LogData::new_unchecked(vec![DEPOSIT_EVENT_ABI_HASH], Bytes::default()),
        };
        let receipts = vec![Receipt {
            status: Eip658Value::Eip658(false),
            logs: vec![log],
            ..Default::default()
        }];
        let input = extract_l1_input(Header::default(), Vec::<L1TxView>::new(), &receipts, &cfg);
        assert!(input.deposit_logs.is_empty());
    }
}
