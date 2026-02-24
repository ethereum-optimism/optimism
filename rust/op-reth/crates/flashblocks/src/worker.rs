use crate::{
    PendingFlashBlock,
    pending_state::PendingBlockState,
    tx_cache::{CachedExecutionMeta, TransactionCache},
};
use alloy_eips::{BlockNumberOrTag, eip2718::WithEncoded};
use alloy_primitives::B256;
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_chain_state::{ComputedTrieData, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm, Evm,
    execute::{
        BlockAssembler, BlockAssemblerInput, BlockBuilder, BlockBuilderOutcome, BlockExecutor,
    },
};
use reth_execution_types::{BlockExecutionOutput, BlockExecutionResult};
use reth_optimism_primitives::OpReceipt;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered, RecoveredBlock,
    SealedHeader, transaction::TxHashRef,
};
use reth_revm::{
    cached::CachedReads,
    database::StateProviderDatabase,
    db::{BundleState, State, states::bundle_state::BundleRetention},
};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{
    BlockReaderIdExt, HashedPostStateProvider, StateProviderFactory, StateRootProvider,
    noop::NoopProvider,
};
use std::{
    sync::Arc,
    time::{Duration, Instant},
};
use tracing::trace;

/// The `FlashBlockBuilder` builds [`PendingBlock`] out of a sequence of transactions.
#[derive(Debug)]
pub(crate) struct FlashBlockBuilder<EvmConfig, Provider> {
    evm_config: EvmConfig,
    provider: Provider,
}

impl<EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider> {
    pub(crate) const fn new(evm_config: EvmConfig, provider: Provider) -> Self {
        Self { evm_config, provider }
    }

    pub(crate) const fn provider(&self) -> &Provider {
        &self.provider
    }
}

pub(crate) struct BuildArgs<I, N: NodePrimitives> {
    pub(crate) base: OpFlashblockPayloadBase,
    pub(crate) transactions: I,
    pub(crate) cached_state: Option<(B256, CachedReads)>,
    pub(crate) last_flashblock_index: u64,
    pub(crate) last_flashblock_hash: B256,
    pub(crate) compute_state_root: bool,
    /// Optional pending parent state for speculative building.
    /// When set, allows building on top of a pending block that hasn't been
    /// canonicalized yet.
    pub(crate) pending_parent: Option<PendingBlockState<N>>,
}

/// Result of a flashblock build operation.
#[derive(Debug)]
pub(crate) struct BuildResult<N: NodePrimitives> {
    /// The built pending flashblock.
    pub(crate) pending_flashblock: PendingFlashBlock<N>,
    /// Cached reads from this build.
    pub(crate) cached_reads: CachedReads,
    /// Pending state that can be used for building subsequent blocks.
    pub(crate) pending_state: PendingBlockState<N>,
}

/// Cached prefix execution data used to resume canonical builds.
#[derive(Debug, Clone)]
struct CachedPrefixExecutionResult<R> {
    /// Number of leading transactions covered by cached execution.
    cached_tx_count: usize,
    /// Cumulative bundle state after executing the cached prefix.
    bundle: BundleState,
    /// Cached receipts for the prefix.
    receipts: Vec<R>,
    /// Total gas used by the cached prefix.
    gas_used: u64,
    /// Total blob/DA gas used by the cached prefix.
    blob_gas_used: u64,
}

/// Receipt requirements for cache-resume flow.
pub trait FlashblockCachedReceipt: Clone {
    /// Adds `gas_offset` to each receipt's `cumulative_gas_used`.
    fn add_cumulative_gas_offset(receipts: &mut [Self], gas_offset: u64);
}

impl FlashblockCachedReceipt for OpReceipt {
    fn add_cumulative_gas_offset(receipts: &mut [Self], gas_offset: u64) {
        if gas_offset == 0 {
            return;
        }

        for receipt in receipts {
            let inner = receipt.as_receipt_mut();
            inner.cumulative_gas_used = inner.cumulative_gas_used.saturating_add(gas_offset);
        }
    }
}

