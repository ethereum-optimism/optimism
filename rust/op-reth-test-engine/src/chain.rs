//! Ephemeral, genesis-initialized OP chain backed by a temp-dir reth provider.

use std::{collections::HashMap, sync::Arc};

use alloy_consensus::{
    BlockHeader as _, Header, TxReceipt,
    transaction::{Recovered, SignerRecoverable},
};
use alloy_eips::{BlockHashOrNumber, eip2718::Encodable2718};
use alloy_genesis::Genesis;
use alloy_primitives::{B256, keccak256};
use op_revm::constants::L1_BLOCK_CONTRACT;
use reth_chain_state::{ExecutedBlock, NewCanonicalChain};
use reth_db::{DatabaseEnv, test_utils::TempDatabase};
use reth_db_common::init::init_genesis;
use reth_evm::{
    ConfigureEvm,
    execute::{BlockBuilder, BlockBuilderOutcome},
};
use reth_node_api::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::{OpEvmConfig, OpNextBlockEnvAttributes, OpRethReceiptBuilder};
use reth_optimism_node::OpNode;
use reth_optimism_primitives::{OpBlock, OpPrimitives, OpReceipt, OpTransactionSigned};
use reth_optimism_rpc::eth::receipt::OpReceiptConverter;
use reth_primitives_traits::{RecoveredBlock, SealedBlock, SealedHeader};
use reth_provider::{
    ProviderError, providers::BlockchainProvider,
    test_utils::create_test_provider_factory_with_node_types,
};
use reth_revm::{database::StateProviderDatabase, db::State};
use reth_rpc_eth_api::transaction::{ConvertReceiptInput, ReceiptConverter};
use reth_storage_api::{
    BlockReader, HeaderProvider, ReceiptProvider, StateProviderBox, StateProviderFactory,
    TransactionsProvider,
};

use alloy_consensus::transaction::TransactionMeta;
use op_alloy_rpc_types::OpTransactionReceipt;

use crate::Error;

/// A block assembled from a parent state plus a set of transactions, with the total gas the
/// stateful builder needs to track remaining block gas.
pub(crate) struct BuiltBlock {
    /// The sealed, recovered block.
    pub block: RecoveredBlock<OpBlock>,
    /// Total gas used by the block.
    pub gas_used: u64,
}

type TestNodeTypes = NodeTypesWithDBAdapter<OpNode, Arc<TempDatabase<DatabaseEnv>>>;
type Provider = BlockchainProvider<TestNodeTypes>;

/// An ephemeral, genesis-initialized OP chain answering read-only block/header/receipt
/// queries and accepting executed blocks onto the canonical chain.
///
/// State and indexing are provided by a reth provider, so trie state roots and receipt lookups are
/// real rather than reimplemented here. Genesis lives in an ephemeral (tmpfs-backed) database;
/// blocks built on top are held in the provider's in-memory canonical state and overlay it, so
/// nothing past genesis is written to disk. The backing storage is discarded when the chain drops.
#[derive(Debug)]
pub struct EphemeralChain {
    provider: Provider,
    chain_spec: Arc<OpChainSpec>,
    genesis_hash: B256,
}

impl EphemeralChain {
    /// Build an ephemeral chain from a genesis spec, initializing genesis state.
    pub fn new(genesis: Genesis) -> crate::Result<Self> {
        Self::from_chain_spec(Arc::new(genesis.into()))
    }

    /// Build an ephemeral chain from an already-constructed chain spec.
    ///
    /// Tests use this to activate hardforks via [`OpChainSpecBuilder`][reth_optimism_chainspec]
    /// (which encodes activations directly) rather than round-tripping them through genesis JSON.
    pub(crate) fn from_chain_spec(chain_spec: Arc<OpChainSpec>) -> crate::Result<Self> {
        let factory = create_test_provider_factory_with_node_types::<OpNode>(chain_spec.clone());
        let genesis_hash = init_genesis(&factory)?;
        let provider = BlockchainProvider::new(factory)?;
        Ok(Self { provider, chain_spec, genesis_hash })
    }

    /// The chain spec this chain was initialized with.
    pub(crate) fn chain_spec(&self) -> Arc<OpChainSpec> {
        self.chain_spec.clone()
    }

    /// The hash of the genesis block.
    pub const fn genesis_hash(&self) -> B256 {
        self.genesis_hash
    }

