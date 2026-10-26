//! Doubles shared by the RPC server tests.

use crate::{L1State, L1WatcherQueries, L1WatcherQuerySender};
use alloy_primitives::B256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot};
use std::sync::Arc;
use tokio::sync::mpsc;

/// The L2 genesis time of the test chain.
pub(crate) const GENESIS_TIME: u64 = 1_000;
/// The block time of the test chain.
const BLOCK_TIME: u64 = 2;

/// The timestamp of L2 block `number` on the test chain.
pub(crate) const fn timestamp_of(number: u64) -> u64 {
    GENESIS_TIME + BLOCK_TIME * number
}

/// A block and the output root the engine computes for it.
#[derive(Debug, Clone, Copy)]
pub(crate) struct BlockOutput {
    pub(crate) block: L2BlockInfo,
    pub(crate) output: OutputRoot,
}

pub(crate) fn query_client(config: Arc<RollupConfig>) -> kona_engine::EngineQueryClient {
    kona_engine::test_utils::test_engine_client(config).0.query_client()
}

pub(crate) fn recorded_query_client(
    config: Arc<RollupConfig>,
    output: Option<BlockOutput>,
    record: Option<kona_safedb::SafeHeadRecord>,
) -> kona_engine::EngineQueryClient {
    let (engine, _, transport) = kona_engine::test_utils::test_engine_client(config);
    if let (Some(output), Some(record)) = (output, record) {
        let block = rpc_block(output);
        let anchor = if record.safe_head.number == output.block.block_info.number {
            block.clone()
        } else {
            fixture_rpc_block(record.safe_head.number)
        };
        let attempts = if anchor.header.hash == record.safe_head.hash { 1 } else { 3 };
        for _ in 0..attempts {
            transport.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([format!("0x{:x}", record.safe_head.number), false]),
                &anchor,
            );
        }
        if anchor.header.hash == record.safe_head.hash {
            for number in (output.block.block_info.number..record.safe_head.number).rev() {
                let parent = fixture_rpc_block(number);
                transport.expect_params(
                    "eth_getBlockByHash",
                    serde_json::json!([parent.header.hash, false]),
                    parent,
                );
            }
            transport.expect_params(
                "eth_getBlockByHash",
                serde_json::json!([block.header.hash, true]),
                &block,
            );
            transport.expect_params(
                "eth_getProof",
                serde_json::json!([
                    kona_protocol::Predeploys::L2_TO_L1_MESSAGE_PASSER,
                    [],
                    block.header.hash
                ]),
                alloy_rpc_types_eth::EIP1186AccountProofResponse {
                    storage_hash: output.output.bridge_storage_root,
                    ..Default::default()
                },
            );
            transport.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([format!("0x{:x}", record.safe_head.number), false]),
                anchor,
            );
        }
    }
    engine.query_client()
}

fn fixture_rpc_block(number: u64) -> alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> {
    rpc_block(BlockOutput {
        block: l2_block(number),
        output: OutputRoot {
            block_hash: l2_block(number).block_info.hash,
            bridge_storage_root: B256::with_last_byte(0x60 + number as u8),
            state_root: B256::with_last_byte(0x40 + number as u8),
        },
    })
}

pub(crate) fn rpc_block(
    output: BlockOutput,
) -> alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> {
    let mut block = alloy_rpc_types_eth::Block::<op_alloy_rpc_types::Transaction>::default();
    block.header.inner.number = output.block.block_info.number;
    block.header.inner.parent_hash = output.block.block_info.parent_hash;
    block.header.inner.timestamp = output.block.block_info.timestamp;
    block.header.inner.state_root = output.output.state_root;
    block.header.hash = block.header.inner.hash_slow();
    let deposit = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
        op_alloy_consensus::TxDeposit {
            input: kona_protocol::L1BlockInfoBedrock::default().encode_calldata(),
            ..Default::default()
        },
    ));
    block.transactions =
        alloy_rpc_types_eth::BlockTransactions::Full(vec![op_alloy_rpc_types::Transaction {
            inner: alloy_rpc_types_eth::Transaction {
                block_hash: None,
                block_number: Some(output.block.block_info.number),
                block_timestamp: None,
                effective_gas_price: Some(1),
                inner: alloy_consensus::transaction::Recovered::new_unchecked(
                    deposit,
                    Default::default(),
                ),
                transaction_index: Some(0),
            },
            deposit_nonce: None,
            deposit_receipt_version: None,
        }]);
    block
}

/// A query sender whose L1 watcher reports `current_l1` and nothing else, for as long as the
/// test runtime lives.
pub(crate) fn l1_watcher_at(current_l1: Option<BlockInfo>) -> L1WatcherQuerySender {
    let (sender, mut receiver) = mpsc::channel(8);
    tokio::spawn(async move {
        while let Some(query) = receiver.recv().await {
            match query {
                L1WatcherQueries::L1State(reply) => {
                    let _ = reply.send(L1State {
                        current_l1,
                        current_l1_finalized: None,
                        head_l1: None,
                        safe_l1: None,
                        finalized_l1: None,
                    });
                }
                L1WatcherQueries::Config(reply) => {
                    let _ = reply.send(RollupConfig::default());
                }
            }
        }
    });
    sender
}

/// An L1 block whose hash encodes its number, so a wrong block is visible in a failure.
pub(crate) fn l1_block(number: u64) -> BlockInfo {
    BlockInfo { hash: B256::with_last_byte(number as u8), number, ..Default::default() }
}

pub(crate) fn l2_block(number: u64) -> L2BlockInfo {
    let mut parent_hash = B256::ZERO;
    let mut block_info = BlockInfo::default();
    for height in 0..=number {
        let header = alloy_consensus::Header {
            number: height,
            parent_hash,
            state_root: B256::with_last_byte(0x40 + height as u8),
            timestamp: timestamp_of(height),
            ..Default::default()
        };
        block_info = BlockInfo {
            hash: header.hash_slow(),
            number: height,
            parent_hash,
            timestamp: timestamp_of(height),
        };
        parent_hash = block_info.hash;
    }
    L2BlockInfo { block_info, ..Default::default() }
}
