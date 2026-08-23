//! Unsafe-insert tests for the post-EL-sync regime.
//!
//! While the execution layer's initial sync is in flight, any admitted gossip payload may be
//! inserted: pushing the tip at the EL is what drives the sync (op-node's EL-sync regime,
//! `op-node/rollup/driver/sync_deriver.go:102-115`). Once EL sync has finished, op-node never
//! inserts a payload that does not directly extend the unsafe head — gossip goes into a payload
//! queue and only the next applicable payload is processed
//! (`op-node/rollup/engine/engine_controller.go:1343-1352`) — and refuses to adopt a head the EL
//! answers `SYNCING` for (`engine_controller.go:586-595`, `:873-879`). This engine has no payload
//! buffer and gossip keeps re-delivering the tip, so the faithful translation of both refusals is
//! to drop the payload: the heads stay put and the queue keeps moving. Adopting the head instead
//! is what wedged the sync-tester verifier suite: consolidation then asked the EL for
//! `local_safe + 1` forever, a block the EL never had.

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

fn insert_task(
    client: Arc<MockEngineClient>,
    cfg: Arc<RollupConfig>,
    payload: OpExecutionPayloadEnvelope,
) -> EngineTask<MockEngineClient> {
    EngineTask::Insert(Box::new(InsertTask::new(
        client,
        cfg,
        payload,
        None,
        Arc::new(NoopBlockSink),
    )))
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

/// After EL sync, a far-ahead gossiped payload is dropped before the `engine_newPayload` round
/// trip: the client below is configured with no `new_payload` answer, so reaching the EL would
/// fail the task. op-node's payload queue holds such a payload back the same way
/// (`op-node/rollup/engine/engine_controller.go:1343-1352`); with no buffer here, the task
/// completes (so the queue moves on) and the unsafe head stays where the execution layer is.
#[tokio::test]
async fn a_far_ahead_gossip_payload_is_dropped_after_el_sync() {
    let payload = payload_at(10, B256::repeat_byte(0xaa));
    let cfg = cfg_for(&payload);
    let client = Arc::new(test_engine_client_builder().with_config(cfg.clone()).build());
    let mut state = state_at(block(1), true);

    tokio::time::timeout(
        std::time::Duration::from_secs(1),
        insert_task(client, cfg, payload).execute(&mut state),
    )
    .await
    .expect("a payload that reached the engine would retry forever; the drop must not")
    .expect("the payload is dropped without reaching the engine: the task completes");

    assert_eq!(
        state.sync_state.unsafe_head(),
        block(1),
        "a payload that does not extend the unsafe head must not be adopted after EL sync"
    );
}

/// During the initial EL sync the same insert is adopted optimistically — it is what points the
/// execution layer at the tip to sync toward (op-node's EL-sync regime,
/// `op-node/rollup/driver/sync_deriver.go:102-115`, `engine_controller.go:587-594`).
#[tokio::test]
async fn a_far_ahead_gossip_payload_drives_el_sync_before_completion() {
    let payload = payload_at(10, B256::repeat_byte(0xaa));
    let cfg = cfg_for(&payload);
    let client = syncing_client(cfg.clone());
    let mut state = state_at(block(1), false);

    insert_task(client, cfg, payload)
        .execute(&mut state)
        .await
        .expect("EL-sync bootstrap accepts the insert");

    assert_eq!(
        state.sync_state.unsafe_head().block_info.number,
        10,
        "during EL sync the gossiped tip is adopted to drive the sync"
    );
}

/// A payload that directly extends the unsafe head but still gets `SYNCING` back from the
/// canonicalizing forkchoice update — an execution layer that lost its state — is dropped without
/// the head being adopted, the no-buffer analog of op-node keeping the payload queued and the
/// head unchanged (`op-node/rollup/engine/engine_controller.go:873-879`).
#[tokio::test]
async fn an_extending_payload_the_el_answers_syncing_for_is_dropped_after_el_sync() {
    let head = block(1);
    let payload = payload_at(2, head.block_info.hash);
    let cfg = cfg_for(&payload);
    let client = syncing_client(cfg.clone());
    let mut state = state_at(head, true);

    insert_task(client, cfg, payload)
        .execute(&mut state)
        .await
        .expect("the payload is dropped, not retried: the task completes");

    assert_eq!(
        state.sync_state.unsafe_head(),
        block(1),
        "a head the execution layer answered SYNCING for must not be adopted after EL sync"
    );
}

/// The applicability rule itself, op-node's
/// (`op-node/rollup/engine/engine_controller.go:1343-1352`): only the payload sitting directly on
/// the unsafe head attaches once EL sync has finished; everything attaches while it is in flight.
#[test]
fn only_directly_extending_payloads_attach_after_el_sync() {
    let head = block(5);
    let cases = [
        // (number, parent, el_sync_finished, applicable)
        (6, head.block_info.hash, true, true),
        (6, B256::repeat_byte(0xaa), true, false),
        (7, B256::repeat_byte(0xaa), true, false),
        (5, head.block_info.parent_hash, true, false),
        (9, B256::repeat_byte(0xaa), false, true),
    ];
    for (number, parent, el_sync_finished, applicable) in cases {
        let payload = payload_at(number, parent);
        let cfg = cfg_for(&payload);
        let task = InsertTask::new(
            syncing_client(cfg.clone()),
            cfg,
            payload,
            None,
            Arc::new(NoopBlockSink),
        );
        assert_eq!(
            task.extends_engine_unsafe_head(&state_at(head, el_sync_finished)),
            applicable,
            "payload {number} (parent {parent}) with el_sync_finished={el_sync_finished}"
        );
    }
}

/// Before the engine has an unsafe head at all there is nothing to attach to, so gossip is
/// admitted whatever it claims — the same bootstrap posture as the local-safe admission check.
#[test]
fn gossip_is_applicable_without_an_unsafe_head() {
    let payload = payload_at(9, B256::repeat_byte(0xaa));
    let cfg = cfg_for(&payload);
    let task =
        InsertTask::new(syncing_client(cfg.clone()), cfg, payload, None, Arc::new(NoopBlockSink));
    let state = EngineState { el_sync_finished: true, ..Default::default() };
    assert!(task.extends_engine_unsafe_head(&state));
}
