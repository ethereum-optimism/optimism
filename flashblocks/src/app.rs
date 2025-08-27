use crate::{ExecutionPayloadBaseV1, FlashBlockService, FlashBlockWsStream};
use futures_util::StreamExt;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::{BlockTy, HeaderTy, NodePrimitives, ReceiptTy};
use reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv;
use reth_rpc_eth_types::PendingBlock;
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use tokio::sync::watch;
use url::Url;

/// Spawns a background task that subscribes over websocket to `ws_url`.
///
/// Returns a [`FlashBlockRx`] that receives the most recent [`PendingBlock`] built from
///  [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub fn launch_wss_flashblocks_service<N, EvmConfig, Provider>(
    ws_url: Url,
    evm_config: EvmConfig,
    provider: Provider,
) -> FlashBlockRx<N>
where
    N: NodePrimitives,
    EvmConfig: ConfigureEvm<
            Primitives = N,
            NextBlockEnvCtx: BuildPendingEnv<N::BlockHeader>
                                 + From<ExecutionPayloadBaseV1>
                                 + Unpin
                                 + 'static,
        > + 'static,
    Provider: StateProviderFactory
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin
        + 'static,
{
    let stream = FlashBlockWsStream::new(ws_url);
    let mut service = FlashBlockService::new(stream, evm_config, provider);
    let (tx, rx) = watch::channel(None);

    tokio::spawn(async move {
        while let Some(block) = service.next().await {
            if let Ok(block) = block.inspect_err(|e| tracing::error!("{e}")) {
                let _ = tx.send(Some(block)).inspect_err(|e| tracing::error!("{e}"));
            }
        }
    });

    rx
}

/// Receiver of the most recent [`PendingBlock`] built out of [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockRx<N> = watch::Receiver<Option<PendingBlock<N>>>;
