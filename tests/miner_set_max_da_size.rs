use common::{RollupBoostTestHarnessBuilder, proxy::BuilderProxyHandler};
use futures::FutureExt as _;
use serde_json::Value;
use std::pin::Pin;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use testcontainers::core::client::docker_client_instance;

mod common;

#[derive(Debug, Clone, Default)]
struct MaxDaSizeHandler {
    found_max_da_value: Arc<Mutex<u64>>,
}

impl MaxDaSizeHandler {
    fn get_found_max_da_value(&self) -> u64 {
        *self.found_max_da_value.lock().unwrap()
    }
}

impl BuilderProxyHandler for MaxDaSizeHandler {
    fn handle(
        &self,
        method: String,
        params: Value,
        _result: Value,
    ) -> Pin<Box<dyn Future<Output = Option<Value>> + Send>> {
        println!("method: {method:?}");
        if method == "miner_setMaxDASize" {
            // decode the params
            let params: Vec<u64> = serde_json::from_value(params).unwrap();
            assert_eq!(params.len(), 2);
            *self.found_max_da_value.lock().unwrap() = params[0];
        }
        async move { None }.boxed()
    }
}

#[tokio::test]
async fn miner_set_max_da_size() -> eyre::Result<()> {
    let handler = Arc::new(MaxDaSizeHandler::default());

    let harness = RollupBoostTestHarnessBuilder::new("miner_set_max_da_size")
        .with_isthmus_block(0)
        .proxy_handler(handler.clone())
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;
    block_generator.generate_builder_blocks(1).await?;

    let engine_api = harness.engine_api()?;

    // stop the builder
    let client = docker_client_instance().await?;
    client.pause_container(harness.builder.id()).await?;
    tokio::time::sleep(Duration::from_secs(2)).await;

    let first_val = 1000000;
    let _res = engine_api.set_max_da_size(first_val, first_val).await;
    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    let found = handler.get_found_max_da_value();
    assert_eq!(found, 0);

    let second_val = 2000000;
    let _res = engine_api.set_max_da_size(second_val, second_val).await;
    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    let found = handler.get_found_max_da_value();
    assert_eq!(found, 0);

    // restart the builder
    client.unpause_container(harness.builder.id()).await?;

    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    let found = handler.get_found_max_da_value();
    assert_eq!(found, second_val);

    Ok(())
}
