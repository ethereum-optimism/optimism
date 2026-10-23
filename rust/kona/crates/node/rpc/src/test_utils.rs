//! Doubles shared by the RPC server tests.

use crate::{EngineRpcClient, L1State, L1WatcherQueries, L1WatcherQuerySender};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot};
use std::{collections::BTreeMap, sync::Arc};
use tokio::sync::{mpsc, watch};

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

/// An engine client answering from fixed state and a fixed set of block outputs.
#[derive(Debug, Clone)]
pub(crate) struct StubEngineClient {
    pub(crate) state: EngineState,
    pub(crate) outputs: Arc<BTreeMap<u64, BlockOutput>>,
}

impl StubEngineClient {
    pub(crate) fn new(state: EngineState, outputs: BTreeMap<u64, BlockOutput>) -> Self {
        Self { state, outputs: Arc::new(outputs) }
    }
}

#[async_trait]
impl EngineRpcClient for StubEngineClient {
    async fn get_config(&self) -> RpcResult<RollupConfig> {
        Ok(RollupConfig::default())
    }
    async fn get_state(&self) -> RpcResult<EngineState> {
        Ok(self.state)
    }
    async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> RpcResult<(L2BlockInfo, OutputRoot, EngineState)> {
        let BlockNumberOrTag::Number(number) = block else {
            return Err(ErrorObject::from(ErrorCode::InvalidParams));
        };
        let found = self.outputs.get(&number).ok_or_else(|| {
            ErrorObject::owned(
                ErrorCode::InvalidParams.code(),
                format!("block {number} not found"),
                None::<()>,
            )
        })?;
        Ok((found.block, found.output, self.state))
    }
    async fn dev_get_task_queue_length(&self) -> RpcResult<usize> {
        unimplemented!("not used by these tests")
    }
    async fn dev_subscribe_to_engine_queue_length(&self) -> RpcResult<watch::Receiver<usize>> {
        unimplemented!("not used by these tests")
    }
    async fn dev_subscribe_to_engine_state(&self) -> RpcResult<watch::Receiver<EngineState>> {
        unimplemented!("not used by these tests")
    }
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

/// An L2 block of the test chain whose hash encodes its number.
pub(crate) fn l2_block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::with_last_byte(0x80 + number as u8),
            number,
            parent_hash: B256::ZERO,
            timestamp: timestamp_of(number),
        },
        ..Default::default()
    }
}
