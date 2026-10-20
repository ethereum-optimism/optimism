use super::*;
use crate::{
    EngineForkchoiceVersion, EnginePhase, EngineState, EngineSyncStateUpdate, ImportedBlockSink,
    NoopBlockSink,
    test_utils::{
        ExpectedCall, MockEngineClient, Script, TestAttributesBuilder, TestEngineStateBuilder,
        test_block_info,
    },
};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3, ForkchoiceUpdated, PayloadId,
    PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3};
use rstest::rstest;
use std::sync::{Arc, Mutex};

#[derive(Debug, Default)]
struct Sink(Mutex<Vec<L2BlockInfo>>);
impl ImportedBlockSink for Sink {
    fn block_imported(&self, _: OpBlock, block: L2BlockInfo) {
        self.0.lock().unwrap().push(block);
    }
}

fn valid() -> PayloadStatus {
    PayloadStatus::from_status(PayloadStatusEnum::Valid)
}
fn invalid() -> PayloadStatus {
    PayloadStatus::from_status(PayloadStatusEnum::Invalid { validation_error: "rejected".into() })
}
fn fcu() -> ForkchoiceUpdated {
    ForkchoiceUpdated {
        payload_id: Some(PayloadId::new([1; 8])),
        ..ForkchoiceUpdated::new(valid())
    }
}

/// Pin the payload as genesis so these operation tests need no fabricated L1-info deposit.
fn payload(cfg: &mut RollupConfig, attrs: &OpAttributesWithParent) -> OpExecutionPayloadEnvelopeV3 {
    let mut block = OpBlock::default();
    block.header.number = attrs.block_number();
    block.header.parent_hash = attrs.parent.block_info.hash;
    block.header.timestamp = attrs.attributes.payload_attributes.timestamp;
    let mut envelope = OpExecutionPayloadEnvelope::V3 {
        payload: ExecutionPayloadV3 {
            payload_inner: ExecutionPayloadV2 {
                payload_inner: ExecutionPayloadV1::from_block_slow(&block),
                withdrawals: vec![],
            },
            blob_gas_used: 0,
            excess_blob_gas: 0,
        },
        parent_beacon_block_root: B256::ZERO,
    };
    let block = envelope.clone().try_into_block::<OpTxEnvelope>().unwrap();
    cfg.genesis.l2 = BlockNumHash { number: block.header.number, hash: block.header.hash_slow() };
    envelope.as_v1_mut().block_hash = cfg.genesis.l2.hash;
    let OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } = envelope else {
        unreachable!()
    };
    OpExecutionPayloadEnvelopeV3 {
        execution_payload: payload,
        parent_beacon_block_root,
        block_value: Default::default(),
        blobs_bundle: Default::default(),
        should_override_builder: false,
    }
}

#[rstest]
#[case::valid(PayloadStatusEnum::Valid)]
#[case::invalid(PayloadStatusEnum::Invalid { validation_error: "rejected".into() })]
#[case::syncing(PayloadStatusEnum::Syncing)]
#[case::accepted(PayloadStatusEnum::Accepted)]
#[tokio::test]
async fn build_versions_and_statuses(
    #[case] status: PayloadStatusEnum,
    #[values(EngineForkchoiceVersion::V2, EngineForkchoiceVersion::V3)]
    version: EngineForkchoiceVersion,
) {
    let parent = test_block_info(0);
    let attrs = TestAttributesBuilder::new().with_parent(parent).build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(match version {
        EngineForkchoiceVersion::V2 => u64::MAX,
        EngineForkchoiceVersion::V3 => 0,
    });
    let method = match version {
        EngineForkchoiceVersion::V2 => "engine_forkchoiceUpdatedV2",
        EngineForkchoiceVersion::V3 => "engine_forkchoiceUpdatedV3",
    };
    let script = Script::new([ExpectedCall::success(
        method,
        ForkchoiceUpdated { payload_status: PayloadStatus::from_status(status.clone()), ..fcu() },
    )]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(15))
        .with_safe_head(parent)
        .with_finalized_head(parent)
        .with_el_sync_finished(false)
        .build();
    let previous = state;
    let result = start_build(&client, &cfg, &mut state, &attrs).await;
    match status {
        PayloadStatusEnum::Valid => {
            assert_eq!(result.unwrap(), PayloadId::new([1; 8]));
            assert_eq!(state.sync_state.unsafe_head(), parent);
            assert!(state.el_sync_finished);
        }
        PayloadStatusEnum::Invalid { .. } => {
            assert!(matches!(result, Err(BuildError::Rejected(_))))
        }
        PayloadStatusEnum::Syncing => assert!(matches!(result, Err(BuildError::Syncing))),
        PayloadStatusEnum::Accepted => {
            assert!(matches!(result, Err(BuildError::UnexpectedStatus(_))))
        }
    }
    if status != PayloadStatusEnum::Valid {
        assert_eq!(state, previous);
    }
    script.assert_finished();
}

