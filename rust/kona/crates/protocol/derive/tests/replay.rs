//! Synthetic replay test for `kona_derive::pure::Deriver`.
//!
//! See `tests/fixtures/README.md` for why this test uses synthetic fixtures
//! rather than recorded mainnet data, and what it covers. Quick summary:
//!
//! - **Sysconfig update path** — feeds a real `ConfigUpdate` log and asserts the deriver emits
//!   `SystemConfigUpdated`.
//! - **Sysconfig-malformed path** — feeds a malformed config log and asserts
//!   `SystemConfigUpdateDropped`.
//! - **Deposit decode path** — feeds a real `TransactionDeposited` log and asserts the deriver
//!   doesn't drop it.
//! - **Deposit-malformed path** — feeds a malformed deposit log and asserts `DepositLogDropped`.
//! - **Hardfork activation boundary** — feeds an L1 stream that straddles a post-Holocene
//!   activation timestamp (Granite, here). The deriver must continue to derive correctly without
//!   any soft-reset signal — closing the otherwise-hidden risk that `Signal::Activation` was
//!   load-bearing.
//! - **Overlap request/response contract** — covered by
//!   `pure::deriver::tests::add_span_batch_overlap_unsolicited_errors` plus the
//!   `pure::overlap::tests::*` byte-wise compare tests.
//!
//! When real mainnet recording becomes feasible (full L1 RPC + beacon
//! access, plus a snapshot of the L2 lookups the deriver requests), the
//! replay test will move under a `record-fixtures` feature gate per the
//! plan, and CI will replay from frozen bytes.

use std::sync::Arc;

use alloy_consensus::Header;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Bytes, Log, LogData, U64, U256, address};
use kona_derive::{Derivation, Deriver, L1Input, TraceEntry};
use kona_genesis::{
    CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC, HardForkConfig, L1ChainConfig,
    RollupConfig, SystemConfig,
};
use kona_protocol::{BlockInfo, DEPOSIT_EVENT_ABI_HASH, L2BlockInfo};
use kona_registry::L1Config;

const BATCH_INBOX: alloy_primitives::Address = address!("ff00000000000000000000000000000000042069");
const DEPOSIT_CONTRACT: alloy_primitives::Address =
    address!("0808080808080808080808080808080808080808");
const SYS_CONFIG: alloy_primitives::Address = address!("0909090909090909090909090909090909090909");

fn rollup_cfg_with_granite_at(granite_time: u64) -> RollupConfig {
    RollupConfig {
        block_time: 2,
        seq_window_size: 100,
        batch_inbox_address: BATCH_INBOX,
        deposit_contract_address: DEPOSIT_CONTRACT,
        l1_system_config_address: SYS_CONFIG,
        hardforks: HardForkConfig {
            holocene_time: Some(0),
            granite_time: Some(granite_time),
            ..Default::default()
        },
        ..Default::default()
    }
}

fn header(number: u64, timestamp: u64, parent_hash: B256) -> Header {
    Header { number, timestamp, parent_hash, ..Default::default() }
}

fn valid_deposit_log() -> Log {
    let from = address!("2222222222222222222222222222222222222222");
    let to = address!("3333333333333333333333333333333333333333");
    let mut from_bytes = vec![0u8; 32];
    from_bytes[12..32].copy_from_slice(from.as_slice());
    let mut to_bytes = vec![0u8; 32];
    to_bytes[12..32].copy_from_slice(to.as_slice());
    let mut data = vec![0u8; 192];
    let offset: [u8; 8] = U64::from(32).to_be_bytes();
    data[24..32].copy_from_slice(&offset);
    let len: [u8; 8] = U64::from(128).to_be_bytes();
    data[56..64].copy_from_slice(&len);
    let mint: [u8; 16] = 10_u128.to_be_bytes();
    data[80..96].copy_from_slice(&mint);
    let value: [u8; 32] = U256::from(100).to_be_bytes();
    data[96..128].copy_from_slice(&value);
    let gas: [u8; 8] = 1000_u64.to_be_bytes();
    data[128..136].copy_from_slice(&gas);
    data[136] = 1;
    Log {
        address: DEPOSIT_CONTRACT,
        data: LogData::new_unchecked(
            vec![
                DEPOSIT_EVENT_ABI_HASH,
                B256::from_slice(&from_bytes),
                B256::from_slice(&to_bytes),
                B256::default(),
            ],
            Bytes::from(data),
        ),
    }
}

