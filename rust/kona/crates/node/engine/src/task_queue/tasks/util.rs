//! Shared steps for building, sealing, and importing blocks.

use super::{BuildTaskError, SealTaskError, get_payload, insert_payload, start_build};
use crate::{EngineClient, EngineState, ImportedBlockSink, InsertTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Error type for build and import operations.
#[derive(Debug, thiserror::Error)]
pub(in crate::task_queue) enum BuildAndImportError {
    /// An error occurred during the build phase.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// An error occurred while sealing or importing the block.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
}

/// Builds a block from derived `attributes`, seals it, and imports it as the new safe head.
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
            return Err(handle_rejected_attributes(
                engine,
                cfg,
                state,
                &attributes,
                true,
                block_sink,
            )
            .await
            .map_or(BuildAndImportError::Build(err), BuildAndImportError::Seal));
        }
        result => result?,
    };

    let payload =
        get_payload(engine, cfg, payload_id, attributes.attributes().payload_attributes.timestamp)
            .await?;
    let new_block_ref = insert_payload_with_holocene_fallback(
        engine,
        cfg,
        state,
        &attributes,
        payload,
        // Derived blocks are safe.
        true,
        block_sink,
    )
    .await?;

    info!(
        target: "engine",
        l2_number = new_block_ref.block_info.number,
        l2_time = new_block_ref.block_info.timestamp,
        "Built and imported new safe block",
    );

    Ok(())
}

/// Inserts a `payload` built from `attributes` into the engine. If the engine rejects the payload
/// after Holocene and the payload was derived, it is replaced with a deposits-only block.
///
/// A successful replacement is reported as [`SealTaskError::HoloceneInvalidFlush`], whose
/// severity tells the engine to flush the derivation pipeline's current channel. A rejected
/// deposits-only payload has no replacement and fails with
/// [`SealTaskError::DepositOnlyPayloadFailed`].
pub(in crate::task_queue) async fn insert_payload_with_holocene_fallback(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    payload: OpExecutionPayloadEnvelope,
    is_payload_safe: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, SealTaskError> {
    match insert_payload(engine, cfg, state, payload, is_payload_safe, block_sink).await {
        Err(err @ InsertTaskError::UnexpectedPayloadStatus(_)) => {
            warn!(target: "engine", %err, "Payload rejected at newPayload");
            Err(handle_rejected_attributes(
                engine,
                cfg,
                state,
                attributes,
                is_payload_safe,
                block_sink,
            )
            .await
            .unwrap_or_else(|| SealTaskError::PayloadInsertionFailed(Box::new(err))))
        }
        Err(e) => {
            error!(target: "engine", "Payload import failed: {e}");
            Err(Box::new(e).into())
        }
        Ok(new_block_ref) => Ok(new_block_ref),
    }
}

/// Applies the deposits-only and Holocene recovery rules to rejected attributes.
/// Sequenced payloads keep their rejection; only derived payloads receive a replacement.
async fn handle_rejected_attributes(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    is_attributes_derived: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Option<SealTaskError> {
    if attributes.is_deposits_only() {
        error!(target: "engine", "Critical: the engine rejected deposits-only attributes");
        return Some(SealTaskError::DepositOnlyPayloadFailed);
    }
    if !is_attributes_derived ||
        !cfg.is_holocene_active(attributes.attributes().payload_attributes.timestamp)
    {
        return None;
    }
    warn!(target: "engine", "Replacing rejected derived attributes with deposits only");
    Some(
        match build_seal_and_insert(
            engine,
            cfg,
            state,
            attributes.as_deposits_only(),
            true,
            block_sink,
        )
        .await
        {
            Ok(block) => {
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
    is_payload_safe: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, BuildAndImportError> {
    let timestamp = attributes.attributes().payload_attributes.timestamp;
    let payload_id = start_build(engine, cfg, state, attributes).await?;
    let payload = get_payload(engine, cfg, payload_id, timestamp).await?;
    insert_payload(engine, cfg, state, payload, is_payload_safe, block_sink)
        .await
        .map_err(|err| SealTaskError::PayloadInsertionFailed(Box::new(err)).into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        NoopBlockSink,
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
            true,
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
}
