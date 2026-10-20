//! Applying derived attributes and trusted upstream safe heads.

use crate::{
    AttributesMatch, BuildError, ConsolidateError, EngineClient, EngineState,
    EngineSyncStateUpdate, ForkchoiceError, ImportMode, ImportedBlockSink, InsertError,
    get_payload, insert_payload, start_build, synchronize,
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};

/// What happened to derived input. The applied prefix is the resulting pending safe head.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Consolidation {
    /// The original input was applied or matched the unsafe chain.
    Applied,
    /// A rejected batch was replaced with its deposits-only block.
    Replaced,
    /// A pre-Holocene invalid batch was consumed without advancing safe.
    Dropped,
}

/// Applies derived attributes using their actual L1 inclusion-source timestamp.
/// The owner checks the original reset epoch and safe-parent context before calling.
pub async fn consolidate(
    client: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    source_timestamp: u64,
    block_sink: &dyn ImportedBlockSink,
) -> Result<Consolidation, ConsolidateError> {
    let holocene = cfg.is_holocene_active(source_timestamp);
    let concluding = holocene || attributes.is_last_in_span;
    if state.sync_state.safe_head().block_info.number <
        state.sync_state.unsafe_head().block_info.number
    {
        let block = client
            .get_l2_block(attributes.block_number().into())
            .full()
            .await
            .map_err(ConsolidateError::Read)?;
        if let Some(block) = block &&
            AttributesMatch::check(cfg, attributes, &block).is_match()
        {
            let block = block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
            if let Ok(info) = L2BlockInfo::from_block_and_genesis(&block, &cfg.genesis) {
                if concluding {
                    synchronize(
                        client,
                        state,
                        EngineSyncStateUpdate {
                            safe_head: Some(info),
                            local_safe_head: Some(info),
                            ..Default::default()
                        },
                    )
                    .await?;
                }
                state.pending_safe_head = info;
                block_sink.block_imported(block, info);
                return Ok(Consolidation::Applied);
            }
        }
    }

    let error = match build_and_import(client, cfg, state, attributes, concluding, block_sink).await
    {
        Ok(()) => return Ok(Consolidation::Applied),
        Err(error) => error,
    };
    if !is_rejected(&error) {
        return Err(error);
    }
    // There is no transaction subset to discard if this block already contains only deposits.
    if attributes.is_deposits_only() {
        return Err(ConsolidateError::DepositsOnlyRejected(error.to_string()));
    }
    if !holocene {
        if matches!(error, ConsolidateError::Build(BuildError::InvalidAttributes(_))) {
            // -38003 may be returned before OR after the EL changed forkchoice. Establish a
            // known parent before continuing derivation; a failed FCU goes back to the owner.
            synchronize(
                client,
                state,
                EngineSyncStateUpdate {
                    unsafe_head: Some(attributes.parent),
                    ..Default::default()
                },
            )
            .await?;
        }
        state.pending_safe_head = state.sync_state.safe_head();
        return Ok(Consolidation::Dropped);
    }

    let deposits = attributes.as_deposits_only();
    match build_and_import(client, cfg, state, &deposits, true, block_sink).await {
        Ok(()) => Ok(Consolidation::Replaced),
        Err(error) if is_rejected(&error) => {
            Err(ConsolidateError::DepositsOnlyRejected(error.to_string()))
        }
        Err(error) => Err(error),
    }
}

const fn is_rejected(error: &ConsolidateError) -> bool {
    matches!(
        error,
        ConsolidateError::Build(BuildError::Rejected(_) | BuildError::InvalidAttributes(_)) |
            ConsolidateError::Insert(InsertError::Rejected(PayloadStatusEnum::Invalid { .. }))
    )
}

/// Build/import is shared by the original derived block and its single deposits-only replacement.
async fn build_and_import(
    client: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes: &OpAttributesWithParent,
    concluding: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<(), ConsolidateError> {
    let id = start_build(client, cfg, state, attributes).await?;
    let payload = get_payload(client, cfg, id, attributes.attributes.payload_attributes.timestamp)
        .await
        .map_err(ConsolidateError::GetPayload)?;
    insert_payload(client, cfg, state, &payload, ImportMode::Derived { concluding }, block_sink)
        .await?;
    Ok(())
}

/// Follows an explicitly trusted upstream safe block, including one the local EL must sync.
/// Missing local bodies do not prevent sending the forkchoice that initiates that sync.
pub async fn follow_safe(
    client: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    safe: L2BlockInfo,
    block_sink: &dyn ImportedBlockSink,
) -> Result<Consolidation, ConsolidateError> {
    let block = client
        .get_l2_block(safe.block_info.number.into())
        .full()
        .await
        .map_err(ConsolidateError::Read)?;
    let matched = block.filter(|block| {
        block.header.hash == safe.block_info.hash && block.header.number == safe.block_info.number
    });
    let decoded = match matched {
        Some(block) => {
            let block = block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
            let info = L2BlockInfo::from_block_and_genesis(&block, &cfg.genesis)
                .map_err(InsertError::BlockInfo)?;
            if info != safe {
                return Err(ForkchoiceError::InvalidState.into());
            }
            Some(block)
        }
        None => None,
    };
    let preserve_unsafe = decoded.is_some() &&
        state.sync_state.unsafe_head().block_info.number > safe.block_info.number;
    synchronize(
        client,
        state,
        EngineSyncStateUpdate {
            unsafe_head: (!preserve_unsafe).then_some(safe),
            safe_head: Some(safe),
            local_safe_head: Some(safe),
            ..Default::default()
        },
    )
    .await?;
    state.pending_safe_head = safe;
    if let Some(block) = decoded {
        block_sink.block_imported(block, safe);
    }
    Ok(Consolidation::Applied)
}