#[tokio::test]
async fn accepted_parent_is_retained_even_without_a_payload_id() {
    let script = Script::new([ExpectedCall::success(
        "engine_forkchoiceUpdatedV2",
        ForkchoiceUpdated::new(valid()),
    )]);
    let cfg = RollupConfig::default();
    let attrs = TestAttributesBuilder::new().build();
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = EngineState::default();
    assert!(matches!(
        start_build(&client, &cfg, &mut state, &attrs).await,
        Err(BuildError::MissingPayloadId)
    ));
    assert_eq!(state.sync_state.unsafe_head(), attrs.parent);
    assert!(state.el_sync_finished);
    assert_eq!(state.phase, EnginePhase::Initializing);
    script.assert_finished();
}

#[rstest]
#[case::valid(PayloadStatusEnum::Valid, true)]
#[case::syncing(PayloadStatusEnum::Syncing, true)]
#[case::invalid(PayloadStatusEnum::Invalid { validation_error: "rejected".into() }, false)]
#[case::accepted(PayloadStatusEnum::Accepted, false)]
#[tokio::test]
async fn synchronize_commits_only_accepted_forkchoice(
    #[case] status: PayloadStatusEnum,
    #[case] accepted: bool,
) {
    let script = Script::new([ExpectedCall::success(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(status.clone())),
    )]);
    let client = script.client(Arc::new(RollupConfig::default()));
    let head = test_block_info(1);
    let mut state = EngineState::default();
    let result = synchronize(
        &client,
        &mut state,
        EngineSyncStateUpdate { unsafe_head: Some(head), ..Default::default() },
    )
    .await;
    if accepted {
        assert_eq!(
            result.unwrap(),
            if status == PayloadStatusEnum::Valid {
                ForkchoiceStatus::Valid
            } else {
                ForkchoiceStatus::Syncing
            }
        );
        assert_eq!(state.sync_state.unsafe_head(), head);
    } else {
        assert!(matches!(result, Err(ForkchoiceError::UnexpectedStatus(_))));
        assert_eq!(state.sync_state, Default::default());
    }
    assert_eq!(state.el_sync_finished, status == PayloadStatusEnum::Valid);
    script.assert_finished();
}

#[rstest]
#[case::invalid_state(-38002)]
#[case::other_error(-32000)]
#[tokio::test]
async fn forkchoice_errors_keep_their_stage(#[case] code: i64) {
    let script = Script::new([ExpectedCall::error("engine_forkchoiceUpdatedV3", code)]);
    let client = script.client(Arc::new(RollupConfig::default()));
    let mut state = EngineState::default();
    let error = synchronize(&client, &mut state, Default::default()).await.unwrap_err();
    assert!(matches!(error, ForkchoiceError::InvalidState) == (code == -38002));
    assert_eq!(state, EngineState::default());
    script.assert_finished();
}

