//! `op-reth-test-engine`: a library-first, ephemeral, OP-flavored execution layer for tests.
//!
//! Tests drive this as a library (Rust action tests, fuzzers, replays) or, via the companion
//! binary, as a subprocess over a Unix socket (Go action tests). It exists so `op-e2e/actions`
//! can run an OP-flavored execution layer without embedding op-geth in-process.
//!
//! The library exposes an ephemeral chain with read-only queries, the one-shot `new_payload`
//! import path, forkchoice updates, and a stateful in-flight payload builder
//! (`forkchoice_updated`-with-attributes → [`include_tx`](TestEngine::include_tx)\* →
//! [`get_payload`](TestEngine::get_payload)).

mod builder;
mod chain;
mod exec;
#[cfg(test)]
mod testsupport;

pub use builder::IncludeTxOutcome;
pub use chain::EphemeralChain;

use std::collections::HashMap;

use alloy_consensus::Header;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use op_alloy_rpc_types_engine::{OpExecutionData, OpPayloadAttributes};
use reth_db_common::init::InitStorageError;
use reth_optimism_primitives::{OpBlock, OpReceipt};
use reth_payload_primitives::{EngineApiMessageVersion, EngineObjectValidationError};
use reth_provider::ProviderError;

use crate::builder::InFlightPayload;

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
    /// A forkchoice update's payload attributes fail the engine API's checks for the forks active
    /// at their timestamp: a field the fork requires is missing, one it does not support is
    /// present, or an OP field is out of range. Carries reth's typed error, which tells invalid
    /// parameters apart from invalid payload attributes and an unsupported fork.
    #[error("malformed payload attributes: {0}")]
    MalformedPayloadAttributes(EngineObjectValidationError),
    /// A forkchoice update's payload attributes cannot open a block on the head they build on:
    /// their timestamp is not after the head's, the next block's environment cannot be derived
    /// from them, or their transactions do not decode or put a deposit after a user transaction.
    #[error("invalid payload attributes: {0}")]
    InvalidPayloadAttributes(String),
    /// A non-zero `safe`/`finalized` forkchoice block is unknown to the chain.
    #[error("forkchoice {which} block {hash} is unknown")]
    UnknownForkchoiceBlock {
        /// Which forkchoice pointer was unknown.
        which: ForkchoicePointer,
        /// The unknown block hash.
        hash: B256,
    },
    /// A block-building operation was requested while no block is being built. Mirrors
    /// `engineapi.ErrNotBuildingBlock`.
    #[error("not currently building a block")]
    NotBuildingBlock,
    /// `get_payload` was asked for a payload id that is not being built.
    #[error("unknown payload id {0}")]
    UnknownPayloadId(PayloadId),
    /// A transaction's declared gas limit exceeds the block gas limit. Mirrors
    /// `engineapi.ErrExceedsGasLimit`.
    #[error("tx gas exceeds block gas limit: tx gas {tx_gas}, block gas limit {block_gas_limit}")]
    ExceedsGasLimit {
        /// The transaction's declared gas limit.
        tx_gas: u64,
        /// The block gas limit.
        block_gas_limit: u64,
    },
    /// A transaction's declared gas limit exceeds the gas remaining in the block. Mirrors
    /// `engineapi.ErrUsesTooMuchGas` ("action takes too much gas").
    #[error("action takes too much gas: {tx_gas}, only have {remaining}")]
    UsesTooMuchGas {
        /// The transaction's declared gas limit.
        tx_gas: u64,
        /// Gas remaining in the block.
        remaining: u64,
    },
    /// A raw transaction could not be decoded.
    #[error("failed to decode transaction: {0}")]
    TxDecode(String),
}

/// A forkchoice pointer other than the head.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ForkchoicePointer {
    /// The `safe` block.
    Safe,
    /// The `finalized` block.
    Finalized,
}

impl core::fmt::Display for ForkchoicePointer {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.write_str(match self {
            Self::Safe => "safe",
            Self::Finalized => "finalized",
        })
    }
}

/// Result alias for this crate.
pub type Result<T> = core::result::Result<T, Error>;

