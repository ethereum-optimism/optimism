#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use serde_json::Value;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;

    use crate::integration::RollupBoostTestHarnessBuilder;
    use crate::server::ExecutionMode;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;

    #[tokio::test]
    async fn test_integration_simple() -> eyre::Result<()> {
        let harness = RollupBoostTestHarnessBuilder::new("test_integration_simple")
            .build()
            .await?;
        let mut block_generator = harness.get_block_generator().await?;

        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );
        }

        Ok(())
    }

    #[tokio::test]
    async fn test_integration_no_tx_pool() -> eyre::Result<()> {
        let harness = RollupBoostTestHarnessBuilder::new("test_integration_no_tx_pool")
            .build()
            .await?;
        let mut block_generator = harness.get_block_generator().await?;

        // start creating 5 empty blocks which are processed by the L2 builder
        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(true).await?;
            assert!(block_creator.is_l2(), "Block creator should be l2");
        }

        // process 5 more non empty blocks which are processed by the builder.
        // The builder should be on sync because it has received the new payload requests from rollup-boost.
        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );
        }

        Ok(())
    }

    #[tokio::test]
    async fn test_integration_execution_mode() -> eyre::Result<()> {
        // Create a counter that increases whenever we receive a new RPC call in the builder
        let counter = Arc::new(Mutex::new(0));

        let counter_for_handler = counter.clone();
        let handler = Box::new(move |_method: &str, _params: Value, _result: Value| {
            let mut counter = counter_for_handler.lock().unwrap();

            *counter += 1;
            None
        });

        let harness = RollupBoostTestHarnessBuilder::new("test_integration_dry_run")
            .proxy_handler(handler)
            .build()
            .await?;
        let mut block_generator = harness.get_block_generator().await?;

        // start creating 5 empty blocks which are processed by the builder
        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );
        }

        let client = harness.get_client().await;

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

    #[tokio::test]
    async fn test_integration_remote_builder_down() -> eyre::Result<()> {
        let mut harness =
            RollupBoostTestHarnessBuilder::new("test_integration_remote_builder_down")
                .build()
                .await?;
        let mut block_generator = harness.get_block_generator().await?;

        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );
        }

        // stop the builder
        let builder_service = harness._framework.get_mut_service("builder")?;
        builder_service.stop()?;

        // create 3 new blocks that are processed by the l2 builder
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be l2");
        }

        // start the builder again
        builder_service.start_and_ready()?;

        // the next block is computed by the l2 builder because the builder is not synced with the previous 3 blocks
        // But, once the builder receives the FCU request from rollup-boost, it will sync up the blocks with the
        // L2 block builder and be ready again.
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(block_creator.is_l2(), "Block creator should be l2");

        // Note: We might add some sleep here if the builder is not synced in time. I have not seen this happen yet.

        // create 3 new blocks that are processed by the l2 builder because the builder is not synced with the previous 3 blocks
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_builder(), "Block creator should be l2");
        }

        Ok(())
    }

    #[tokio::test]
    async fn test_integration_builder_full_delay() -> eyre::Result<()> {
        // Create a dynamic handler that delays all the calls by 2 seconds
        let delay = Arc::new(Mutex::new(Duration::from_secs(0)));

        let delay_for_handler = delay.clone();
        let handler = Box::new(move |_method: &str, _params: Value, _result: Value| {
            let delay = delay_for_handler.lock().unwrap();
            // sleep the amount of time specified in the delay
            std::thread::sleep(*delay);
            None
        });

        // This integration test checks that if the builder has a general delay in processing ANY of the requests,
        // rollup-boost does not stop building blocks.
        let harness = RollupBoostTestHarnessBuilder::new("test_integration_builder_full_delay")
            .proxy_handler(handler)
            .build()
            .await?;

        let mut block_generator = harness.get_block_generator().await?;

        // create 3 blocks that are processed by the builder
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );
        }

        // add the delay
        *delay.lock().unwrap() = Duration::from_secs(2);

        // create 3 blocks that are processed by the builder
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be the builder");
        }

        Ok(())
    }

    #[tokio::test]
    async fn test_integration_builder_returns_incorrect_block() -> eyre::Result<()> {
        // Test that the builder returns a block with an incorrect state root and that rollup-boost
        // does not process it.
        let handler = Box::new(move |method: &str, _params: Value, _result: Value| {
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
        });

        let harness =
            RollupBoostTestHarnessBuilder::new("test_integration_builder_returns_incorrect_block")
                .proxy_handler(handler)
                .build()
                .await?;

        let mut block_generator = harness.get_block_generator().await?;

        // create 3 blocks that are processed by the builder
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be the builder");
        }

        Ok(())
    }
}
