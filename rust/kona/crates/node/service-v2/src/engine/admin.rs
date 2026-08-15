//! RPC-only administration capability for Engine-owned unsafe processing.

use crate::engine::{network::NetworkClient, unsafe_chain::SequencerHandle};
use alloy_primitives::B256;
use kona_gossip::P2pRpcRequest;
use kona_rpc::{NetworkAdminQuery, SequencerAdminAPIError};
use tokio::sync::mpsc;

/// Narrow Engine administration adapter supplied only to the RPC router.
#[derive(Debug, Clone)]
pub struct EngineAdminAdapter {
    sequencer: SequencerHandle,
    network: NetworkClient,
}

impl EngineAdminAdapter {
    pub(super) const fn new(sequencer: SequencerHandle, network: NetworkClient) -> Self {
        Self { sequencer, network }
    }

    /// Returns whether local production is active.
    pub fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer.configured_status()?.active)
    }

    /// Returns whether transaction-pool recovery mode is enabled.
    pub fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer.configured_status()?.recovery_mode)
    }

    /// Returns whether an HA conductor is configured.
    pub fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer.configured_status()?.conductor_enabled)
    }

    /// Starts local block production with fresh workflow state.
    pub async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        self.sequencer.start().await
    }

    /// Stops local production after its accepted block action completes.
    pub async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        self.sequencer.stop().await
    }

    /// Changes transaction-pool recovery mode at a block boundary.
    pub async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.sequencer.set_recovery_mode(mode).await
    }

    /// Requests HA conductor leadership override.
    pub async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.sequencer.override_leader().await
    }

    /// Returns the request sender used by the P2P RPC namespace.
    pub fn p2p_sender(&self) -> mpsc::Sender<P2pRpcRequest> {
        self.network.p2p_sender()
    }

    /// Returns the request sender used by admin payload injection.
    pub fn network_admin_sender(&self) -> mpsc::Sender<NetworkAdminQuery> {
        self.network.admin_sender()
    }
}
