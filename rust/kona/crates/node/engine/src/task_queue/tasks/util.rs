//! Shared steps for building, sealing, and importing blocks.

use super::{EngineBuildError, SealTaskError, get_payload, insert_payload, start_build};
use crate::{EngineClient, EngineState, ImportedBlockSink, InsertTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Error type for build and import operations.
#[derive(Debug, thiserror::Error)]
pub(in crate::task_queue) enum BuildAndImportError {
    /// An error occurred during the build phase.
    #[error(transparent)]
    Build(#[from] EngineBuildError),
    /// An error occurred while sealing or importing the block.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
}

/// Builds and imports derived `attributes` as the next derivation parent. Only the last block
/// in a pre-Holocene span advances the completed safe head.
///
/// Build and import run inside the same engine task, so no other task can move the unsafe head in
/// between, and no staleness check is needed. A derivation-driven reorg legitimately builds on a
/// parent that differs from the current unsafe head.
pub(in crate::task_queue) async fn build_and_import(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: OpAttributesWithParent,
    block_sink: &dyn ImportedBlockSink,
) -> Result<(), BuildAndImportError> {
    let payload_id = match start_build(engine, cfg, state, attributes.clone()).await {
        Err(err) if err.is_invalid_attributes() => {
            warn!(target: "engine", %err, "Derived attributes rejected at forkchoiceUpdated");
            return Err(handle_rejected_attributes(engine, cfg, state, &attributes, block_sink)
                .await
                .map_or(BuildAndImportError::Build(err), BuildAndImportError::Seal));
        }
        result => result?,
    };

    let payload =
        get_payload(engine, cfg, payload_id, attributes.attributes().payload_attributes.timestamp)
            .await?;
    let new_block_ref =
        insert_payload_with_holocene_fallback(engine, cfg, state, &attributes, payload, block_sink)
            .await?;

    info!(
        target: "engine",
        l2_number = new_block_ref.block_info.number,
        l2_time = new_block_ref.block_info.timestamp,
        "Built and imported derived block",
    );

    Ok(())
}

/// Inserts a `payload` built from `attributes` into the engine. If the engine rejects the payload
/// after Holocene and the payload was derived, it is replaced with a deposits-only block.
///
/// A successful replacement is reported as [`SealTaskError::HoloceneInvalidFlush`], whose
/// severity tells the engine to flush the derivation pipeline's current channel. A rejected
/// deposits-only payload has no replacement and fails with
/// [`SealTaskError::DepositOnlyPayloadFailed`]. Sequenced blocks never take this fallback: a
/// replacement block would need its own conductor commit.
async fn insert_payload_with_holocene_fallback(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    payload: OpExecutionPayloadEnvelope,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, SealTaskError> {
    // Holocene validates singular blocks independently, including blocks inside a span.
    let completes_batch = attributes.is_last_in_span || attributes.is_holocene_active(cfg);
    match insert_payload(engine, cfg, state, payload, completes_batch, block_sink).await {
        Err(err @ InsertTaskError::UnexpectedPayloadStatus(_)) => {
            warn!(target: "engine", %err, "Payload rejected at newPayload");
            Err(handle_rejected_attributes(engine, cfg, state, attributes, block_sink)
                .await
                .unwrap_or_else(|| SealTaskError::PayloadInsertionFailed(Box::new(err))))
        }
        Err(e) => {
            error!(target: "engine", "Payload import failed: {e}");
            Err(Box::new(e).into())
        }
        Ok(new_block_ref) => {
            state.pending_safe_head = (!completes_batch).then_some(new_block_ref);
            Ok(new_block_ref)
        }
    }
}

/// Applies the deposits-only and Holocene recovery rules to rejected attributes.
/// Sequenced payloads keep their rejection; only derived payloads receive a replacement.
async fn handle_rejected_attributes(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    block_sink: &dyn ImportedBlockSink,
) -> Option<SealTaskError> {
    if attributes.is_deposits_only() {
        error!(target: "engine", "Critical: the engine rejected deposits-only attributes");
        return Some(SealTaskError::DepositOnlyPayloadFailed);
    }
    if !attributes.is_holocene_active(cfg) {
        return None;
    }
    warn!(target: "engine", "Replacing rejected derived attributes with deposits only");
    Some(
        match build_seal_and_insert(engine, cfg, state, attributes.as_deposits_only(), block_sink)
            .await
        {
            Ok(block) => {
                state.pending_safe_head = None;
                info!(
                    target: "engine",
                    l2_number = block.block_info.number,
                    hash = %block.block_info.hash,
                    "Successfully imported deposits-only payload"
                );
                SealTaskError::HoloceneInvalidFlush
            }
            Err(err) => {
                error!(target: "engine", %err, "Critical: deposits-only replacement failed");
                SealTaskError::DepositOnlyPayloadReattemptFailed
            }
        },
    )
}

/// Builds a block from `attributes`, seals it, and inserts it, without the Holocene fallback.
async fn build_seal_and_insert(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: OpAttributesWithParent,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, BuildAndImportError> {
    let timestamp = attributes.attributes().payload_attributes.timestamp;
    let payload_id = start_build(engine, cfg, state, attributes).await?;
    let payload = get_payload(engine, cfg, payload_id, timestamp).await?;
    insert_payload(engine, cfg, state, payload, true, block_sink)
        .await
        .map_err(|err| SealTaskError::PayloadInsertionFailed(Box::new(err)).into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineTaskError, EngineTaskErrorSeverity, NoopBlockSink,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
        },
    };
    use alloy_consensus::Block;
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3, ForkchoiceUpdated, PayloadId,
        PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_consensus::{OpTxEnvelope, OpTxType};
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
    use rstest::rstest;
    use std::sync::Arc;

    /// A derivation-driven reorg builds on a parent other than the current unsafe head. Unlike a
    /// queued seal, the build and import must proceed instead of aborting as stale.
    #[tokio::test]
    async fn build_and_import_proceeds_on_a_parent_other_than_the_unsafe_head() {
        let parent = test_block_info(10);
        let attributes = TestAttributesBuilder::new().with_parent(parent).build();
        let mut cfg = RollupConfig::default();
        // Use `engine_forkchoiceUpdatedV3` to start the build.
        cfg.hardforks.ecotone_time = Some(attributes.attributes().payload_attributes.timestamp);
        let (engine, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated {
                payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                payload_id: Some(PayloadId::new([1u8; 8])),
            },
        );
        l2.expect_error("engine_getPayloadV3");
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(15)).build();

        let result = build_and_import(&engine, &cfg, &mut state, attributes, &NoopBlockSink).await;

        // The injected payload-fetch failure shows the build was not aborted as stale.
        assert!(
            matches!(result, Err(BuildAndImportError::Seal(SealTaskError::GetPayloadFailed(_)))),
            "{result:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Recovery must import the replacement and request a channel flush at either rejection point.
    #[rstest]
    #[tokio::test]
    async fn successful_replacement_still_flushes(#[values(true, false)] rejected_at_build: bool) {
        let mut block = Block::<OpTxEnvelope>::default();
        block.header.base_fee_per_gas = Some(0);
        block.header.withdrawals_root = Some(alloy_consensus::EMPTY_ROOT_HASH);
        block.header.parent_beacon_block_root = Some(Default::default());
        block.header.blob_gas_used = Some(0);
        block.header.excess_blob_gas = Some(0);
        let payload = OpExecutionPayloadEnvelope::V3 {
            payload: ExecutionPayloadV3::from_block_slow(&block),
            parent_beacon_block_root: Default::default(),
        };
        let attributes = TestAttributesBuilder::new()
            .with_transactions(vec![Bytes::from(vec![OpTxType::Legacy as u8, 0xaa])])
            .build();
        let mut cfg = RollupConfig::default();
        cfg.hardforks.holocene_time = Some(0);
        // Pin the synthetic replacement to genesis so decoding needs no L1-info transaction.
        cfg.genesis.l2 =
            alloy_eips::BlockNumHash::new(payload.block_number(), payload.block_hash());
        let (engine, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        if rejected_at_build {
            l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        } else {
            l2.expect(
                "engine_newPayloadV3",
                PayloadStatus::from_status(PayloadStatusEnum::Invalid {
                    validation_error: "invalid".into(),
                }),
            );
        }
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated { payload_status: valid(), payload_id: Some(PayloadId::new([1; 8])) },
        );
        l2.expect(
            "engine_getPayloadV3",
            OpExecutionPayloadEnvelopeV3 {
                execution_payload: ExecutionPayloadV3::from_block_slow(&block),
                block_value: Default::default(),
                blobs_bundle: Default::default(),
                should_override_builder: false,
                parent_beacon_block_root: Default::default(),
            },
        );
        l2.expect("engine_newPayloadV3", valid());
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated { payload_status: valid(), payload_id: None },
        );
        let mut state = EngineState::default();

        let err = if rejected_at_build {
            match build_and_import(&engine, &cfg, &mut state, attributes, &NoopBlockSink).await {
                Err(BuildAndImportError::Seal(err)) => err,
                other => panic!("expected a seal error after recovery, got {other:?}"),
            }
        } else {
            insert_payload_with_holocene_fallback(
                &engine,
                &cfg,
                &mut state,
                &attributes,
                payload.clone(),
                &NoopBlockSink,
            )
            .await
            .expect_err("successful recovery must request a flush")
        };

        assert!(matches!(err, SealTaskError::HoloceneInvalidFlush), "{err:?}");
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Flush);
        assert_eq!(state.sync_state.unsafe_head().block_info.hash, payload.block_hash());
        assert_eq!(state.sync_state.safe_head(), state.sync_state.unsafe_head());
        assert_eq!(state.sync_state.local_safe_head(), state.sync_state.unsafe_head());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// How the engine responds when the payload is rejected and the Holocene fallback is or is not
    /// available.
    #[derive(Debug)]
    enum Fallback {
        /// The rejected payload was already deposits-only, so nothing can replace it.
        DepositsOnlyRejected,
        /// Holocene is not active, so the rejection is returned as is.
        PreHolocene,
        /// Holocene is active, and the deposits-only replacement is rejected too.
        ReplacementRejected,
    }

    #[rstest]
    #[case::deposits_only_rejected(Fallback::DepositsOnlyRejected)]
    #[case::pre_holocene(Fallback::PreHolocene)]
    #[case::replacement_rejected(Fallback::ReplacementRejected)]
    #[tokio::test]
    async fn rejected_payload_fallback(#[case] fallback: Fallback) {
        let payload = ExecutionPayloadV1::from_block_slow(&Block::<OpTxEnvelope>::default());
        let mut attributes = TestAttributesBuilder::new().with_parent(test_block_info(0));
        if !matches!(fallback, Fallback::DepositsOnlyRejected) {
            attributes =
                attributes.with_transactions(vec![Bytes::from(vec![OpTxType::Legacy as u8, 0xaa])]);
        }
        let attributes = attributes.build();
        let mut cfg = RollupConfig::default();
        if !matches!(fallback, Fallback::PreHolocene) {
            cfg.hardforks.holocene_time = Some(0);
        }
        let (engine, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        let invalid = || {
            PayloadStatus::from_status(PayloadStatusEnum::Invalid {
                validation_error: "invalid".into(),
            })
        };
        l2.expect("engine_newPayloadV1", invalid());
        if matches!(fallback, Fallback::ReplacementRejected) {
            l2.expect(
                "engine_forkchoiceUpdatedV3",
                ForkchoiceUpdated {
                    payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                    payload_id: Some(PayloadId::new([1u8; 8])),
                },
            );
            l2.expect(
                "engine_getPayloadV3",
                OpExecutionPayloadEnvelopeV3 {
                    execution_payload: ExecutionPayloadV3 {
                        payload_inner: ExecutionPayloadV2 {
                            payload_inner: payload.clone(),
                            withdrawals: vec![],
                        },
                        blob_gas_used: 0,
                        excess_blob_gas: 0,
                    },
                    block_value: Default::default(),
                    blobs_bundle: Default::default(),
                    should_override_builder: false,
                    parent_beacon_block_root: Default::default(),
                },
            );
            l2.expect("engine_newPayloadV3", invalid());
        }
        let mut state = EngineState::default();

        let result = insert_payload_with_holocene_fallback(
            &engine,
            &cfg,
            &mut state,
            &attributes,
            OpExecutionPayloadEnvelope::V1(payload),
            &NoopBlockSink,
        )
        .await;

        match fallback {
            Fallback::DepositsOnlyRejected => {
                assert!(
                    matches!(result, Err(SealTaskError::DepositOnlyPayloadFailed)),
                    "{result:?}"
                )
            }
            Fallback::PreHolocene => assert!(
                matches!(
                    &result,
                    Err(SealTaskError::PayloadInsertionFailed(err))
                        if matches!(**err, InsertTaskError::UnexpectedPayloadStatus(_))
                ),
                "{result:?}"
            ),
            Fallback::ReplacementRejected => assert!(
                matches!(result, Err(SealTaskError::DepositOnlyPayloadReattemptFailed)),
                "{result:?}"
            ),
        }
        assert_eq!(state, EngineState::default(), "a rejected payload must not change forkchoice");
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Derived blocks are imported as the new safe head in the same operation that builds them.
    #[rstest]
    #[case::completed_batch(true, false)]
    #[case::holocene_prefix(false, true)]
    #[tokio::test]
    async fn build_and_import_inserts_the_block_as_the_safe_head(
        #[case] last_in_span: bool,
        #[case] holocene: bool,
    ) {
        use super::super::canonicalize::tests::{PayloadFixture, RecordingSink, payload_fixture};
        use alloy_rpc_types_engine::{ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2};

        let PayloadFixture { payload: expected, mut cfg } = payload_fixture();
        if holocene {
            Arc::make_mut(&mut cfg).hardforks.holocene_time = Some(10);
        }
        let op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(payload) = expected.clone()
        else {
            panic!("fixture must be V1");
        };
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        let (engine, l1, l2) = test_engine_client(cfg.clone());
        l2.expect(
            "engine_forkchoiceUpdatedV2",
            ForkchoiceUpdated {
                payload_status: valid(),
                payload_id: Some(PayloadId::new([1u8; 8])),
            },
        );
        l2.expect(
            "engine_getPayloadV2",
            ExecutionPayloadEnvelopeV2 {
                execution_payload: ExecutionPayloadFieldV2::V1(payload),
                block_value: Default::default(),
            },
        );
        l2.expect("engine_newPayloadV1", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        let sink = RecordingSink::default();
        let mut state = EngineState::default();
        let mut attributes = TestAttributesBuilder::new()
            .with_parent(kona_protocol::L2BlockInfo::default())
            .with_timestamp(0)
            .build();
        attributes.is_last_in_span = last_in_span;
        if holocene {
            attributes.derived_from =
                Some(kona_protocol::BlockInfo { timestamp: 10, ..Default::default() });
        }

        build_and_import(&engine, &cfg, &mut state, attributes, &sink).await.unwrap();

        assert_eq!(state.sync_state.unsafe_head().block_info.hash, expected.block_hash());
        assert_eq!(state.sync_state.safe_head().block_info.hash, expected.block_hash());
        assert_eq!(sink.0.lock().unwrap().len(), 1);
        l1.assert_finished();
        l2.assert_finished();
    }
}
