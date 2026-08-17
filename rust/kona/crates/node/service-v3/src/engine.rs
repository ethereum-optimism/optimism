//! Stateful execution-engine facade.

use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated, PayloadStatus};
use alloy_transport::TransportResult;
use kona_engine::{EngineClient, EngineState};
use kona_genesis::RollupConfig;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;

/// A stateful execution engine that is not thread-safe.
#[derive(Debug)]
pub struct Engine<Client> {
    client: Arc<Client>,
    config: Arc<RollupConfig>,
    state: EngineState,
}

impl<Client> Engine<Client> {
    /// Creates an engine with an empty authoritative state.
    pub fn new(client: Arc<Client>, config: Arc<RollupConfig>) -> Self {
        Self { client, config, state: EngineState::default() }
    }

    /// Returns the raw execution client.
    pub const fn client(&self) -> &Arc<Client> {
        &self.client
    }

    /// Returns the rollup configuration.
    pub const fn config(&self) -> &Arc<RollupConfig> {
        &self.config
    }

    /// Returns the authoritative engine state.
    pub const fn state(&self) -> &EngineState {
        &self.state
    }

    /// Returns mutable access to the authoritative engine state.
    pub(crate) const fn state_mut(&mut self) -> &mut EngineState {
        &mut self.state
    }
}

impl<Client> Engine<Client>
where
    Client: EngineClient,
{
    /// Sends a payload to the execution layer using the version represented by its envelope.
    pub async fn new_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> TransportResult<PayloadStatus> {
        self.client.new_payload(payload).await
    }

    /// Sends a forkchoice update without payload attributes.
    pub async fn forkchoice_updated(
        &self,
        forkchoice: ForkchoiceState,
    ) -> TransportResult<ForkchoiceUpdated> {
        self.client.fork_choice_updated_v3(forkchoice, None).await
    }
}
