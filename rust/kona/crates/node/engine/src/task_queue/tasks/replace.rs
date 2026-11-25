//! Replacing a block that a cross-chain verifier denied with the deposits-only block.

use super::{InsertTaskError, SealTaskError, get_payload, new_payload, start_build};
use crate::{
    AttributesMatch, BuildTaskError, DeniedBlocksUnknown, EngineClient, EngineState,
    EngineTaskError, ImportedBlockSink, SynchronizeTaskError,
    state::EngineSyncStateUpdate,
    task_queue::{synchronize, tasks::task::EngineTaskErrorSeverity},
};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::{FromBlockError, L2BlockInfo, OpAttributesWithParent};
use op_alloy_consensus::OpBlock;
use thiserror::Error;

/// Why a denied block was not replaced, or that it was.
#[derive(Debug, Error)]
pub enum ReplaceDeniedError {
    /// The verifier's denied blocks are not known yet, so no derived block can be imported.
    #[error(transparent)]
    Unknown(#[from] DeniedBlocksUnknown),
    /// The denied block was replaced. As after an invalid payload, the derivation pipeline
    /// flushes its channel: the rest of it continues the denied block's chain.
    #[error("Replaced denied block {denied:?} with the deposits-only block {replacement:?}")]
    Replaced {
        /// The denied block.
        denied: BlockNumHash,
        /// The deposits-only block that replaced it.
        replacement: BlockNumHash,
    },
    /// Starting the replacement's build failed.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// Fetching the built replacement failed.
    #[error(transparent)]
    GetPayload(#[from] SealTaskError),
    /// Inserting the replacement failed.
    #[error(transparent)]
    Insert(#[from] InsertTaskError),
    /// The engine did not validate the replacement, so it is not made the head.
    #[error("The engine answered {0} to the deposits-only replacement")]
    NotValidated(PayloadStatusEnum),
    /// The forkchoice update to the replacement failed.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// After the forkchoice update, the engine's block at the height is not the replacement.
    #[error("The engine's block {number} is {found:?}, not the replacement {expected}")]
    NotCanonical {
        /// The height of the replacement.
        number: u64,
        /// The replacement's hash.
        expected: B256,
        /// The hash of the engine's block at the height, if it has one.
        found: Option<B256>,
    },
    /// The denied block is its own deposits-only form, so nothing can replace it.
    #[error("The denied block {0:?} has only deposits, so nothing can replace it")]
    DepositsOnlyDenied(BlockNumHash),
    /// A block lookup failed.
    #[error(transparent)]
    Lookup(#[from] RpcError<TransportErrorKind>),
    /// A block could not be turned into an [`L2BlockInfo`].
    #[error(transparent)]
    FromBlock(#[from] FromBlockError),
}

impl EngineTaskError for ReplaceDeniedError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::Replaced { .. } => EngineTaskErrorSeverity::Flush,
            Self::NotValidated(PayloadStatusEnum::Invalid { .. }) |
            Self::DepositsOnlyDenied(_) |
            Self::FromBlock(_) => EngineTaskErrorSeverity::Critical,
            Self::Unknown(_) |
            Self::NotValidated(_) |
            Self::NotCanonical { .. } |
            Self::Lookup(_) => EngineTaskErrorSeverity::Temporary,
            Self::Build(inner) => inner.severity(),
            Self::GetPayload(inner) => inner.severity(),
            Self::Insert(inner) => inner.severity(),
            Self::Forkchoice(inner) => inner.severity(),
        }
    }
}

/// Imports the deposits-only form of derived `attributes` in place of `denied`, the block they
/// produce, which a cross-chain verifier denied. On success, fails with
/// [`ReplaceDeniedError::Replaced`], which flushes the derivation pipeline's channel.
///
/// The replacement is the block the interop protocol defines for a denied block: the derived
/// attributes keeping only their deposits, as op-node builds it, which matches the block the
/// proof program builds from the denied block's header and deposits. If the engine's block at the
/// height already is the replacement, as when a node re-derives a block it replaced before a
/// restart, it is adopted as local-safe. Otherwise the replacement is built on the attributes'
/// parent and made the head only once the engine has validated it, and read back after.
pub(in crate::task_queue) async fn replace_denied(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    denied: BlockNumHash,
    block_sink: &dyn ImportedBlockSink,
) -> ReplaceDeniedError {
    match import_replacement(engine, cfg, state, attributes, denied, block_sink).await {
        Ok(replacement) => {
            let replacement = replacement.block_info.id();
            warn!(target: "engine", ?denied, ?replacement, "Replaced a denied block with the deposits-only block");
            ReplaceDeniedError::Replaced { denied, replacement }
        }
        Err(err) => err,
    }
}

async fn import_replacement(
    engine: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    denied: BlockNumHash,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, ReplaceDeniedError> {
    let attributes = attributes.as_deposits_only();
    let number = attributes.block_number();

    if let Some(block) = engine.get_l2_block(BlockId::number(number)).full().await? &&
        block.header.hash != denied.hash &&
        AttributesMatch::check(cfg, &attributes, &block).is_match()
    {
        let block: OpBlock =
            block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
        let replacement = L2BlockInfo::from_block_and_genesis(&block, &cfg.genesis)?;
        warn!(target: "engine", replacement = ?replacement.block_info.id(), "Adopting the deposits-only replacement the engine already has");
        block_sink.block_imported(block, replacement);
        let unsafe_head =
            (state.sync_state.unsafe_head().block_info.number < number).then_some(replacement);
        synchronize(
            engine,
            state,
            EngineSyncStateUpdate {
                unsafe_head,
                local_safe_head: Some(replacement),
                ..Default::default()
            },
        )
        .await?;
        return Ok(replacement);
    }

    let timestamp = attributes.attributes().payload_attributes.timestamp;
    let payload_id =
        start_build(engine, cfg, state, attributes).await.map_err(BuildTaskError::from)?;
    let payload = get_payload(engine, cfg, payload_id, timestamp).await?;
    // A forkchoice update to a block the engine has not validated would skip its checks, so the
    // replacement becomes the head only once newPayload validated it.
    match new_payload(engine, payload.clone()).await? {
        PayloadStatusEnum::Valid => {}
        status => return Err(ReplaceDeniedError::NotValidated(status)),
    }
    let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
    let replacement = L2BlockInfo::from_block_and_genesis(&block, &cfg.genesis)?;
    if replacement.block_info.id() == denied {
        return Err(ReplaceDeniedError::DepositsOnlyDenied(denied));
    }
    synchronize(
        engine,
        state,
        EngineSyncStateUpdate {
            unsafe_head: Some(replacement),
            local_safe_head: Some(replacement),
            ..Default::default()
        },
    )
    .await?;

    let found = engine.get_l2_block(BlockId::number(number)).await?.map(|block| block.header.hash);
    if found != Some(replacement.block_info.hash) {
        return Err(ReplaceDeniedError::NotCanonical {
            number,
            expected: replacement.block_info.hash,
            found,
        });
    }
    block_sink.block_imported(block, replacement);
    Ok(replacement)
}