#[rstest]
#[case::accepted(PayloadStatusEnum::Accepted)]
#[case::syncing(PayloadStatusEnum::Syncing)]
#[tokio::test]
async fn derived_import_does_not_apply_unvalidated_payloads(#[case] status: PayloadStatusEnum) {
    let attrs = TestAttributesBuilder::new().build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let reply = payload(&mut cfg, &attrs);
    let script = Script::new([
        ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
        ExpectedCall::success("engine_getPayloadV3", reply),
        ExpectedCall::success("engine_newPayloadV3", PayloadStatus::from_status(status)),
    ]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = EngineState::default();
    let safe = state.sync_state.safe_head();
    let sink = Sink::default();
    assert!(matches!(
        consolidate(&client, &cfg, &mut state, &attrs, 0, &sink).await,
        Err(ConsolidateError::Insert(InsertError::Rejected(_)))
    ));
    assert_eq!(state.sync_state.safe_head(), safe);
    assert!(sink.0.lock().unwrap().is_empty());
    script.assert_finished();
}

#[rstest]
#[case::source_active(10, 20, true)]
#[case::source_inactive(20, 10, false)]
#[tokio::test]
async fn source_timestamp_controls_derived_replacement(
    #[case] l2_time: u64,
    #[case] source_time: u64,
    #[case] replace: bool,
) {
    let attrs = TestAttributesBuilder::new()
        .with_timestamp(l2_time)
        .with_transactions(vec![Bytes::from_static(&[0x02, 0x00])])
        .build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    cfg.hardforks.holocene_time = Some(15);
    let reply = payload(&mut cfg, &attrs);
    let mut calls = vec![ExpectedCall::success(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(invalid()),
    )];
    if replace {
        calls.extend([
            ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
            ExpectedCall::success("engine_getPayloadV3", reply),
            ExpectedCall::success("engine_newPayloadV3", valid()),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid())),
        ]);
    }
    let script = Script::new(calls);
    let client = script.client(Arc::new(cfg.clone()));
    let sink = Sink::default();
    let mut state = EngineState::default();
    let previous_safe = state.sync_state.safe_head();
    let outcome = consolidate(&client, &cfg, &mut state, &attrs, source_time, &sink).await.unwrap();
    assert_eq!(outcome, if replace { Consolidation::Replaced } else { Consolidation::Dropped });
    if replace {
        assert_eq!(state.sync_state.safe_head().block_info.id(), cfg.genesis.l2);
        assert_eq!(*sink.0.lock().unwrap(), [state.sync_state.safe_head()]);
        assert_eq!(
            script.calls()[1].params[1],
            serde_json::to_value(attrs.as_deposits_only().attributes()).unwrap()
        );
    } else {
        assert_eq!(state.sync_state.safe_head(), previous_safe);
        assert!(sink.0.lock().unwrap().is_empty());
    }
    script.assert_finished();
}

#[tokio::test]
async fn invalid_attributes_are_reconciled_before_a_pre_holocene_drop() {
    let attrs =
        TestAttributesBuilder::new().with_transactions(vec![Bytes::from_static(&[2])]).build();
    let cfg = RollupConfig::default();
    let script = Script::new([
        ExpectedCall::success("eth_getBlockByNumber", serde_json::Value::Null),
        ExpectedCall::error("engine_forkchoiceUpdatedV2", -38003),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid())),
    ]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(15))
        .with_safe_head(attrs.parent)
        .with_finalized_head(attrs.parent)
        .build();
    let previous_safe = state.sync_state.safe_head();
    let outcome = consolidate(&client, &cfg, &mut state, &attrs, 0, &NoopBlockSink).await.unwrap();
    assert_eq!(outcome, Consolidation::Dropped);
    assert_eq!(state.sync_state.safe_head(), previous_safe);
    assert_eq!(state.sync_state.unsafe_head(), attrs.parent);
    assert!(script.calls()[2].params[1].is_null());
    script.assert_finished();
}

#[rstest]
#[case::before_holocene(0)]
#[case::after_holocene(20)]
#[tokio::test]
async fn already_deposits_only_rejection_is_terminal(#[case] source_time: u64) {
    let attrs = TestAttributesBuilder::new().with_timestamp(10).build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.holocene_time = Some(15);
    let script = Script::new([ExpectedCall::success(
        "engine_forkchoiceUpdatedV2",
        ForkchoiceUpdated::new(invalid()),
    )]);
    let client = script.client(Arc::new(cfg.clone()));
    assert!(matches!(
        consolidate(
            &client,
            &cfg,
            &mut EngineState::default(),
            &attrs,
            source_time,
            &NoopBlockSink
        )
        .await,
        Err(ConsolidateError::DepositsOnlyRejected(_))
    ));
    script.assert_finished();
}