    /// The sealed header of the current canonical head.
    #[cfg(test)]
    pub(crate) fn tip_header(&self) -> SealedHeader {
        self.provider.canonical_in_memory_state().get_canonical_head()
    }

    /// A state provider for the chain tip (a memory overlay over the genesis DB state once
    /// blocks have been built), used as the parent state for executing the next block.
    #[cfg(test)]
    pub(crate) fn latest_state(&self) -> crate::Result<StateProviderBox> {
        Ok(self.provider.latest()?)
    }

    /// A state provider rooted at `hash`, or `None` if that block is unknown to the chain.
    pub(crate) fn state_at(&self, hash: B256) -> crate::Result<Option<StateProviderBox>> {
        match self.provider.state_by_block_hash(hash) {
            Ok(state) => Ok(Some(state)),
            Err(ProviderError::StateForHashNotFound(_)) => Ok(None),
            Err(err) => Err(err.into()),
        }
    }

    /// The sealed header of the block `hash`, or `None` if unknown. The hash is trusted rather than
    /// recomputed, since it came from a header the provider already indexed.
    pub(crate) fn sealed_header(&self, hash: B256) -> crate::Result<Option<SealedHeader>> {
        Ok(self.provider.header(hash)?.map(|header| SealedHeader::new(header, hash)))
    }

    /// The chain id.
    pub fn chain_id(&self) -> u64 {
        self.chain_spec.chain().id()
    }

    /// The sealed header of the current canonical head (the `latest` block).
    pub fn latest_header(&self) -> SealedHeader {
        self.provider.canonical_in_memory_state().get_canonical_head()
    }

    /// The sealed header of the current `safe` block, or `None` if unset.
    pub fn safe_header(&self) -> Option<SealedHeader> {
        self.provider.canonical_in_memory_state().get_safe_header()
    }

    /// The sealed header of the current `finalized` block, or `None` if unset.
    pub fn finalized_header(&self) -> Option<SealedHeader> {
        self.provider.canonical_in_memory_state().get_finalized_header()
    }

    /// Assemble a block on top of `parent_hash` from `next_env` and `txs`, computing all roots.
    ///
    /// This drives reth's [`BlockBuilder`] end-to-end: it opens a fresh bundle state over the
    /// parent, applies pre-execution system changes, executes each transaction in order, and seals
    /// the block via the OP block assembler (which fills in Holocene `extraData`, the Isthmus
    /// withdrawals root, and so on). Nothing is committed — the caller round-trips the result
    /// through [`new_payload`](crate::TestEngine::new_payload). Executing the same `txs` twice is
    /// deterministic, which is what lets the stateful builder re-run the accumulated list on each
    /// `include_tx`/`get_payload` call rather than holding a live executor across RPC round-trips.
    pub(crate) fn assemble_block(
        &self,
        parent_hash: B256,
        next_env: OpNextBlockEnvAttributes,
        txs: &[OpTransactionSigned],
    ) -> crate::Result<BuiltBlock> {
        let evm_config: OpEvmConfig =
            OpEvmConfig::new(self.chain_spec(), OpRethReceiptBuilder::default());
        let parent = self
            .sealed_header(parent_hash)?
            .ok_or_else(|| Error::Execution(format!("parent block {parent_hash} is unknown")))?;
        let state = self
            .state_at(parent_hash)?
            .ok_or_else(|| Error::Execution(format!("no state for parent block {parent_hash}")))?;

        let mut db = State::builder()
            .with_database(StateProviderDatabase::new(&state))
            .with_bundle_update()
            .build();
        // Assembling an OP block reads the DA-footprint scalar from the L1Block predeploy; a cold
        // cache there panics, so preload it.
        db.load_cache_account(L1_BLOCK_CONTRACT).map_err(exec_err)?;

        let mut builder =
            evm_config.builder_for_next_block(&mut db, &parent, next_env).map_err(exec_err)?;
        builder.apply_pre_execution_changes().map_err(exec_err)?;
        for tx in txs {
            let recovered = tx.clone().try_into_recovered().map_err(|_| {
                Error::Execution("failed to recover transaction sender".to_string())
            })?;
            builder.execute_transaction(recovered).map_err(exec_err)?;
        }
        let BlockBuilderOutcome { block, execution_result, .. } =
            builder.finish(&state, None).map_err(exec_err)?;
        Ok(BuiltBlock { block, gas_used: execution_result.gas_used })
    }

