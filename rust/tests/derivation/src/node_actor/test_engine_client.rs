//! Test implementation of [`DerivationEngineClient`] that captures derived attributes.

use async_trait::async_trait;
use kona_engine::ConsolidateInput;
use kona_node_service::{DerivationEngineClient, EngineClientResult};
use std::sync::{Arc, Mutex};
use tracing::info;

/// Captured outputs from the derivation engine client.
#[derive(Debug, Default, Clone)]
pub struct CapturedEngineOutputs {
    /// Attributes sent to the engine for consolidation.
    pub consolidation_signals: Vec<ConsolidateInput>,
    /// Finalized L2 block numbers.
    pub finalized_blocks: Vec<u64>,
    /// Number of forkchoice resets.
    pub reset_count: u32,
}

/// A test engine client that captures derived attributes instead of
/// sending them to a real execution engine.
#[derive(Debug, Clone)]
pub struct TestEngineClient {
    /// Captured outputs from the engine.
    pub outputs: Arc<Mutex<CapturedEngineOutputs>>,
}

impl TestEngineClient {
    /// Creates a new [`TestEngineClient`].
    pub fn new() -> Self {
        Self {
            outputs: Arc::new(Mutex::new(CapturedEngineOutputs::default())),
        }
    }

    /// Returns a snapshot of the captured outputs.
    pub fn captured_outputs(&self) -> CapturedEngineOutputs {
        self.outputs.lock().unwrap().clone()
    }
}

#[async_trait]
impl DerivationEngineClient for TestEngineClient {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        info!(target: "test_engine", "Received forkchoice reset");
        let mut outputs = self.outputs.lock().unwrap();
        outputs.reset_count += 1;
        Ok(())
    }

    async fn send_finalized_l2_block(&self, block_number: u64) -> EngineClientResult<()> {
        info!(target: "test_engine", block_number, "Received finalized L2 block");
        let mut outputs = self.outputs.lock().unwrap();
        outputs.finalized_blocks.push(block_number);
        Ok(())
    }

    async fn send_safe_l2_signal(&self, signal: ConsolidateInput) -> EngineClientResult<()> {
        info!(target: "test_engine", ?signal, "Received safe L2 signal");
        let mut outputs = self.outputs.lock().unwrap();
        outputs.consolidation_signals.push(signal);
        Ok(())
    }
}
