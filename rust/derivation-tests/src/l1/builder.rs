//! L1 chain builder for constructing deterministic L1 blocks with real EVM execution.

use alloy_consensus::{
    Header, SignableTransaction, TxEip1559, TxEip4844, TxEnvelope,
    transaction::{Recovered, SignerRecoverable},
};
use alloy_eips::{Encodable2718, eip4895::Withdrawals, eip7685::EMPTY_REQUESTS_HASH};
use alloy_evm::{
    EvmEnv, EvmFactory as _,
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
};
use alloy_primitives::{B256, Bloom, Bytes, Sealable, TxKind, U256, hex};
use alloy_signer::SignerSync;
use alloy_signer_local::PrivateKeySigner;
use reth_evm_ethereum::{EthEvmConfig, revm_spec_by_timestamp_and_block_number};
use revm::context::{BlockEnv, CfgEnv};
use std::{borrow::Cow, collections::BTreeMap, sync::Arc};

use alloy_genesis::GenesisAccount;

use crate::{
    config::{DeterministicConfig, PREFUNDED_ACCOUNT, PREFUNDED_ACCOUNT_KEY},
    state::{StateSnapshot, TestStateDb, rebuild_cache_db, roots::EMPTY_ROOT_HASH},
};

use super::types::{BatchSubmission, BlobWithCommitment, L1Block, SystemConfigUpdate};

/// Builds a deterministic L1 chain block by block with real EVM execution.
///
/// Mirrors the `L2ChainBuilder` pattern: transactions execute through reth's Ethereum
/// block executor, producing real state roots, receipts, gas accounting, and bloom filters.
#[allow(missing_debug_implementations)]
pub struct L1ChainBuilder {
    config: DeterministicConfig,
    state: TestStateDb,
    blocks: Vec<L1Block>,
    snapshots: Vec<StateSnapshot>,
    /// Blobs indexed by beacon slot.
    blobs: BTreeMap<u64, Vec<BlobWithCommitment>>,
    /// Batcher transaction nonce counter.
    batcher_nonce: u64,
    /// `SystemConfig` owner (`PREFUNDED_ACCOUNT`) nonce counter.
    owner_nonce: u64,
    /// L1 chain spec for `EthEvmConfig`.
    chain_spec: Arc<reth_chainspec::ChainSpec>,
}

