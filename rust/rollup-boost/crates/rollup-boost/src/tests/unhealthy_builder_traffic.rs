use super::common::{RollupBoostTestHarnessBuilder, proxy::BuilderProxyHandler};
use crate::ExecutionMode;
use futures::FutureExt as _;
use serde_json::Value;
use std::pin::Pin;
use std::sync::{Arc, Mutex};
use std::time::Duration;

struct CounterHandler {
    counter: Arc<Mutex<u32>>,
}

impl BuilderProxyHandler for CounterHandler {
    fn handle(
        &self,
        method: String,
        _params: Value,
        _result: Value,
    ) -> Pin<Box<dyn Future<Output = Option<Value>> + Send>> {
        // Only count Engine API calls, not health check calls
        if method != "eth_getBlockByNumber" {
            *self.counter.lock().unwrap() += 1;
            tracing::info!("Proxy handler intercepted Engine API call: {}", method);
        } else {
            tracing::debug!("Proxy handler intercepted health check call: {}", method);
        }
        async move { None }.boxed()
    }
}

#[tokio::test]
async fn no_traffic_to_unhealthy_builder_when_flag_disabled() -> eyre::Result<()> {
    // Create a counter that tracks Engine API calls to the builder
    let counter = Arc::new(Mutex::new(0));
    let handler = Arc::new(CounterHandler {
        counter: counter.clone(),
    });

    // Create test harness with:
    // - ignore_unhealthy_builders=true (key test parameter)
    // - short max_unsafe_interval=1 to make builder unhealthy quickly
    let harness = RollupBoostTestHarnessBuilder::new("no_traffic_unhealthy")
        .proxy_handler(handler)
        .with_ignore_unhealthy_builders(true)
        .with_max_unsafe_interval(1)
        .build()
        .await?;

    // Override max_unsafe_interval to 1 second for this test
    // We'll need to modify the config after build - let's access the rollup boost args

    let mut block_generator = harness.block_generator().await?;
    let client = harness.debug_client().await;

    // Step 1: Disable execution mode so L2 moves ahead and builder falls behind
    let response = client
        .set_execution_mode(ExecutionMode::Disabled)
        .await
        .unwrap();
    assert_eq!(response.execution_mode, ExecutionMode::Disabled);

    // Step 2: Let L2 move ahead by generating some blocks
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_l2(),
            "Blocks should be created by L2 when execution disabled"
        );
    }

    // Step 3: Re-enable execution mode
    let response = client
        .set_execution_mode(ExecutionMode::Enabled)
        .await
        .unwrap();
    assert_eq!(response.execution_mode, ExecutionMode::Enabled);

    // Step 4:Wait for health check to run again and mark builder as unhealthy
    // since execution mode is now enabled and builder timestamp is stale
    tokio::time::sleep(Duration::from_secs(2)).await;

    // Reset counter after builder should be marked unhealthy
    *counter.lock().unwrap() = 0;

    // Step 5: Generate blocks - builder should now be unhealthy and with flag=false,
    // no engine API calls should go to the builder
    for _i in 0..5 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        // Blocks should be created by L2 since builder is unhealthy and flag is false
        assert!(
            block_creator.is_l2(),
            "Block creator should be L2 when builder is unhealthy and flag is false"
        );
    }

    let final_count = *counter.lock().unwrap();

    // With flag=false and unhealthy builder, should see zero engine API calls
    assert_eq!(
        final_count, 0,
        "Should see no Engine API calls to unhealthy builder when flag is false"
    );

    Ok(())
}
