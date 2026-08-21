//! The ways a supernode query can fail, and how each one crosses the wire.
//!
//! Every variant here is a condition op-supernode also fails the call on. That symmetry is the
//! point: a consumer that retries on an error and acts on an answer must not be handed an answer
//! where op-supernode would have errored, because a partial super root is well-formed and wrong.
//! The two conditions that are *not* errors — a chain that has not derived the requested timestamp
//! yet, and a timestamp the verifier has not reached — are reported inside a successful response,
//! as an absent chain and an absent `data` respectively.

use crate::interop::InteropQueryError;
use alloy_primitives::ChainId;
use jsonrpsee::types::{ErrorObjectOwned, error::INTERNAL_ERROR_CODE};

/// A failed supernode query.
#[derive(Debug, thiserror::Error)]
pub(crate) enum QueryError {
    /// The chains have not been composed yet, so there is nothing to aggregate.
    ///
    /// The process-wide RPC binds before composition — that is what lets a launching harness learn
    /// its address from one log line, and fail fast if the process dies instead — so a caller can
    /// reach this server before the chains it answers for exist. Saying so beats reporting a
    /// supernode with no chains.
    #[error("the supernode is still starting: its chains are not composed yet")]
    Starting,
    /// A timestamp before one chain's L2 genesis.
    ///
    /// op-supernode fails here too: `TargetBlockNumber` rejects a timestamp below genesis, and the
    /// error propagates out of the optimistic branch.
    #[error("chain {chain_id} has no block at timestamp {timestamp}: it is before L2 genesis")]
    BeforeGenesis {
        /// The chain that has no such block.
        chain_id: ChainId,
        /// The timestamp asked about.
        timestamp: u64,
    },
    /// The safe-head history that would pair a chain's block with an L1 block is gone.
    ///
    /// Permanent, and op-supernode's `ErrHistoryUnavailable`.
    #[error(
        "chain {chain_id} no longer records which L1 block made its block at timestamp \
         {timestamp} safe"
    )]
    HistoryUnavailable {
        /// The chain whose history is gone.
        chain_id: ChainId,
        /// The timestamp asked about.
        timestamp: u64,
    },
    /// A chain could not be read.
    ///
    /// The cause is carried rendered rather than as a `#[source]`: the underlying failures are
    /// jsonrpsee error objects and store errors with no common error type, and a field named
    /// `source` would make `thiserror` demand one.
    #[error("chain {chain_id}: {reason}")]
    Chain {
        /// The chain that could not be read.
        chain_id: ChainId,
        /// What went wrong, rendered.
        reason: String,
    },
    /// The interop verifier could not be read.
    #[error(transparent)]
    Interop(#[from] InteropQueryError),
    /// The verified frontier names a different chain set than this supernode hosts.
    ///
    /// Either side would produce a super root that disagrees with peers running the full set, so
    /// this is refused rather than served over the chains they have in common. op-supernode calls
    /// the same condition a dep-set mismatch and refuses it too.
    #[error(
        "the verified frontier at {timestamp} covers {verified} chains but this supernode hosts \
         {hosted}"
    )]
    ChainSetMismatch {
        /// The timestamp whose frontier disagrees.
        timestamp: u64,
        /// How many chains the frontier names.
        verified: usize,
        /// How many chains this supernode hosts.
        hosted: usize,
    },
    /// The verified frontier names a chain this supernode does not host, or omits one it does.
    #[error("the verified frontier at {timestamp} says nothing about chain {chain_id}")]
    ChainNotVerified {
        /// The timestamp whose frontier is incomplete.
        timestamp: u64,
        /// The chain the frontier says nothing about.
        chain_id: ChainId,
    },
    /// The chain's canonical block at a verified height is not the block that was verified.
    ///
    /// A chain that reorged below a block the verifier declared cross-safe. Serving the canonical
    /// block's output root here would answer with a super root over a branch nothing verified, so
    /// the call fails and the verifier reports the reorg on its own next round.
    #[error(
        "chain {chain_id} block {number} is {canonical} but the verified frontier at {timestamp} \
         names {verified}"
    )]
    ReorgedBelowVerified {
        /// The timestamp whose frontier no longer matches the chain.
        timestamp: u64,
        /// The chain that reorged.
        chain_id: ChainId,
        /// The height they disagree at.
        number: u64,
        /// The block the frontier named.
        verified: alloy_primitives::B256,
        /// The block the chain has there now.
        canonical: alloy_primitives::B256,
    },
}

impl From<QueryError> for ErrorObjectOwned {
    fn from(err: QueryError) -> Self {
        // One code for every variant: these are all "this supernode cannot answer that right
        // now", and the Go consumers of these RPCs discriminate on nothing but success.
        Self::owned(INTERNAL_ERROR_CODE, err.to_string(), None::<()>)
    }
}
