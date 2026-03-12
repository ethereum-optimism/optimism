//! L2 chain builder that constructs valid OP Stack L2 blocks with real EVM execution.

use alloy_consensus::{
    EMPTY_OMMER_ROOT_HASH, Header, Receipt, ReceiptWithBloom, Transaction as _,
    transaction::SignerRecoverable,
};
use alloy_eips::{
    Encodable2718, Typed2718 as _, eip1559::BaseFeeParams, eip7685::EMPTY_REQUESTS_HASH,
};
use alloy_primitives::{Address, B256, Bloom, Bytes, Log, Sealable, U256};
use either::Either;
use op_alloy_consensus::{OpDepositReceipt, OpReceiptEnvelope, OpTxEnvelope};
use op_revm::{
    L1BlockInfo, OpBuilder, OpSpecId, OpTransaction, transaction::deposit::DepositTransactionParts,
};
use revm::{
    Context, ExecuteEvm, Journal,
    context::{BlockEnv, CfgEnv, TxEnv},
    context_interface::{block::BlobExcessGasAndPrice, result::ExecutionResult},
    database::CacheDB,
    handler::system_call::SystemCallEvm,
};

use crate::{
    config::DeterministicConfig,
    l1::L1Block,
    state::{StateSnapshot, TestStateDb, compute_receipts_root, compute_transactions_root},
};

use super::{
    deposit::l1_info_deposit_tx,
    types::{L2Block, L2BlockRef},
};

/// Builds a deterministic L2 chain block by block with real EVM execution.
#[allow(missing_debug_implementations)]
pub struct L2ChainBuilder {
    config: DeterministicConfig,
    state: TestStateDb,
    blocks: Vec<L2Block>,
    snapshots: Vec<StateSnapshot>,
    current_epoch: Option<EpochRef>,
    seq_num: u64,
    gas_limit: u64,
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

        // Read gas_limit from the rollup config's genesis system config
        let gas_limit = config
            .rollup_config()
            .genesis
            .system_config
            .as_ref()
            .map(|sc| sc.gas_limit)
            .expect("rollup config must have genesis system config with gas_limit");

        // Isthmus requires requests_hash per EIP-7685
        let requests_hash = config
            .hardforks
            .isthmus_time
            .filter(|&t| config.genesis_timestamp >= t)
            .map(|_| EMPTY_REQUESTS_HASH);

        // Three-way withdrawals_root per OP Stack spec (matching kona assemble.rs):
        // - Isthmus active: L2ToL1MessagePasser storage root
        // - Canyon active (pre-Isthmus): EMPTY_ROOT_HASH
        // - Pre-Canyon: None
        let withdrawals_root =
            if config.hardforks.isthmus_time.is_some_and(|t| config.genesis_timestamp >= t) {
                Some(message_passer_storage_root_from_snapshot(&genesis_snapshot))
            } else if config.hardforks.canyon_time.is_some_and(|t| config.genesis_timestamp >= t) {
                Some(crate::state::roots::EMPTY_ROOT_HASH)
            } else {
                None
            };

