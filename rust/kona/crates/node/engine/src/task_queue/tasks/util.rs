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
    engine: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: OpAttributesWithParent,
    block_sink: &dyn ImportedBlockSink,
) -> Result<(), BuildAndImportError> {
    let payload = build_payload(engine, cfg, state, attributes.clone()).await?;
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

/// Inserts a `payload` built from `attributes` into the engine with [`insert_payload`].
///
/// This preserves the existing fallback policy: `UnexpectedPayloadStatus` (including `INVALID`
/// and `ACCEPTED`) triggers a deposits-only replacement when Holocene is active at the L2
/// attributes timestamp. This applies to both derived and sequenced payloads.
///
/// A successful replacement is reported as [`SealTaskError::HoloceneInvalidFlush`], whose
/// severity tells the engine to flush the derivation pipeline's current channel. A rejected
/// deposits-only payload has no replacement and fails with
/// [`SealTaskError::DepositOnlyPayloadFailed`].
pub(in crate::task_queue) async fn insert_payload_with_holocene_fallback(
    engine: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    payload: OpExecutionPayloadEnvelope,
    is_payload_safe: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, SealTaskError> {
    match insert_payload(engine, cfg, state, payload, is_payload_safe, block_sink).await {
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
            let replacement: Result<_, BuildAndImportError> = async {
                let payload =
                    build_payload(engine, cfg, state, attributes.as_deposits_only()).await?;
                insert_payload(engine, cfg, state, payload, is_payload_safe, block_sink)
                    .await
                    .map_err(|error| SealTaskError::PayloadInsertionFailed(Box::new(error)).into())
            }
            .await;
            match replacement {
                Ok(block) => {
                    info!(
                        target: "engine",
                        l2_number = block.block_info.number,
                        hash = %block.block_info.hash,
                        "Successfully imported deposits-only payload"
                    );
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

/// Starts and retrieves a payload; callers decide how to import it and handle rejection.
async fn build_payload(
    engine: &impl EngineClient,
    cfg: &RollupConfig,
    state: &EngineState,
    attributes: OpAttributesWithParent,
) -> Result<OpExecutionPayloadEnvelope, BuildAndImportError> {
    let timestamp = attributes.attributes().payload_attributes.timestamp;
    let payload_id = start_build(engine, cfg, state, attributes).await?;
    get_payload(engine, cfg, payload_id, timestamp).await.map_err(Into::into)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        ConsolidateTask, EngineTaskError, EngineTaskErrorSeverity, EngineTaskExt,
        task_queue::tasks::test_utils::{ExpectedCall, Script},
        test_utils::{TestAttributesBuilder, test_block_info},
    };
    use alloy_primitives::{B256, Bytes};
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3, ForkchoiceUpdated, PayloadId,
        PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_consensus::{OpBlock, OpTxEnvelope};
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
    use rstest::rstest;
    use std::sync::{Arc, Mutex};

    #[derive(Debug, Default)]
    struct Sink(Mutex<Vec<L2BlockInfo>>);

    impl ImportedBlockSink for Sink {
        fn block_imported(&self, _: OpBlock, block: L2BlockInfo) {
            self.0.lock().unwrap().push(block);
        }
    }

    fn valid_forkchoice() -> ForkchoiceUpdated {
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid))
    }

    fn build_started() -> ForkchoiceUpdated {
        ForkchoiceUpdated { payload_id: Some(PayloadId::new([1; 8])), ..valid_forkchoice() }
    }

    fn invalid() -> PayloadStatus {
        PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "rejected".into(),
        })
    }

    /// Pin the fetched block as genesis so this fixture needs no synthetic L1-info transaction.
    fn replacement(
        cfg: &mut RollupConfig,
        attributes: &OpAttributesWithParent,
    ) -> OpExecutionPayloadEnvelopeV3 {
        let mut block = OpBlock::default();
        block.header.number = attributes.block_number();
        block.header.parent_hash = attributes.parent.block_info.hash;
        block.header.timestamp = attributes.attributes().payload_attributes.timestamp;
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
        cfg.genesis.l2 = alloy_eips::BlockNumHash {
            number: block.header.number,
            hash: block.header.hash_slow(),
        };
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

    #[tokio::test]
    async fn mismatching_consolidation_can_reorg_a_different_unsafe_head() {
        let attributes = TestAttributesBuilder::new().with_parent(test_block_info(0)).build();
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        let payload = replacement(&mut cfg, &attributes);
        let mismatching_block = alloy_rpc_types_eth::Block::<op_alloy_rpc_types::Transaction>::new(
            alloy_rpc_types_eth::Header::new(alloy_consensus::Header::default()),
            alloy_rpc_types_eth::BlockTransactions::Full(vec![]),
        );
        assert_ne!(
            mismatching_block.header.timestamp,
            attributes.attributes().payload_attributes.timestamp
        );
        let script = Script::new([
            ExpectedCall::success("eth_getBlockByNumber", mismatching_block),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", build_started()),
            ExpectedCall::success("engine_getPayloadV3", payload),
            ExpectedCall::success(
                "engine_newPayloadV3",
                PayloadStatus::from_status(PayloadStatusEnum::Valid),
            ),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", valid_forkchoice()),
        ]);
        let cfg = Arc::new(cfg);
        let engine = Arc::new(script.client(cfg.clone()));
        let mut state = EngineState::default();
        state.sync_state = state.sync_state.apply_update(crate::EngineSyncStateUpdate {
            unsafe_head: Some(test_block_info(15)),
            ..Default::default()
        });
        let sink = Arc::new(Sink::default());
        ConsolidateTask::new(engine, cfg.clone(), attributes.into(), sink.clone())
            .execute(&mut state)
            .await
            .unwrap();
        assert_eq!(state.sync_state.unsafe_head().block_info.id(), cfg.genesis.l2);
        assert_eq!(state.sync_state.safe_head(), state.sync_state.unsafe_head());
        assert_eq!(state.sync_state.local_safe_head(), state.sync_state.unsafe_head());
        assert_eq!(*sink.0.lock().unwrap(), [state.sync_state.unsafe_head()]);
        script.assert_finished();
    }

    #[rstest]
    #[case::derived(true)]
    #[case::sequenced(false)]
    #[tokio::test]
    async fn successful_replacement_commits_before_returning_flush(#[case] is_payload_safe: bool) {
        let mut attributes = TestAttributesBuilder::new()
            .with_parent(test_block_info(0))
            .with_transactions(vec![
                Bytes::from_static(&[0x7e, 0x00]),
                Bytes::from_static(&[0x02, 0x00]),
            ])
            .build();
        attributes.derived_from = Some(Default::default());
        attributes.is_last_in_span = true;
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        cfg.hardforks.holocene_time = Some(0);
        let payload = replacement(&mut cfg, &attributes);
        let script = Script::new([
            ExpectedCall::success("engine_newPayloadV1", invalid()),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", build_started()),
            ExpectedCall::success("engine_getPayloadV3", payload),
            ExpectedCall::success(
                "engine_newPayloadV3",
                PayloadStatus::from_status(PayloadStatusEnum::Valid),
            ),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", valid_forkchoice()),
        ]);
        let engine = script.client(Arc::new(cfg.clone()));
        let mut state = EngineState::default();
        state.sync_state = state.sync_state.apply_update(crate::EngineSyncStateUpdate {
            unsafe_head: Some(test_block_info(15)),
            ..Default::default()
        });
        let old_safe = state.sync_state.safe_head();
        let sink = Sink::default();
        let original = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &OpBlock::default(),
        ));
        let error = insert_payload_with_holocene_fallback(
            &engine,
            &cfg,
            &mut state,
            &attributes,
            original,
            is_payload_safe,
            &sink,
        )
        .await
        .unwrap_err();
        assert!(matches!(error, SealTaskError::HoloceneInvalidFlush));
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Flush);
        assert_eq!(state.sync_state.unsafe_head().block_info.id(), cfg.genesis.l2);
        let safe = if is_payload_safe { state.sync_state.unsafe_head() } else { old_safe };
        assert_eq!(state.sync_state.safe_head(), safe);
        assert_eq!(state.sync_state.local_safe_head(), safe);
        assert_eq!(*sink.0.lock().unwrap(), [state.sync_state.unsafe_head()]);
        let calls = script.calls();
        let deposits = attributes.as_deposits_only();
        assert_eq!(calls[1].params[1], serde_json::to_value(deposits.attributes()).unwrap());
        assert_eq!(deposits.parent, attributes.parent);
        assert_eq!(deposits.derived_from, attributes.derived_from);
        assert_eq!(deposits.is_last_in_span, attributes.is_last_in_span);
        assert_eq!(calls[4].params[1], serde_json::Value::Null);
        script.assert_finished();
    }

    #[derive(Clone, Copy, Debug)]
    enum Rejection {
        DepositsOnly,
        PreHolocene,
        Replacement,
    }

    #[rstest]
    #[case::deposits_only(Rejection::DepositsOnly)]
    #[case::pre_holocene(Rejection::PreHolocene)]
    #[case::replacement(Rejection::Replacement)]
    #[tokio::test]
    async fn rejected_payloads_preserve_state_and_error_policy(#[case] rejection: Rejection) {
        let txs = match rejection {
            Rejection::DepositsOnly => vec![Bytes::from_static(&[0x7e, 0x00])],
            _ => vec![Bytes::from_static(&[0x02, 0x00])],
        };
        let attributes = TestAttributesBuilder::new().with_transactions(txs).build();
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        cfg.hardforks.holocene_time = Some(match rejection {
            Rejection::PreHolocene => u64::MAX,
            _ => 0,
        });
        let mut calls = vec![ExpectedCall::success("engine_newPayloadV1", invalid())];
        if matches!(rejection, Rejection::Replacement) {
            calls.extend([
                ExpectedCall::success("engine_forkchoiceUpdatedV3", build_started()),
                ExpectedCall::success("engine_getPayloadV3", replacement(&mut cfg, &attributes)),
                ExpectedCall::success("engine_newPayloadV3", invalid()),
            ]);
        }
        let script = Script::new(calls);
        let engine = script.client(Arc::new(cfg.clone()));
        let mut state = EngineState::default();
        let sink = Sink::default();
        let original = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &OpBlock::default(),
        ));
        let error = insert_payload_with_holocene_fallback(
            &engine,
            &cfg,
            &mut state,
            &attributes,
            original,
            true,
            &sink,
        )
        .await
        .unwrap_err();
        match rejection {
            Rejection::DepositsOnly => {
                assert!(matches!(&error, SealTaskError::DepositOnlyPayloadFailed))
            }
            Rejection::PreHolocene => assert!(matches!(&error,
                SealTaskError::PayloadInsertionFailed(inner)
                    if matches!(inner.as_ref(), InsertTaskError::UnexpectedPayloadStatus(_)))),
            Rejection::Replacement => {
                assert!(matches!(&error, SealTaskError::DepositOnlyPayloadReattemptFailed))
            }
        }
        let severity = match rejection {
            Rejection::PreHolocene => EngineTaskErrorSeverity::Temporary,
            _ => EngineTaskErrorSeverity::Critical,
        };
        assert_eq!(error.severity(), severity);
        assert_eq!(state, EngineState::default());
        assert!(sink.0.lock().unwrap().is_empty());
        // Building a replacement sends an FCU with attributes, but rejection must not send a
        // later attribute-less FCU to canonicalize the rejected block.
        assert!(
            script.calls().into_iter().all(
                |call| call.method != "engine_forkchoiceUpdatedV3" || !call.params[1].is_null()
            )
        );
        script.assert_finished();
    }
}
