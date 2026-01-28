//! Contains helper methods for testing the traversal stages in the pipeline.

use crate::{PollingTraversal, test_utils::TestChainProvider};
use alloc::{sync::Arc, vec};
use alloy_consensus::Receipt;
use alloy_primitives::{Address, B256, Bytes, Log, LogData, address, hex};
use kona_genesis::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC, RollupConfig};
use kona_protocol::BlockInfo;

/// [`TraversalTestHelper`] encapsulates useful testing methods for traversal stages.
#[derive(Debug, Clone)]
pub struct TraversalTestHelper;

impl TraversalTestHelper {
    /// The address of the l1 system config contract.
    pub const L1_SYS_CONFIG_ADDR: Address = address!("1337000000000000000000000000000000000000");

    /// Creates a new [`Log`] for the update batcher event.
    pub fn new_update_batcher_log() -> Log {
        Log {
            address: Self::L1_SYS_CONFIG_ADDR,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO, // Update type
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        }
    }

    /// Creates a new [`Receipt`] with the update batcher log and a bad log.
    pub fn new_receipts() -> alloc::vec::Vec<Receipt> {
        let mut receipt =
            Receipt { status: alloy_consensus::Eip658Value::Eip658(true), ..Receipt::default() };
        let bad = Log::new(
            Address::from([2; 20]),
            vec![CONFIG_UPDATE_TOPIC, B256::default()],
            Bytes::default(),
        )
        .unwrap();
        receipt.logs = vec![Self::new_update_batcher_log(), bad, Self::new_update_batcher_log()];
        vec![receipt.clone(), Receipt::default(), receipt]
    }

    /// Creates a new [`PollingTraversal`] with the given blocks and receipts.
    pub fn new_from_blocks(
        blocks: alloc::vec::Vec<BlockInfo>,
        receipts: alloc::vec::Vec<Receipt>,
    ) -> PollingTraversal<TestChainProvider> {
        let mut provider = TestChainProvider::default();
        let rollup_config = RollupConfig {
            l1_system_config_address: Self::L1_SYS_CONFIG_ADDR,
            ..RollupConfig::default()
        };
        for (i, block) in blocks.iter().enumerate() {
            provider.insert_block(i as u64, *block);
        }
        for (i, receipt) in receipts.iter().enumerate() {
            let hash = blocks.get(i).map(|b| b.hash).unwrap_or_default();
            provider.insert_receipts(hash, vec![receipt.clone()]);
        }
        PollingTraversal::new(provider, Arc::new(rollup_config))
    }

    /// Creates a new [`PollingTraversal`] with two default blocks and populated receipts.
    pub fn new_populated() -> PollingTraversal<TestChainProvider> {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = Self::new_receipts();
        Self::new_from_blocks(blocks, receipts)
    }
}