#[tokio::test]
async fn replacement_imports_actual_payload_once_and_rejects_a_second_invalid_payload() {
    let attrs =
        TestAttributesBuilder::new().with_transactions(vec![Bytes::from_static(&[2])]).build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    cfg.hardforks.holocene_time = Some(0);
    let reply = payload(&mut cfg, &attrs);
    for replacement_valid in [true, false] {
        let mut calls = vec![
            ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
            ExpectedCall::success("engine_getPayloadV3", reply.clone()),
            ExpectedCall::success("engine_newPayloadV3", invalid()),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
            ExpectedCall::success("engine_getPayloadV3", reply.clone()),
            ExpectedCall::success(
                "engine_newPayloadV3",
                if replacement_valid { valid() } else { invalid() },
            ),
        ];
        if replacement_valid {
            calls.push(ExpectedCall::success(
                "engine_forkchoiceUpdatedV3",
                ForkchoiceUpdated::new(valid()),
            ));
        }
        let script = Script::new(calls);
        let client = script.client(Arc::new(cfg.clone()));
        let sink = Sink::default();
        let mut state = EngineState::default();
        let result = consolidate(&client, &cfg, &mut state, &attrs, 0, &sink).await;
        if replacement_valid {
            assert_eq!(result.unwrap(), Consolidation::Replaced);
            assert_eq!(state.sync_state.safe_head().block_info.id(), cfg.genesis.l2);
            assert_eq!(*sink.0.lock().unwrap(), [state.sync_state.safe_head()]);
        } else {
            assert!(matches!(result, Err(ConsolidateError::DepositsOnlyRejected(_))));
            assert!(sink.0.lock().unwrap().is_empty());
            assert_eq!(state.sync_state.safe_head(), EngineState::default().sync_state.safe_head());
        }
        script.assert_finished();
    }
}

#[tokio::test]
async fn derived_parent_move_survives_a_later_get_payload_failure() {
    let attrs = TestAttributesBuilder::new().build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let script = Script::new([
        ExpectedCall::success("eth_getBlockByNumber", serde_json::Value::Null),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
        ExpectedCall::error("engine_getPayloadV3", -32000),
    ]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(15))
        .with_safe_head(attrs.parent)
        .with_finalized_head(attrs.parent)
        .build();
    assert!(matches!(
        consolidate(&client, &cfg, &mut state, &attrs, 0, &NoopBlockSink).await,
        Err(ConsolidateError::GetPayload(_))
    ));
    assert_eq!(state.sync_state.unsafe_head(), attrs.parent);
    script.assert_finished();
}

#[tokio::test]
async fn follow_safe_can_start_sync_without_a_local_body() {
    let safe = test_block_info(10);
    let script = Script::new([
        ExpectedCall::success("eth_getBlockByNumber", serde_json::Value::Null),
        ExpectedCall::success(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
        ),
    ]);
    let cfg = RollupConfig::default();
    let client = script.client(Arc::new(cfg.clone()));
    let sink = Sink::default();
    let mut state = EngineState::default();
    assert_eq!(
        follow_safe(&client, &cfg, &mut state, safe, &sink).await.unwrap(),
        Consolidation::Applied
    );
    assert_eq!(state.sync_state.safe_head(), safe);
    assert_eq!(state.sync_state.unsafe_head(), safe);
    assert!(!state.el_sync_finished);
    assert!(sink.0.lock().unwrap().is_empty());
    script.assert_finished();
}

#[tokio::test]
async fn same_head_reset_still_attempts_forkchoice() {
    let script = Script::new([ExpectedCall::success(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(valid()),
    )]);
    let client = script.client(Arc::new(RollupConfig::default()));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(1)).build();
    state.phase = EnginePhase::Ready;
    synchronize(&client, &mut state, Default::default()).await.unwrap();
    assert!(script.calls().is_empty());
    state.phase = EnginePhase::Resetting;
    synchronize(&client, &mut state, Default::default()).await.unwrap();
    assert_eq!(script.calls().len(), 1);
    script.assert_finished();
}

