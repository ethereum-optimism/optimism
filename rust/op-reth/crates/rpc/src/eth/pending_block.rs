//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError, eth::memory_overlay::MemoryOverlayStateProvider};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use reth_errors::RethError;
use reth_evm::ConfigureEvm;
use reth_optimism_flashblocks::PendingFlashBlock;
use reth_primitives_traits::NodePrimitives;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert, RpcNodeCore, RpcNodeCoreExt,
    helpers::{LoadPendingBlock, SpawnBlocking, pending_block::PendingEnvBuilder},
};
use reth_rpc_eth_types::{
    EthApiError, PendingBlock, PendingBlockEnv, PendingBlockEnvOrigin, block::BlockAndReceipts,
    builder::config::PendingBlockKind, error::FromEthApiError,
};
use reth_storage_api::{
    BlockReader, BlockReaderIdExt, StateProviderBox, StateProviderFactory, errors::ProviderResult,
};

#[inline]
const fn pending_state_history_lookup_hash<N: NodePrimitives>(
    pending_block: &PendingFlashBlock<N>,
) -> B256 {
    pending_block.canonical_anchor_hash
}

/// Returns the pending state: the tip's bundle on its canonical anchor's state.
///
/// The sequencer streams flashblocks ahead of block import, so the flashblocks worker can build
/// past the canonical head. With canonical head N-1, it builds pending N on N-1, then speculative
/// N+1 on N, and so on, all anchored at N-1. A tip on the anchor, such as N, opens through reth's
/// `state_with_block_appended`. A speculative tip opens through the vendored
/// [`MemoryOverlayStateProvider`]: its bundle holds every write since the anchor, so account,
/// storage and code reads need only the tip. Block hash lookups are the exception: they see only
/// the tip, not the pending blocks before it.
///
/// reth's `state_with_block_appended(parent_hash, block)` can't open a speculative tip such as N+1:
///
/// - With N, it fails with `BlockHashNotFound`. N is in neither the database nor the overlay
///   manager, which holds only blocks the engine executed.
/// - With N-1, it panics on the `debug_assert` in `OverlayBuilder::with_appended_block` that the
///   hash is the header's parent.
///
/// Appending the whole chain instead needs it at the RPC
/// (<https://github.com/ethereum-optimism/optimism/issues/23998>) and reth access we lack:
/// `state_with_block_appended` takes one block, and `ProviderFactory::overlay_manager` is
/// crate-private.
fn pending_flashblock_state<P, N>(
    provider: &P,
    pending_block: PendingFlashBlock<N>,
) -> ProviderResult<StateProviderBox>
where
    N: NodePrimitives,
    P: StateProviderFactory<Primitives = N>,
{
    let anchor = pending_state_history_lookup_hash(&pending_block);
    let speculative = pending_block.pending.parent_hash() != anchor;
    let tip = pending_block.pending.executed_block;
    if !speculative {
        return provider.state_with_block_appended(anchor, tip);
    }
    let anchor_historical = provider.history_by_block_hash(anchor)?;
    Ok(MemoryOverlayStateProvider::new(anchor_historical, vec![tip]).boxed())
}

