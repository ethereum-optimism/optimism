//! L2 chain builder that constructs valid OP Stack L2 blocks.

use alloy_consensus::Header;
use alloy_primitives::{Bloom, Sealable};
use op_alloy_consensus::{OpReceiptEnvelope, OpTxEnvelope};

use crate::{
    config::DeterministicConfig,
    l1::L1Block,
    state::{StateSnapshot, TestStateDb, compute_receipts_root, compute_transactions_root},
};

use super::{
    deposit::l1_info_deposit_tx,
    types::{L2Block, L2BlockRef},
};

/// Builds a deterministic L2 chain block by block.
#[allow(missing_debug_implementations)]
pub struct L2ChainBuilder {
    config: DeterministicConfig,
    state: TestStateDb,
    blocks: Vec<L2Block>,
    snapshots: Vec<StateSnapshot>,
    current_epoch: Option<EpochRef>,
    seq_num: u64,
}

/// Reference to the current L1 epoch (origin block).
#[derive(Debug, Clone)]
struct EpochRef {
    l1_block: L1Block,
}

impl L2ChainBuilder {
    /// Create a new L2 chain builder, initialized from genesis.
    pub fn new(config: &DeterministicConfig) -> Self {
        let mut state = TestStateDb::new();
        let allocs = config.l2_genesis_allocs();
        state.init_genesis(&allocs);

        let genesis_snapshot = state.snapshot();
        let genesis_state_root = genesis_snapshot.state_root;

        // Build genesis L2 block
        let genesis_header = Header {
            number: 0,
            timestamp: config.genesis_timestamp,
            state_root: genesis_state_root,
            transactions_root: crate::state::roots::EMPTY_ROOT_HASH,
            receipts_root: crate::state::roots::EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        let genesis_block = L2Block { header: sealed, transactions: vec![], receipts: vec![] };

        Self {
            config: config.clone(),
            state,
            blocks: vec![genesis_block],
            snapshots: vec![genesis_snapshot],
            current_epoch: None,
            seq_num: 0,
        }
    }

    /// Set the L1 epoch (origin block) for subsequent L2 blocks.
    pub fn set_epoch(&mut self, l1_block: &L1Block) {
        self.current_epoch = Some(EpochRef { l1_block: l1_block.clone() });
        self.seq_num = 0;
    }

    /// Build the next L2 block with only the L1 info deposit transaction (empty block).
    pub fn build_empty_block(&mut self) -> Result<L2BlockRef, Box<dyn std::error::Error>> {
        self.build_block(vec![])
    }

    /// Build the next L2 block with the given user transactions.
    pub fn build_block(
        &mut self,
        user_txs: Vec<OpTxEnvelope>,
    ) -> Result<L2BlockRef, Box<dyn std::error::Error>> {
        let epoch = self.current_epoch.as_ref().ok_or("must set epoch before building blocks")?;

        let prev = self.blocks.last().expect("always have genesis");
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l2_block_time;

        // Build L1 info deposit tx
        let deposit_tx = l1_info_deposit_tx(&self.config, &epoch.l1_block, self.seq_num)?;
        self.seq_num += 1;

        // Combine deposit + user txs
        let mut all_txs = vec![deposit_tx];
        all_txs.extend(user_txs);

        // Build receipts (deposit tx always succeeds with empty receipt)
        let receipts: Vec<OpReceiptEnvelope> =
            all_txs.iter().enumerate().map(|(i, tx)| build_receipt(tx, i as u64)).collect();

        let transactions_root = compute_transactions_root(&all_txs);
        let receipts_root = compute_receipts_root(&receipts);

        // Take a snapshot for state root
        let snapshot = self.state.snapshot();
        let state_root = snapshot.state_root;

        let header = Header {
            parent_hash: prev.header.hash(),
            number: block_num,
            timestamp,
            state_root,
            transactions_root,
            receipts_root,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        let block = L2Block { header: sealed, transactions: all_txs, receipts };

        self.blocks.push(block);
        self.snapshots.push(snapshot);

        Ok(L2BlockRef { index: self.blocks.len() - 1 })
    }

    /// Build N empty (deposit-only) L2 blocks.
    pub fn build_empty_blocks(
        &mut self,
        count: usize,
    ) -> Result<Vec<L2BlockRef>, Box<dyn std::error::Error>> {
        let mut refs = Vec::with_capacity(count);
        for _ in 0..count {
            refs.push(self.build_empty_block()?);
        }
        Ok(refs)
    }

    /// Get all built blocks.
    pub fn blocks(&self) -> &[L2Block] {
        &self.blocks
    }

    /// Get the latest block.
    pub fn head(&self) -> &L2Block {
        self.blocks.last().expect("always have genesis")
    }

    /// Get a block by index.
    pub fn block(&self, block_ref: L2BlockRef) -> &L2Block {
        &self.blocks[block_ref.index]
    }

    /// Get the state snapshot at a given block.
    pub fn snapshot_at(&self, block_ref: L2BlockRef) -> &StateSnapshot {
        &self.snapshots[block_ref.index]
    }

    /// Get the current state database.
    pub const fn state(&self) -> &TestStateDb {
        &self.state
    }

    /// Get the config.
    pub const fn config(&self) -> &DeterministicConfig {
        &self.config
    }

    /// Get the head snapshot.
    pub fn head_snapshot(&self) -> &StateSnapshot {
        self.snapshots.last().expect("always have genesis snapshot")
    }
}

/// Build a minimal receipt for a transaction.
fn build_receipt(tx: &OpTxEnvelope, _index: u64) -> OpReceiptEnvelope {
    use alloy_consensus::{Eip658Value, Receipt, ReceiptWithBloom};
    use op_alloy_consensus::OpDepositReceipt;

    match tx {
        OpTxEnvelope::Deposit(_) => {
            let receipt = OpDepositReceipt {
                inner: Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 0,
                    logs: vec![],
                },
                deposit_nonce: Some(0),
                deposit_receipt_version: Some(1),
            };
            OpReceiptEnvelope::Deposit(ReceiptWithBloom::new(receipt, Bloom::default()))
        }
        _ => {
            let receipt = Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 21_000,
                logs: vec![],
            };
            OpReceiptEnvelope::Eip1559(ReceiptWithBloom::new(receipt, Bloom::default()))
        }
    }
}
