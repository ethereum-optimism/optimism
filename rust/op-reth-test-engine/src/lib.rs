//! `op-reth-test-engine`: a library-first, ephemeral, OP-flavored execution layer for tests.
//!
//! Tests drive this as a library (Rust action tests, fuzzers, replays) or, via the companion
//! binary, as a subprocess over a Unix socket (Go action tests). It exists so `op-e2e/actions`
//! can run an OP-flavored execution layer without embedding op-geth in-process.
//!
//! The library exposes an ephemeral chain with read-only queries, the one-shot `new_payload`
//! import path, and attrs-less `forkchoice_updated`.

mod chain;
mod exec;

pub use chain::EphemeralChain;

use alloy_consensus::Header;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use op_alloy_rpc_types_engine::{OpExecutionData, OpPayloadAttributes};
use reth_db_common::init::InitStorageError;
use reth_optimism_primitives::{OpBlock, OpReceipt};
use reth_provider::ProviderError;

/// Errors produced by the test engine.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// A reth provider or storage query failed.
    #[error(transparent)]
    Provider(#[from] ProviderError),
    /// Genesis initialization failed.
    #[error(transparent)]
    InitStorage(#[from] InitStorageError),
    /// Executing or building a block failed for a reason that is not attributable to the block
    /// itself (an internal/EVM-wiring error, distinct from an `INVALID` payload).
    #[error("block execution failed: {0}")]
    Execution(String),
    /// A non-zero `safe`/`finalized` forkchoice block is unknown to the chain.
    #[error("forkchoice {which} block {hash} is unknown")]
    UnknownForkchoiceBlock {
        /// Which forkchoice pointer was unknown: `"safe"` or `"finalized"`.
        which: &'static str,
        /// The unknown block hash.
        hash: B256,
    },
    /// An unsupported operation was requested.
    #[error("unsupported: {0}")]
    Unsupported(&'static str),
}

/// Result alias for this crate.
pub type Result<T> = core::result::Result<T, Error>;

/// The in-test OP execution engine.
///
/// Constructs an ephemeral genesis-initialized chain, answers read-only chain queries, imports
/// blocks via [`new_payload`](Self::new_payload), and advances the head via attrs-less
/// [`forkchoice_updated`](Self::forkchoice_updated).
#[derive(Debug)]
pub struct TestEngine {
    chain: EphemeralChain,
}

impl TestEngine {
    /// Construct an engine over a fresh ephemeral chain initialized from `genesis`.
    pub fn new(genesis: Genesis) -> Result<Self> {
        Ok(Self { chain: EphemeralChain::new(genesis)? })
    }

    /// Import a complete execution payload (`engine_newPayload`).
    ///
    /// Validates the payload's layout, executes it against its parent state with OP semantics,
    /// verifies the post-state root, and—on success—commits it as the new canonical head.
    /// Returns `VALID` (with the block hash as `latestValidHash`), `INVALID` (with the parent
    /// hash), or `SYNCING` when the parent block is unknown.
    pub fn new_payload(&self, payload: OpExecutionData) -> Result<PayloadStatus> {
        exec::import_payload(&self.chain, payload)
    }

    /// Update the forkchoice (`engine_forkchoiceUpdated`).
    ///
    /// With `attributes == None` this advances the canonical/safe/finalized pointers: a known
    /// `head` yields `VALID`, an unknown `head` yields `SYNCING` (never a silent `VALID`), and an
    /// unknown non-zero `safe`/`finalized` is an [`Error::UnknownForkchoiceBlock`]. Payload
    /// building (`attributes == Some`) is not supported.
    pub fn forkchoice_updated(
        &self,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated> {
        if attributes.is_some() {
            return Err(Error::Unsupported(
                "forkchoice_updated with payload attributes (payload building)",
            ));
        }
        let head = state.head_block_hash;
        if !self.chain.advance_forkchoice(
            head,
            state.safe_block_hash,
            state.finalized_block_hash,
        )? {
            return Ok(ForkchoiceUpdated::from_status(PayloadStatusEnum::Syncing));
        }
        Ok(ForkchoiceUpdated::from_status(PayloadStatusEnum::Valid).with_latest_valid_hash(head))
    }

    /// Fetch a block by number, or `None` if unknown.
    pub fn block_by_number(&self, number: u64) -> Result<Option<OpBlock>> {
        self.chain.block_by_number(number)
    }

    /// Fetch a block by hash, or `None` if unknown.
    pub fn block_by_hash(&self, hash: B256) -> Result<Option<OpBlock>> {
        self.chain.block_by_hash(hash)
    }

    /// Fetch a header by number, or `None` if unknown.
    pub fn header_by_number(&self, number: u64) -> Result<Option<Header>> {
        self.chain.header_by_number(number)
    }

    /// Fetch the receipts of a block by hash, or `None` if unknown.
    pub fn receipts_by_block_hash(&self, hash: B256) -> Result<Option<Vec<OpReceipt>>> {
        self.chain.receipts_by_block_hash(hash)
    }
}
