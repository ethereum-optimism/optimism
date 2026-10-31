//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use reth_chain_state::BlockState;
use reth_errors::RethError;
use reth_evm::ConfigureEvm;
use reth_optimism_flashblocks::PendingFlashBlock;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert, RpcNodeCore, RpcNodeCoreExt,
    helpers::{LoadPendingBlock, SpawnBlocking, pending_block::PendingEnvBuilder},
};
use reth_rpc_eth_types::{
    EthApiError, PendingBlock, PendingBlockEnv, PendingBlockEnvOrigin, block::BlockAndReceipts,
    builder::config::PendingBlockKind, error::FromEthApiError,
};
use reth_storage_api::{BlockReader, BlockReaderIdExt, StateProviderBox, StateProviderFactory};
use std::sync::Arc;

#[inline]
const fn pending_state_history_lookup_hash<N: reth_primitives_traits::NodePrimitives>(
    pending_block: &PendingFlashBlock<N>,
) -> B256 {
    pending_block.canonical_anchor_hash
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

    /// UPSTREAM-MIRROR(override): reth@op-v2.6.0-1
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
                .map(|(block, receipts)| BlockAndReceipts {
                    block: Arc::new(block),
                    receipts: Arc::new(receipts),
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

    /// Returns a [`StateProviderBox`] on a mem-pool built pending block overlaying latest.
    async fn local_pending_state(&self) -> Result<Option<StateProviderBox>, Self::Error>
    where
        Self: SpawnBlocking,
    {
        let Ok(Some(pending_block)) = self.pending_flashblock().await else {
            return Ok(None);
        };
        let canonical_anchor_hash = pending_state_history_lookup_hash(&pending_block);
        let state = BlockState::from(pending_block.pending);

        let anchor_historical = self
            .provider()
            .history_by_block_hash(canonical_anchor_hash)
            .map_err(Self::Error::from_eth_err)?;

        Ok(Some(Box::new(state.state_provider(anchor_historical)) as StateProviderBox))
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
            .map(|(block, receipts)| BlockAndReceipts { block, receipts });
        Ok(latest)
    }
}

#[cfg(test)]
mod tests {
    use super::pending_state_history_lookup_hash;
    use alloy_primitives::B256;
    use reth_chain_state::ExecutedBlock;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_rpc_eth_types::PendingBlock;
    use std::time::Instant;

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

    mod pending_call {
        use crate::{
            OpEthApi,
            eth::{receipt::OpReceiptConverter, transaction::OpTxInfoMapper},
        };
        use alloy_consensus::{Block, Header};
        use alloy_eips::BlockId;
        use alloy_primitives::{Address, B256, Bytes, U256, address, bytes};
        use alloy_rpc_types_eth::{TransactionRequest, state::EvmOverrides};
        use op_alloy_network::Optimism;
        use reth_chain_state::ExecutedBlock;
        use reth_network_api::noop::NoopNetwork;
        use reth_optimism_chainspec::{OP_MAINNET, OpChainSpec};
        use reth_optimism_evm::{OpEvmConfig, tx::OpTxEnvConverter};
        use reth_optimism_flashblocks::{FlashblocksListeners, PendingFlashBlock};
        use reth_optimism_primitives::OpPrimitives;
        use reth_optimism_txpool::OpPooledTransaction;
        use reth_primitives_traits::{RecoveredBlock, SealedHeader};
        use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
        use reth_rpc::EthApiBuilder;
        use reth_rpc_eth_api::{RpcConverter, helpers::EthCall, node::RpcNodeCoreAdapter};
        use reth_rpc_eth_types::PendingBlock;
        use reth_storage_api::BlockReaderIdExt;
        use reth_transaction_pool::noop::NoopTransactionPool;
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

        type Provider = MockEthProvider<OpPrimitives, Arc<OpChainSpec>>;
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
            let provider = MockEthProvider::<OpPrimitives>::new()
                .with_chain_spec(OP_MAINNET.clone())
                .with_genesis_block();
            provider.add_account(TARGET, ExtendedAccount::new(0, U256::from(1)));
            provider.add_account(
                ENV_PROBE,
                ExtendedAccount::new(0, U256::ZERO).with_bytecode(ENV_PROBE_CODE),
            );
            provider.add_account(
                BALANCE_GUARD,
                ExtendedAccount::new(0, U256::ZERO).with_bytecode(BALANCE_GUARD_CODE),
            );
            provider
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
            let state = BundleState::builder(0..=0)
                .state_original_account_info(
                    TARGET,
                    AccountInfo { balance: U256::from(1), ..Default::default() },
                )
                .state_present_account_info(
                    TARGET,
                    AccountInfo { balance: U256::from(2), ..Default::default() },
                )
                .build();
            let executed = ExecutedBlock::<OpPrimitives>::default();
            let mut execution_output = (*executed.execution_output).clone();
            execution_output.state = state;
            let executed = ExecutedBlock {
                recovered_block: Arc::new(RecoveredBlock::new_unhashed(
                    Block::new(header, Default::default()),
                    Vec::new(),
                )),
                execution_output: Arc::new(execution_output),
                ..executed
            };

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
            let eth_api = EthApiBuilder::new(
                provider,
                NoopTransactionPool::new(),
                NoopNetwork::default(),
                OpEvmConfig::optimism(OP_MAINNET.clone()),
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
