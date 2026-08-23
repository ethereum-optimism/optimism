//! Unsafe-insert tests for the EL-sync catch-up channel.
//!
//! A gossiped payload far ahead of the execution layer's chain gets `SYNCING` back from both
//! `engine_newPayload` and the canonicalizing forkchoice update. The insert adopts the payload's
//! head anyway — during the initial EL sync *and* afterwards: this engine has no reqresp sync
//! client and no unsafe payload buffer, so pointing the EL at the gossiped tip is its only
//! unsafe-chain catch-up channel, op-node's EL-sync regime
//! (`op-node/rollup/driver/sync_deriver.go:102-115`,
//! `op-node/rollup/engine/engine_controller.go:587-594`). A verifier that falls behind — a
//! restart, a dropped peer — re-enters exactly this path; gating it on `el_sync_finished`
//! stranded such verifiers behind a live sequencer with no recovery channel
//! (`TestUnsafeChainNotStalling_*`, `TestSequencerRestart`, `TestSyncUnsafeBecomesSafe`).
//! When the adopted head names blocks the EL cannot serve, consolidation's fetch miss takes
//! op-node's stall/reset split instead of retrying forever (see the consolidate task).

use crate::{
    EngineState, EngineTask, EngineTaskExt, InsertTask, NoopBlockSink,
    test_utils::{MockEngineClient, TestEngineStateBuilder, test_engine_client_builder},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::{ChainGenesis, RollupConfig};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rstest::rstest;
use std::sync::Arc;

fn block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(number as u8),
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
            timestamp: number * 2,
        },
        ..Default::default()
    }
}

/// An empty payload at `number` whose parent is `parent_hash`.
fn payload_at(number: u64, parent_hash: B256) -> OpExecutionPayloadEnvelope {
    let mut payload =
        ExecutionPayloadV1::from_block_slow(&alloy_consensus::Block::<OpTxEnvelope>::default());
    payload.block_number = number;
    payload.parent_hash = parent_hash;
    OpExecutionPayloadEnvelope::V1(payload)
}

/// A rollup config whose genesis is pinned to `payload`'s block, so its `L2BlockInfo` decodes
/// without an L1-info deposit.
fn cfg_for(payload: &OpExecutionPayloadEnvelope) -> Arc<RollupConfig> {
    let block: OpBlock = payload.clone().try_into_block().unwrap();
    Arc::new(RollupConfig {
        genesis: ChainGenesis {
            l2: BlockNumHash { number: block.header.number, hash: block.header.hash_slow() },
            ..Default::default()
        },
        ..Default::default()
    })
}

/// A client that answers `SYNCING` to both the payload insert and the forkchoice update, the way
/// an execution layer that does not have the payload's ancestry answers.
fn syncing_client(cfg: Arc<RollupConfig>) -> Arc<MockEngineClient> {
    Arc::new(
        test_engine_client_builder()
            .with_config(cfg)
            .with_new_payload_v1_response(PayloadStatus::new(PayloadStatusEnum::Syncing, None))
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Syncing,
                None,
            )))
            .build(),
    )
}

/// A state whose heads sit at `head`, with EL sync in the given phase.
fn state_at(head: L2BlockInfo, el_sync_finished: bool) -> EngineState {
    TestEngineStateBuilder::new()
        .with_unsafe_head(head)
        .with_local_safe_head(head)
        .with_finalized_head(head)
        .with_el_sync_finished(el_sync_finished)
        .build()
}

/// A far-ahead gossiped payload the EL answers `SYNCING` for is adopted as the unsafe head in
/// both EL-sync phases — it is what points the execution layer at the tip to sync toward.
#[rstest]
#[case::during_initial_el_sync(false)]
#[case::after_el_sync(true)]
#[tokio::test]
async fn a_far_ahead_gossip_payload_drives_el_sync(#[case] el_sync_finished: bool) {
    let payload = payload_at(10, B256::repeat_byte(0xaa));
    let cfg = cfg_for(&payload);
    let client = syncing_client(cfg.clone());
    let mut state = state_at(block(1), el_sync_finished);

    EngineTask::Insert(Box::new(InsertTask::new(
        client,
        cfg,
        payload,
        None,
        Arc::new(NoopBlockSink),
    )))
    .execute(&mut state)
    .await
    .expect("the EL-sync catch-up channel accepts the insert");

    assert_eq!(
        state.sync_state.unsafe_head().block_info.number,
        10,
        "the gossiped tip is adopted to drive EL sync (el_sync_finished={el_sync_finished})"
    );
}