    /// Commit an executed block as the new canonical head.
    ///
    /// Both calls are required: `update_chain` inserts the block into the in-memory map (making it
    /// queryable), while `set_canonical_head` advances the head pointer that `latest()` and
    /// `best_block_number` read.
    pub(crate) fn commit_block(&self, executed: ExecutedBlock<OpPrimitives>) {
        let head = executed.recovered_block.clone_sealed_header();
        let state = self.provider.canonical_in_memory_state();
        state.update_chain(NewCanonicalChain::Commit { new: vec![executed] });
        state.set_canonical_head(head);
    }

    /// Point the canonical/safe/finalized heads at the given hashes (attrs-less forkchoice).
    ///
    /// Canonicalizes `head`: a linear reset or an alternate-fork head both reorg the in-memory
    /// canonical chain onto it (via [`reorg_to`](Self::reorg_to)), so provider reads (`latest`,
    /// block-by-number) reflect exactly the new chain. `side_blocks` supplies executed blocks that
    /// are not currently canonical — those buffered by `new_payload` and any previously reorged-out
    /// — so an alternate fork can be materialized.
    ///
    /// Returns `Ok(false)` without mutating anything if `head` (or an ancestor needed to reach the
    /// current chain) is unknown, which the engine maps to `SYNCING`. A zero `safe`/`finalized`
    /// hash is skipped; a non-zero one that is unknown is an [`Error::UnknownForkchoiceBlock`].
    /// All three are resolved before any pointer is moved.
    pub(crate) fn advance_forkchoice(
        &self,
        head: B256,
        safe: B256,
        finalized: B256,
        side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
    ) -> crate::Result<bool> {
        let Some(head_header) = self.resolve_block_header(head, side_blocks)? else {
            return Ok(false);
        };
        // Resolve safe/finalized against the canonical chain or the buffered blocks before mutating
        // anything, so a bad pointer fails atomically.
        let safe_header = self.resolve_forkchoice_block("safe", safe, side_blocks)?;
        let finalized_header =
            self.resolve_forkchoice_block("finalized", finalized, side_blocks)?;

        if !self.reorg_to(&head_header, side_blocks)? {
            return Ok(false);
        }

        let state = self.provider.canonical_in_memory_state();
        if let Some(header) = safe_header {
            state.set_safe(header);
        }
        if let Some(header) = finalized_header {
            state.set_finalized(header);
        }
        Ok(true)
    }

    /// Reorg the in-memory canonical chain so `new_head` becomes the tip.
    ///
    /// Traces `new_head` back to the first ancestor already on the current canonical chain (the
    /// fork point), then applies a single [`NewCanonicalChain::Reorg`] that adds the blocks
    /// from the fork point up to `new_head` and removes the canonical blocks above the fork
    /// point, followed by a `set_canonical_head` — the exact pair reth's own engine tree uses.
    /// Genesis lives on disk and always terminates the walk, so a full reset to genesis simply
    /// drops every in-memory block. Returns `Ok(false)` if a block on the path to the fork
    /// point can't be resolved from either the canonical chain or `side_blocks` (mapped to
    /// `SYNCING`).
    fn reorg_to(
        &self,
        new_head: &SealedHeader,
        side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
    ) -> crate::Result<bool> {
        let state = self.provider.canonical_in_memory_state();
        let cur_head = state.get_canonical_head();
        if new_head.hash() == cur_head.hash() {
            return Ok(true);
        }

        // The current canonical chain as hash -> number, tip down to (and including) genesis.
        let mut canonical: HashMap<B256, u64> = HashMap::new();
        let mut hash = cur_head.hash();
        loop {
            let Some(header) = self.sealed_header(hash)? else {
                return Ok(false);
            };
            canonical.insert(hash, header.number);
            if header.number == 0 {
                break;
            }
            hash = header.parent_hash;
        }

        // Walk the new head down to the fork point, collecting the blocks to add (newest first).
        let mut new_blocks: Vec<ExecutedBlock<OpPrimitives>> = Vec::new();
        let mut cursor = new_head.hash();
        let fork_number = loop {
            if let Some(&number) = canonical.get(&cursor) {
                break number;
            }
            let Some(block) = self.executed_block(cursor, side_blocks) else {
                return Ok(false);
            };
            let parent = block.recovered_block().parent_hash();
            new_blocks.push(block);
            cursor = parent;
        };

        // The canonical blocks strictly above the fork point are removed. They are always in memory
        // (only genesis is on disk, and it can never be above the fork point).
        let old_blocks: Vec<ExecutedBlock<OpPrimitives>> = canonical
            .iter()
            .filter(|&(_, &number)| number > fork_number)
            .filter_map(|(&hash, _)| state.state_by_hash(hash).map(|s| s.block_ref().clone()))
            .collect();

        new_blocks.reverse();
        state.update_chain(NewCanonicalChain::Reorg { new: new_blocks, old: old_blocks });
        state.set_canonical_head(new_head.clone());
        Ok(true)
    }

