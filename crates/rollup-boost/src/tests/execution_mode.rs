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
async fn execution_mode() -> eyre::Result<()> {
    // Create a counter that increases whenever we receive a new RPC call in the builder
    let counter = Arc::new(Mutex::new(0));
    let handler = Arc::new(CounterHandler {
        counter: counter.clone(),
    });

    let harness = RollupBoostTestHarnessBuilder::new("execution_mode")
        .proxy_handler(handler)
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;

    // start creating 5 empty blocks which are processed by the builder
    for _ in 0..5 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    let client = harness.debug_client().await;

    // enable dry run mode
    {
        let response = client
            .set_execution_mode(ExecutionMode::DryRun)
            .await
            .unwrap();
        assert_eq!(response.execution_mode, ExecutionMode::DryRun);

        // the new valid block should be created the the l2 builder
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(block_creator.is_l2(), "Block creator should be l2");
    }

    // toggle again dry run mode
    {
        let response = client
            .set_execution_mode(ExecutionMode::Enabled)
            .await
            .unwrap();
        assert_eq!(response.execution_mode, ExecutionMode::Enabled);

        // the new valid block should be created the the builder
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    // sleep for 1 second so that it has time to send the last FCU request to the builder
    // and there is not a race condition with the disable call
    std::thread::sleep(Duration::from_secs(1));

    tracing::info!("Setting execution mode to disabled");

    // Set the execution mode to disabled and reset the counter in the proxy to 0
    // to track the number of calls to the builder during the disabled mode which
    // should be 0
    {
        let response = client
            .set_execution_mode(ExecutionMode::Disabled)
            .await
            .unwrap();
        assert_eq!(response.execution_mode, ExecutionMode::Disabled);

        // reset the counter in the proxy
        *counter.lock().unwrap() = 0;

        // create 5 blocks which are processed by the l2 clients
        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be l2");
        }

        assert_eq!(
            *counter.lock().unwrap(),
            0,
            "Number of calls to the builder should be 0",
        );
    }

    Ok(())
}