impl L1ChainBuilder {
    /// Create a new L1 chain builder with a genesis block.
    pub fn new(config: &DeterministicConfig) -> Self {
        // Initialize state from L1 genesis allocs, plus fund test accounts for gas.
        // The batcher and owner (PREFUNDED_ACCOUNT) aren't in the L1 genesis allocs from
        // op-deployer — they're Hardhat default accounts. We include them in the initial
        // allocs so they survive state DB rebuilds across block executions.
        let mut allocs = config.l1_genesis_allocs();
        let test_balance = U256::from(1_000_000_000_000_000_000_000u128); // 1000 ETH
        allocs
            .entry(config.batcher)
            .or_insert_with(|| GenesisAccount::default().with_balance(test_balance));
        allocs
            .entry(PREFUNDED_ACCOUNT)
            .or_insert_with(|| GenesisAccount::default().with_balance(test_balance));

        let mut state = TestStateDb::new();
        state.init_genesis(&allocs);

        // Take the genesis snapshot. Note: the state root will differ from go-ethereum's
        // because we've added test accounts. The genesis HEADER still uses go-ethereum's
        // state root to preserve the expected genesis block hash.
        let genesis_snapshot = state.snapshot();

        let chain_spec = config.l1_chain_spec();

        // Read genesis account nonces (0 for newly funded accounts)
        let batcher_nonce = state.account(&config.batcher).map_or(0, |a| a.nonce);
        let owner_nonce = state.account(&PREFUNDED_ACCOUNT).map_or(0, |a| a.nonce);

        // Read header fields from L1 genesis JSON
        let (timestamp, gas_limit, base_fee, extra_data, coinbase, _difficulty, mix_hash) =
            config.l1_genesis_header_fields();
        let excess_blob_gas = config.l1_genesis_excess_blob_gas();

        // Determine which fork-dependent header fields to include based on L1 chain config.
        // This mimics go-ethereum's Genesis.ToBlock() behavior.
        let l1_chain_config = config.l1_chain_config();
        let is_shanghai = l1_chain_config.shanghai_time.is_some();
        let is_cancun = l1_chain_config.cancun_time.is_some();
        let is_prague = l1_chain_config.prague_time.is_some();

        let genesis_header = Header {
            number: 0,
            timestamp,
            state_root: config.l1_genesis_state_root,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            withdrawals_root: is_shanghai.then_some(EMPTY_ROOT_HASH),
            base_fee_per_gas: Some(base_fee),
            blob_gas_used: is_cancun.then_some(0),
            excess_blob_gas: is_cancun.then(|| excess_blob_gas.unwrap_or(0)),
            parent_beacon_block_root: is_cancun.then_some(B256::ZERO),
            requests_hash: is_prague.then_some(EMPTY_REQUESTS_HASH),
            gas_limit,
            beneficiary: coinbase,
            mix_hash,
            extra_data,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        // Verify genesis hash matches rollup config
        let expected_hash = config.rollup_config().genesis.l1.hash;
        assert_eq!(
            sealed.hash(),
            expected_hash,
            "L1 genesis hash mismatch: builder produced {:?} but rollup config expects {:?}",
            sealed.hash(),
            expected_hash,
        );

        let genesis_block = L1Block { header: sealed, transactions: vec![], receipts: vec![] };

        Self {
            config: config.clone(),
            state,
            blocks: vec![genesis_block],
            snapshots: vec![genesis_snapshot],
            blobs: BTreeMap::new(),
            batcher_nonce,
            owner_nonce,
            chain_spec,
        }
    }

    /// Emit an empty L1 block with no transactions.
    pub fn emit_empty_block(&mut self) {
        let prev = self.blocks.last().expect("always have genesis");
        let parent_hash = prev.header.hash();
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;
        let base_fee = self.next_l1_base_fee();

        let block_env = self.build_block_env(block_num, timestamp, base_fee);
        let (receipts, gas_used, blob_gas_used) = self
            .execute_transactions(&[], block_env, parent_hash, Some(B256::ZERO))
            .expect("empty block execution should not fail");

        let snapshot = self.state.snapshot();

        let header = Header {
            parent_hash,
            number: block_num,
            timestamp,
            state_root: snapshot.state_root,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_used,
            gas_limit: self.gas_limit(),
            base_fee_per_gas: Some(base_fee),
            blob_gas_used: Some(blob_gas_used),
            excess_blob_gas: Some(self.current_excess_blob_gas()),
            parent_beacon_block_root: Some(B256::ZERO),
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            requests_hash: Some(EMPTY_REQUESTS_HASH),
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions: vec![], receipts });
        self.snapshots.push(snapshot);
    }

