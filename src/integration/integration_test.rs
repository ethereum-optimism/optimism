#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use serde_json::Value;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;

    use crate::debug_api::SetDryRunRequestAction;
    use crate::integration::RollupBoostTestHarnessBuilder;
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

            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
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
    async fn test_integration_dry_run() -> eyre::Result<()> {
        let harness = RollupBoostTestHarnessBuilder::new("test_integration_dry_run")
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
                .set_dry_run(SetDryRunRequestAction::SetDryRun(true))
                .await
                .unwrap();
            assert!(response.dry_run_state, "Dry run mode should be enabled");

            // the new valid block should be created the the l2 builder
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be l2");
        }

        // toggle again dry run mode
        {
            let response = client
                .set_dry_run(SetDryRunRequestAction::SetDryRun(false))
                .await
                .unwrap();
            assert!(!response.dry_run_state, "Dry run mode should be disabled");

            // the new valid block should be created the the builder
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
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

            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
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
            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        }

        // add the delay
        *delay.lock().unwrap() = Duration::from_secs(2);

        // create 3 blocks that are processed by the builder
        for _ in 0..3 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(block_creator.is_l2(), "Block creator should be the builder");
            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
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

        let mut harness =
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

        // check that at some point we had the log "builder payload was not valid" which signals
        // that the builder returned a payload that was not valid and rollup-boost did not process it.
        let rb_service = harness._framework.get_mut_service("rollup-boost")?;
        let logs = rb_service.get_logs()?;
        assert!(
            logs.contains("builder payload was not valid"),
            "Logs should contain the message 'builder payload was not valid'"
        );

        Ok(())
    }
}