impl<N, Rpc> LoadPendingBlock for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    #[inline]
    fn pending_block(&self) -> &tokio::sync::Mutex<Option<PendingBlock<N::Primitives>>> {
        self.inner.eth_api.pending_block()
    }

    #[inline]
    fn pending_env_builder(&self) -> &dyn PendingEnvBuilder<Self::Evm> {
        self.inner.eth_api.pending_env_builder()
    }

    #[inline]
    fn pending_block_kind(&self) -> PendingBlockKind {
        self.inner.eth_api.pending_block_kind()
    }

    /// UPSTREAM-MIRROR(override): reth@op-v2.7.0-1
    /// `reth_rpc_eth_api::helpers::pending_block::LoadPendingBlock::pending_block_env_and_cfg`
    ///
    /// Uses the served flashblock as the actual pending block, so calls at `pending` run on it.
    fn pending_block_env_and_cfg(&self) -> Result<PendingBlockEnv<Self::Evm>, Self::Error> {
        let actual_pending = match self.current_pending_flashblock() {
            Some(pending) => Some(pending.pending.into_block_and_receipts()),
            None => self
                .provider()
                .pending_block_and_receipts()
                .map_err(Self::Error::from_eth_err)?
                .map(|pending| {
                    let (block, output) = pending.into_parts();
                    BlockAndReceipts { block, receipts: output.into() }
                }),
        };

        if let Some(BlockAndReceipts { block, receipts }) = actual_pending {
            let evm_env = self
                .evm_config()
                .evm_env(block.header())
                .map_err(RethError::other)
                .map_err(Self::Error::from_eth_err)?;

            return Ok(PendingBlockEnv::new(
                evm_env,
                PendingBlockEnvOrigin::ActualPending(block, receipts),
            ));
        }

        let latest = self
            .provider()
            .latest_header()
            .map_err(Self::Error::from_eth_err)?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;

        let evm_env = self
            .evm_config()
            .next_evm_env(&latest, &self.next_env_attributes(&latest)?)
            .map_err(RethError::other)
            .map_err(Self::Error::from_eth_err)?;

        Ok(PendingBlockEnv::new(evm_env, PendingBlockEnvOrigin::DerivedFromLatest(latest)))
    }

    /// UPSTREAM-MIRROR(override): reth@op-v2.7.0-1
    /// `reth_rpc_eth_api::helpers::pending_block::LoadPendingBlock::local_pending_state`
    ///
    /// Serves the flashblock pending state instead of the pool-built one. Keep upstream's blocking
    /// discipline: callers await this on the async runtime, so the reads that open the state go
    /// through `spawn_blocking_io`, as the default does.
    ///
    /// Returns a [`StateProviderBox`] on a mem-pool built pending block overlaying latest.
    async fn local_pending_state(&self) -> Result<Option<StateProviderBox>, Self::Error>
    where
        Self: SpawnBlocking,
    {
        let Ok(Some(pending_block)) = self.pending_flashblock().await else {
            return Ok(None);
        };
        // Callers resolve pending state before taking a blocking thread, so the reads that open it
        // take one here, as reth's default does.
        self.spawn_blocking_io(move |this| {
            pending_flashblock_state(this.provider(), pending_block)
                .map(Some)
                .map_err(Self::Error::from_eth_err)
        })
        .await
    }

    /// Returns the locally built pending block
    async fn local_pending_block(
        &self,
    ) -> Result<Option<BlockAndReceipts<Self::Primitives>>, Self::Error> {
        if let Ok(Some(pending)) = self.pending_flashblock().await {
            return Ok(Some(pending.pending.into_block_and_receipts()));
        }

        // See: <https://github.com/ethereum-optimism/op-geth/blob/f2e69450c6eec9c35d56af91389a1c47737206ca/miner/worker.go#L367-L375>
        let latest = self
            .provider()
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;

        let latest = self
            .cache()
            .get_block_and_receipts(latest.hash())
            .await
            .map_err(Self::Error::from_eth_err)?
            .map(|(block, receipts)| BlockAndReceipts::new(block, receipts));
        Ok(latest)
    }
}

#[cfg(test)]
mod tests {
    use super::{pending_flashblock_state, pending_state_history_lookup_hash};
    use alloy_consensus::Header;
    use alloy_primitives::{Address, B256, U256};
    use reth_chain_state::ExecutedBlock;
    use reth_chainspec::MAINNET;
    use reth_db_common::init::init_genesis;
    use reth_ethereum_primitives::{Block, EthPrimitives};
    use reth_execution_types::BlockExecutionOutput;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_primitives_traits::{RecoveredBlock, SealedBlock};
    use reth_provider::{providers::BlockchainProvider, test_utils::create_test_provider_factory};
    use reth_rpc_eth_types::PendingBlock;
    use reth_storage_api::{AccountReader, StateProvider, StateProviderFactory};
    use reth_trie::{ComputedTrieData, HashedPostState, KeccakKeyHasher};
    use revm::{
        database::{AccountStatus, BundleState},
        state::AccountInfo,
    };
    use rstest::rstest;
    use std::{sync::Arc, time::Instant};

