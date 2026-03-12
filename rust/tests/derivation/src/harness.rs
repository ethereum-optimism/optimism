//! Test environment builder and equivalence assertion harness.

use crate::{
    genesis::{self, TEST_BATCHER_ADDR, TEST_BATCH_INBOX},
    mock_batcher::MockBatcher,
    mock_l1::MockL1Chain,
    mock_sequencer::{MockL2Block, MockSequencer},
    node_actor::{TestEngineClient, TestL2Provider, TestNodeActor, TestPipeline},
};
use alloy_primitives::Bytes;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_node_service::DerivationError;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The test environment for derivation pipeline equivalence testing.
#[derive(Debug)]
pub struct TestEnv {
    /// Rollup configuration.
    pub config: Arc<RollupConfig>,
    /// Mock L1 chain.
    pub l1: MockL1Chain,
    /// Mock batcher.
    pub batcher: MockBatcher,
    /// Mock sequencer.
    pub sequencer: MockSequencer,
    /// Node actor (derivation path A).
    pub node_actor: TestNodeActor,
    /// Test engine client to capture derived attributes.
    pub engine_client: TestEngineClient,
    /// L2 provider for the derivation pipeline.
    pub l2_provider: TestL2Provider,
    /// Genesis L2 block info.
    pub genesis_l2: L2BlockInfo,
}

/// Builder for [`TestEnv`].
#[derive(Debug)]
pub struct TestEnvBuilder {
    config: Option<Arc<RollupConfig>>,
}

impl TestEnvBuilder {
    /// Creates a new builder.
    pub fn new() -> Self {
        Self { config: None }
    }

    /// Sets a custom rollup config.
    pub fn with_rollup_config(mut self, config: Arc<RollupConfig>) -> Self {
        self.config = Some(config);
        self
    }

    /// Builds the test environment.
    pub async fn build(self) -> TestEnv {
        let config = self.config.unwrap_or_else(genesis::test_rollup_config_arc);
        let l1_genesis = genesis::default_l1_genesis();
        let l2_genesis_info = genesis::default_l2_genesis();

        // Create mock L1 chain with genesis
        let l1 = MockL1Chain::new(l1_genesis);

        // Create L2 provider with genesis state
        let l2_provider = TestL2Provider::new();
        l2_provider.insert_block(l2_genesis_info);

        // Insert genesis system config
        if let Some(ref sys_config) = config.genesis.system_config {
            l2_provider.insert_system_config(0, sys_config.clone());
        }

        // Create mock sequencer
        let sequencer = MockSequencer::new(
            l2_genesis_info.block_info.hash,
            l2_genesis_info.block_info.timestamp,
            config.block_time,
        );

        // Create mock batcher
        let batcher = MockBatcher::new(TEST_BATCH_INBOX, TEST_BATCHER_ADDR);

        // Create engine client
        let engine_client = TestEngineClient::new();

        // Create the test derivation pipeline
        let l1_config = Arc::new(L1ChainConfig::default());
        let pipeline = TestPipeline::new(
            l1.clone(),
            l2_provider.clone(),
            config.clone(),
            l1_config,
            l1_genesis,
        );

        // Create the node actor
        let node_actor = TestNodeActor::new(engine_client.clone(), pipeline).await;

        TestEnv {
            config,
            l1,
            batcher,
            sequencer,
            node_actor,
            engine_client,
            l2_provider,
            genesis_l2: l2_genesis_info,
        }
    }
}

impl TestEnv {
    /// Creates a new builder for the test environment.
    pub fn builder() -> TestEnvBuilder {
        TestEnvBuilder::new()
    }

    /// Produces an L2 block via the sequencer with the given transactions.
    /// The block references the latest L1 block as its L1 origin.
    pub fn produce_l2_block(&mut self, txs: Vec<Bytes>) -> MockL2Block {
        let l1_origin = self.l1.latest();
        self.sequencer.produce_block(txs, l1_origin)
    }

    /// Buffers the given L2 block for batch submission, then encodes and submits
    /// all buffered blocks as L1 transactions. Mines a new L1 block containing
    /// the batch transactions and advances safe/finalized heads.
    pub fn batch_and_mine(&mut self) {
        // Buffer the latest sequencer block
        if let Some(block) = self.sequencer.latest_block() {
            self.batcher.buffer_block(block);
        }

        // Submit all buffered blocks as L1 txs
        let txs = self.batcher.submit_all();

        // Mine a new L1 block with these transactions
        self.l1.act_start_block(12); // 12s L1 block time
        for tx in txs {
            self.l1.act_include_tx(tx);
        }
        self.l1.act_end_block(12);
    }

    /// Advances L1 safe and finalized heads by one block each.
    pub fn advance_l1_safe_and_finalized(&self) {
        self.l1.act_safe_next();
        self.l1.act_finalize_next();
    }

    /// Convenience: produce block, batch, mine, and advance safe/finalized.
    pub fn produce_batch_mine_finalize(&mut self, txs: Vec<Bytes>) -> MockL2Block {
        let block = self.produce_l2_block(txs);
        self.batch_and_mine();
        self.advance_l1_safe_and_finalized();
        block
    }

    /// Steps the derivation actor until it yields (needs more data).
    /// Returns Ok(()) on yield, or propagates critical errors.
    pub async fn derive_until_yield(&mut self) -> Result<(), DerivationError> {
        loop {
            match self.node_actor.step().await {
                Ok(()) => continue,
                Err(DerivationError::Yield) => return Ok(()),
                Err(e) => return Err(e),
            }
        }
    }
}
