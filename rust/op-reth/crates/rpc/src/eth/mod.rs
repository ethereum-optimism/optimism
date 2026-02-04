//! OP-Reth `eth_` endpoint implementation.

pub mod ext;
pub mod receipt;
pub mod transaction;

mod block;
mod call;
mod pending_block;

use crate::{
    OpEthApiError, SequencerClient,
    eth::{receipt::OpReceiptConverter, transaction::OpTxInfoMapper},
};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U256};
use alloy_rpc_types_eth::{Filter, Log};
use alloy_transport_http::reqwest::Url;
use eyre::WrapErr;
use futures::StreamExt;
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
pub use receipt::{OpReceiptBuilder, OpReceiptFieldsBuilder};
use reth_chainspec::{EthereumHardforks, Hardforks};
use reth_evm::ConfigureEvm;
use reth_node_api::{FullNodeComponents, FullNodeTypes, HeaderTy, NodeTypes};
use reth_node_builder::rpc::{EthApiBuilder, EthApiCtx};
use reth_optimism_flashblocks::{
    FlashBlockBuildInfo, FlashBlockCompleteSequence, FlashBlockCompleteSequenceRx,
    FlashBlockConsensusClient, FlashBlockRx, FlashBlockService, FlashblocksListeners,
    PendingBlockRx, PendingFlashBlock, WsFlashBlockStream,
};
use reth_rpc::eth::core::EthApiInner;
use reth_rpc_eth_api::{
    EthApiTypes, FromEvmError, FullEthApiServer, RpcConvert, RpcConverter, RpcNodeCore,
    RpcNodeCoreExt, RpcTypes,
    helpers::{
        EthApiSpec, EthFees, EthState, LoadFee, LoadPendingBlock, LoadState, SpawnBlocking, Trace,
        pending_block::BuildPendingEnv,
    },
};
use reth_rpc_eth_types::{
    EthStateCache, FeeHistoryCache, GasPriceOracle, PendingBlock,
    logs_utils::matching_block_logs_with_tx_hashes,
};
use reth_storage_api::{BlockReaderIdExt, ProviderHeader};
use reth_tasks::{
    TaskSpawner,
    pool::{BlockingTaskGuard, BlockingTaskPool},
};
use std::{
    fmt::{self, Formatter},
    marker::PhantomData,
    sync::Arc,
    time::Duration,
};
use tokio::{sync::watch, time};
use tokio_stream::{Stream, wrappers::BroadcastStream};
use tracing::info;

/// Maximum duration to wait for a fresh flashblock when one is being built.
const MAX_FLASHBLOCK_WAIT_DURATION: Duration = Duration::from_millis(50);

/// Adapter for [`EthApiInner`], which holds all the data required to serve core `eth_` API.
pub type EthApiNodeBackend<N, Rpc> = EthApiInner<N, Rpc>;

/// OP-Reth `Eth` API implementation.
///
/// This type provides the functionality for handling `eth_` related requests.
///
/// This wraps a default `Eth` implementation, and provides additional functionality where the
/// optimism spec deviates from the default (ethereum) spec, e.g. transaction forwarding to the
/// sequencer, receipts, additional RPC fields for transaction receipts.
///
/// This type implements the [`FullEthApi`](reth_rpc_eth_api::helpers::FullEthApi) by implemented
/// all the `Eth` helper traits and prerequisite traits.
pub struct OpEthApi<N: RpcNodeCore, Rpc: RpcConvert> {
    /// Gateway to node's core components.
    inner: Arc<OpEthApiInner<N, Rpc>>,
}

impl<N: RpcNodeCore, Rpc: RpcConvert> Clone for OpEthApi<N, Rpc> {
    fn clone(&self) -> Self {
        Self { inner: self.inner.clone() }
    }
}

impl<N: RpcNodeCore, Rpc: RpcConvert> OpEthApi<N, Rpc> {
    /// Creates a new `OpEthApi`.
    pub fn new(
        eth_api: EthApiNodeBackend<N, Rpc>,
        sequencer_client: Option<SequencerClient>,
        min_suggested_priority_fee: U256,
        flashblocks: Option<FlashblocksListeners<N::Primitives>>,
    ) -> Self {
        let inner = Arc::new(OpEthApiInner {
            eth_api,
            sequencer_client,
            min_suggested_priority_fee,
            flashblocks,
        });
        Self { inner }
    }

