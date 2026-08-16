//! Shared stateful execution-engine facade.

use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::Mutex;

/// The shared engine type used by the safe and unsafe chain tasks.
pub type SharedEngine<Client> = Arc<Mutex<Engine<Client>>>;

/// Owns the raw execution client and the node's authoritative execution state.
///
/// Service V3 serializes complete semantic operations by holding the [`Mutex`] around this value.
/// The operation implementations will be added incrementally as the safe and unsafe builders are
/// ported.
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

    /// Wraps this engine in the shared Tokio mutex used by the core tasks.
    pub fn shared(self) -> SharedEngine<Client> {
        Arc::new(Mutex::new(self))
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