    /// Resolve a block header by hash from the canonical chain or the buffered `side_blocks`, or
    /// `None` if it is unknown to both.
    fn resolve_block_header(
        &self,
        hash: B256,
        side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
    ) -> crate::Result<Option<SealedHeader>> {
        if let Some(header) = self.sealed_header(hash)? {
            return Ok(Some(header));
        }
        Ok(side_blocks.get(&hash).map(|block| block.recovered_block().clone_sealed_header()))
    }

    /// The executed block for `hash`: a buffered/reorged-out `side_blocks` entry if present, else
    /// the in-memory canonical block state.
    fn executed_block(
        &self,
        hash: B256,
        side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
    ) -> Option<ExecutedBlock<OpPrimitives>> {
        side_blocks.get(&hash).cloned().or_else(|| {
            self.provider
                .canonical_in_memory_state()
                .state_by_hash(hash)
                .map(|s| s.block_ref().clone())
        })
    }

    /// Look up a non-zero `safe`/`finalized` forkchoice block from the canonical chain or the
    /// buffered `side_blocks`, erroring if it is unknown. A zero hash means "unset" (`None`).
    fn resolve_forkchoice_block(
        &self,
        which: &'static str,
        hash: B256,
        side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
    ) -> crate::Result<Option<SealedHeader>> {
        if hash.is_zero() {
            return Ok(None);
        }
        self.resolve_block_header(hash, side_blocks)?
            .map(Some)
            .ok_or(crate::Error::UnknownForkchoiceBlock { which, hash })
    }

    /// Fetch a block by number, or `None` if unknown.
    pub fn block_by_number(&self, number: u64) -> crate::Result<Option<OpBlock>> {
        Ok(self.provider.block_by_number(number)?)
    }

    /// Fetch a block by hash, or `None` if unknown.
    pub fn block_by_hash(&self, hash: B256) -> crate::Result<Option<OpBlock>> {
        Ok(self.provider.block_by_hash(hash)?)
    }

    /// Fetch a header by number, or `None` if unknown.
    pub fn header_by_number(&self, number: u64) -> crate::Result<Option<Header>> {
        Ok(self.provider.header_by_number(number)?)
    }

    /// Fetch the receipts of a block by hash, or `None` if unknown.
    pub fn receipts_by_block_hash(&self, hash: B256) -> crate::Result<Option<Vec<OpReceipt>>> {
        Ok(self.provider.receipts_by_block(BlockHashOrNumber::Hash(hash))?)
    }

    /// Build the OP-enriched RPC receipts of a block by hash, or `None` if the block is unknown.
    ///
    /// This is reth's production receipt path: the consensus [`OpReceipt`]s are folded with the
    /// block's L1 fee info (via `OpReceiptConverter`, which reads the `L1Block` predeploy from the
    /// L1-info deposit) into the `l1Fee`/`l1GasUsed`/…, deposit-nonce, and deposit-receipt-version
    /// fields op-node's `FetchReceipts` and the action tests read. The running cumulative-gas and
    /// log-index accounting mirrors reth's `LoadReceipt`.
    pub fn rpc_receipts_by_block_hash(
        &self,
        hash: B256,
    ) -> crate::Result<Option<Vec<OpTransactionReceipt>>> {
        let Some(block) = self.block_by_hash(hash)? else {
            return Ok(None);
        };
        let Some(receipts) = self.receipts_by_block_hash(hash)? else {
            return Ok(None);
        };
        let sealed = SealedBlock::<OpBlock>::new_unchecked(block, hash);
        Ok(Some(self.convert_receipts(&sealed, &receipts)?))
    }

    /// Fetch a committed transaction and its inclusion metadata (block hash/number, index,
    /// base fee, timestamp) by hash, or `None` if unknown.
    pub fn transaction_by_hash(
        &self,
        hash: B256,
    ) -> crate::Result<Option<(OpTransactionSigned, TransactionMeta)>> {
        Ok(self.provider.transaction_by_hash_with_meta(hash)?)
    }