    /// Build a [`OpEthApi`] using [`OpEthApiBuilder`].
    pub const fn builder() -> OpEthApiBuilder<Rpc> {
        OpEthApiBuilder::new()
    }

    /// Returns a reference to the [`EthApiNodeBackend`].
    pub fn eth_api(&self) -> &EthApiNodeBackend<N, Rpc> {
        self.inner.eth_api()
    }
    /// Returns the configured sequencer client, if any.
    pub fn sequencer_client(&self) -> Option<&SequencerClient> {
        self.inner.sequencer_client()
    }

    /// Returns a cloned pending block receiver, if any.
    pub fn pending_block_rx(&self) -> Option<PendingBlockRx<N::Primitives>> {
        self.inner.flashblocks.as_ref().map(|f| f.pending_block_rx.clone())
    }

    /// Returns a new subscription to received flashblocks.
    pub fn subscribe_received_flashblocks(&self) -> Option<FlashBlockRx> {
        self.inner.flashblocks.as_ref().map(|f| f.received_flashblocks.subscribe())
    }

    /// Returns a new subscription to flashblock sequences.
    pub fn subscribe_flashblock_sequence(&self) -> Option<FlashBlockCompleteSequenceRx> {
        self.inner.flashblocks.as_ref().map(|f| f.flashblocks_sequence.subscribe())
    }

    /// Returns a stream of matching flashblock receipts, if any.
    ///
    /// This will yield all new matching receipts received from _new_ flashblocks.
    pub fn flashblock_receipts_stream(
        &self,
        filter: Filter,
    ) -> Option<impl Stream<Item = Log> + Send + Unpin> {
        self.subscribe_received_flashblocks().map(|rx| {
            BroadcastStream::new(rx)
                .scan(
                    None::<(u64, u64)>, // state buffers base block number and timestamp
                    move |state, result| {
                        let fb = match result.ok() {
                            Some(fb) => fb,
                            None => return futures::future::ready(None),
                        };

                        // Update state from base flashblock for block level meta data.
                        if let Some(base) = &fb.base {
                            *state = Some((base.block_number, base.timestamp));
                        }

                        let Some((block_number, timestamp)) = *state else {
                            // we haven't received a new flashblock sequence yet, so we can skip
                            // until we receive the first index 0 (base)
                            return futures::future::ready(Some(Vec::new()));
                        };

                        let receipts =
                            fb.metadata.receipts.iter().map(|(tx, receipt)| (*tx, receipt));

                        let all_logs = matching_block_logs_with_tx_hashes(
                            &filter,
                            BlockNumHash::new(block_number, fb.diff.block_hash),
                            timestamp,
                            receipts,
                            false,
                        );

                        futures::future::ready(Some(all_logs))
                    },
                )
                .flat_map(futures::stream::iter)
        })
    }

    /// Returns information about the flashblock currently being built, if any.
    fn flashblock_build_info(&self) -> Option<FlashBlockBuildInfo> {
        self.inner.flashblocks.as_ref().and_then(|f| *f.in_progress_rx.borrow())
    }

    /// Extracts pending block if it matches the expected parent hash.
    fn extract_matching_block(
        &self,
        block: Option<&PendingFlashBlock<N::Primitives>>,
        parent_hash: B256,
    ) -> Option<PendingBlock<N::Primitives>> {
        block.filter(|b| b.block().parent_hash() == parent_hash).map(|b| b.pending.clone())
    }

