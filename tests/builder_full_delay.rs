use common::RollupBoostTestHarnessBuilder;
use common::proxy::BuilderProxyHandler;
use futures::FutureExt;
use serde_json::Value;
use std::pin::Pin;
use std::sync::{Arc, Mutex};
use std::time::Duration;

mod common;

// Create a dynamic handler that delays all the calls by 2 seconds
struct DelayHandler {
    delay: Arc<Mutex<Duration>>,
}

impl BuilderProxyHandler for DelayHandler {
    fn handle(
        &self,
        _method: String,
        _params: Value,
        _result: Value,
    ) -> Pin<Box<dyn Future<Output = Option<Value>> + Send>> {
        let delay = *self.delay.lock().unwrap();
        async move {
            tokio::time::sleep(delay).await;
            None
        }
        .boxed()
    }
}

#[tokio::test]
async fn builder_full_delay() -> eyre::Result<()> {
    let delay = Arc::new(Mutex::new(Duration::from_secs(0)));

    let handler = Arc::new(DelayHandler {
        delay: delay.clone(),
    });

    // This integration test checks that if the builder has a general delay in processing ANY of the requests,
    // rollup-boost does not stop building blocks.
    let harness = RollupBoostTestHarnessBuilder::new("builder_full_delay")
        .proxy_handler(handler)
        .build()
        .await?;

    let mut block_generator = harness.block_generator().await?;

    // create 3 blocks that are processed by the builder
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    // create 3 blocks that are processed by the builder
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    // add the delay
    *delay.lock().unwrap() = Duration::from_secs(5);

    // create 3 blocks that are processed by the builder
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(block_creator.is_l2(), "Block creator should be the l2");
    }

    Ok(())
}