    /// Build the single OP-enriched RPC receipt for a transaction hash, or `None` if unknown.
    pub fn rpc_receipt_by_tx_hash(
        &self,
        tx_hash: B256,
    ) -> crate::Result<Option<OpTransactionReceipt>> {
        let Some((_, meta)) = self.provider.transaction_by_hash_with_meta(tx_hash)? else {
            return Ok(None);
        };
        let Some(mut receipts) = self.rpc_receipts_by_block_hash(meta.block_hash)? else {
            return Ok(None);
        };
        if (meta.index as usize) >= receipts.len() {
            return Ok(None);
        }
        Ok(Some(receipts.swap_remove(meta.index as usize)))
    }

    /// Convert a sealed block's consensus receipts to OP RPC receipts via reth's
    /// `OpReceiptConverter`.
    fn convert_receipts(
        &self,
        sealed: &SealedBlock<OpBlock>,
        receipts: &[OpReceipt],
    ) -> crate::Result<Vec<OpTransactionReceipt>> {
        let header = sealed.header();
        let base_fee = header.base_fee_per_gas;
        let block_number = header.number;
        let block_hash = sealed.hash();
        let timestamp = header.timestamp;
        let excess_blob_gas = header.excess_blob_gas;

        let txs = &sealed.body().transactions;
        let mut inputs: Vec<ConvertReceiptInput<'_, OpPrimitives>> =
            Vec::with_capacity(receipts.len());
        let mut prev_cumulative_gas = 0u64;
        let mut next_log_index = 0usize;
        for (index, (tx, receipt)) in txs.iter().zip(receipts.iter()).enumerate() {
            let signer = tx
                .recover_signer()
                .map_err(|err| Error::Execution(format!("recover receipt signer: {err}")))?;
            let cumulative_gas = receipt.cumulative_gas_used();
            let meta = TransactionMeta {
                tx_hash: keccak256(tx.encoded_2718()),
                index: index as u64,
                block_hash,
                block_number,
                base_fee,
                excess_blob_gas,
                timestamp,
            };
            inputs.push(ConvertReceiptInput {
                receipt: receipt.clone(),
                tx: Recovered::new_unchecked(tx, signer),
                gas_used: cumulative_gas - prev_cumulative_gas,
                next_log_index,
                meta,
            });
            prev_cumulative_gas = cumulative_gas;
            next_log_index += receipt.logs().len();
        }

        OpReceiptConverter::new(self.provider.clone())
            .convert_receipts_with_block(inputs, sealed)
            .map_err(|err| Error::Execution(format!("convert receipts: {err}")))
    }
}

/// Wrap a builder/EVM error as an internal execution error (distinct from an `INVALID` payload).
fn exec_err(err: impl core::fmt::Display) -> Error {
    Error::Execution(err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{Address, U256};

    fn test_genesis() -> Genesis {
        Genesis::default().extend_accounts([(
            Address::with_last_byte(0x42),
            GenesisAccount { balance: U256::from(1_000_000u64), ..Default::default() },
        )])
    }

    #[test]
    fn genesis_roundtrip() {
        let chain = EphemeralChain::new(test_genesis()).expect("build chain");
        let header = chain.header_by_number(0).expect("query").expect("genesis header present");
        // The hash reth recorded for genesis matches the header we read back.
        assert_eq!(header.hash_slow(), chain.genesis_hash());
        // Genesis state root is a real trie root (alloc applied), not the zero hash.
        assert_ne!(header.state_root, B256::ZERO);
        // The full genesis block is queryable and hashes to the same genesis hash.
        let block = chain.block_by_number(0).expect("query").expect("genesis block present");
        assert_eq!(block.header.hash_slow(), chain.genesis_hash());
    }

    #[test]
    fn queries_on_genesis_only_chain() {
        let chain = EphemeralChain::new(test_genesis()).expect("build chain");
        // Genesis is the latest (and only) block.
        assert!(chain.block_by_number(0).expect("query").is_some());
        // Unknown inputs return `None` across all four read methods.
        let unknown = B256::repeat_byte(0xab);
        assert!(chain.block_by_hash(unknown).expect("query").is_none());
        assert!(chain.header_by_number(99).expect("query").is_none());
        assert!(chain.receipts_by_block_hash(unknown).expect("query").is_none());
    }
}