impl<N, EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider>
where
    N: NodePrimitives,
    N::Receipt: FlashblockCachedReceipt,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>,
    Provider: StateProviderFactory
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin,
{
    /// Returns the [`PendingFlashBlock`] made purely out of transactions and
    /// [`OpFlashblockPayloadBase`] in `args`.
    ///
    /// This method supports two building modes:
    /// 1. **Canonical mode**: Parent matches local tip - uses state from storage
    /// 2. **Speculative mode**: Parent is a pending block - uses pending state
    ///
    /// When a `tx_cache` is provided and we're in canonical mode, the builder will
    /// attempt to resume from cached state if the transaction list is a continuation
    /// of what was previously executed.
    ///
    /// Returns `None` if:
    /// - In canonical mode: flashblock doesn't attach to the latest header
    /// - In speculative mode: no pending parent state provided
    pub(crate) fn execute<I: IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>(
        &self,
        mut args: BuildArgs<I, N>,
        tx_cache: Option<&mut TransactionCache<N>>,
    ) -> eyre::Result<Option<BuildResult<N>>> {
        trace!(target: "flashblocks", "Attempting new pending block from flashblocks");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        // Determine build mode: canonical (parent is local tip) or speculative (parent is pending)
        let is_canonical = args.base.parent_hash == latest_hash;
        let has_pending_parent = args.pending_parent.is_some();

        if !is_canonical && !has_pending_parent {
            trace!(
                target: "flashblocks",
                flashblock_parent = ?args.base.parent_hash,
                local_latest = ?latest.num_hash(),
                "Skipping non-consecutive flashblock (no pending parent available)"
            );
            return Ok(None);
        }

        // Collect transactions and extract hashes for cache lookup
        let transactions: Vec<_> = args.transactions.into_iter().collect();
        let tx_hashes: Vec<B256> = transactions.iter().map(|tx| *tx.tx_hash()).collect();

        // Get state provider and parent header context.
        // For speculative builds, use the canonical anchor hash (not the pending parent hash)
        // for storage reads, but execute with the pending parent's sealed header context.
        let (state_provider, canonical_anchor, parent_header) = if is_canonical {
            (self.provider.history_by_block_hash(latest.hash())?, latest.hash(), &latest)
        } else {
            // For speculative building, we need to use the canonical anchor
            // and apply the pending state's bundle on top of it
            let pending = args.pending_parent.as_ref().unwrap();
            let Some(parent_header) = pending.sealed_header.as_ref() else {
                trace!(
                    target: "flashblocks",
                    pending_block_number = pending.block_number,
                    pending_block_hash = ?pending.block_hash,
                    "Skipping speculative build: pending parent header is unavailable"
                );
                return Ok(None);
            };
            if !is_consistent_speculative_parent_hashes(
                args.base.parent_hash,
                pending.block_hash,
                parent_header.hash(),
            ) {
                trace!(
                    target: "flashblocks",
                    incoming_parent_hash = ?args.base.parent_hash,
                    pending_block_hash = ?pending.block_hash,
                    pending_sealed_hash = ?parent_header.hash(),
                    pending_block_number = pending.block_number,
                    "Skipping speculative build: inconsistent pending parent hashes"
                );
                return Ok(None);
            }
            trace!(
                target: "flashblocks",
                pending_block_number = pending.block_number,
                pending_block_hash = ?pending.block_hash,
                canonical_anchor = ?pending.canonical_anchor_hash,
                "Building speculatively on pending state"
            );
            (
                self.provider.history_by_block_hash(pending.canonical_anchor_hash)?,
                pending.canonical_anchor_hash,
                parent_header,
            )
        };

        // Set up cached reads
        let cache_key = if is_canonical { latest_hash } else { args.base.parent_hash };
        let mut request_cache = args
            .cached_state
            .take()
            .filter(|(hash, _)| hash == &cache_key)
            .map(|(_, state)| state)
            .unwrap_or_else(|| {
                // For speculative builds, use cached reads from pending parent
                args.pending_parent.as_ref().map(|p| p.cached_reads.clone()).unwrap_or_default()
            });

        let cached_db = request_cache.as_db_mut(StateProviderDatabase::new(&state_provider));

        // Check for resumable canonical execution state.
        let canonical_parent_hash = args.base.parent_hash;
        let cached_prefix = if is_canonical {
            tx_cache.as_ref().and_then(|cache| {
                cache
                    .get_resumable_state_with_execution_meta_for_parent(
                        args.base.block_number,
                        canonical_parent_hash,
                        &tx_hashes,
                    )
                    .map(
                        |(
                            bundle,
                            receipts,
                            _requests,
                            gas_used,
                            blob_gas_used,
                            cached_tx_count,
                        )| {
                            trace!(
                                target: "flashblocks",
                                cached_tx_count,
                                total_txs = tx_hashes.len(),
                                "Cache hit (executing only uncached suffix)"
                            );
                            CachedPrefixExecutionResult {
                                cached_tx_count,
                                bundle: bundle.clone(),
                                receipts: receipts.to_vec(),
                                gas_used,
                                blob_gas_used,
                            }
                        },
                    )
            })
        } else {
            None
        };

        // Build state with appropriate prestate
        // - Speculative builds use pending parent prestate
        // - Canonical cache-hit builds use cached prefix prestate
        let mut state = if let Some(ref pending) = args.pending_parent {
            State::builder()
                .with_database(cached_db)
                .with_bundle_prestate(pending.execution_outcome.state.clone())
                .with_bundle_update()
                .build()
        } else if let Some(ref cached_prefix) = cached_prefix {
            State::builder()
                .with_database(cached_db)
                .with_bundle_prestate(cached_prefix.bundle.clone())
                .with_bundle_update()
                .build()
        } else {
            State::builder().with_database(cached_db).with_bundle_update().build()
        };

        let (execution_result, block, hashed_state, bundle) = if let Some(cached_prefix) =
            cached_prefix
        {
            // Cached prefix execution model:
            // - The cached bundle prestate already includes pre-execution state changes
            //   (blockhash/beacon root updates, create2deployer), so we do NOT call
            //   apply_pre_execution_changes() again.
            // - The only pre-execution effect we need is set_state_clear_flag, which configures EVM
            //   empty-account handling (OP Stack chains activate Spurious Dragon at genesis, so
            //   this is always true).
            // - Suffix transactions execute against the warm prestate.
            // - Post-execution (finish()) runs once on the suffix executor, producing correct
            //   results for the full block. For OP Stack post-merge, the
            //   post_block_balance_increments are empty (no block rewards, no ommers, no
            //   withdrawals passed), so finish() only seals execution state.
            let attrs = args.base.clone().into();
            let evm_env =
                self.evm_config.next_evm_env(parent_header, &attrs).map_err(RethError::other)?;
            let execution_ctx = self
                .evm_config
                .context_for_next_block(parent_header, attrs)
                .map_err(RethError::other)?;

            // The cached bundle prestate already includes pre-execution state changes.
            // Only set the state clear flag (Spurious Dragon empty-account handling).
            state.set_state_clear_flag(true);
            let evm = self.evm_config.evm_with_env(&mut state, evm_env);
            let mut executor = self.evm_config.create_executor(evm, execution_ctx.clone());

            for tx in transactions.iter().skip(cached_prefix.cached_tx_count).cloned() {
                let _gas_used = executor.execute_transaction(tx)?;
            }

            let (evm, suffix_execution_result) = executor.finish()?;
            let (db, evm_env) = evm.finish();
            db.merge_transitions(BundleRetention::Reverts);

            let execution_result =
                Self::merge_cached_and_suffix_results(cached_prefix, suffix_execution_result);

            let (hashed_state, state_root) = if args.compute_state_root {
                trace!(target: "flashblocks", "Computing block state root");
                let hashed_state = state_provider.hashed_post_state(&db.bundle_state);
                let (state_root, _) = state_provider
                    .state_root_with_updates(hashed_state.clone())
                    .map_err(RethError::other)?;
                (hashed_state, state_root)
            } else {
                let noop_provider = NoopProvider::default();
                let hashed_state = noop_provider.hashed_post_state(&db.bundle_state);
                let (state_root, _) = noop_provider
                    .state_root_with_updates(hashed_state.clone())
                    .map_err(RethError::other)?;
                (hashed_state, state_root)
            };
            let bundle = db.take_bundle();

            let (block_transactions, senders): (Vec<_>, Vec<_>) =
                transactions.iter().map(|tx| tx.1.clone().into_parts()).unzip();
            let block = self
                .evm_config
                .block_assembler()
                .assemble_block(BlockAssemblerInput::new(
                    evm_env,
                    execution_ctx,
                    parent_header,
                    block_transactions,
                    &execution_result,
                    &bundle,
                    &state_provider,
                    state_root,
                ))
                .map_err(RethError::other)?;
            let block = RecoveredBlock::new_unhashed(block, senders);

            (execution_result, block, hashed_state, bundle)
        } else {
            let mut builder = self
                .evm_config
                .builder_for_next_block(&mut state, parent_header, args.base.clone().into())
                .map_err(RethError::other)?;

            builder.apply_pre_execution_changes()?;

            for tx in transactions {
                let _gas_used = builder.execute_transaction(tx)?;
            }

            let BlockBuilderOutcome { execution_result, block, hashed_state, .. } =
                if args.compute_state_root {
                    trace!(target: "flashblocks", "Computing block state root");
                    builder.finish(&state_provider)?
                } else {
                    builder.finish(NoopProvider::default())?
                };
            let bundle = state.take_bundle();

            (execution_result, block, hashed_state, bundle)
        };

        // Update transaction cache if provided (only in canonical mode)
        if let Some(cache) = tx_cache &&
            is_canonical
        {
            cache.update_with_execution_meta_for_parent(
                args.base.block_number,
                canonical_parent_hash,
                tx_hashes,
                bundle.clone(),
                execution_result.receipts.clone(),
                CachedExecutionMeta {
                    requests: execution_result.requests.clone(),
                    gas_used: execution_result.gas_used,
                    blob_gas_used: execution_result.blob_gas_used,
                },
            );
        }

        let execution_outcome = BlockExecutionOutput { state: bundle, result: execution_result };
        let execution_outcome = Arc::new(execution_outcome);

        // Create pending state for subsequent builds.
        // Use the locally built block hash for both parent matching and speculative
        // execution context to avoid split-hash ambiguity.
        let local_block_hash = block.hash();
        if local_block_hash != args.last_flashblock_hash {
            trace!(
                target: "flashblocks",
                local_block_hash = ?local_block_hash,
                sequencer_block_hash = ?args.last_flashblock_hash,
                block_number = block.number(),
                "Local block hash differs from sequencer-provided hash; speculative chaining will follow local hash"
            );
        }
        let sealed_header = SealedHeader::new(block.header().clone(), local_block_hash);
        let pending_state = PendingBlockState::new(
            local_block_hash,
            block.number(),
            args.base.parent_hash,
            canonical_anchor,
            execution_outcome.clone(),
            request_cache.clone(),
        )
        .with_sealed_header(sealed_header);

        let pending_block = PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock::new(
                block.into(),
                execution_outcome,
                ComputedTrieData::without_trie_input(
                    Arc::new(hashed_state.into_sorted()),
                    Arc::default(),
                ),
            ),
        );
        let pending_flashblock = PendingFlashBlock::new(
            pending_block,
            canonical_anchor,
            args.last_flashblock_index,
            args.last_flashblock_hash,
            args.compute_state_root,
        );

        Ok(Some(BuildResult { pending_flashblock, cached_reads: request_cache, pending_state }))
    }

    fn merge_cached_and_suffix_results(
        cached_prefix: CachedPrefixExecutionResult<N::Receipt>,
        mut suffix_result: BlockExecutionResult<N::Receipt>,
    ) -> BlockExecutionResult<N::Receipt> {
        N::Receipt::add_cumulative_gas_offset(&mut suffix_result.receipts, cached_prefix.gas_used);

        let mut receipts = cached_prefix.receipts;
        receipts.extend(suffix_result.receipts);

        // Use only suffix requests: the suffix executor's finish() produces
        // post-execution requests from the complete block state (cached prestate +
        // suffix changes). The cached prefix requests came from an intermediate
        // state and must not be merged.
        let requests = suffix_result.requests;

        BlockExecutionResult {
            receipts,
            requests,
            gas_used: cached_prefix.gas_used.saturating_add(suffix_result.gas_used),
            blob_gas_used: cached_prefix.blob_gas_used.saturating_add(suffix_result.blob_gas_used),
        }
    }
}

