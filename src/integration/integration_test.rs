#[cfg(test)]
mod tests {
    use crate::integration::{RollupBoostTestHarness, SimpleBlockGenerator};

    #[tokio::test]
    async fn test_integration() {
        let harness = RollupBoostTestHarness::new().await.unwrap();

        let mut block_generator = SimpleBlockGenerator::new(&harness.engine_api);
        block_generator.init().await.unwrap();

        for _ in 0..10 {
            let block = block_generator.generate_block().await.unwrap();
            println!("Generated block: {:?}", block);
        }
    }
}
