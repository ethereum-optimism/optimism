//! The observation seam: everything the round loop reads, and nothing it writes.
//!
//! One trait per source. [`InteropChain`] is one L2 chain as the verifier sees it — a chain
//! controller for the safety snapshot, and a read-only execution layer for receipts and outputs.
//! [`L1Canonical`] is the L1 they all derive from, consulted only to ask whether a block the
//! verifier already relied on is still canonical.
//!
//! Both traits are deliberately narrow and read-only. The round loop's only writes go to the
//! stores in this crate and, in a later phase, to the single cross-safe promotion entry point —
//! never through here.

use crate::error::StoreError;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId, Log};
use async_trait::async_trait;
use kona_engine::{LocalSafeAtTimestamp, LocalSafeSnapshot};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, OutputRoot};
use std::fmt::Debug;

/// Why an observation could not be answered *yet*.
///
/// Every variant here is transient: ask again and the answer may differ. A permanent inability to
/// answer is not an error but a verdict — [`ChainAt::HistoryUnavailable`] — so that the one
/// condition which stops the verifier for good has exactly one spelling.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum ChainError {
    /// The source cannot answer yet: it is still syncing, or has not recorded what was asked for.
    #[error("not ready")]
    NotReady,
    /// The source could not be reached, or answered with an error.
    #[error("unreachable: {0}")]
    Unreachable(String),
}

/// Where a round's timestamp falls on one chain, and the pairing when it falls on a block.
///
/// These are the four verdicts of the engine's local-safe-at-timestamp query, carried through to
/// the round loop rather than flattened into "an answer or no answer". The round loop treats them
/// very differently: [`Self::NotYet`] is waited out, [`Self::HistoryUnavailable`] halts the
/// verifier, and [`Self::BeforeGenesis`] means the chain set and the verification start disagree.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChainAt {
    /// The chain's local-safe block at the timestamp, and the L1 block it was derived from.
    Derived {
        /// The L2 block carrying the timestamp.
        block: BlockNumHash,
        /// The L1 block it was derived from.
        l1: BlockNumHash,
    },
    /// The chain has not derived a local-safe block at that timestamp yet. Transient.
    NotYet,
    /// The chain had a local-safe block there, but which L1 block made it safe is no longer
    /// recorded on this node and cannot be recovered. Permanent.
    HistoryUnavailable,
    /// No block of this chain carries that timestamp, because the chain did not exist yet.
    /// Permanent.
    BeforeGenesis,
}

impl ChainAt {
    /// Maps the engine's snapshot onto a verdict, or [`None`] when the snapshot defers to history.
    ///
    /// The live engine state holds the L1 origin of exactly one L2 block — the local-safe head —
    /// so it can pair only the head's own timestamp. A timestamp behind the head is a
    /// [`LocalSafeAtTimestamp::BehindHead`], which is not an absent answer but a redirection: the
    /// pairing exists, recorded in the safe-head database, and the caller looks it up there. That
    /// redirection is why this returns an [`Option`] rather than a verdict — flattening
    /// `BehindHead` into [`Self::NotYet`] would make a lagging verifier wait forever, and
    /// flattening it into [`Self::HistoryUnavailable`] would halt a healthy node.
    ///
    /// An unpaired head is [`Self::NotYet`]: its writer held no L1 key for it, so nothing can be
    /// verified against it until derivation writes one.
    pub fn from_snapshot(snapshot: &LocalSafeSnapshot) -> Option<Self> {
        if !snapshot.el_sync_finished {
            // The heads describe a node still catching up, not the chain.
            return Some(Self::NotYet);
        }
        match snapshot.local_safe_at {
            LocalSafeAtTimestamp::BeforeGenesis => Some(Self::BeforeGenesis),
            LocalSafeAtTimestamp::NotLocalSafeYet => Some(Self::NotYet),
            LocalSafeAtTimestamp::BehindHead => None,
            LocalSafeAtTimestamp::Head(head) => {
                Some(head.derived_from_l1().map_or(Self::NotYet, |l1| Self::Derived {
                    block: head.head.block_info.id(),
                    l1: l1.id(),
                }))
            }
        }
    }
}

/// A block and the logs it emitted, in block order.
///
/// The logs are the whole block's, not only the interop ones: log indices are global within the
/// block, so an initiating message's index is only meaningful against the full sequence.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BlockLogs {
    /// The block the logs came from.
    pub block: BlockInfo,
    /// Every log the block emitted, in index order starting at zero.
    pub logs: Vec<Log>,
}