#[tokio::test]
async fn finalization_checks_canonical_hash_and_local_source_before_fcu() {
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    let block: Block<op_alloy_rpc_types::Transaction> = Block::new(
        Header::new(alloy_consensus::Header { number: 10, ..Default::default() }),
        BlockTransactions::Full(vec![]),
    );
    let mut cfg = RollupConfig::default();
    cfg.genesis.l2 = BlockNumHash { number: 10, hash: block.header.hash };
    let canonical =
        L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &cfg.genesis).unwrap();
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(canonical)
        .with_safe_head(canonical)
        .with_finalized_head(test_block_info(0))
        .build();
    let l1: Block = Block::new(
        Header::new(alloy_consensus::Header { number: 5, ..Default::default() }),
        BlockTransactions::Hashes(vec![]),
    );
    let source = BlockNumHash { number: 5, hash: l1.header.hash };
    let client = MockEngineClient::builder()
        .with_config(Arc::new(cfg.clone()))
        .with_l2_block(BlockId::number(10), block)
        .with_l1_block(BlockId::number(5), l1)
        .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(valid()))
        .build();
    let before = state;
    assert!(matches!(
        finalize(
            &client,
            &cfg,
            &mut state,
            BlockNumHash { hash: B256::ZERO, ..cfg.genesis.l2 },
            FinalizationSource::Delegated
        )
        .await,
        Err(FinalizeError::L2Mismatch(_))
    ));
    assert!(matches!(
        finalize(
            &client,
            &cfg,
            &mut state,
            cfg.genesis.l2,
            FinalizationSource::Local(BlockNumHash { hash: B256::ZERO, ..source })
        )
        .await,
        Err(FinalizeError::L1Mismatch(_))
    ));
    assert_eq!(state, before);
    finalize(&client, &cfg, &mut state, cfg.genesis.l2, FinalizationSource::Local(source))
        .await
        .unwrap();
    assert_eq!(state.sync_state.finalized_head(), canonical);
    let conflict = BlockNumHash { number: 10, hash: B256::ZERO };
    assert!(matches!(
        finalize(&client, &cfg, &mut state, conflict, FinalizationSource::Delegated).await,
        Err(FinalizeError::L2Mismatch(_))
    ));
    finalize(
        &client,
        &cfg,
        &mut state,
        BlockNumHash { number: 9, hash: B256::ZERO },
        FinalizationSource::Delegated,
    )
    .await
    .unwrap();
    assert_eq!(state.sync_state.finalized_head(), canonical);
}

#[tokio::test]
async fn unsafe_import_can_initiate_el_sync_without_claiming_safe() {
    let attrs = TestAttributesBuilder::new().build();
    let mut cfg = RollupConfig::default();
    let reply = payload(&mut cfg, &attrs);
    let envelope = OpExecutionPayloadEnvelope::V3 {
        payload: reply.execution_payload,
        parent_beacon_block_root: reply.parent_beacon_block_root,
    };
    let syncing = PayloadStatus::from_status(PayloadStatusEnum::Syncing);
    let script = Script::new([
        ExpectedCall::success("engine_newPayloadV3", syncing.clone()),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(syncing)),
    ]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = EngineState::default();
    let old_safe = state.sync_state.safe_head();
    let sink = Sink::default();
    insert_payload(&client, &cfg, &mut state, &envelope, ImportMode::Unsafe, &sink).await.unwrap();
    assert_eq!(state.sync_state.unsafe_head().block_info.id(), cfg.genesis.l2);
    assert_eq!(state.sync_state.safe_head(), old_safe);
    assert!(!state.el_sync_finished);
    script.assert_finished();
}

