//! L2 chain builder that constructs valid OP Stack L2 blocks with real EVM execution.

use alloy_consensus::{
    EMPTY_OMMER_ROOT_HASH, Header,
    transaction::{Recovered, SignerRecoverable},
};
use alloy_eips::{eip1559::BaseFeeParams, eip7685::EMPTY_REQUESTS_HASH};
use alloy_evm::{
    EvmEnv, EvmFactory as _,
    block::{BlockExecutor, BlockExecutorFactory},
};
use alloy_op_evm::{
    OpBlockExecutionCtx, OpBlockExecutorFactory, block::receipt_builder::OpAlloyReceiptBuilder,
};
use alloy_primitives::{B256, Bloom, Bytes, Sealable, U256};
use op_alloy_consensus::{OpReceiptEnvelope, OpTxEnvelope};
use op_revm::OpSpecId;
use revm::{
    context::{BlockEnv, CfgEnv},
    context_interface::block::BlobExcessGasAndPrice,
};

use crate::{
    config::DeterministicConfig,
    l1::L1Block,
    state::{
        StateSnapshot, TestStateDb, compute_receipts_root, compute_transactions_root,
        rebuild_cache_db,
    },
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
        // Use the authoritative genesis state root computed by go-ethereum's Genesis.ToBlock(),
        // NOT our own trie computation. The framework's trie is used for post-genesis blocks,
        // but the genesis state root must match what op-program/op-geth expect.
        let genesis_state_root = config.l2_genesis_state_root;

        // Read gas_limit from the rollup config's genesis system config
        let gas_limit = config
            .rollup_config()
            .genesis
            .system_config
            .as_ref()
            .map(|sc| sc.gas_limit)
            .expect("rollup config must have genesis system config with gas_limit");

        let hardforks = &config.rollup_config().hardforks;

        // Isthmus requires requests_hash per EIP-7685
        let requests_hash = hardforks
            .isthmus_time
            .filter(|&t| config.genesis_timestamp >= t)
            .map(|_| EMPTY_REQUESTS_HASH);

        // Three-way withdrawals_root per OP Stack spec (matching kona assemble.rs):
        // - Isthmus active: L2ToL1MessagePasser storage root
        // - Canyon active (pre-Isthmus): EMPTY_ROOT_HASH
        // - Pre-Canyon: None
        let withdrawals_root =
            if hardforks.isthmus_time.is_some_and(|t| config.genesis_timestamp >= t) {
                Some(message_passer_storage_root_from_snapshot(&genesis_snapshot))
            } else if hardforks.canyon_time.is_some_and(|t| config.genesis_timestamp >= t) {
                Some(crate::state::roots::EMPTY_ROOT_HASH)
            } else {
                None
            };

        // Read genesis header fields from the JSON
        let (timestamp, json_gas_limit, base_fee, extra_data, coinbase, _difficulty, mix_hash) =
            config.l2_genesis_header_fields();

        // Prefer the gas limit from JSON if it matches, otherwise use rollup config
        let header_gas_limit = if json_gas_limit > 0 { json_gas_limit } else { gas_limit };

        // blob gas fields from genesis JSON
        let blob_gas_used = Some(0);
        let excess_blob_gas = config.l2_genesis_excess_blob_gas().or(Some(0));

        let genesis_header = Header {
            number: 0,
            timestamp,
            state_root: genesis_state_root,
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            transactions_root: crate::state::roots::EMPTY_ROOT_HASH,
            receipts_root: crate::state::roots::EMPTY_ROOT_HASH,
            gas_limit: header_gas_limit,
            beneficiary: coinbase,
            mix_hash,
            withdrawals_root,
            base_fee_per_gas: Some(base_fee),
            blob_gas_used,
            excess_blob_gas,
            parent_beacon_block_root: Some(B256::ZERO),
            extra_data,
            requests_hash,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        // Verify the genesis hash matches what the rollup config expects
        let expected_hash = config.rollup_config().genesis.l2.hash;
        assert_eq!(
            sealed.hash(),
            expected_hash,
            "L2 genesis hash mismatch: builder produced {:?} but rollup config expects {:?}",
            sealed.hash(),
            expected_hash,
        );

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

    /// Advance the sequence number by 1 without building a block.
    /// Used when the genesis block already consumed `seq_num` 0 for the genesis epoch.
    pub const fn advance_seq_num(&mut self) {
        self.seq_num += 1;
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

        // Extract all values from the parent block before taking &mut self for execution.
        let parent_header = self.blocks.last().expect("always have genesis").header.inner().clone();
        let parent_hash = self.blocks.last().expect("always have genesis").header.hash();
        let block_num = parent_header.number + 1;
        let timestamp = parent_header.timestamp + self.config.l2_block_time;

        // Compute base fee from parent header using Holocene EIP-1559 params
        let base_fee = next_base_fee(&parent_header, &self.config);

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

        // In OP Stack, the L2 block's parent_beacon_block_root comes from the L1 origin
        let parent_beacon_block_root = epoch.l1_block.header.inner().parent_beacon_block_root;

        // Execute all transactions through OpBlockExecutor (same executor used by op-reth).
        let (receipts, gas_used, da_footprint) = self.execute_transactions(
            &all_txs,
            spec_id,
            block_env,
            parent_hash,
            parent_beacon_block_root,
        )?;

        // Compute roots from real execution results
        let transactions_root = compute_transactions_root(&all_txs);
        let receipts_root = compute_receipts_root(&receipts);
        let logs_bloom = {
            let mut bloom = Bloom::default();
            for receipt in &receipts {
                bloom.accrue_bloom(receipt.logs_bloom());
            }
            bloom
        };

        // Snapshot state after applying changes (for state root)
        let snapshot = self.state.snapshot();
        let state_root = snapshot.state_root;

        let hardforks = &self.config.rollup_config().hardforks;

        // Three-way withdrawals_root per OP Stack spec (matching kona assemble.rs):
        // - Isthmus active: L2ToL1MessagePasser storage root
        // - Canyon active (pre-Isthmus): EMPTY_ROOT_HASH
        // - Pre-Canyon: None
        let withdrawals_root = if hardforks.isthmus_time.is_some_and(|t| timestamp >= t) {
            Some(message_passer_storage_root_from_state(&self.state))
        } else if hardforks.canyon_time.is_some_and(|t| timestamp >= t) {
            Some(crate::state::roots::EMPTY_ROOT_HASH)
        } else {
            None
        };

        // Isthmus requires requests_hash per EIP-7685
        let requests_hash =
            hardforks.isthmus_time.filter(|&t| timestamp >= t).map(|_| EMPTY_REQUESTS_HASH);

        // Build EIP-1559 extra data for non-genesis blocks.
        // Jovian: 17 bytes [0x01, denom_be32, elasticity_be32, min_base_fee_be64]
        // Holocene: 9 bytes [0x00, denom_be32, elasticity_be32]
        let extra_data = if hardforks.jovian_time.is_some_and(|t| timestamp >= t) {
            let denom = self.config.eip1559_denominator();
            let elasticity = self.config.eip1559_elasticity();
            let min_base_fee = self.config.min_base_fee();
            let eip_1559_params = alloy_primitives::B64::from({
                let mut b = [0u8; 8];
                b[0..4].copy_from_slice(&denom.to_be_bytes());
                b[4..8].copy_from_slice(&elasticity.to_be_bytes());
                b
            });
            let default_params =
                alloy_eips::eip1559::BaseFeeParams::new(denom as u128, elasticity as u128);
            op_alloy_consensus::encode_jovian_extra_data(
                eip_1559_params,
                default_params,
                min_base_fee,
            )
            .expect("failed to encode jovian extra data")
        } else {
            crate::config::holocene_extra_data(
                self.config.eip1559_denominator(),
                self.config.eip1559_elasticity(),
            )
        };

        // Jovian repurposes blob_gas_used to store the DA footprint
        let blob_gas_used =
            if hardforks.jovian_time.is_some_and(|t| timestamp >= t) { da_footprint } else { 0 };

        let header = Header {
            parent_hash,
            beneficiary: self.config.fee_recipient,
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
            blob_gas_used: Some(blob_gas_used),
            excess_blob_gas: Some(0),
            parent_beacon_block_root,
            // EIP-4399: mix_hash carries L1 origin's prevRandao post-merge
            mix_hash: prev_randao,
            // Holocene EIP-1559 params in extra data
            extra_data,
            requests_hash,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        let block = L2Block { header: sealed, transactions: all_txs, receipts, withdrawals_root };

        self.blocks.push(block);
        self.snapshots.push(snapshot);

        Ok(L2BlockRef { index: self.blocks.len() - 1 })
    }

    /// Execute all transactions through the OP Stack EVM using `OpBlockExecutor`.
    ///
    /// This uses the same block executor as op-reth, which handles system calls (EIP-4788
    /// beacon block root, EIP-2935 block hash history), transaction execution, receipt
    /// construction, and post-block balance increments canonically.
    fn execute_transactions(
        &mut self,
        txs: &[OpTxEnvelope],
        spec_id: OpSpecId,
        block_env: BlockEnv,
        parent_hash: B256,
        parent_beacon_block_root: Option<B256>,
    ) -> Result<(Vec<OpReceiptEnvelope>, u64, u64), Box<dyn std::error::Error>> {
        let cfg_env: CfgEnv<OpSpecId> = CfgEnv::new()
            .with_chain_id(self.config.l2_chain_id)
            .with_spec_and_mainnet_gas_params(spec_id);

        // Wrap the CacheDB in a State wrapper (required by OpBlockExecutor for StateDB trait)
        let db = self.state.db.clone();
        let mut state_db =
            revm::database::State::builder().with_database(db).with_bundle_update().build();

        // Create the EVM factory and block executor factory
        let evm_factory =
            alloy_op_evm::OpEvmFactory::<op_revm::OpTransaction<revm::context::TxEnv>>::default();
        let executor_factory = OpBlockExecutorFactory::new(
            OpAlloyReceiptBuilder::default(),
            self.config.rollup_config().clone(),
            evm_factory,
        );

        // Create the EVM with the block and cfg environments
        let evm_env = EvmEnv::new(cfg_env, block_env);
        let evm = executor_factory.evm_factory().create_evm(&mut state_db, evm_env);

        // Create the block execution context
        let ctx = OpBlockExecutionCtx {
            parent_hash,
            parent_beacon_block_root,
            extra_data: Bytes::default(),
        };

        // Create the executor
        let mut executor = executor_factory.create_executor(evm, ctx);

        // Apply pre-execution changes (system calls: EIP-4788, EIP-2935, create2 deployer)
        executor
            .apply_pre_execution_changes()
            .map_err(|e| format!("pre-execution changes failed: {e:?}"))?;

        // Execute each transaction
        for (i, tx) in txs.iter().enumerate() {
            let sender = match tx {
                OpTxEnvelope::Deposit(sealed_deposit) => sealed_deposit.inner().from,
                _ => tx.recover_signer()?,
            };
            let recovered = Recovered::new_unchecked(tx.clone(), sender);

            executor
                .execute_transaction(&recovered)
                .map_err(|e| format!("EVM error tx {i}: {e:?}"))?;
        }

        // Finish execution (applies post-block balance increments)
        let (evm, result) =
            executor.finish().map_err(|e| format!("block execution finish failed: {e:?}"))?;

        let receipts = result.receipts;
        let gas_used = result.gas_used;
        let da_footprint = result.blob_gas_used;

        // Extract the State wrapper from the EVM and get the bundle state.
        // The EVM holds &mut State<CacheDB<EmptyDB>>; finish() returns it.
        let (state_db_ref, _evm_env): (_, EvmEnv<OpSpecId>) = alloy_evm::Evm::finish(evm);
        state_db_ref
            .merge_transitions(revm_database::states::bundle_state::BundleRetention::PlainState);
        let bundle = state_db_ref.take_bundle();

        // Extract the underlying CacheDB from the State wrapper.
        // The State's cache contains all committed changes, but we need a plain CacheDB.
        // Rebuild it from the State's cache.
        let post_db = rebuild_cache_db(state_db_ref);

        // Apply the bundle state to our tracked state
        self.state.apply_bundle_state(&bundle, post_db);

        Ok((receipts, gas_used, da_footprint))
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

/// Compute the next block's base fee from the parent header using Holocene EIP-1559 parameters.
///
/// Decodes elasticity and denominator from the parent's `extra_data` field and uses the
/// standard EIP-1559 formula. Falls back to the config's `chain_op_config` values.
fn next_base_fee(parent: &Header, config: &DeterministicConfig) -> u64 {
    // Try Jovian (17-byte) format first, then Holocene (9-byte), then fallback to config defaults.
    let (elasticity, denominator, min_base_fee) =
        op_alloy_consensus::decode_jovian_extra_data(&parent.extra_data)
            .map(|(e, d, mbf)| (e, d, Some(mbf)))
            .or_else(|_| {
                op_alloy_consensus::decode_holocene_extra_data(&parent.extra_data)
                    .map(|(e, d)| (e, d, None))
            })
            .unwrap_or_else(|_| (config.eip1559_elasticity(), config.eip1559_denominator(), None));
    let params = BaseFeeParams::new(denominator as u128, elasticity as u128);
    let mut base_fee =
        parent.next_block_base_fee(params).unwrap_or_else(|| parent.base_fee_per_gas.unwrap_or(1));
    // Apply minimum base fee floor (Jovian+)
    if let Some(mbf) = min_base_fee {
        base_fee = base_fee.max(mbf);
    }
    base_fee
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
        // Extra data should be set (from genesis JSON)
        assert!(!genesis.header.inner().extra_data.is_empty());
    }

    #[test]
    fn genesis_state_root_matches_go() {
        let (config, _l1, l2) = setup_builder();
        let snapshot = l2.head_snapshot();
        let go_state_root = config.l2_genesis_state_root;
        let framework_state_root = snapshot.state_root;
        assert_eq!(
            framework_state_root, go_state_root,
            "Framework's trie-computed genesis state root must match Go's Genesis.ToBlock() state root"
        );
    }

    #[test]
    fn genesis_hash_matches_rollup_config() {
        let (config, _l1, l2) = setup_builder();
        let rollup = config.rollup_config();
        let builder_hash = l2.head().header.hash();
        let config_hash = rollup.genesis.l2.hash;
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
        // max_fee_per_gas must be >= the current base fee (which starts at
        // 1 gwei from the op-deployer genesis and decays each empty block).
        let base_fee = l2.head().header.inner().base_fee_per_gas.unwrap_or(1_000_000_000);
        // The prefunded account has nonce > 0 from op-deployer genesis deployment.
        let prefunded_nonce = l2.state().account(&PREFUNDED_ACCOUNT).map_or(0, |a| a.nonce);
        let tx = TxEip1559 {
            chain_id: config.l2_chain_id,
            nonce: prefunded_nonce,
            gas_limit: 21_000,
            max_fee_per_gas: base_fee as u128,
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
