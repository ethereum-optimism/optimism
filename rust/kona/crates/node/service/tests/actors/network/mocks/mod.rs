//! Tests interactions with sequencer actor's inputs channels.

use std::{str::FromStr, time::Duration};

use backon::{ExponentialBuilder, Retryable};
use discv5::Enr;
use kona_gossip::{PeerDump, PeerInfo};
use kona_node_service::NetworkActorError;
use kona_rpc::{OpP2PApiServer, P2pRpc};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::{sync::mpsc, task::JoinHandle};

pub(crate) mod builder;

pub(crate) struct TestNetwork {
    pub(super) p2p_rpc: P2pRpc,
    #[allow(dead_code)]
    pub(super) admin_rpc_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    pub(super) signer: kona_node_service::signer::Handle,
    pub(super) blocks_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    #[allow(dead_code)]
    handle: JoinHandle<Result<(), NetworkActorError>>,
}

#[derive(Debug, thiserror::Error)]
pub(crate) enum TestNetworkError {
    #[error("P2p RPC failed: {0}")]
    Rpc(#[from] jsonrpsee::types::ErrorObjectOwned),
    #[error("Peer info missing ENR")]
    PeerInfoMissingEnr,
    #[error("Invalid ENR: {0}")]
    InvalidEnr(String),
    #[error("Peer not connected")]
    PeerNotConnected,
}

impl TestNetwork {
    pub(super) async fn peer_info(&self) -> Result<PeerInfo, TestNetworkError> {
        Ok(self.p2p_rpc.opp2p_self().await?)
    }

    pub(super) async fn peers(&self) -> Result<PeerDump, TestNetworkError> {
        Ok(self.p2p_rpc.opp2p_peers(true).await?)
    }

    pub(super) async fn is_connected_to(&self, other: &Self) -> Result<(), TestNetworkError> {
        let other_peer_id = other.peer_id().await?;
        let peers = self.peers().await?;
        if !peers.peers.contains_key(&other_peer_id) {
            return Err(TestNetworkError::PeerNotConnected);
        }
        Ok(())
    }

    /// Like `is_connected_to`, but retries a couple of times until the connection is established.
    pub(super) async fn is_connected_to_with_retries(
        &self,
        other: &Self,
    ) -> Result<(), TestNetworkError> {
        (async || self.is_connected_to(other).await)
            .retry(ExponentialBuilder::default().with_total_delay(Some(Duration::from_secs(360))))
            // When to retry
            .when(|e| matches!(e, TestNetworkError::PeerNotConnected))
            .notify(|e, duration| tracing::info!(target: "network", "Retrying connection. Error: {e:?}, duration: {duration:?}"))
            .await
    }

    pub(super) async fn peer_enr(&self) -> Result<Enr, TestNetworkError> {
        let enr = self.peer_info().await?.enr.ok_or(TestNetworkError::PeerInfoMissingEnr)?;
        // Parse the ENR
        let enr = Enr::from_str(&enr).map_err(TestNetworkError::InvalidEnr)?;
        Ok(enr)
    }

    pub(super) async fn peer_id(&self) -> Result<String, TestNetworkError> {
        Ok(self.peer_info().await?.peer_id)
    }
}