#[rstest]
#[case::pre_holocene_prefix(false, false)]
#[case::pre_holocene_conclusion(true, false)]
#[case::holocene_prefix(false, true)]
#[tokio::test]
async fn matching_consolidation_tracks_pending_and_completed_safe(
    #[case] last_in_span: bool,
    #[case] holocene: bool,
) {
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    let mut attrs = TestAttributesBuilder::new().build();
    attrs.is_last_in_span = last_in_span;
    let block: Block<op_alloy_rpc_types::Transaction> = Block::new(
        Header::new(alloy_consensus::Header {
            number: attrs.block_number(),
            parent_hash: attrs.parent.block_info.hash,
            timestamp: attrs.attributes.payload_attributes.timestamp,
            mix_hash: attrs.attributes.payload_attributes.prev_randao,
            beneficiary: attrs.attributes.payload_attributes.suggested_fee_recipient,
            gas_limit: attrs.attributes.gas_limit.unwrap(),
            parent_beacon_block_root: attrs.attributes.payload_attributes.parent_beacon_block_root,
            ..Default::default()
        }),
        BlockTransactions::Full(vec![]),
    );
    let mut cfg = RollupConfig::default();
    cfg.genesis.l2 = BlockNumHash { number: block.header.number, hash: block.header.hash };
    let source_time = attrs.attributes.payload_attributes.timestamp + 1;
    if holocene {
        cfg.hardforks.holocene_time = Some(source_time);
    }
    assert!(crate::AttributesMatch::check(&cfg, &attrs, &block).is_match());
    let concluding = last_in_span || holocene;
    let mut calls = vec![ExpectedCall::success("eth_getBlockByNumber", block)];
    if concluding {
        calls.push(ExpectedCall::success(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(valid()),
        ));
    } else {
        // A later rejected block in the span must discard this prefix without flushing
        // alternatives.
        calls.extend([
            ExpectedCall::success("eth_getBlockByNumber", serde_json::Value::Null),
            ExpectedCall::success("engine_forkchoiceUpdatedV2", ForkchoiceUpdated::new(invalid())),
        ]);
    }
    let script = Script::new(calls);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(15))
        .with_safe_head(attrs.parent)
        .with_finalized_head(attrs.parent)
        .build();
    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(attrs.parent),
        ..Default::default()
    });
    let sink = Sink::default();
    assert_eq!(
        consolidate(&client, &cfg, &mut state, &attrs, source_time, &sink).await.unwrap(),
        Consolidation::Applied
    );
    let prefix = state.pending_safe_head;
    assert_eq!(prefix.block_info.id(), cfg.genesis.l2);
    assert_eq!(*sink.0.lock().unwrap(), [prefix]);
    if concluding {
        assert_eq!(state.sync_state.safe_head(), prefix);
        assert_eq!(state.sync_state.local_safe_head(), prefix);
    } else {
        assert_eq!(state.sync_state.safe_head(), attrs.parent);
        assert_eq!(state.sync_state.local_safe_head(), attrs.parent);
        assert!(matches!(
            finalize(
                &client,
                &cfg,
                &mut state,
                prefix.block_info.id(),
                FinalizationSource::Delegated
            )
            .await,
            Err(FinalizeError::NotSafe)
        ));
        let rejected = TestAttributesBuilder::new()
            .with_parent(prefix)
            .with_timestamp(attrs.attributes.payload_attributes.timestamp + cfg.block_time)
            .with_transactions(vec![Bytes::from_static(&[2])])
            .build();
        assert_eq!(
            consolidate(&client, &cfg, &mut state, &rejected, source_time, &sink).await.unwrap(),
            Consolidation::Dropped
        );
        assert_eq!(state.pending_safe_head, attrs.parent);
        assert_eq!(state.sync_state.safe_head(), attrs.parent);
        assert_eq!(*sink.0.lock().unwrap(), [prefix]);
    }
    script.assert_finished();
}

#[rstest]
#[case::prefix(false)]
#[case::conclusion(true)]
#[tokio::test]
async fn derived_import_advances_only_its_completed_safety_level(#[case] concluding: bool) {
    let attrs = TestAttributesBuilder::new().build();
    let mut cfg = RollupConfig::default();
    let reply = payload(&mut cfg, &attrs);
    let envelope = OpExecutionPayloadEnvelope::V3 {
        payload: reply.execution_payload,
        parent_beacon_block_root: reply.parent_beacon_block_root,
    };
    let script = Script::new([
        ExpectedCall::success("engine_newPayloadV3", valid()),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid())),
    ]);
    let client = script.client(Arc::new(cfg.clone()));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(attrs.parent).build();
    let accepted = insert_payload(
        &client,
        &cfg,
        &mut state,
        &envelope,
        ImportMode::Derived { concluding },
        &NoopBlockSink,
    )
    .await
    .unwrap();
    assert_eq!(state.pending_safe_head, accepted);
    assert_eq!(state.sync_state.unsafe_head(), accepted);
    let completed = if concluding { accepted } else { attrs.parent };
    assert_eq!(state.sync_state.safe_head(), completed);
    assert_eq!(state.sync_state.local_safe_head(), completed);
    assert_eq!(
        script.calls()[1].params[0]["safeBlockHash"],
        serde_json::to_value(completed.hash()).unwrap()
    );
    script.assert_finished();
}
