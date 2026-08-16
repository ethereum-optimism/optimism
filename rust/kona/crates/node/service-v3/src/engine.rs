//! Stateful execution-engine facade.

use kona_engine::EngineState;
use kona_genesis::RollupConfig;
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
}
