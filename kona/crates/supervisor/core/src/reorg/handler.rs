use super::metrics::Metrics;
use crate::{ReorgHandlerError, reorg::task::ReorgTask};
use alloy_primitives::ChainId;
use alloy_rpc_client::RpcClient;
use derive_more::Constructor;
use futures::future;
use kona_protocol::BlockInfo;
use kona_supervisor_metrics::observe_metrics_for_result_async;
use kona_supervisor_storage::{DbReader, StorageRewinder};
use std::{collections::HashMap, sync::Arc};
use tracing::{error, info, trace};

/// Handles L1 reorg operations for multiple chains
#[derive(Debug, Constructor)]
pub struct ReorgHandler<DB> {
    /// The Alloy RPC client for L1.
    rpc_client: RpcClient,
    /// Per chain dbs.
    chain_dbs: HashMap<ChainId, Arc<DB>>,
}

impl<DB> ReorgHandler<DB>
where
    DB: DbReader + StorageRewinder + Send + Sync + 'static,
{
    /// Initializes the metrics for the reorg handler
    pub fn with_metrics(self) -> Self {
        // Initialize metrics for all chains
        for chain_id in self.chain_dbs.keys() {
            Metrics::init(*chain_id);
        }

        self
    }

    /// Wrapper method for segregating concerns between the startup and L1 watcher reorg handlers.
    pub async fn verify_l1_consistency(&self) -> Result<(), ReorgHandlerError> {
        info!(
            target: "supervisor::reorg_handler",
            "Verifying L1 consistency for each chain..."
        );

        self.verify_and_handle_chain_reorg().await
    }

    /// Processes a reorg for all chains when a new latest L1 block is received
    pub async fn handle_l1_reorg(&self, latest_block: BlockInfo) -> Result<(), ReorgHandlerError> {
        trace!(
            target: "supervisor::reorg_handler",
            l1_block_number = latest_block.number,
            "Potential reorg detected, processing..."
        );

        self.verify_and_handle_chain_reorg().await
    }

    /// Verifies the consistency of each chain with the L1 chain and handles any reorgs, if any.
    async fn verify_and_handle_chain_reorg(&self) -> Result<(), ReorgHandlerError> {
        let mut handles = Vec::with_capacity(self.chain_dbs.len());

        for (chain_id, chain_db) in &self.chain_dbs {
            let reorg_task =
                ReorgTask::new(*chain_id, Arc::clone(chain_db), self.rpc_client.clone());

            let chain_id = *chain_id;

            let handle = tokio::spawn(async move {
                observe_metrics_for_result_async!(
                    Metrics::SUPERVISOR_REORG_SUCCESS_TOTAL,
                    Metrics::SUPERVISOR_REORG_ERROR_TOTAL,
                    Metrics::SUPERVISOR_REORG_DURATION_SECONDS,
                    Metrics::SUPERVISOR_REORG_METHOD_PROCESS_CHAIN_REORG,
                    async {
                        reorg_task.process_chain_reorg().await
                    },
                    "chain_id" => chain_id.to_string()
                )
            });
            handles.push(handle);
        }

        let results = future::join_all(handles).await;
        for result in results {
            if let Err(err) = result {
                error!(target: "supervisor::reorg_handler", %err, "Reorg task failed");
            }
        }

        Ok(())
    }
}