        let genesis_header = Header {
            number: 0,
            timestamp: config.genesis_timestamp,
            state_root: genesis_state_root,
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            transactions_root: crate::state::roots::EMPTY_ROOT_HASH,
            receipts_root: crate::state::roots::EMPTY_ROOT_HASH,
            gas_limit,
            // Post-Shanghai/Cancun fields required for correct RLP hashing
            withdrawals_root,
            base_fee_per_gas: Some(1),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            parent_beacon_block_root: Some(alloy_primitives::B256::ZERO),
            // Holocene EIP-1559 params in extra data: [version, denom_be32, elasticity_be32]
            extra_data: crate::config::holocene_extra_data(),
            requests_hash,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        let genesis_block = L2Block {
            header: sealed,
            transactions: vec![],
            receipts: vec![],
            withdrawals_root: None,
        };

        Self {
            config: config.clone(),
            state,
            blocks: vec![genesis_block],
            snapshots: vec![genesis_snapshot],
            current_epoch: None,
            seq_num: 0,
            gas_limit,
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
    ///
    /// Executes all transactions through the OP Stack EVM, collecting real receipts,
    /// computing state/transactions/receipts roots from execution results, and applying
    /// state changes to the underlying database.
    pub fn build_block(
        &mut self,
        user_txs: Vec<OpTxEnvelope>,
    ) -> Result<L2BlockRef, Box<dyn std::error::Error>> {
        let epoch = self.current_epoch.as_ref().ok_or("must set epoch before building blocks")?;

        let prev = self.blocks.last().expect("always have genesis");
        let parent_header = prev.header.inner();
        let block_num = parent_header.number + 1;
        let timestamp = parent_header.timestamp + self.config.l2_block_time;

        // Compute base fee from parent header using Holocene EIP-1559 params
        let base_fee = next_base_fee(parent_header);

        // Build L1 info deposit tx
        let deposit_tx = l1_info_deposit_tx(&self.config, &epoch.l1_block, self.seq_num)?;
        self.seq_num += 1;

        // Combine deposit + user txs
        let mut all_txs = vec![deposit_tx];
        all_txs.extend(user_txs);

        // Determine spec ID from rollup config
        let spec_id = self.config.rollup_config().spec_id(timestamp);

        // EIP-4399: L2 block's prevRandao comes from L1 origin's mix_hash
        let prev_randao = epoch.l1_block.header.inner().mix_hash;

        let block_env = BlockEnv {
            number: U256::from(block_num),
            beneficiary: self.config.fee_recipient,
            timestamp: U256::from(timestamp),
            gas_limit: self.gas_limit,
            basefee: base_fee,
            prevrandao: Some(prev_randao),
            // Cancun+ requires blob gas configuration for correct EIP-7623 gas accounting
            blob_excess_gas_and_price: Some(BlobExcessGasAndPrice {
                excess_blob_gas: 0,
                blob_gasprice: 1,
            }),
            ..Default::default()
        };

        let parent_hash = prev.header.hash();

        // In OP Stack, the L2 block's parent_beacon_block_root comes from the L1 origin
        let parent_beacon_block_root = epoch.l1_block.header.inner().parent_beacon_block_root;

        // Execute all transactions through the EVM (including pre-block system calls)
        let (receipts, evm_state) = self.execute_transactions(
            &all_txs,
            spec_id,
            block_env,
            block_num,
            parent_hash,
            parent_beacon_block_root,
        )?;

        // Apply state changes to our tracked state
        self.state.apply_evm_result(&evm_state);

        // Compute roots from real execution results
        let transactions_root = compute_transactions_root(&all_txs);
        let receipts_root = compute_receipts_root(&receipts);
        let logs_bloom = receipts_bloom(&receipts);
        let gas_used = cumulative_gas_from_receipts(&receipts);

        // Snapshot state after applying changes (for state root)
        let snapshot = self.state.snapshot();
        let state_root = snapshot.state_root;

        // Three-way withdrawals_root per OP Stack spec (matching kona assemble.rs):
        // - Isthmus active: L2ToL1MessagePasser storage root
        // - Canyon active (pre-Isthmus): EMPTY_ROOT_HASH
        // - Pre-Canyon: None
        let withdrawals_root = if self.config.hardforks.isthmus_time.is_some_and(|t| timestamp >= t)
        {
            Some(message_passer_storage_root_from_state(&self.state))
        } else if self.config.hardforks.canyon_time.is_some_and(|t| timestamp >= t) {
            Some(crate::state::roots::EMPTY_ROOT_HASH)
        } else {
            None
        };

        // Isthmus requires requests_hash per EIP-7685
        let requests_hash = self
            .config
            .hardforks
            .isthmus_time
            .filter(|&t| timestamp >= t)
            .map(|_| EMPTY_REQUESTS_HASH);

        let header = Header {
            parent_hash: prev.header.hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            number: block_num,
            timestamp,
            state_root,
            transactions_root,
            receipts_root,
            logs_bloom,
            gas_used,
            gas_limit: self.gas_limit,
            withdrawals_root,
            // Post-Cancun fields required for correct RLP hashing
            base_fee_per_gas: Some(base_fee),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            parent_beacon_block_root,
            // EIP-4399: mix_hash carries L1 origin's prevRandao post-merge
            mix_hash: prev_randao,
            // Holocene EIP-1559 params in extra data
            extra_data: crate::config::holocene_extra_data(),
            requests_hash,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        let block = L2Block { header: sealed, transactions: all_txs, receipts, withdrawals_root };

        self.blocks.push(block);
        self.snapshots.push(snapshot);

        Ok(L2BlockRef { index: self.blocks.len() - 1 })
    }

    /// Execute all transactions through the OP Stack EVM.
    ///
    /// Applies EIP-4788 (beacon block root) and EIP-2935 (block hash history) system
    /// calls before processing transactions, matching go-ethereum's block processing.
    fn execute_transactions(
        &self,
        txs: &[OpTxEnvelope],
        spec_id: OpSpecId,
        block_env: BlockEnv,
        block_num: u64,
        parent_hash: B256,
        parent_beacon_block_root: Option<B256>,
    ) -> Result<(Vec<OpReceiptEnvelope>, revm::state::EvmState), Box<dyn std::error::Error>> {
        let cfg_env: CfgEnv<OpSpecId> = CfgEnv::new()
            .with_chain_id(self.config.l2_chain_id)
            .with_spec_and_mainnet_gas_params(spec_id);

        let default_tx = OpTransaction::builder().build_fill();
        let db = self.state.db.clone();

        type EvmDb = CacheDB<revm::database::EmptyDB>;
        let base_ctx: Context<BlockEnv, TxEnv, CfgEnv<OpSpecId>, EvmDb, Journal<EvmDb>, ()> =
            Context::new(db, spec_id);

        let ctx = base_ctx
            .with_block(block_env)
            .with_cfg(cfg_env)
            .with_tx(default_tx)
            .with_chain(L1BlockInfo::default());

        let mut evm = ctx.build_op();

        // Pre-block system calls (skip genesis block per EIP specs)
        if block_num > 0 {
            // EIP-4788: store parent beacon block root (Cancun+)
            if let Some(beacon_root) = parent_beacon_block_root {
                evm.system_call_one(
                    alloy_eips::eip4788::BEACON_ROOTS_ADDRESS,
                    beacon_root.0.into(),
                )
                .map_err(|e| format!("EIP-4788 system call failed: {e:?}"))?;
            }

            // EIP-2935: store parent block hash (Prague+ / Isthmus+)
            if spec_id.is_enabled_in(OpSpecId::ISTHMUS) {
                evm.system_call_one(
                    alloy_eips::eip2935::HISTORY_STORAGE_ADDRESS,
                    parent_hash.0.into(),
                )
                .map_err(|e| format!("EIP-2935 system call failed: {e:?}"))?;
            }
        }

        let mut receipts = Vec::with_capacity(txs.len());
        let mut cumulative_gas_used: u64 = 0;

        for (i, tx) in txs.iter().enumerate() {
            let is_deposit = matches!(tx, OpTxEnvelope::Deposit(_));
            let sender = deposit_sender(tx);
            let op_tx = envelope_to_op_transaction(tx)?;

            let result = evm.transact_one(op_tx).map_err(|e| format!("EVM error tx {i}: {e:?}"))?;

            // For deposit transactions, read the sender's post-execution nonce from the
            // journal state. Per the OP Stack spec (Canyon+), the deposit nonce in the
            // receipt must be the depositor's nonce after execution.
            let deposit_nonce = if is_deposit {
                sender.and_then(|addr| {
                    evm.0.journaled_state.state.get(&addr).map(|acc| acc.info.nonce)
                })
            } else {
                None
            };

            let (gas_used, logs, success) = match &result {
                ExecutionResult::Success { gas_used, logs, .. } => (*gas_used, logs.clone(), true),
                ExecutionResult::Revert { gas_used, .. } |
                ExecutionResult::Halt { gas_used, .. } => (*gas_used, vec![], false),
            };

            cumulative_gas_used += gas_used;

            let is_canyon = spec_id.is_enabled_in(OpSpecId::CANYON);
            let receipt = build_execution_receipt(
                tx,
                success,
                cumulative_gas_used,
                logs,
                deposit_nonce,
                is_canyon,
            );
            receipts.push(receipt);
        }

        let evm_state = evm.finalize();

        Ok((receipts, evm_state))
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

/// Extract the sender address from a deposit transaction.
const fn deposit_sender(tx: &OpTxEnvelope) -> Option<Address> {
    match tx {
        OpTxEnvelope::Deposit(sealed_deposit) => Some(sealed_deposit.inner().from),
        _ => None,
    }
}

/// Convert an `OpTxEnvelope` to an `OpTransaction<TxEnv>` for revm execution.
fn envelope_to_op_transaction(
    tx: &OpTxEnvelope,
) -> Result<OpTransaction<TxEnv>, Box<dyn std::error::Error>> {
    match tx {
        OpTxEnvelope::Deposit(sealed_deposit) => {
            let deposit = sealed_deposit.inner();
            let base = TxEnv {
                tx_type: 0x7E,
                caller: deposit.from,
                gas_limit: deposit.gas_limit,
                gas_price: 0,
                kind: deposit.to,
                value: deposit.value,
                data: deposit.input.clone(),
                nonce: 0,
                chain_id: None,
                gas_priority_fee: None,
                ..Default::default()
            };

            Ok(OpTransaction {
                base,
                enveloped_tx: None,
                deposit: DepositTransactionParts {
                    source_hash: deposit.source_hash,
                    mint: (deposit.mint > 0).then_some(deposit.mint),
                    is_system_transaction: deposit.is_system_transaction,
                },
            })
        }
        _ => {
            let sender = tx.recover_signer()?;

            // Encode the transaction envelope for L1 cost calculation
            let mut encoded = Vec::new();
            tx.encode_2718(&mut encoded);

            let base = TxEnv {
                tx_type: tx.ty(),
                caller: sender,
                gas_limit: tx.gas_limit(),
                gas_price: tx.max_fee_per_gas(),
                kind: tx.kind(),
                value: tx.value(),
                data: tx.input().clone(),
                nonce: tx.nonce(),
                chain_id: tx.chain_id(),
                gas_priority_fee: tx.max_priority_fee_per_gas(),
                access_list: tx.access_list().cloned().unwrap_or_default(),
                // EIP-4844 blob txs cannot appear as L2 user transactions (L1-only),
                // so blob fields use defaults.
                blob_hashes: Vec::new(),
                max_fee_per_blob_gas: 0,
                authorization_list: tx
                    .authorization_list()
                    .map(|auths| auths.iter().cloned().map(Either::Left).collect())
                    .unwrap_or_default(),
            };

            Ok(OpTransaction {
                base,
                enveloped_tx: Some(Bytes::from(encoded)),
                deposit: DepositTransactionParts::default(),
            })
        }
    }
}

/// Build a receipt from EVM execution results.
fn build_execution_receipt(
    tx: &OpTxEnvelope,
    success: bool,
    cumulative_gas_used: u64,
    logs: Vec<Log>,
    deposit_nonce: Option<u64>,
    is_canyon: bool,
) -> OpReceiptEnvelope {
    use alloy_consensus::Eip658Value;

    let bloom = alloy_primitives::logs_bloom(logs.iter());
    let status = Eip658Value::Eip658(success);

    match tx {
        OpTxEnvelope::Deposit(_) => {
            let receipt = OpDepositReceipt {
                inner: Receipt { status, cumulative_gas_used, logs },
                deposit_nonce,
                deposit_receipt_version: is_canyon.then_some(1),
            };
            OpReceiptEnvelope::Deposit(ReceiptWithBloom::new(receipt, bloom))
        }
        OpTxEnvelope::Legacy(_) => {
            let receipt = Receipt { status, cumulative_gas_used, logs };
            OpReceiptEnvelope::Legacy(ReceiptWithBloom::new(receipt, bloom))
        }
        OpTxEnvelope::Eip2930(_) => {
            let receipt = Receipt { status, cumulative_gas_used, logs };
            OpReceiptEnvelope::Eip2930(ReceiptWithBloom::new(receipt, bloom))
        }
        OpTxEnvelope::Eip1559(_) => {
            let receipt = Receipt { status, cumulative_gas_used, logs };
            OpReceiptEnvelope::Eip1559(ReceiptWithBloom::new(receipt, bloom))
        }
        OpTxEnvelope::Eip7702(_) => {
            let receipt = Receipt { status, cumulative_gas_used, logs };
            OpReceiptEnvelope::Eip7702(ReceiptWithBloom::new(receipt, bloom))
        }
    }
}

/// Compute the aggregate logs bloom from all receipts.
fn receipts_bloom(receipts: &[OpReceiptEnvelope]) -> Bloom {
    let mut bloom = Bloom::default();
    for receipt in receipts {
        bloom.accrue_bloom(receipt.logs_bloom());
    }
    bloom
}

/// Get the cumulative gas used from the last receipt.
fn cumulative_gas_from_receipts(receipts: &[OpReceiptEnvelope]) -> u64 {
    receipts.last().map_or(0, |r| r.cumulative_gas_used())
}

/// Compute the next block's base fee from the parent header using Holocene EIP-1559 parameters.
///
/// Decodes elasticity and denominator from the parent's `extra_data` field and uses the
/// standard EIP-1559 formula. Falls back to the parent's base fee if params can't be decoded.
fn next_base_fee(parent: &Header) -> u64 {
    let (elasticity, denominator) =
        op_alloy_consensus::decode_holocene_extra_data(&parent.extra_data)
            .unwrap_or((crate::config::EIP1559_ELASTICITY, crate::config::EIP1559_DENOMINATOR));
    let params = BaseFeeParams::new(denominator as u128, elasticity as u128);
    parent.next_block_base_fee(params).unwrap_or_else(|| parent.base_fee_per_gas.unwrap_or(1))
}

/// Compute the `L2ToL1MessagePasser` storage root from a state snapshot.
fn message_passer_storage_root_from_snapshot(snapshot: &StateSnapshot) -> B256 {
    snapshot
        .storage
        .get(&crate::config::L2_TO_L1_MESSAGE_PASSER)
        .map(|storage| {
            let mut node_store = crate::state::roots::TrieNodeStore::new();
            crate::state::roots::compute_storage_root(storage, &mut node_store)
        })
        .unwrap_or(crate::state::roots::EMPTY_ROOT_HASH)
}

/// Compute the `L2ToL1MessagePasser` storage root from the current state database.
fn message_passer_storage_root_from_state(state: &TestStateDb) -> B256 {
    state
        .account_storage(&crate::config::L2_TO_L1_MESSAGE_PASSER)
        .map(|storage| {
            let mut node_store = crate::state::roots::TrieNodeStore::new();
            crate::state::roots::compute_storage_root(storage, &mut node_store)
        })
        .unwrap_or(crate::state::roots::EMPTY_ROOT_HASH)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        config::{DeterministicConfig, PREFUNDED_ACCOUNT, PREFUNDED_ACCOUNT_KEY},
        l1::L1ChainBuilder,
        state::roots::EMPTY_ROOT_HASH,
    };
    use alloy_primitives::B256;

    fn setup_builder() -> (DeterministicConfig, L1ChainBuilder, L2ChainBuilder) {
        let config = DeterministicConfig::default();
        let l1 = L1ChainBuilder::new(&config);
        let l2 = L2ChainBuilder::new(&config);
        (config, l1, l2)
    }

    #[test]
    fn genesis_block_properties() {
        let (_config, _l1, l2) = setup_builder();
        let genesis = l2.head();

        assert_eq!(genesis.header.inner().number, 0);
        assert_ne!(genesis.header.inner().state_root, B256::ZERO);
        assert_ne!(genesis.header.inner().state_root, EMPTY_ROOT_HASH);
        // Genesis has no parent, so parent_hash is zero
        assert_eq!(genesis.header.inner().parent_hash, B256::ZERO);
        assert!(genesis.transactions.is_empty(), "genesis should have no transactions");
        // Holocene extra data should be set
        assert_eq!(genesis.header.inner().extra_data.len(), 9);
    }

    #[test]
    fn genesis_hash_matches_rollup_config() {
        let (config, _l1, l2) = setup_builder();
        let rollup = config.rollup_config();
        let builder_hash = l2.head().header.hash();
        let config_hash = rollup.genesis.l2.hash;
        eprintln!("Builder genesis hash: {:?}", builder_hash);
        eprintln!("Config genesis hash:  {:?}", config_hash);
        eprintln!("Builder extra_data:   {:?}", l2.head().header.inner().extra_data);
        assert_eq!(builder_hash, config_hash, "Genesis hash from builder must match rollup config");
    }

    #[test]
    fn block_with_transfer() {
        use alloy_consensus::{SignableTransaction, TxEip1559};
        use alloy_primitives::{Address, TxKind};
        use alloy_signer::SignerSync;
        use alloy_signer_local::PrivateKeySigner;

        let (config, mut l1, mut l2) = setup_builder();
        l1.emit_empty_block();
        let l1_block = l1.block_at(1).unwrap().clone();
        l2.set_epoch(&l1_block);

        let genesis_root = l2.head_snapshot().state_root;

        let signer = PrivateKeySigner::from_bytes(&PREFUNDED_ACCOUNT_KEY).expect("valid key");
        assert_eq!(signer.address(), PREFUNDED_ACCOUNT);

        // The deposit tx in build_block will bump the nonce for the L1 info depositor,
        // but the prefunded account starts at nonce 0.
        let tx = TxEip1559 {
            chain_id: config.l2_chain_id,
            nonce: 0,
            gas_limit: 21_000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 0,
            to: TxKind::Call(Address::with_last_byte(0x42)),
            value: U256::from(1u64),
            ..Default::default()
        };

        let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing works");
        let signed = tx.into_signed(sig);
        let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
        let op_tx = OpTxEnvelope::try_from_eth_envelope(eth_envelope)
            .expect("should convert ETH envelope to OP envelope");

        l2.build_block(vec![op_tx]).unwrap();

        let post_root = l2.head_snapshot().state_root;
        assert_ne!(post_root, genesis_root, "state root should change after transfer");
    }

    #[test]
    fn epoch_change() {
        let (config, mut l1, mut l2) = setup_builder();

        // Create two L1 blocks for two epochs
        l1.emit_empty_block(); // block 1
        l1.emit_empty_block(); // block 2

        let l1_block_1 = l1.block_at(1).unwrap().clone();
        let l1_block_2 = l1.block_at(2).unwrap().clone();

        // Build blocks in epoch 1
        l2.set_epoch(&l1_block_1);
        l2.build_empty_block().unwrap();
        l2.build_empty_block().unwrap();

        // Switch to epoch 2
        l2.set_epoch(&l1_block_2);
        l2.build_empty_block().unwrap();

        let blocks = l2.blocks();
        // 4 blocks total: genesis + 3 built
        assert_eq!(blocks.len(), 4);

        // The deposit tx in block 3 (index 3) should reference L1 block 2.
        // Deposit txs contain the L1 origin info encoded in the input data,
        // but since we use kona's L1BlockInfoTx the exact encoding is complex.
        // Instead, we verify timestamps are consistent: block 3 was built after
        // epoch change, and timestamps still increment by L2 block time.
        for window in blocks.windows(2) {
            assert_eq!(
                window[1].header.inner().timestamp,
                window[0].header.inner().timestamp + config.l2_block_time,
            );
        }

        // Each non-genesis block should have at least one deposit tx
        for (i, block) in blocks.iter().enumerate().skip(1) {
            assert!(
                block.transactions.iter().any(|tx| matches!(tx, OpTxEnvelope::Deposit(_))),
                "block {i} should have a deposit tx"
            );
        }
    }
}