    /// Emit an L1 block containing batch submissions.
    pub fn emit_block_with_batches(&mut self, batches: Vec<BatchSubmission>) {
        let prev = self.blocks.last().expect("always have genesis");
        let parent_hash = prev.header.hash();
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;
        let base_fee = self.next_l1_base_fee();

        let signer =
            PrivateKeySigner::from_bytes(&self.config.batcher_key).expect("valid batcher key");

        let mut tx_envelopes = Vec::new();
        let mut raw_txs = Vec::new();

        for batch in batches {
            match batch {
                BatchSubmission::Calldata(data) => {
                    let (envelope, raw) = self.sign_batcher_tx(&signer, data, base_fee);
                    tx_envelopes.push(envelope);
                    raw_txs.push(raw);
                }
                BatchSubmission::Blob(blob_data) => {
                    let slot = self.timestamp_to_slot(timestamp);
                    let versioned_hash = blob_data.versioned_hash;
                    self.blobs.entry(slot).or_default().push(blob_data);
                    let (envelope, raw) =
                        self.sign_blob_tx(&signer, vec![versioned_hash], base_fee);
                    tx_envelopes.push(envelope);
                    raw_txs.push(raw);
                }
            }
        }

        let block_env = self.build_block_env(block_num, timestamp, base_fee);
        let (receipts, gas_used, blob_gas_used) = self
            .execute_transactions(&tx_envelopes, block_env, parent_hash, Some(B256::ZERO))
            .expect("batch block execution should not fail");

        let snapshot = self.state.snapshot();

        let transactions_root = compute_raw_transactions_root(&raw_txs);
        let receipts_root = compute_l1_receipts_root(&receipts);
        let logs_bloom = aggregate_logs_bloom(&receipts);

        let header = Header {
            parent_hash,
            number: block_num,
            timestamp,
            state_root: snapshot.state_root,
            transactions_root,
            receipts_root,
            logs_bloom,
            gas_used,
            gas_limit: self.gas_limit(),
            base_fee_per_gas: Some(base_fee),
            blob_gas_used: Some(blob_gas_used),
            excess_blob_gas: Some(self.current_excess_blob_gas()),
            parent_beacon_block_root: Some(B256::ZERO),
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            requests_hash: Some(EMPTY_REQUESTS_HASH),
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions: raw_txs, receipts });
        self.snapshots.push(snapshot);
    }

    /// Emit an L1 block containing a system config update.
    ///
    /// Calls the `SystemConfig` contract directly (owned by `PREFUNDED_ACCOUNT`) to produce
    /// real `ConfigUpdate` log events via EVM execution.
    pub fn emit_block_with_system_config_update(&mut self, update: SystemConfigUpdate) {
        let prev = self.blocks.last().expect("always have genesis");
        let parent_hash = prev.header.hash();
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;
        let base_fee = self.next_l1_base_fee();

        let owner_signer =
            PrivateKeySigner::from_bytes(&PREFUNDED_ACCOUNT_KEY).expect("valid owner key");

        let calldata = encode_system_config_calldata(&update);

        let tx = TxEip1559 {
            chain_id: self.config.l1_chain_id,
            nonce: self.owner_nonce,
            gas_limit: 200_000,
            max_fee_per_gas: base_fee as u128 + 1,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(self.config.system_config),
            value: U256::ZERO,
            input: calldata,
            ..Default::default()
        };
        self.owner_nonce += 1;

        let sig =
            owner_signer.sign_hash_sync(&tx.signature_hash()).expect("signing should not fail");
        let signed = tx.into_signed(sig);
        let envelope = TxEnvelope::Eip1559(signed);

        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        let raw_tx = Bytes::from(buf);

        let block_env = self.build_block_env(block_num, timestamp, base_fee);
        let (receipts, gas_used, blob_gas_used) = self
            .execute_transactions(&[envelope], block_env, parent_hash, Some(B256::ZERO))
            .expect("system config update execution should not fail");

        let snapshot = self.state.snapshot();

        let transactions_root = compute_raw_transactions_root(std::slice::from_ref(&raw_tx));
        let receipts_root = compute_l1_receipts_root(&receipts);
        let logs_bloom = aggregate_logs_bloom(&receipts);

        let header = Header {
            parent_hash,
            number: block_num,
            timestamp,
            state_root: snapshot.state_root,
            transactions_root,
            receipts_root,
            logs_bloom,
            gas_used,
            gas_limit: self.gas_limit(),
            base_fee_per_gas: Some(base_fee),
            blob_gas_used: Some(blob_gas_used),
            excess_blob_gas: Some(self.current_excess_blob_gas()),
            parent_beacon_block_root: Some(B256::ZERO),
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            requests_hash: Some(EMPTY_REQUESTS_HASH),
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions: vec![raw_tx], receipts });
        self.snapshots.push(snapshot);
    }

    /// Execute transactions through reth's Ethereum block executor.
    ///
    /// Mirrors the `L2ChainBuilder` pattern: creates an `EthEvmConfig`, wraps state in a
    /// State wrapper, creates an executor, executes transactions, and applies results.
    /// The reth Ethereum executor uses `EthereumTxEnvelope<TxEip4844>` (without variant)
    /// as its transaction type. We accept `TxEnvelope` (with variant) and convert internally.
    fn execute_transactions(
        &mut self,
        txs: &[TxEnvelope],
        block_env: BlockEnv,
        parent_hash: B256,
        parent_beacon_block_root: Option<B256>,
    ) -> Result<(Vec<alloy_consensus::ReceiptEnvelope>, u64, u64), Box<dyn std::error::Error>> {
        // Convert TxEnvelope (with TxEip4844Variant) to reth's TransactionSigned (TxEip4844)
        type TransactionSigned = alloy_consensus::EthereumTxEnvelope<alloy_consensus::TxEip4844>;
        let reth_txs: Vec<TransactionSigned> = txs.iter().map(|tx| tx.clone().into()).collect();
        let timestamp = block_env.timestamp.to::<u64>();
        let block_number = block_env.number.to::<u64>();
        let spec_id = revm_spec_by_timestamp_and_block_number(
            self.chain_spec.as_ref(),
            timestamp,
            block_number,
        );

        let cfg_env = CfgEnv::new()
            .with_chain_id(self.config.l1_chain_id)
            .with_spec_and_mainnet_gas_params(spec_id);

        // Clone state DB and wrap in State (same pattern as L2)
        let db = self.state.db.clone();
        let mut state_db =
            revm::database::State::builder().with_database(db).with_bundle_update().build();

        // Create the Ethereum EVM config and extract its executor factory
        let evm_config = EthEvmConfig::ethereum(self.chain_spec.clone());
        let evm_env = EvmEnv::new(cfg_env, block_env);
        let evm = evm_config.executor_factory.evm_factory().create_evm(&mut state_db, evm_env);

        // Post-merge: no ommers, empty withdrawals
        let empty_withdrawals = Withdrawals::default();
        let ctx = EthBlockExecutionCtx {
            parent_hash,
            parent_beacon_block_root,
            ommers: &[],
            withdrawals: Some(Cow::Borrowed(&empty_withdrawals)),
            extra_data: Bytes::default(),
            tx_count_hint: Some(txs.len()),
        };

        let mut executor = evm_config.executor_factory.create_executor(evm, ctx);

        // Apply pre-execution changes (EIP-4788 beacon root, EIP-2935 block hashes)
        executor
            .apply_pre_execution_changes()
            .map_err(|e| format!("pre-execution changes failed: {e:?}"))?;

        // Execute each transaction
        for (i, tx) in reth_txs.iter().enumerate() {
            let sender = tx.recover_signer()?;
            let recovered = Recovered::new_unchecked(tx.clone(), sender);

            executor
                .execute_transaction(&recovered)
                .map_err(|e| format!("EVM error tx {i}: {e:?}"))?;
        }

        // Finish execution
        let (evm, result) =
            executor.finish().map_err(|e| format!("block execution finish failed: {e:?}"))?;

        // Convert reth EthereumReceipt to alloy ReceiptEnvelope
        let receipts: Vec<alloy_consensus::ReceiptEnvelope> =
            result.receipts.into_iter().map(Into::into).collect();
        let gas_used = result.gas_used;
        let blob_gas_used = result.blob_gas_used;

        // Extract bundle state and apply to tracked state (same as L2)
        let (state_db_ref, _evm_env) = alloy_evm::Evm::finish(evm);
        state_db_ref
            .merge_transitions(revm_database::states::bundle_state::BundleRetention::PlainState);
        let bundle = state_db_ref.take_bundle();
        let post_db = rebuild_cache_db(state_db_ref);
        self.state.apply_bundle_state(&bundle, post_db);

        Ok((receipts, gas_used, blob_gas_used))
    }

    /// Sign a batcher transaction, returning both the decoded envelope and raw bytes.
    fn sign_batcher_tx(
        &mut self,
        signer: &PrivateKeySigner,
        calldata: Bytes,
        base_fee: u64,
    ) -> (TxEnvelope, Bytes) {
        let tx = TxEip1559 {
            chain_id: self.config.l1_chain_id,
            nonce: self.batcher_nonce,
            gas_limit: 1_000_000,
            max_fee_per_gas: base_fee as u128 + 1,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(self.config.batch_inbox),
            value: U256::ZERO,
            input: calldata,
            ..Default::default()
        };
        self.batcher_nonce += 1;

        let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing should not fail");
        let signed = tx.into_signed(sig);
        let envelope = TxEnvelope::Eip1559(signed);

        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        (envelope, Bytes::from(buf))
    }

    /// Sign an EIP-4844 blob transaction, returning both the decoded envelope and raw bytes.
    fn sign_blob_tx(
        &mut self,
        signer: &PrivateKeySigner,
        blob_versioned_hashes: Vec<B256>,
        base_fee: u64,
    ) -> (TxEnvelope, Bytes) {
        let tx = TxEip4844 {
            chain_id: self.config.l1_chain_id,
            nonce: self.batcher_nonce,
            gas_limit: 1_000_000,
            max_fee_per_gas: base_fee as u128 + 1,
            max_priority_fee_per_gas: 1,
            to: self.config.batch_inbox,
            value: U256::ZERO,
            input: Bytes::new(),
            blob_versioned_hashes,
            max_fee_per_blob_gas: 1,
            access_list: Default::default(),
        };
        self.batcher_nonce += 1;

        let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing should not fail");
        let signed = tx.into_signed(sig);
        let envelope: TxEnvelope = signed.into();

        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        (envelope, Bytes::from(buf))
    }

    /// Compute the next block's base fee using EIP-1559 rules.
    fn next_l1_base_fee(&self) -> u64 {
        let parent = self.blocks.last().expect("always have genesis").header.inner();
        let params = alloy_eips::eip1559::BaseFeeParams::ethereum();
        parent.next_block_base_fee(params).unwrap_or_else(|| parent.base_fee_per_gas.unwrap_or(1))
    }

    /// Get the current excess blob gas for the next block.
    fn current_excess_blob_gas(&self) -> u64 {
        let parent = self.blocks.last().expect("always have genesis").header.inner();
        alloy_eips::eip4844::calc_excess_blob_gas(
            parent.excess_blob_gas.unwrap_or(0),
            parent.blob_gas_used.unwrap_or(0),
        )
    }

    /// Get the L1 gas limit (from genesis header).
    fn gas_limit(&self) -> u64 {
        self.blocks.first().expect("always have genesis").header.inner().gas_limit
    }

    /// Build a `BlockEnv` for a new block.
    fn build_block_env(&self, block_num: u64, timestamp: u64, base_fee: u64) -> BlockEnv {
        BlockEnv {
            number: U256::from(block_num),
            timestamp: U256::from(timestamp),
            gas_limit: self.gas_limit(),
            basefee: base_fee,
            prevrandao: Some(B256::ZERO),
            blob_excess_gas_and_price: Some(
                revm::context_interface::block::BlobExcessGasAndPrice {
                    excess_blob_gas: self.current_excess_blob_gas(),
                    blob_gasprice: 1,
                },
            ),
            ..Default::default()
        }
    }

    /// Get all blocks.
    pub fn blocks(&self) -> &[L1Block] {
        &self.blocks
    }

    /// Get the latest block.
    pub fn head(&self) -> &L1Block {
        self.blocks.last().expect("always have genesis")
    }

    /// Get a block by number.
    pub fn block_at(&self, number: u64) -> Option<&L1Block> {
        self.blocks.get(number as usize)
    }

    /// Get blobs at a particular slot.
    pub fn blobs_at_slot(&self, slot: u64) -> Option<&Vec<BlobWithCommitment>> {
        self.blobs.get(&slot)
    }

    /// Convert a timestamp to a beacon slot number.
    pub const fn timestamp_to_slot(&self, timestamp: u64) -> u64 {
        (timestamp - self.config.genesis_timestamp) / self.config.seconds_per_slot
    }

    /// Get all blobs indexed by slot.
    pub const fn blobs(&self) -> &BTreeMap<u64, Vec<BlobWithCommitment>> {
        &self.blobs
    }

    /// Get the config.
    pub const fn config(&self) -> &DeterministicConfig {
        &self.config
    }
}