    /// Awaits a fresh flashblock if one is being built, otherwise returns current.
    async fn flashblock(
        &self,
        parent_hash: B256,
    ) -> eyre::Result<Option<PendingBlock<N::Primitives>>> {
        let Some(rx) = self.inner.flashblocks.as_ref().map(|f| &f.pending_block_rx) else {
            return Ok(None);
        };

        // Check if a flashblock is being built
        if let Some(build_info) = self.flashblock_build_info() {
            let current_index = rx.borrow().as_ref().map(|b| b.last_flashblock_index);

            // Check if this is the first flashblock or the next consecutive index
            let is_next_index = current_index.is_none_or(|idx| build_info.index == idx + 1);

            // Wait only for relevant flashblocks: matching parent and next in sequence
            if build_info.parent_hash == parent_hash && is_next_index {
                let mut rx_clone = rx.clone();
                // Wait up to MAX_FLASHBLOCK_WAIT_DURATION for a new flashblock to arrive
                let _ = time::timeout(MAX_FLASHBLOCK_WAIT_DURATION, rx_clone.changed()).await;
            }
        }

        // Fall back to current block
        Ok(self.extract_matching_block(rx.borrow().as_ref(), parent_hash))
    }

    /// Returns a [`PendingBlock`] that is built out of flashblocks.
    ///
    /// If flashblocks receiver is not set, then it always returns `None`.
    ///
    /// It may wait up to 50ms for a fresh flashblock if one is currently being built.
    pub async fn pending_flashblock(&self) -> eyre::Result<Option<PendingBlock<N::Primitives>>>
    where
        OpEthApiError: FromEvmError<N::Evm>,
        Rpc: RpcConvert<Primitives = N::Primitives>,
    {
        let Some(latest) = self.provider().latest_header()? else {
            return Ok(None);
        };

        self.flashblock(latest.hash()).await
    }
}

impl<N, Rpc> EthApiTypes for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    type Error = OpEthApiError;
    type NetworkTypes = Rpc::Network;
    type RpcConvert = Rpc;

    fn converter(&self) -> &Self::RpcConvert {
        self.inner.eth_api.converter()
    }
}

impl<N, Rpc> RpcNodeCore for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives>,
{
    type Primitives = N::Primitives;
    type Provider = N::Provider;
    type Pool = N::Pool;
    type Evm = N::Evm;
    type Network = N::Network;

    #[inline]
    fn pool(&self) -> &Self::Pool {
        self.inner.eth_api.pool()
    }

    #[inline]
    fn evm_config(&self) -> &Self::Evm {
        self.inner.eth_api.evm_config()
    }

    #[inline]
    fn network(&self) -> &Self::Network {
        self.inner.eth_api.network()
    }

    #[inline]
    fn provider(&self) -> &Self::Provider {
        self.inner.eth_api.provider()
    }
}

impl<N, Rpc> RpcNodeCoreExt for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives>,
{
    #[inline]
    fn cache(&self) -> &EthStateCache<N::Primitives> {
        self.inner.eth_api.cache()
    }
}

impl<N, Rpc> EthApiSpec for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    #[inline]
    fn starting_block(&self) -> U256 {
        self.inner.eth_api.starting_block()
    }
}

impl<N, Rpc> SpawnBlocking for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    #[inline]
    fn io_task_spawner(&self) -> impl TaskSpawner {
        self.inner.eth_api.task_spawner()
    }

    #[inline]
    fn tracing_task_pool(&self) -> &BlockingTaskPool {
        self.inner.eth_api.blocking_task_pool()
    }

    #[inline]
    fn tracing_task_guard(&self) -> &BlockingTaskGuard {
        self.inner.eth_api.blocking_task_guard()
    }

    #[inline]
    fn blocking_io_task_guard(&self) -> &Arc<tokio::sync::Semaphore> {
        self.inner.eth_api.blocking_io_request_semaphore()
    }
}

impl<N, Rpc> LoadFee for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    #[inline]
    fn gas_oracle(&self) -> &GasPriceOracle<Self::Provider> {
        self.inner.eth_api.gas_oracle()
    }

    #[inline]
    fn fee_history_cache(&self) -> &FeeHistoryCache<ProviderHeader<N::Provider>> {
        self.inner.eth_api.fee_history_cache()
    }

    async fn suggested_priority_fee(&self) -> Result<U256, Self::Error> {
        self.inner
            .eth_api
            .gas_oracle()
            .op_suggest_tip_cap(self.inner.min_suggested_priority_fee)
            .await
            .map_err(Into::into)
    }
}

impl<N, Rpc> LoadState for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives>,
    Self: LoadPendingBlock,
{
}

impl<N, Rpc> EthState for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
    Self: LoadPendingBlock,
{
    #[inline]
    fn max_proof_window(&self) -> u64 {
        self.inner.eth_api.eth_proof_window()
    }
}