#[inline]
fn is_consistent_speculative_parent_hashes(
    incoming_parent_hash: B256,
    pending_block_hash: B256,
    pending_sealed_hash: B256,
) -> bool {
    incoming_parent_hash == pending_block_hash && pending_block_hash == pending_sealed_hash
}

impl<EvmConfig: Clone, Provider: Clone> Clone for FlashBlockBuilder<EvmConfig, Provider> {
    fn clone(&self) -> Self {
        Self { evm_config: self.evm_config.clone(), provider: self.provider.clone() }
    }
}

#[cfg(test)]
mod tests {
    use super::{BuildArgs, FlashBlockBuilder, is_consistent_speculative_parent_hashes};
    use crate::{TransactionCache, tx_cache::CachedExecutionMeta};
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_network::TxSignerSync;
    use alloy_primitives::{Address, B256, StorageKey, StorageValue, TxKind, U256};
    use alloy_signer_local::PrivateKeySigner;
    use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
    use op_revm::constants::L1_BLOCK_CONTRACT;
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_evm::OpEvmConfig;
    use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
    use reth_primitives_traits::{AlloyBlockHeader, Recovered, SignerRecoverable};
    use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
    use reth_storage_api::BlockReaderIdExt;
    use std::str::FromStr;

    fn signed_transfer_tx(
        signer: &PrivateKeySigner,
        nonce: u64,
        recipient: Address,
    ) -> OpTransactionSigned {
        let mut tx = TxEip1559 {
            chain_id: 10, // OP Mainnet chain id
            nonce,
            gas_limit: 100_000,
            max_priority_fee_per_gas: 1_000_000_000,
            max_fee_per_gas: 2_000_000_000,
            to: TxKind::Call(recipient),
            value: U256::from(1),
            ..Default::default()
        };
        let signature = signer.sign_transaction_sync(&mut tx).expect("signing tx succeeds");
        tx.into_signed(signature).into()
    }

