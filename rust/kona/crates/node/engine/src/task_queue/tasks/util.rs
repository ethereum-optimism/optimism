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
    let payload_id = start_build(engine, cfg, state, attributes.clone()).await?;

    let payload =
        get_payload(engine, cfg, payload_id, attributes.attributes().payload_attributes.timestamp)
            .await?;
    let new_block_ref =
        import_payload(engine, cfg, state, &attributes, payload, true, block_sink).await?;

    info!(
        target: "engine",
        l2_number = new_block_ref.block_info.number,
        l2_time = new_block_ref.block_info.timestamp,
        "Built and imported new safe block",
    );

    Ok(())
}

/// Inserts a `payload` built from `attributes` into the engine. If the engine rejects the payload
/// after Holocene, it is replaced with a block built from only the deposits in `attributes`.
///
/// A successful replacement is reported as [`SealTaskError::HoloceneInvalidFlush`], whose
/// severity tells the engine to flush the derivation pipeline's current channel.
pub(in crate::task_queue) async fn import_payload(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    payload: OpExecutionPayloadEnvelope,
    is_attributes_derived: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, SealTaskError> {
    match insert_payload(engine, cfg, state, payload, is_attributes_derived, block_sink).await {
        Err(InsertTaskError::UnexpectedPayloadStatus(e)) if attributes.is_deposits_only() => {
            error!(target: "engine", error = ?e, "Critical: Deposit-only payload import failed");
            Err(SealTaskError::DepositOnlyPayloadFailed)
        }
        Err(InsertTaskError::UnexpectedPayloadStatus(e))
            if cfg.is_holocene_active(attributes.attributes().payload_attributes.timestamp) =>
        {
            warn!(target: "engine", error = ?e, "Re-attempting payload import with deposits only.");

            // HOLOCENE: Re-attempt payload import with deposits only. The deposits-only block gets
            // no further fallback: if it fails to import, nothing can replace it.
            match build_seal_and_insert(
                engine,
                cfg,
                state,
                attributes.as_deposits_only(),
                is_attributes_derived,
                block_sink,
            )
            .await
            {
                Ok(_) => {
                    info!(target: "engine", "Successfully imported deposits-only payload");
                    Err(SealTaskError::HoloceneInvalidFlush)
                }
                Err(err) => {
                    error!(target: "engine", ?err, "Deposits-only payload re-attempt failed");
                    Err(SealTaskError::DepositOnlyPayloadReattemptFailed)
                }
            }
        }
        Err(e) => {
            error!(target: "engine", "Payload import failed: {e}");
            Err(Box::new(e).into())
        }
        Ok(new_block_ref) => {
            info!(target: "engine", "Successfully imported payload");
            Ok(new_block_ref)
        }
    }
}

/// Builds a block from `attributes`, seals it, and inserts it, without the Holocene fallback.
async fn build_seal_and_insert(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: OpAttributesWithParent,
    is_attributes_derived: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, BuildAndImportError> {
    let timestamp = attributes.attributes().payload_attributes.timestamp;
    let payload_id = start_build(engine, cfg, state, attributes).await?;
    let payload = get_payload(engine, cfg, payload_id, timestamp).await?;
    insert_payload(engine, cfg, state, payload, is_attributes_derived, block_sink)
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
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum};
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
}