impl BlockLogs {
    /// Returns the block's number and hash.
    pub const fn id(&self) -> BlockNumHash {
        BlockNumHash { number: self.block.number, hash: self.block.hash }
    }
}

/// One L2 chain, as the verification round reads it.
#[async_trait]
pub trait InteropChain: Debug + Send + Sync {
    /// The chain's id.
    fn chain_id(&self) -> ChainId;

    /// The chain's rollup config.
    ///
    /// The verifier reads the block time and the interop activation time from here, so the
    /// activation invariant is evaluated per chain against that chain's own config — which is what
    /// keeps it identical to the rule the proof side applies.
    fn rollup_config(&self) -> &RollupConfig;

    /// Where an L2 timestamp falls on this chain, and the L1 block that made it local-safe.
    ///
    /// The engine's own query answers this atomically for the local-safe head; an implementation
    /// answers the rest from the safe-head database, whose records are themselves paired. Either
    /// way the L2 block and its L1 key must come from one read — assembling them from two is the
    /// race this seam exists to avoid.
    async fn local_safe_at(&self, timestamp: u64) -> Result<ChainAt, ChainError>;

    /// Every log of `block`, for indexing it into the chain's log store.
    async fn block_logs(&self, block: BlockNumHash) -> Result<BlockLogs, ChainError>;

    /// The output root commitment at `number`, whose block hash is the canonical one at that
    /// height.
    async fn output_at(&self, number: u64) -> Result<OutputRoot, ChainError>;

    /// The earliest timestamp this chain's safe-head history covers.
    ///
    /// [`ChainError::NotReady`] means the chain has recorded no safe head yet, which is the normal
    /// answer while a cold-starting node is still catching up.
    async fn first_safe_head_timestamp(&self) -> Result<u64, ChainError>;

    /// The number of the block covering `timestamp`, flooring onto the preceding block when the
    /// timestamp falls between two of them.
    async fn block_number_at_timestamp(&self, timestamp: u64) -> Result<u64, ChainError>;
}

/// The L1 the chains derive from, as the verifier reads it.
#[async_trait]
pub trait L1Canonical: Debug + Send + Sync {
    /// The hash of the canonical L1 block at `number`.
    async fn canonical_hash_at(&self, number: u64) -> Result<B256, ChainError>;
}

impl<T: L1Canonical + ?Sized> L1CanonicalExt for T {}

/// Convenience checks over an [`L1Canonical`].
#[async_trait]
pub trait L1CanonicalExt: L1Canonical {
    /// Returns whether every one of `blocks` is still the canonical L1 block at its height.
    ///
    /// A block at height zero is skipped rather than checked: it is the "no L1 block" value the
    /// unpaired case carries, and asking the L1 about it would answer about genesis.
    async fn all_canonical(&self, blocks: &[BlockNumHash]) -> Result<bool, ChainError> {
        for block in blocks {
            if block.number == 0 && block.hash.is_zero() {
                continue;
            }
            if self.canonical_hash_at(block.number).await? != block.hash {
                return Ok(false);
            }
        }
        Ok(true)
    }
}

/// Errors the round loop itself can fail with, above the individual observations.
#[derive(Debug, thiserror::Error)]
pub enum RoundError {
    /// One chain could not be observed.
    #[error("chain {chain_id}: {source}")]
    Chain {
        /// The chain that could not be observed.
        chain_id: ChainId,
        /// Why.
        source: ChainError,
    },
    /// The L1 could not be consulted.
    #[error("l1: {0}")]
    L1(#[source] ChainError),
    /// A store read or write failed.
    #[error(transparent)]
    Store(#[from] StoreError),
    /// The round reached a state the code does not know how to continue from.
    #[error("{0}")]
    Invariant(String),
    /// A condition this node can never recover from on its own.
    #[error("{0}")]
    Permanent(String),
}

impl RoundError {
    /// Attaches a chain id to a [`ChainError`].
    pub const fn chain(chain_id: ChainId, source: ChainError) -> Self {
        Self::Chain { chain_id, source }
    }

    /// Returns whether retrying the round could succeed.
    ///
    /// An untrue answer here is costly in both directions: treating a permanent failure as
    /// transient spins forever, and treating a transient one as permanent halts a healthy node.
    pub fn is_transient(&self) -> bool {
        match self {
            // Every `ChainError` is by construction an "ask again".
            Self::Chain { .. } | Self::L1(_) => true,
            // A damaged store is permanent; every other store failure is a failed write to retry.
            Self::Store(err) => !matches!(err, StoreError::DataCorruption(_)),
            Self::Invariant(_) | Self::Permanent(_) => false,
        }
    }
}
