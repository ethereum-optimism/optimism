//! Example for a simple Execution Extension
//!
//! Run with
//!
//! ```sh
//! cargo run -p example-exex-hello-world -- node --dev --dev.block-time 5s
//! ```

use clap::Parser;
use futures::TryStreamExt;
use reth_ethereum::{
    chainspec::EthereumHardforks,
    exex::{ExExContext, ExExEvent, ExExNotification},
    node::{
        api::{FullNodeComponents, NodeTypes},
        builder::rpc::RpcHandle,
        EthereumNode,
    },
    rpc::api::eth::helpers::FullEthApi,
};
use reth_tracing::tracing::info;
use tokio::sync::oneshot;

/// Additional CLI arguments
#[derive(Parser)]
struct ExExArgs {
    /// whether to launch an op-reth node
    #[arg(long)]
    optimism: bool,
}

/// A basic subscription loop of new blocks.
async fn my_exex<Node: FullNodeComponents>(mut ctx: ExExContext<Node>) -> eyre::Result<()> {
    while let Some(notification) = ctx.notifications.try_next().await? {
        match &notification {
            ExExNotification::ChainCommitted { new } => {
                info!(committed_chain = ?new.range(), "Received commit");
            }
            ExExNotification::ChainReorged { old, new } => {
                info!(from_chain = ?old.range(), to_chain = ?new.range(), "Received reorg");
            }
            ExExNotification::ChainReverted { old } => {
                info!(reverted_chain = ?old.range(), "Received revert");
            }
        };

        if let Some(committed_chain) = notification.committed_chain() {
            ctx.events.send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
        }
    }

    Ok(())
}

/// This is an example of how to access the [`RpcHandle`] inside an ExEx. It receives the
/// [`RpcHandle`] once the node is launched fully.
///
/// This function supports both Opstack Eth API and ethereum Eth API.
///
/// The received handle gives access to the `EthApi` has full access to all eth api functionality
/// [`FullEthApi`]. And also gives access to additional eth-related rpc method handlers, such as eth
/// filter.
async fn ethapi_exex<Node, EthApi>(
    mut ctx: ExExContext<Node>,
    rpc_handle: oneshot::Receiver<RpcHandle<Node, EthApi>>,
) -> eyre::Result<()>
where
    Node: FullNodeComponents<Types: NodeTypes<ChainSpec: EthereumHardforks>>,
    EthApi: FullEthApi,
{
    // Wait for the ethapi to be sent from the main function
    let rpc_handle = rpc_handle.await?;
    info!("Received rpc handle inside exex");

    // obtain the ethapi from the rpc handle
    let ethapi = rpc_handle.eth_api();

    // EthFilter type that provides all eth_getlogs related logic
    let _eth_filter = rpc_handle.eth_handlers().filter.clone();
    // EthPubSub type that provides all eth_subscribe logic
    let _eth_pubsub = rpc_handle.eth_handlers().pubsub.clone();
    // The TraceApi type that provides all the trace_ handlers
    let _trace_api = rpc_handle.trace_api();
    // The DebugApi type that provides all the debug_ handlers
    let _debug_api = rpc_handle.debug_api();

    while let Some(notification) = ctx.notifications.try_next().await? {
        if let Some(committed_chain) = notification.committed_chain() {
            ctx.events.send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;

            // can use the eth api to interact with the node
            let _rpc_block = ethapi.rpc_block(committed_chain.tip().hash().into(), true).await?;
        }
    }

    Ok(())
}

fn main() -> eyre::Result<()> {
    let args = ExExArgs::parse();

    if args.optimism {
        reth_op::cli::Cli::parse_args().run(|builder, _| {
            let (rpc_handle_tx, rpc_handle_rx) = oneshot::channel();
            Box::pin(async move {
                let handle = builder
                    .node(reth_op::node::OpNode::default())
                    .install_exex("my-exex", async move |ctx| Ok(my_exex(ctx)))
                    .install_exex("ethapi-exex", async move |ctx| {
                        Ok(ethapi_exex(ctx, rpc_handle_rx))
                    })
                    .launch()
                    .await?;

                // Retrieve the rpc handle from the node and send it to the exex
                rpc_handle_tx
                    .send(handle.node.add_ons_handle.clone())
                    .expect("Failed to send ethapi to ExEx");

                handle.wait_for_node_exit().await
            })
        })
    } else {
        reth_ethereum::cli::Cli::parse_args().run(|builder, _| {
            Box::pin(async move {
                let (rpc_handle_tx, rpc_handle_rx) = oneshot::channel();
                let handle = builder
                    .node(EthereumNode::default())
                    .install_exex("my-exex", async move |ctx| Ok(my_exex(ctx)))
                    .install_exex("ethapi-exex", async move |ctx| {
                        Ok(ethapi_exex(ctx, rpc_handle_rx))
                    })
                    .launch()
                    .await?;

                // Retrieve the rpc handle from the node and send it to the exex
                rpc_handle_tx
                    .send(handle.node.add_ons_handle.clone())
                    .expect("Failed to send ethapi to ExEx");

                handle.wait_for_node_exit().await
            })
        })
    }
}