    #[test]
    fn pending_state_prefers_canonical_anchor_over_parent_hash() {
        let pending = PendingBlock::<OpPrimitives>::with_executed_block(
            Instant::now(),
            ExecutedBlock::<OpPrimitives>::default(),
        );
        let parent_hash = pending.parent_hash();
        let canonical_anchor_hash = B256::from([0x11; 32]);
        assert_ne!(canonical_anchor_hash, parent_hash);

        let pending_flashblock = PendingFlashBlock::<OpPrimitives>::new(
            pending,
            canonical_anchor_hash,
            0,
            B256::ZERO,
            false,
        );

        assert_eq!(pending_state_history_lookup_hash(&pending_flashblock), canonical_anchor_hash);
    }

    /// Builds `blocks` pending blocks on the canonical genesis and reads the tip's state the way
    /// `local_pending_state` does: through reth's overlay for a tip on the anchor, through the
    /// vendored one for a speculative tip. Block `n` sets the balance of account `n` to `n`, and
    /// the first block also writes a storage slot and destroys a genesis account. As in the
    /// flashblocks worker, a speculative tip's bundle holds every write since the anchor.
    ///
    /// These are spot checks that the speculative gap is bridged, not a parity suite against reth's
    /// `OverlayStateProvider`: the vendored overlay is meant to go away (see its module doc).
    #[rstest]
    #[case::canonical(1)]
    #[case::speculative(2)]
    fn pending_flashblock_state_includes_every_pending_block(#[case] blocks: u64) {
        let factory = create_test_provider_factory();
        let anchor = init_genesis(&factory).unwrap();
        let provider = BlockchainProvider::new(factory).unwrap();

        let account = |number: u64| Address::with_last_byte(number as u8);
        let destroyed = *MAINNET.genesis.alloc.keys().next().expect("mainnet genesis has accounts");
        assert!(provider.latest().unwrap().basic_account(&destroyed).unwrap().is_some());
        let (slot, value) = (U256::from(1), U256::from(42));

        let mut bundle = BundleState::builder(1..=blocks)
            .state_storage(account(1), std::iter::once((slot, (U256::ZERO, value))).collect())
            .state_original_account_info(destroyed, AccountInfo::default());
        let mut parent_hash = anchor;
        let mut tip = None;
        for number in 1..=blocks {
            bundle = bundle.state_present_account_info(
                account(number),
                AccountInfo { balance: U256::from(number), ..Default::default() },
            );
            let header = Header { number, parent_hash, ..Default::default() };
            let block = SealedBlock::seal_slow(Block { header, body: Default::default() });
            parent_hash = block.hash();
            tip = Some(block);
        }
        let mut bundle = bundle.build();
        bundle.state.get_mut(&destroyed).expect("in the bundle").status = AccountStatus::Destroyed;
        let hashed_state = HashedPostState::from_bundle_state::<KeccakKeyHasher>(bundle.state());
        let executed = ExecutedBlock::<EthPrimitives>::new(
            Arc::new(RecoveredBlock::new_sealed(tip.expect("at least one block"), Vec::new())),
            Arc::new(BlockExecutionOutput { state: bundle, ..Default::default() }),
            ComputedTrieData::new(Arc::new(hashed_state.into_sorted()), Arc::default()),
        );
        let pending = PendingFlashBlock::new(
            PendingBlock::with_executed_block(Instant::now(), executed),
            anchor,
            0,
            B256::ZERO,
            false,
        );

        let state = pending_flashblock_state(&provider, pending).unwrap();
        for number in 1..=blocks {
            let balance = state.basic_account(&account(number)).unwrap().map(|a| a.balance);
            assert_eq!(balance, Some(U256::from(number)), "account {number}");
        }
        assert_eq!(state.storage(account(1), B256::from(slot)).unwrap(), Some(value));
        assert_eq!(state.basic_account(&destroyed).unwrap(), None, "destroyed account");
    }