impl<N, Rpc> EthFees for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}

impl<N, Rpc> Trace for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
}

impl<N: RpcNodeCore, Rpc: RpcConvert> fmt::Debug for OpEthApi<N, Rpc> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpEthApi").finish_non_exhaustive()
    }
}

/// Container type `OpEthApi`
pub struct OpEthApiInner<N: RpcNodeCore, Rpc: RpcConvert> {
    /// Gateway to node's core components.
    eth_api: EthApiNodeBackend<N, Rpc>,
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_client: Option<SequencerClient>,
    /// Minimum priority fee enforced by OP-specific logic.
    ///
    /// See also <https://github.com/ethereum-optimism/op-geth/blob/d4e0fe9bb0c2075a9bff269fb975464dd8498f75/eth/gasprice/optimism-gasprice.go#L38-L38>
    min_suggested_priority_fee: U256,
    /// Flashblocks listeners.
    ///
    /// If set, provides receivers for pending blocks, flashblock sequences, and build status.
    flashblocks: Option<FlashblocksListeners<N::Primitives>>,
}

impl<N: RpcNodeCore, Rpc: RpcConvert> fmt::Debug for OpEthApiInner<N, Rpc> {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpEthApiInner").finish()
    }
}

impl<N: RpcNodeCore, Rpc: RpcConvert> OpEthApiInner<N, Rpc> {
    /// Returns a reference to the [`EthApiNodeBackend`].
    const fn eth_api(&self) -> &EthApiNodeBackend<N, Rpc> {
        &self.eth_api
    }

    /// Returns the configured sequencer client, if any.
    const fn sequencer_client(&self) -> Option<&SequencerClient> {
        self.sequencer_client.as_ref()
    }
}

/// Converter for OP RPC types.
pub type OpRpcConvert<N, NetworkT> = RpcConverter<
    NetworkT,
    <N as FullNodeComponents>::Evm,
    OpReceiptConverter<<N as FullNodeTypes>::Provider>,
    (),
    OpTxInfoMapper<<N as FullNodeTypes>::Provider>,
>;

/// Builds [`OpEthApi`] for Optimism.
#[derive(Debug)]
pub struct OpEthApiBuilder<NetworkT = Optimism> {
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_url: Option<String>,
    /// Headers to use for the sequencer client requests.
    sequencer_headers: Vec<String>,
    /// Minimum suggested priority fee (tip)
    min_suggested_priority_fee: u64,
    /// A URL pointing to a secure websocket connection (wss) that streams out [flashblocks].
    ///
    /// [flashblocks]: reth_optimism_flashblocks
    flashblocks_url: Option<Url>,
    /// Enable flashblock consensus client to drive the chain forward.
    ///
    /// When enabled, flashblock sequences are submitted to the engine API via
    /// `newPayload` and `forkchoiceUpdated` calls, advancing the canonical chain state.
    /// Requires `flashblocks_url` to be set.
    flashblock_consensus: bool,
    /// Marker for network types.
    _nt: PhantomData<NetworkT>,
}

impl<NetworkT> Default for OpEthApiBuilder<NetworkT> {
    fn default() -> Self {
        Self {
            sequencer_url: None,
            sequencer_headers: Vec::new(),
            min_suggested_priority_fee: 1_000_000,
            flashblocks_url: None,
            flashblock_consensus: false,
            _nt: PhantomData,
        }
    }
}

impl<NetworkT> OpEthApiBuilder<NetworkT> {
    /// Creates a [`OpEthApiBuilder`] instance from core components.
    pub const fn new() -> Self {
        Self {
            sequencer_url: None,
            sequencer_headers: Vec::new(),
            min_suggested_priority_fee: 1_000_000,
            flashblocks_url: None,
            flashblock_consensus: false,
            _nt: PhantomData,
        }
    }

    /// With a [`SequencerClient`].
    pub fn with_sequencer(mut self, sequencer_url: Option<String>) -> Self {
        self.sequencer_url = sequencer_url;
        self
    }

    /// With headers to use for the sequencer client requests.
    pub fn with_sequencer_headers(mut self, sequencer_headers: Vec<String>) -> Self {
        self.sequencer_headers = sequencer_headers;
        self
    }

