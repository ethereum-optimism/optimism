//! Server lifecycle management.

use jsonrpsee::server::ServerHandle;
use std::sync::Arc;
use tokio::net::TcpListener;

use crate::{config::DeterministicConfig, l1::L1ChainBuilder, l2::L2ChainBuilder};

use super::{
    beacon::{self, BeaconState},
    l1_rpc::{L1RpcImpl, L1RpcServer},
    l2_rpc::{L2RpcImpl, L2RpcServer},
};

/// Running test servers for L1 RPC, L2 RPC, and beacon API.
#[allow(missing_debug_implementations)]
pub struct TestServers {
    l1_handle: ServerHandle,
    l2_handle: ServerHandle,
    beacon_handle: tokio::task::JoinHandle<()>,
    l1_url: String,
    l2_url: String,
    beacon_url: String,
}

impl TestServers {
    /// Start all three servers on random available ports.
    pub async fn start(
        config: &DeterministicConfig,
        l1: &L1ChainBuilder,
        l2: &L2ChainBuilder,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        // Start L1 RPC server
        let l1_server = jsonrpsee::server::ServerBuilder::default().build("127.0.0.1:0").await?;
        let l1_addr = l1_server.local_addr()?;
        let l1_impl = L1RpcImpl::new(l1.blocks().to_vec(), config.l1_chain_id);
        let l1_handle = l1_server.start(l1_impl.into_rpc());

        // Start L2 RPC server
        let l2_server = jsonrpsee::server::ServerBuilder::default().build("127.0.0.1:0").await?;
        let l2_addr = l2_server.local_addr()?;
        let l2_impl = L2RpcImpl::new(
            l2.blocks().to_vec(),
            // Collect snapshots
            (0..l2.blocks().len())
                .map(|i| l2.snapshot_at(crate::l2::L2BlockRef { index: i }).clone())
                .collect(),
            config.l2_chain_id,
        );
        let l2_handle = l2_server.start(l2_impl.into_rpc());

        // Start beacon API server
        let beacon_listener = TcpListener::bind("127.0.0.1:0").await?;
        let beacon_addr = beacon_listener.local_addr()?;
        let beacon_state = Arc::new(BeaconState {
            config: config.clone(),
            blobs: std::collections::BTreeMap::new(), // TODO: extract from L1 builder
        });
        let beacon_app = beacon::beacon_router(beacon_state);
        let beacon_handle = tokio::spawn(async move {
            axum::serve(beacon_listener, beacon_app).await.expect("beacon server failed");
        });

        Ok(Self {
            l1_handle,
            l2_handle,
            beacon_handle,
            l1_url: format!("http://127.0.0.1:{}", l1_addr.port()),
            l2_url: format!("http://127.0.0.1:{}", l2_addr.port()),
            beacon_url: format!("http://127.0.0.1:{}", beacon_addr.port()),
        })
    }

    /// L1 RPC URL.
    pub fn l1_rpc_url(&self) -> &str {
        &self.l1_url
    }

    /// L2 RPC URL.
    pub fn l2_rpc_url(&self) -> &str {
        &self.l2_url
    }

    /// Beacon API URL.
    pub fn beacon_url(&self) -> &str {
        &self.beacon_url
    }

    /// Stop all servers.
    pub fn stop(self) {
        let _ = self.l1_handle.stop();
        let _ = self.l2_handle.stop();
        self.beacon_handle.abort();
    }
}
