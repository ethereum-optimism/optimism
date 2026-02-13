use std::{pin::Pin, sync::Arc};

use super::common::{RollupBoostTestHarnessBuilder, proxy::BuilderProxyHandler};
use alloy_primitives::B256;
use futures::FutureExt as _;
use serde_json::Value;

use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;

struct ZeroStateRootHandler;

impl BuilderProxyHandler for ZeroStateRootHandler {
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

            // Set state root to zero to simulate builder payload without computed state root
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
async fn test_l2_state_root_computation() -> eyre::Result<()> {
    // Test that when builder returns payload with zero state root and L2 state root computation
    // is enabled, rollup-boost uses L2 client to compute the correct state root
    let harness = RollupBoostTestHarnessBuilder::new("l2_state_root_computation")
        .proxy_handler(Arc::new(ZeroStateRootHandler))
        .with_l2_state_root_computation(true)
        .build()
        .await?;

    let mut block_generator = harness.block_generator().await?;

    // Generate blocks and verify they are processed successfully by the builder
    // with L2 computing the correct state root
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    // Check logs to verify L2 state root computation was used
    let logs = std::fs::read_to_string(harness.rollup_boost.args().log_file.clone().unwrap())?;
    assert!(
        logs.contains("sent FCU to l2 to calculate new state root"),
        "Logs should contain message indicating L2 state root computation was used"
    );
    assert!(
        logs.contains("received new state root payload from l2"),
        "Logs should contain message indicating L2 returned corrected payload"
    );

    Ok(())
}
