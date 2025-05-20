mod common;

use common::RollupBoostTestHarnessBuilder;
use common::proxy::ProxyHandler;
use futures::FutureExt;
use serde_json::Value;
use std::pin::Pin;
use std::sync::Arc;
use std::time::Duration;

// TODO: Use the same implementation as in builder_full_delay.rs
struct DelayHandler {
    delay: Duration,
}

impl DelayHandler {
    pub fn new(delay: Duration) -> Self {
        Self { delay }
    }
}

impl ProxyHandler for DelayHandler {
    fn handle(
        &self,
        _method: String,
        _params: Value,
        _result: Value,
    ) -> Pin<Box<dyn Future<Output = Option<Value>> + Send>> {
        let delay = self.delay;
        async move {
            tokio::time::sleep(delay).await;
            None
        }
        .boxed()
    }
}

#[tokio::test]
async fn fcu_no_block_time_delay() -> eyre::Result<()> {
    // This test ensures that even with delay in between RB and the external builder (50ms) the
    // builder can still build the block if there is an avalanche of FCUs without block time delay
    let delay = DelayHandler::new(Duration::from_millis(50));

    let harness = RollupBoostTestHarnessBuilder::new("fcu_no_block_time_delay")
        .proxy_handler(Arc::new(delay))
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;
    block_generator.set_block_time(Duration::from_millis(0));

    for _ in 0..30 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    Ok(())
}