    /// With minimum suggested priority fee (tip).
    pub const fn with_min_suggested_priority_fee(mut self, min: u64) -> Self {
        self.min_suggested_priority_fee = min;
        self
    }

    /// With a subscription to flashblocks secure websocket connection.
    pub fn with_flashblocks(mut self, flashblocks_url: Option<Url>) -> Self {
        self.flashblocks_url = flashblocks_url;
        self
    }

    /// With flashblock consensus client enabled to drive chain forward
    pub const fn with_flashblock_consensus(mut self, flashblock_consensus: bool) -> Self {
        self.flashblock_consensus = flashblock_consensus;
        self
    }
}

impl<N, NetworkT> EthApiBuilder<N> for OpEthApiBuilder<NetworkT>
where
    N: FullNodeComponents<
            Evm: ConfigureEvm<
                NextBlockEnvCtx: BuildPendingEnv<HeaderTy<N::Types>>
                                     + From<OpFlashblockPayloadBase>
                                     + Unpin,
            >,
            Types: NodeTypes<
                ChainSpec: Hardforks + EthereumHardforks,
                Payload: reth_node_api::PayloadTypes<
                    ExecutionData: for<'a> TryFrom<
                        &'a FlashBlockCompleteSequence,
                        Error: std::fmt::Display,
                    >,
                >,
            >,
        >,
    NetworkT: RpcTypes,
    OpRpcConvert<N, NetworkT>: RpcConvert<Network = NetworkT>,
    OpEthApi<N, OpRpcConvert<N, NetworkT>>:
        FullEthApiServer<Provider = N::Provider, Pool = N::Pool>,
{
    type EthApi = OpEthApi<N, OpRpcConvert<N, NetworkT>>;

    async fn build_eth_api(self, ctx: EthApiCtx<'_, N>) -> eyre::Result<Self::EthApi> {
        let Self {
            sequencer_url,
            sequencer_headers,
            min_suggested_priority_fee,
            flashblocks_url,
            flashblock_consensus,
            ..
        } = self;
        let rpc_converter =
            RpcConverter::new(OpReceiptConverter::new(ctx.components.provider().clone()))
                .with_mapper(OpTxInfoMapper::new(ctx.components.provider().clone()));

        let sequencer_client = if let Some(url) = sequencer_url {
            Some(
                SequencerClient::new_with_headers(&url, sequencer_headers)
                    .await
                    .wrap_err_with(|| format!("Failed to init sequencer client with: {url}"))?,
            )
        } else {
            None
        };

        let flashblocks = if let Some(ws_url) = flashblocks_url {
            info!(target: "reth:cli", %ws_url, "Launching flashblocks service");

            let (tx, pending_rx) = watch::channel(None);
            let stream = WsFlashBlockStream::new(ws_url);
            let service = FlashBlockService::new(
                stream,
                ctx.components.evm_config().clone(),
                ctx.components.provider().clone(),
                ctx.components.task_executor().clone(),
                // enable state root calculation if flashblock_consensus is enabled.
                flashblock_consensus,
            );

            let flashblocks_sequence = service.block_sequence_broadcaster().clone();
            let received_flashblocks = service.flashblocks_broadcaster().clone();
            let in_progress_rx = service.subscribe_in_progress();
            ctx.components.task_executor().spawn(Box::pin(service.run(tx)));

            if flashblock_consensus {
                info!(target: "reth::cli", "Launching FlashBlockConsensusClient");
                let flashblock_client = FlashBlockConsensusClient::new(
                    ctx.engine_handle.clone(),
                    flashblocks_sequence.subscribe(),
                )?;
                ctx.components.task_executor().spawn(Box::pin(flashblock_client.run()));
            }

            Some(FlashblocksListeners::new(
                pending_rx,
                flashblocks_sequence,
                in_progress_rx,
                received_flashblocks,
            ))
        } else {
            None
        };

        let eth_api = ctx.eth_api_builder().with_rpc_converter(rpc_converter).build_inner();

        Ok(OpEthApi::new(
            eth_api,
            sequencer_client,
            U256::from(min_suggested_priority_fee),
            flashblocks,
        ))
    }
}