    fn into_encoded_recovered(
        tx: OpTransactionSigned,
        signer: Address,
    ) -> alloy_eips::eip2718::WithEncoded<Recovered<OpTransactionSigned>> {
        let encoded = tx.encoded_2718();
        Recovered::new_unchecked(tx, signer).into_encoded_with(encoded)
    }

    #[test]
    fn speculative_parent_hashes_must_all_match() {
        let h = B256::repeat_byte(0x11);
        assert!(is_consistent_speculative_parent_hashes(h, h, h));
    }

    #[test]
    fn speculative_parent_hashes_reject_any_mismatch() {
        let incoming = B256::repeat_byte(0x11);
        let pending = B256::repeat_byte(0x22);
        let sealed = B256::repeat_byte(0x33);

        assert!(!is_consistent_speculative_parent_hashes(incoming, pending, sealed));
        assert!(!is_consistent_speculative_parent_hashes(incoming, incoming, sealed));
        assert!(!is_consistent_speculative_parent_hashes(incoming, pending, pending));
    }

    #[test]
    fn canonical_build_reuses_cached_prefix_execution() {
        let provider = MockEthProvider::<OpPrimitives>::new()
            .with_chain_spec(OP_MAINNET.clone())
            .with_genesis_block();

        let recipient = Address::repeat_byte(0x22);
        let signer = PrivateKeySigner::random();
        let tx_a = signed_transfer_tx(&signer, 0, recipient);
        let tx_b = signed_transfer_tx(&signer, 1, recipient);
        let tx_c = signed_transfer_tx(&signer, 2, recipient);
        let signer = tx_a.recover_signer().expect("tx signer recovery succeeds");

        provider.add_account(signer, ExtendedAccount::new(0, U256::from(1_000_000_000_000_000u64)));
        provider.add_account(recipient, ExtendedAccount::new(0, U256::ZERO));
        provider.add_account(
            L1_BLOCK_CONTRACT,
            ExtendedAccount::new(1, U256::ZERO).extend_storage([
                (StorageKey::with_last_byte(1), StorageValue::from(1_000_000_000u64)),
                (StorageKey::with_last_byte(5), StorageValue::from(188u64)),
                (StorageKey::with_last_byte(6), StorageValue::from(684_000u64)),
                (
                    StorageKey::with_last_byte(3),
                    StorageValue::from_str(
                        "0x0000000000000000000000000000000000001db0000d27300000000000000005",
                    )
                    .expect("valid L1 fee scalar storage value"),
                ),
            ]),
        );

        let latest = provider
            .latest_header()
            .expect("provider latest header query succeeds")
            .expect("genesis header exists");

        let base = OpFlashblockPayloadBase {
            parent_hash: latest.hash(),
            parent_beacon_block_root: B256::ZERO,
            fee_recipient: Address::ZERO,
            prev_randao: B256::repeat_byte(0x55),
            block_number: latest.number() + 1,
            gas_limit: 30_000_000,
            timestamp: latest.timestamp() + 2,
            extra_data: Default::default(),
            base_fee_per_gas: U256::from(1_000_000_000u64),
        };
        let base_parent_hash = base.parent_hash;

        let tx_a_hash = B256::from(*tx_a.tx_hash());
        let tx_b_hash = B256::from(*tx_b.tx_hash());
        let tx_c_hash = B256::from(*tx_c.tx_hash());

        let tx_a = into_encoded_recovered(tx_a, signer);
        let tx_b = into_encoded_recovered(tx_b, signer);
        let tx_c = into_encoded_recovered(tx_c, signer);

        let evm_config = OpEvmConfig::optimism(OP_MAINNET.clone());
        let builder = FlashBlockBuilder::new(evm_config, provider);
        let mut tx_cache = TransactionCache::<OpPrimitives>::new();

        let first = builder
            .execute(
                BuildArgs {
                    base: base.clone(),
                    transactions: vec![tx_a.clone(), tx_b.clone()],
                    cached_state: None,
                    last_flashblock_index: 0,
                    last_flashblock_hash: B256::repeat_byte(0xA0),
                    compute_state_root: false,
                    pending_parent: None,
                },
                Some(&mut tx_cache),
            )
            .expect("first build succeeds")
            .expect("first build is canonical");

        assert_eq!(first.pending_state.execution_outcome.result.receipts.len(), 2);

        let cached_hashes = vec![tx_a_hash, tx_b_hash];
        let (bundle, receipts, requests, gas_used, blob_gas_used, skip) = tx_cache
            .get_resumable_state_with_execution_meta_for_parent(
                base.block_number,
                base_parent_hash,
                &cached_hashes,
            )
            .expect("cache should contain first build execution state");
        assert_eq!(skip, 2);

        let mut tampered_receipts = receipts.to_vec();
        tampered_receipts[0].as_receipt_mut().cumulative_gas_used =
            tampered_receipts[0].as_receipt().cumulative_gas_used.saturating_add(17);
        let expected_tampered_gas = tampered_receipts[0].as_receipt().cumulative_gas_used;

        tx_cache.update_with_execution_meta_for_parent(
            base.block_number,
            base_parent_hash,
            cached_hashes,
            bundle.clone(),
            tampered_receipts,
            CachedExecutionMeta { requests: requests.clone(), gas_used, blob_gas_used },
        );

        let second_hashes = vec![tx_a_hash, tx_b_hash, tx_c_hash];
        let (_, _, _, _, _, skip) = tx_cache
            .get_resumable_state_with_execution_meta_for_parent(
                base.block_number,
                base_parent_hash,
                &second_hashes,
            )
            .expect("second tx list should extend cached prefix");
        assert_eq!(skip, 2);

        let second = builder
            .execute(
                BuildArgs {
                    base,
                    transactions: vec![tx_a, tx_b, tx_c],
                    cached_state: None,
                    last_flashblock_index: 1,
                    last_flashblock_hash: B256::repeat_byte(0xA1),
                    compute_state_root: false,
                    pending_parent: None,
                },
                Some(&mut tx_cache),
            )
            .expect("second build succeeds")
            .expect("second build is canonical");

        let receipts = &second.pending_state.execution_outcome.result.receipts;
        assert_eq!(receipts.len(), 3);
        assert_eq!(receipts[0].as_receipt().cumulative_gas_used, expected_tampered_gas);
        assert!(
            receipts[2].as_receipt().cumulative_gas_used >
                receipts[1].as_receipt().cumulative_gas_used
        );
    }
}