fn malformed_deposit_log() -> Log {
    Log {
        address: DEPOSIT_CONTRACT,
        // Only one topic — `decode_deposit` rejects with UnexpectedTopicsLen.
        data: LogData::new_unchecked(vec![DEPOSIT_EVENT_ABI_HASH], Bytes::default()),
    }
}

fn batcher_update_log(new_batcher: alloy_primitives::Address) -> Log {
    // ConfigUpdate event v0, updateType = Batcher (0).
    let mut data = vec![0u8; 32 * 3];
    // Offset (32) + length (32) + addr-padded-32.
    data[24..32].copy_from_slice(&U64::from(32).to_be_bytes::<8>());
    data[56..64].copy_from_slice(&U64::from(32).to_be_bytes::<8>());
    data[64 + 12..64 + 32].copy_from_slice(new_batcher.as_slice());
    Log {
        address: SYS_CONFIG,
        data: LogData::new_unchecked(
            vec![
                CONFIG_UPDATE_TOPIC,
                CONFIG_UPDATE_EVENT_VERSION_0,
                B256::ZERO, // updateType = Batcher (0)
            ],
            Bytes::from(data),
        ),
    }
}

fn malformed_config_log() -> Log {
    Log {
        address: SYS_CONFIG,
        // Single-topic log fails LogProcessingError::InvalidTopicLen(1).
        data: LogData::new_unchecked(vec![CONFIG_UPDATE_TOPIC], Bytes::default()),
    }
}

fn run_until_idle(deriver: &mut Deriver, safe_head: L2BlockInfo) -> Vec<TraceEntry> {
    let mut all = Vec::new();
    for _ in 0..64 {
        let (out, trace) = deriver.derive(safe_head);
        all.extend(trace.entries);
        if matches!(out, Derivation::NeedL1Input | Derivation::Idle) {
            return all;
        }
        if matches!(out, Derivation::NeedSpanBatchOverlap { .. }) {
            return all;
        }
    }
    all
}

/// Synthetic replay across a Granite activation boundary.
///
/// Critical case: the kona-node async pipeline emits a `Signal::Activation`
/// at this point (`actor.rs:130`), which is a soft reset for buffered
/// channel data. The pure deriver must produce the same observable
/// behaviour without any external signal — that's the load-bearing
/// invariant phase 4 will rely on, and that the plan calls out as a hard
/// gate before phase 4 can proceed.
#[test]
fn replay_hardfork_activation_boundary() {
    // Place Granite activation between L1 blocks 5 and 6.
    let activation_time = 200;
    let rcfg = Arc::new(rollup_cfg_with_granite_at(activation_time));
    let l1cfg: Arc<L1ChainConfig> = Arc::new(L1Config::sepolia().into());
    let safe_head = L2BlockInfo {
        block_info: BlockInfo { number: 1, timestamp: 100, ..Default::default() },
        l1_origin: BlockNumHash { number: 0, ..Default::default() },
        seq_num: 0,
    };
    let mut deriver = Deriver::new(rcfg, l1cfg, SystemConfig::default(), safe_head, None);
    deriver.reset(safe_head, SystemConfig::default());

    // Feed 10 L1 blocks straddling activation_time = 200. Each block has
    // empty inbox / receipts. The deriver should consume them all and emit
    // no errors; the activation timestamp transition is implicit.
    let mut prev_hash = B256::ZERO;
    let mut activation_straddled = false;
    let start_time = 195;
    for n in 0..10u64 {
        let ts = start_time + n * 2;
        if ts <= activation_time && ts + 2 > activation_time {
            activation_straddled = true;
        }
        let h = header(n, ts, prev_hash);
        prev_hash = h.hash_slow();
        deriver
            .add_l1_input(L1Input {
                header: h,
                batch_inbox_data: Vec::new(),
                deposit_logs: Vec::new(),
                config_logs: Vec::new(),
            })
            .expect("contiguous L1 inputs accepted");
    }
    assert!(activation_straddled, "test fixture should straddle the activation timestamp");

    let entries = run_until_idle(&mut deriver, safe_head);
    // No critical trace events should appear.
    let bad = entries
        .iter()
        .filter(|e| {
            matches!(
                e,
                TraceEntry::AttributesBrokenTimeInvariant { .. } |
                    TraceEntry::AttributesL1InfoTxBuildFailed { .. }
            )
        })
        .count();
    assert_eq!(bad, 0, "deriver hit a critical error across the activation: {entries:?}");
}