    mod pending_call {
        use crate::{
            OpEthApi,
            eth::{receipt::OpReceiptConverter, transaction::OpTxInfoMapper},
        };
        use alloy_consensus::{Block, Header};
        use alloy_eips::BlockId;
        use alloy_genesis::{Genesis, GenesisAccount};
        use alloy_primitives::{Address, B256, Bytes, U256, address, bytes};
        use alloy_rpc_types_eth::{TransactionRequest, state::EvmOverrides};
        use op_alloy_network::Optimism;
        use reth_chain_state::ExecutedBlock;
        use reth_chainspec::{Chain, ChainSpecProvider};
        use reth_db::{DatabaseEnv, test_utils::TempDatabase};
        use reth_db_common::init::init_genesis;
        use reth_execution_types::BlockExecutionOutput;
        use reth_network_api::noop::NoopNetwork;
        use reth_node_api::{AnyNodeTypes, NodeTypesWithDBAdapter};
        use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
        use reth_optimism_evm::{OpEvmConfig, tx::OpTxEnvConverter};
        use reth_optimism_flashblocks::{FlashblocksListeners, PendingFlashBlock};
        use reth_optimism_payload_builder::OpPayloadTypes;
        use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
        use reth_optimism_txpool::OpPooledTransaction;
        use reth_primitives_traits::{RecoveredBlock, SealedBlock, SealedHeader};
        use reth_provider::{
            providers::BlockchainProvider, test_utils::create_test_provider_factory_with_node_types,
        };
        use reth_rpc::EthApiBuilder;
        use reth_rpc_eth_api::{RpcConverter, helpers::EthCall, node::RpcNodeCoreAdapter};
        use reth_rpc_eth_types::PendingBlock;
        use reth_storage_api::{BlockReaderIdExt, EmptyBodyStorage};
        use reth_transaction_pool::noop::NoopTransactionPool;
        use reth_trie::{ComputedTrieData, HashedPostState, KeccakKeyHasher};
        use revm::{database::BundleState, state::AccountInfo};
        use std::{sync::Arc, time::Instant};
        use tokio::sync::{broadcast, watch};

        const TARGET: Address = address!("0x00000000000000000000000000000000000000bb");
        /// Returns `BALANCE(TARGET)`, `TIMESTAMP`, `NUMBER` and `PREVRANDAO`.
        const ENV_PROBE: Address = address!("0x00000000000000000000000000000000000000cc");
        const ENV_PROBE_CODE: Bytes = bytes!(
            "7300000000000000000000000000000000000000bb3160005242602052436040524460605260806000f3"
        );
        /// Reverts unless `BALANCE(TARGET) == 2`.
        const BALANCE_GUARD: Address = address!("0x00000000000000000000000000000000000000dd");
        const BALANCE_GUARD_CODE: Bytes =
            bytes!("7300000000000000000000000000000000000000bb31600214602057600080fd5b00");

        type TestNode = AnyNodeTypes<
            OpPrimitives,
            OpChainSpec,
            EmptyBodyStorage<OpTransactionSigned, Header>,
            OpPayloadTypes,
        >;
        // Database-backed: the pending state opens through `state_with_block_appended`, which
        // `MockEthProvider` does not support.
        type Provider =
            BlockchainProvider<NodeTypesWithDBAdapter<TestNode, Arc<TempDatabase<DatabaseEnv>>>>;
        type TestEthApi = OpEthApi<
            RpcNodeCoreAdapter<
                Provider,
                NoopTransactionPool<OpPooledTransaction>,
                NoopNetwork,
                OpEvmConfig,
            >,
            RpcConverter<
                Optimism,
                OpEvmConfig,
                OpReceiptConverter<Provider>,
                (),
                OpTxInfoMapper<Provider>,
                (),
                (),
                OpTxEnvConverter,
            >,
        >;

        fn provider() -> Provider {
            let genesis = Genesis::default().with_gas_limit(30_000_000).extend_accounts([
                (TARGET, GenesisAccount::default().with_balance(U256::from(1))),
                (ENV_PROBE, GenesisAccount::default().with_code(Some(ENV_PROBE_CODE))),
                (BALANCE_GUARD, GenesisAccount::default().with_code(Some(BALANCE_GUARD_CODE))),
            ]);
            let chain_spec = OpChainSpecBuilder::default()
                .chain(Chain::dev())
                .genesis(genesis)
                .regolith_activated()
                .build();
            let factory =
                create_test_provider_factory_with_node_types::<TestNode>(Arc::new(chain_spec));
            init_genesis(&factory).unwrap();
            BlockchainProvider::new(factory).unwrap()
        }