/// The in-test OP execution engine.
///
/// Constructs an ephemeral genesis-initialized chain, answers read-only chain queries, imports
/// blocks via [`new_payload`](Self::new_payload), advances the head via
/// [`forkchoice_updated`](Self::forkchoice_updated), and builds payloads statefully
/// (forkchoice-with-attributes → [`include_tx`](Self::include_tx)\* →
/// [`get_payload`](Self::get_payload)).
#[derive(Debug)]
pub struct TestEngine {
    chain: EphemeralChain,
    /// Payloads currently being built, keyed by id (library-first: several may coexist).
    in_flight: HashMap<PayloadId, InFlightPayload>,
    /// The most recently opened payload — the implicit target of `optest_*`-style calls that don't
    /// name an id.
    current: Option<PayloadId>,
}

impl TestEngine {
    /// Construct an engine over a fresh ephemeral chain initialized from `genesis`.
    pub fn new(genesis: Genesis) -> Result<Self> {
        Ok(Self { chain: EphemeralChain::new(genesis)?, in_flight: HashMap::new(), current: None })
    }

    /// Construct an engine over an already-built ephemeral chain. Tests use this to activate
    /// hardforks via the chain-spec builder rather than round-tripping them through genesis JSON.
    #[cfg(test)]
    pub(crate) fn from_chain(chain: EphemeralChain) -> Self {
        Self { chain, in_flight: HashMap::new(), current: None }
    }

    /// Import a complete execution payload (`engine_newPayload`).
    ///
    /// Validates the payload's layout, executes it against its parent state with OP semantics,
    /// verifies the post-state root, and—on success—commits it as the new canonical head.
    /// Returns `VALID` (with the block hash as `latestValidHash`), `INVALID` (with the parent
    /// hash), or `SYNCING` when the parent block is unknown.
    pub fn new_payload(&mut self, payload: OpExecutionData) -> Result<PayloadStatus> {
        let status = exec::import_payload(&self.chain, payload)?;
        // A committed payload advances the head, stranding any in-flight payload still open on the
        // now-superseded parent.
        self.evict_stale_in_flight();
        Ok(status)
    }