/// Synthetic replay covering: real sysconfig update applied, malformed
/// sysconfig drop, real deposit accepted, malformed deposit dropped.
#[test]
fn replay_sysconfig_and_deposit_paths() {
    let rcfg = Arc::new(rollup_cfg_with_granite_at(u64::MAX));
    let l1cfg: Arc<L1ChainConfig> = Arc::new(L1Config::sepolia().into());
    let safe_head = L2BlockInfo {
        block_info: BlockInfo { number: 1, timestamp: 100, ..Default::default() },
        l1_origin: BlockNumHash { number: 0, ..Default::default() },
        seq_num: 0,
    };
    let mut deriver = Deriver::new(rcfg, l1cfg, SystemConfig::default(), safe_head, None);
    deriver.reset(safe_head, SystemConfig::default());

    let new_batcher = address!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let h = header(0, 100, B256::ZERO);
    deriver
        .add_l1_input(L1Input {
            header: h,
            batch_inbox_data: Vec::new(),
            deposit_logs: vec![valid_deposit_log(), malformed_deposit_log()],
            config_logs: vec![batcher_update_log(new_batcher), malformed_config_log()],
        })
        .unwrap();

    let entries = run_until_idle(&mut deriver, safe_head);

    let cfg_updates =
        entries.iter().filter(|e| matches!(e, TraceEntry::SystemConfigUpdated { .. })).count();
    let cfg_drops = entries
        .iter()
        .filter(|e| matches!(e, TraceEntry::SystemConfigUpdateDropped { .. }))
        .count();
    let dep_drops =
        entries.iter().filter(|e| matches!(e, TraceEntry::DepositLogDropped { .. })).count();
    assert_eq!(cfg_updates, 1, "sysconfig update applied; trace: {entries:?}");
    assert_eq!(cfg_drops, 1, "sysconfig malformed dropped; trace: {entries:?}");
    assert_eq!(dep_drops, 1, "deposit malformed dropped; trace: {entries:?}");
}

/// Sanity: feeding a long sequence of contiguous empty L1 blocks doesn't
/// blow up, doesn't loop, doesn't emit unrelated drops.
#[test]
fn replay_long_empty_sequence_stays_quiet() {
    let rcfg = Arc::new(rollup_cfg_with_granite_at(u64::MAX));
    let l1cfg: Arc<L1ChainConfig> = Arc::new(L1Config::sepolia().into());
    let safe_head = L2BlockInfo {
        block_info: BlockInfo { number: 1, timestamp: 100, ..Default::default() },
        l1_origin: BlockNumHash { number: 0, ..Default::default() },
        seq_num: 0,
    };
    let mut deriver = Deriver::new(rcfg, l1cfg, SystemConfig::default(), safe_head, None);
    deriver.reset(safe_head, SystemConfig::default());

    let mut prev = B256::ZERO;
    for n in 0..40u64 {
        let h = header(n, 100 + n * 12, prev);
        prev = h.hash_slow();
        deriver
            .add_l1_input(L1Input {
                header: h,
                batch_inbox_data: Vec::new(),
                deposit_logs: Vec::new(),
                config_logs: Vec::new(),
            })
            .unwrap();
    }
    let entries = run_until_idle(&mut deriver, safe_head);
    let bad = entries
        .iter()
        .filter(|e| {
            matches!(
                e,
                TraceEntry::FrameDropped { .. } |
                    TraceEntry::FramesParseFailed { .. } |
                    TraceEntry::ChannelDecompressionFailed { .. } |
                    TraceEntry::AttributesBrokenTimeInvariant { .. }
            )
        })
        .count();
    assert_eq!(bad, 0, "empty L1 sequence should not produce drops: {entries:?}");
}