/// Compute the transactions trie root from raw RLP-encoded transaction bytes.
fn compute_raw_transactions_root(txs: &[Bytes]) -> B256 {
    if txs.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    kona_mpt::ordered_trie_with_encoder(txs, |tx, buf| buf.put_slice(tx)).root()
}

/// Compute the receipts trie root from L1 receipt envelopes.
fn compute_l1_receipts_root(receipts: &[alloy_consensus::ReceiptEnvelope]) -> B256 {
    if receipts.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    kona_mpt::ordered_trie_with_encoder(receipts, |r, buf| r.encode_2718(buf)).root()
}

/// Aggregate logs bloom from receipts.
fn aggregate_logs_bloom(receipts: &[alloy_consensus::ReceiptEnvelope]) -> Bloom {
    let mut bloom = Bloom::default();
    for receipt in receipts {
        bloom.accrue_bloom(receipt.logs_bloom());
    }
    bloom
}

/// Encode a `SystemConfig` update as ABI-encoded calldata for the `SystemConfig` contract.
fn encode_system_config_calldata(update: &SystemConfigUpdate) -> Bytes {
    match update {
        SystemConfigUpdate::BatcherAddress(addr) => {
            // setBatcherHash(bytes32): selector 0xc9b26f61
            let mut data = hex!("c9b26f61").to_vec();
            let batcher_hash = B256::left_padding_from(addr.as_slice());
            data.extend_from_slice(batcher_hash.as_slice());
            Bytes::from(data)
        }
        SystemConfigUpdate::GasConfig { overhead, scalar } => {
            // setGasConfig(uint256,uint256): selector 0x935f029e
            let mut data = hex!("935f029e").to_vec();
            data.extend_from_slice(&overhead.to_be_bytes::<32>());
            data.extend_from_slice(&scalar.to_be_bytes::<32>());
            Bytes::from(data)
        }
    }
}