    /// Update the forkchoice (`engine_forkchoiceUpdated`).
    ///
    /// `version` is the message version of the `engine_forkchoiceUpdated` method called. It only
    /// matters with `attributes`, which must carry exactly the fields that version and the forks
    /// active at their timestamp require.
    ///
    /// Advances the canonical/safe/finalized pointers: a known `head` yields `VALID`, an unknown
    /// `head` yields `SYNCING` (never a silent `VALID`), and an unknown non-zero `safe`/`finalized`
    /// is an [`Error::UnknownForkchoiceBlock`]. When `attributes` is `Some`, a new payload is
    /// opened on top of `head` and its [`PayloadId`] is returned. Attributes that are malformed for
    /// the forks active at their timestamp are an [`Error::MalformedPayloadAttributes`], and ones
    /// not later than `head` an [`Error::InvalidPayloadAttributes`]; the attributes' forced
    /// transactions are applied
    /// immediately, so an invalid one errors here too (mirroring op-geth's `startBlock`).
    pub fn forkchoice_updated(
        &mut self,
        version: EngineApiMessageVersion,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated> {
        let head = state.head_block_hash;
        // An unknown head reports SYNCING — never a silent VALID.
        if self.chain.sealed_header(head)?.is_none() {
            return Ok(ForkchoiceUpdated::from_status(PayloadStatusEnum::Syncing));
        }
        // head is known, so this only fails on an unknown non-zero safe/finalized pointer.
        self.chain.advance_forkchoice(head, state.safe_block_hash, state.finalized_block_hash)?;
        // The head may have moved past an earlier in-flight payload's parent; drop it so a later
        // get_payload for it reports UnknownPayloadId rather than re-sealing a stale block.
        self.evict_stale_in_flight();

        let valid =
            ForkchoiceUpdated::from_status(PayloadStatusEnum::Valid).with_latest_valid_hash(head);
        let Some(attributes) = attributes else {
            return Ok(valid);
        };

        let in_flight = InFlightPayload::open(&self.chain, version, head, attributes)?;
        let id = in_flight.id();
        self.in_flight.insert(id, in_flight);
        self.current = Some(id);
        Ok(valid.with_payload_id(id))
    }

    /// Include a raw pool transaction in the block being built (`optest_includeTx`).
    ///
    /// Targets `id`, or the most recently opened payload when `id` is `None`. Returns
    /// [`IncludeTxOutcome::Skipped`] under force-empty, and errors with
    /// [`Error::NotBuildingBlock`], [`Error::ExceedsGasLimit`], or [`Error::UsesTooMuchGas`] as
    /// the op-geth engine API does.
    pub fn include_tx(&mut self, id: Option<PayloadId>, raw: &[u8]) -> Result<IncludeTxOutcome> {
        let id = self.resolve_id(id)?;
        let chain = &self.chain;
        self.in_flight.get_mut(&id).ok_or(Error::NotBuildingBlock)?.include_tx(chain, raw)
    }

    /// Gas remaining in the block being built (`optest_remainingBlockGas`). Returns `0` when no
    /// block is being built or it is already sealed, mirroring `L2EngineAPI.RemainingBlockGas`.
    pub fn remaining_block_gas(&self, id: Option<PayloadId>) -> u64 {
        self.in_flight_ref(id).map_or(0, InFlightPayload::remaining_block_gas)
    }

    /// Whether force-empty is set for the block being built (`optest_forcedEmpty`). Returns `false`
    /// when no block is being built, mirroring `L2EngineAPI.ForcedEmpty`.
    pub fn forced_empty(&self, id: Option<PayloadId>) -> bool {
        self.in_flight_ref(id).is_some_and(InFlightPayload::forced_empty)
    }

    /// Set the force-empty flag for the block being built (`optest_setForceEmpty`). Mirrors
    /// `L2EngineAPI.SetForceEmpty`.
    pub fn set_force_empty(&mut self, id: Option<PayloadId>, value: bool) -> Result<()> {
        let id = self.resolve_id(id)?;
        self.in_flight.get_mut(&id).ok_or(Error::NotBuildingBlock)?.set_force_empty(value);
        Ok(())
    }

    /// Seal and return the block being built (`engine_getPayload`); feed the result to
    /// [`new_payload`](Self::new_payload) to commit it.
    ///
    /// Sealing closes the build: later inclusions fail with [`Error::NotBuildingBlock`] and no
    /// block gas remains. Fetching the payload again returns the same sealed block until the head
    /// moves off its parent, after which the id is unknown.
    pub fn get_payload(&mut self, id: PayloadId) -> Result<OpExecutionData> {
        let chain = &self.chain;
        self.in_flight.get_mut(&id).ok_or(Error::UnknownPayloadId(id))?.get_payload(chain)
    }

    /// Resolve an explicit id, falling back to the current payload; errors if neither is building.
    fn resolve_id(&self, id: Option<PayloadId>) -> Result<PayloadId> {
        id.or(self.current).ok_or(Error::NotBuildingBlock)
    }

    /// Borrow the in-flight payload named by `id`, or the current one, if any.
    fn in_flight_ref(&self, id: Option<PayloadId>) -> Option<&InFlightPayload> {
        id.or(self.current).and_then(|id| self.in_flight.get(&id))
    }

    /// Drop in-flight payloads whose parent is no longer the canonical head.
    ///
    /// op-geth discards a build job once its parent stops being the head; a subsequent `getPayload`
    /// for it then reports an unknown payload. Called on every head-advancing path (a committed
    /// `new_payload`, a forkchoice update) so the same `getPayload` returns `UnknownPayloadId`.
    fn evict_stale_in_flight(&mut self) {
        let head = self.chain.latest_header().hash();
        self.in_flight.retain(|_, payload| payload.parent_hash() == head);
        if self.current.is_some_and(|id| !self.in_flight.contains_key(&id)) {
            self.current = None;
        }
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