        /// A flashblock on `latest` that raises the balance of [`TARGET`] to 2.
        fn pending_flashblock(latest: &SealedHeader) -> PendingFlashBlock<OpPrimitives> {
            let header = Header {
                parent_hash: latest.hash(),
                number: latest.number + 1,
                timestamp: latest.timestamp + 2,
                mix_hash: B256::repeat_byte(0x55),
                gas_limit: latest.gas_limit,
                base_fee_per_gas: latest.base_fee_per_gas,
                ..Default::default()
            };
            let state = BundleState::builder(header.number..=header.number)
                .state_original_account_info(
                    TARGET,
                    AccountInfo { balance: U256::from(1), ..Default::default() },
                )
                .state_present_account_info(
                    TARGET,
                    AccountInfo { balance: U256::from(2), ..Default::default() },
                )
                .build();
            let hashed_state = HashedPostState::from_bundle_state::<KeccakKeyHasher>(state.state());
            let executed = ExecutedBlock::<OpPrimitives>::new(
                Arc::new(RecoveredBlock::new_sealed(
                    SealedBlock::seal_slow(Block::new(header, Default::default())),
                    Vec::new(),
                )),
                Arc::new(BlockExecutionOutput { state, ..Default::default() }),
                ComputedTrieData::new(Arc::new(hashed_state.into_sorted()), Arc::default()),
            );

            PendingFlashBlock::new(
                PendingBlock::with_executed_block(Instant::now(), executed),
                latest.hash(),
                0,
                B256::ZERO,
                false,
            )
        }

        fn eth_api(
            provider: Provider,
            pending: Option<PendingFlashBlock<OpPrimitives>>,
        ) -> TestEthApi {
            let flashblocks = pending.map(|pending| {
                FlashblocksListeners::new(
                    watch::channel(Some(pending)).1,
                    broadcast::channel(1).0,
                    watch::channel(None).1,
                    broadcast::channel(1).0,
                )
            });
            let converter = RpcConverter::new(OpReceiptConverter::new(provider.clone()))
                .with_mapper(OpTxInfoMapper::new(provider.clone()))
                .with_tx_env_converter(OpTxEnvConverter);
            let evm_config = OpEvmConfig::optimism(provider.chain_spec());
            let eth_api = EthApiBuilder::new(
                provider,
                NoopTransactionPool::new(),
                NoopNetwork::default(),
                evm_config,
            )
            .with_rpc_converter(converter)
            .build_inner();

            OpEthApi::new(eth_api, None, U256::ZERO, flashblocks, false)
        }

        async fn probe_pending(eth_api: &TestEthApi) -> [U256; 4] {
            let out = EthCall::call(
                eth_api,
                TransactionRequest::default().to(ENV_PROBE).into(),
                Some(BlockId::pending()),
                EvmOverrides::default(),
            )
            .await
            .unwrap();
            std::array::from_fn(|i| U256::from_be_slice(&out[i * 32..(i + 1) * 32]))
        }

        #[tokio::test]
        async fn calls_at_pending_run_on_the_pending_flashblock() {
            let provider = provider();
            let latest = provider.latest_header().unwrap().unwrap();
            let pending = pending_flashblock(&latest);
            let header = pending.block().clone_sealed_header();
            let eth_api = eth_api(provider, Some(pending));

            assert_eq!(
                probe_pending(&eth_api).await,
                [
                    U256::from(2),
                    U256::from(header.timestamp),
                    U256::from(header.number),
                    U256::from_be_bytes(header.mix_hash.0),
                ]
            );
            EthCall::estimate_gas_at(
                &eth_api,
                TransactionRequest::default().to(BALANCE_GUARD).into(),
                BlockId::pending(),
                EvmOverrides::default(),
            )
            .await
            .unwrap();
        }

        #[tokio::test]
        async fn calls_at_pending_without_flashblocks_run_on_latest() {
            let provider = provider();
            let latest = provider.latest_header().unwrap().unwrap();
            let eth_api = eth_api(provider, None);

            let [balance, _, number, _] = probe_pending(&eth_api).await;
            assert_eq!(balance, U256::from(1));
            assert_eq!(number, U256::from(latest.number + 1));
        }
    }
}
