//! The inputs to engine operations.

use alloy_eips::BlockNumHash;
use alloy_rpc_types_eth::Block;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types::Transaction;

/// Input for consolidation - either derived attributes or safe L2 block
#[derive(Debug, Clone)]
pub enum ConsolidateInput {
    /// Consolidate based on derived attributes.
    Attributes(Box<OpAttributesWithParent>),
    /// Derivation Delegation: consolidate based on safe L2 block info.
    BlockInfo(L2BlockInfo),
}

impl From<L2BlockInfo> for ConsolidateInput {
    fn from(v: L2BlockInfo) -> Self {
        Self::BlockInfo(v)
    }
}

impl From<OpAttributesWithParent> for ConsolidateInput {
    fn from(v: OpAttributesWithParent) -> Self {
        Self::Attributes(Box::new(v))
    }
}

impl ConsolidateInput {
    /// Returns the block number for this consolidation input.
    pub(super) const fn l2_block_number(&self) -> u64 {
        match self {
            Self::Attributes(attributes) => attributes.block_number(),
            Self::BlockInfo(info) => info.block_info.number,
        }
    }

    /// Checks if the block is consistent with this consolidation input.
    pub(super) fn is_consistent_with_block(
        &self,
        cfg: &RollupConfig,
        block: &Block<Transaction>,
    ) -> bool {
        match self {
            Self::Attributes(attributes) => {
                kona_engine::AttributesMatch::check(cfg, attributes, block).is_match()
            }
            Self::BlockInfo(info) => block.header.hash == info.block_info.hash,
        }
    }

    /// Returns true if this is `Attributes` and `attributes.is_last_in_span` is true.
    pub(super) const fn is_attributes_last_in_span(&self) -> bool {
        matches!(
            self,
            Self::Attributes(attributes)
                if attributes.is_last_in_span
        )
    }
}

/// Identifies the L2 block to finalize.
///
/// Two semantic regimes drive finalization in kona-node:
///
/// - [`FinalizeBlockId::ByNumber`] is used when the engine's own canonical chain is the
///   authoritative source for the block at a given height (e.g. local L1-finality driven by
///   `L2Finalizer`). The number is sufficient because there is only one chain to consult.
/// - [`FinalizeBlockId::ByHash`] is used when an upstream supplies `(number, hash)` and the engine
///   must finalize the specific block identified by hash. If the engine does not have that hash,
///   finalization must error rather than silently finalize the engine's canonical block at the same
///   height.
///
/// EL finalization is irreversible, so finalizing the wrong block is unrecoverable. Forcing every
/// caller to pick a variant prevents accidental hash-loss at the boundary.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FinalizeBlockId {
    /// Finalize whatever block the engine has at the given L2 block number.
    ByNumber(u64),
    /// Finalize the block matching the given `(number, hash)` pair. Finalization fails if the
    /// engine does not have that hash.
    ByHash(BlockNumHash),
}

impl FinalizeBlockId {
    /// Returns the L2 block number identified by this id.
    pub const fn number(&self) -> u64 {
        match self {
            Self::ByNumber(n) => *n,
            Self::ByHash(id) => id.number,
        }
    }
}
