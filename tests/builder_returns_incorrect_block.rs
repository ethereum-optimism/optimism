use std::{pin::Pin, sync::Arc};

use alloy_primitives::B256;
use common::{RollupBoostTestHarnessBuilder, proxy::BuilderProxyHandler};
use futures::FutureExt as _;
use serde_json::Value;

use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;

mod common;

struct Handler;

impl BuilderProxyHandler for Handler {
    fn handle(
        &self,
        method: String,
        _params: Value,
        _result: Value,
    ) -> Pin<Box<dyn Future<Output = Option<Value>> + Send>> {
        async move {
            if method != "engine_getPayloadV3" {
                return None;
            }

            let mut payload =
                serde_json::from_value::<OpExecutionPayloadEnvelopeV3>(_result).unwrap();

            // modify the state root field
            payload
                .execution_payload
                .payload_inner
                .payload_inner
                .state_root = B256::ZERO;

            let result = serde_json::to_value(&payload).unwrap();
            Some(result)
        }
        .boxed()
    }
}

#[tokio::test]
async fn builder_returns_incorrect_block() -> eyre::Result<()> {
    // Test that the builder returns a block with an incorrect state root and that rollup-boost
    // does not process it.
    let harness = RollupBoostTestHarnessBuilder::new("builder_returns_incorrect_block")
        .proxy_handler(Arc::new(Handler))
        .build()
        .await?;

    let mut block_generator = harness.block_generator().await?;

    // create 3 blocks that are processed by the builder
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(block_creator.is_l2(), "Block creator should be the l2");
    }
    // check that at some point we had the log "builder payload was not valid" which signals
    // that the builder returned a payload that was not valid and rollup-boost did not process it.
    // read lines
    let logs = std::fs::read_to_string(harness.rollup_boost.args().log_file.clone().unwrap())?;
    assert!(
        logs.contains("Invalid payload"),
        "Logs should contain the message 'builder payload was not valid'"
    );

    Ok(())
}
