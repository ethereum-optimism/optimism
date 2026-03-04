//! Flashblock emitter for native flashblock production.
//!
//! Emits flashblock snapshots on a timer during payload building.
//! Created per payload build. The [`tokio::sync::mpsc::Sender`] is cloned
//! from a long-lived channel that feeds `FlashBlockService` for the node's
//! entire lifetime.

use alloc::{collections::BTreeMap, vec, vec::Vec};
use alloy_primitives::{B256, Bloom, Bytes};
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::{
    OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
    OpFlashblockPayloadMetadata,
};
use std::time::{Duration, Instant};
use tokio::sync::mpsc::Sender;
use tracing::trace;

/// Emits flashblock snapshots on a timer during payload building.
///
/// One emitter is created per payload build. It accumulates executed
/// transactions and periodically flushes them as flashblock deltas
/// via [`Sender::blocking_send`], which is safe from `spawn_blocking`
/// contexts.
#[derive(Debug)]
pub struct FlashblockEmitter {
    /// Channel sender — cloned from the long-lived node-lifetime channel.
    tx: Sender<OpFlashblockPayload>,
    /// Minimum interval between flashblock emissions.
    interval: Duration,
    /// When the last flashblock was emitted.
    last_emit: Instant,
    /// Sequential flashblock index within this payload build.
    index: u64,
    /// The payload ID for this build (tags each flashblock).
    payload_id: PayloadId,
    /// Immutable base payload, sent with index 0 only.
    base: Option<OpFlashblockPayloadBase>,
    /// Raw-encoded transactions accumulated since last emission.
    pending_txs: Vec<Bytes>,
    /// Cumulative gas used at the time of the last emission.
    cumulative_gas_used_at_last_emit: u64,
}

impl FlashblockEmitter {
    /// Creates a new emitter for a single payload build.
    pub fn new(
        tx: Sender<OpFlashblockPayload>,
        interval: Duration,
        payload_id: PayloadId,
    ) -> Self {
        Self {
            tx,
            interval,
            last_emit: Instant::now(),
            index: 0,
            payload_id,
            base: None,
            pending_txs: Vec::new(),
            cumulative_gas_used_at_last_emit: 0,
        }
    }

    /// Sets the immutable base payload (block env fields).
    /// Called once at build start, before any emissions.
    pub fn set_base(&mut self, base: OpFlashblockPayloadBase) {
        self.base = Some(base);
    }

    /// Records a successfully executed transaction for the next flashblock delta.
    pub fn add_tx(&mut self, raw_tx: Bytes) {
        self.pending_txs.push(raw_tx);
    }

    /// Returns `true` if enough time has elapsed to emit a flashblock.
    pub fn should_emit(&self) -> bool {
        self.last_emit.elapsed() >= self.interval
    }

    /// Emits a flashblock snapshot with all transactions accumulated since the last emission.
    ///
    /// Skips emission (except for index 0) if no new transactions have been added.
    /// Uses [`Sender::blocking_send`] which is designed for `spawn_blocking` contexts.
    /// If the channel is full, this blocks briefly (backpressure).
    /// If the receiver is dropped, the error is ignored (node shutdown).
    pub fn emit_snapshot(&mut self, cumulative_gas_used: u64, block_number: u64) {
        let txs = core::mem::take(&mut self.pending_txs);
        if txs.is_empty() && self.index > 0 {
            return;
        }

        let diff = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: cumulative_gas_used,
            block_hash: B256::ZERO,
            transactions: txs,
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        };

        let metadata = OpFlashblockPayloadMetadata {
            block_number,
            new_account_balances: BTreeMap::new(),
            receipts: BTreeMap::new(),
        };

        let fb = OpFlashblockPayload {
            payload_id: self.payload_id,
            index: self.index,
            base: if self.index == 0 { self.base.take() } else { None },
            diff,
            metadata,
        };

        trace!(
            target: "payload_builder",
            payload_id = %self.payload_id,
            index = self.index,
            gas_used = cumulative_gas_used,
            "emitting flashblock"
        );

        // blocking_send from spawn_blocking context.
        // If channel is full, blocks briefly — acceptable backpressure.
        // If receiver dropped, returns Err — service shutdown means
        // flashblocks are no longer needed.
        let _ = self.tx.blocking_send(fb);

        self.index += 1;
        self.last_emit = Instant::now();
        self.cumulative_gas_used_at_last_emit = cumulative_gas_used;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::U256;
    use tokio::sync::mpsc;

    #[test]
    fn test_emitter_timing() {
        let (tx, mut rx) = mpsc::channel(16);
        let payload_id = PayloadId::new([1u8; 8]);
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(50), payload_id);

        emitter.set_base(OpFlashblockPayloadBase {
            parent_hash: B256::ZERO,
            fee_recipient: Default::default(),
            block_number: 100,
            timestamp: 1234567890,
            prev_randao: B256::ZERO,
            gas_limit: 30_000_000,
            extra_data: Bytes::default(),
            base_fee_per_gas: U256::from(1_000_000_000u64),
            parent_beacon_block_root: B256::ZERO,
        });

        // Index 0 should emit even with no txs (base flashblock)
        emitter.emit_snapshot(0, 100);
        let fb = rx.try_recv().expect("should receive index 0");
        assert_eq!(fb.index, 0);
        assert_eq!(fb.payload_id, payload_id);
        assert!(fb.base.is_some());

        // Index 1 with no txs should NOT emit
        emitter.emit_snapshot(0, 100);
        assert!(rx.try_recv().is_err());

        // Index 1 with txs should emit
        emitter.add_tx(Bytes::from(vec![1, 2, 3]));
        emitter.emit_snapshot(21000, 100);
        let fb = rx.try_recv().expect("should receive index 1");
        assert_eq!(fb.index, 1);
        assert!(fb.base.is_none());
        assert_eq!(fb.diff.transactions.len(), 1);
        assert_eq!(fb.diff.gas_used, 21000);
    }

    #[test]
    fn test_emitter_should_emit() {
        let (tx, _rx) = mpsc::channel(16);
        let emitter = FlashblockEmitter::new(tx, Duration::from_millis(50), PayloadId::new([0; 8]));
        // Just created, should not emit yet (timer just started)
        // Note: this could be flaky if test runs very slowly
        assert!(!emitter.should_emit());

        // After waiting, should emit
        std::thread::sleep(Duration::from_millis(60));
        assert!(emitter.should_emit());
    }

    #[test]
    fn test_concurrent_emitters() {
        let (tx, mut rx) = mpsc::channel(64);
        let id_a = PayloadId::new([1u8; 8]);
        let id_b = PayloadId::new([2u8; 8]);

        let mut emitter_a = FlashblockEmitter::new(tx.clone(), Duration::from_millis(200), id_a);
        let mut emitter_b = FlashblockEmitter::new(tx, Duration::from_millis(200), id_b);

        emitter_a.set_base(OpFlashblockPayloadBase::default());
        emitter_b.set_base(OpFlashblockPayloadBase::default());

        emitter_a.add_tx(Bytes::from(vec![1]));
        emitter_a.emit_snapshot(21000, 100);

        emitter_b.add_tx(Bytes::from(vec![2]));
        emitter_b.emit_snapshot(42000, 100);

        let fb_a = rx.try_recv().expect("should receive from emitter A");
        assert_eq!(fb_a.payload_id, id_a);
        assert_eq!(fb_a.index, 0);

        let fb_b = rx.try_recv().expect("should receive from emitter B");
        assert_eq!(fb_b.payload_id, id_b);
        assert_eq!(fb_b.index, 0);
    }
}
