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
mod ethcall;
mod exec;
pub mod rpc;
#[cfg(test)]
mod testsupport;

pub use builder::{IncludeNextOutcome, IncludeTxOutcome};
pub use chain::EphemeralChain;

use std::collections::{BTreeMap, HashMap};

use alloy_consensus::{Header, Transaction as _, transaction::SignerRecoverable};
use alloy_eips::eip2718::Decodable2718;
use alloy_genesis::Genesis;
use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use op_alloy_rpc_types::OpTransactionReceipt;
use op_alloy_rpc_types_engine::{OpExecutionData, OpPayloadAttributes};
use reth_chain_state::ExecutedBlock;
use reth_db_common::init::InitStorageError;
use reth_optimism_primitives::{OpBlock, OpPrimitives, OpReceipt, OpTransactionSigned};
use reth_payload_primitives::{EngineApiMessageVersion, EngineObjectValidationError};
use reth_provider::ProviderError;
use reth_storage_api::StateProvider as _;

use crate::{builder::InFlightPayload, exec::ImportOutcome};

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
    /// from them, their transactions do not decode or put a deposit after a user transaction, or
    /// a forced transaction cannot be applied. op-node maps this to a payload error and (under
    /// Holocene, for a derived block) requests a deposits-only replacement rather than retrying or
    /// resetting forever.
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
    /// A non-zero `safe`/`finalized` forkchoice block is known but not the head or one of its
    /// ancestors.
    #[error("forkchoice {which} block {hash} is not on the head's chain")]
    NonCanonicalForkchoiceBlock {
        /// Which forkchoice pointer was off the head's chain.
        which: ForkchoicePointer,
        /// The offending block hash.
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
    /// A transaction cannot be applied to the state it executes on (unrecoverable signer, nonce,
    /// balance, gas, …).
    #[error("invalid transaction: {0}")]
    InvalidTransaction(String),
    /// A raw transaction could not be decoded.
    #[error("failed to decode transaction: {0}")]
    TxDecode(String),
    /// An `eth_call`/`eth_estimateGas` execution reverted. Carries the revert output so the RPC
    /// layer can attach it as error data (geth revert errors do the same).
    #[error("execution reverted")]
    Revert(Bytes),
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
    /// Per-sender, nonce-keyed buffer of raw transactions that `eth_sendRawTransaction` parked and
    /// [`include_next_tx`](Self::include_next_tx) includes. Sending a transaction whose nonce is
    /// already parked replaces the parked one. A parked transaction stays parked when a build
    /// includes it, so it keeps counting towards the pending nonce and survives a build that is
    /// never sealed. Entries below the sender's nonce in a build's parent state are pruned only
    /// when `include_next_tx` next runs for that sender, so a transaction whose block is reorged
    /// out before then is still parked and can be included again, while one pruned earlier is
    /// gone. This is deliberately *not* a transaction pool: parked transactions are never
    /// auto-included, gossiped or revalidated, and a reorg never re-parks anything — the buffer
    /// only exists so the Go action tests' `SendTransaction` / `PendingNonceAt` /
    /// `ActL2IncludeTx(from)` sequence keeps working unchanged over the socket (the concrete
    /// `*ethclient.Client` they use cannot be intercepted Go-side).
    pending: HashMap<Address, BTreeMap<u64, Bytes>>,
    /// Every block that has passed [`new_payload`](Self::new_payload) validation, keyed by hash
    /// and never evicted. A processed payload is only recorded here (not committed) until a
    /// forkchoice update canonicalizes it; retaining committed and reorged-out blocks alike lets
    /// a later forkchoice update reorg onto an alternate fork or flip back to a previously
    /// abandoned one — the reorg support op-node's derivation relies on.
    known_blocks: HashMap<B256, ExecutedBlock<OpPrimitives>>,
    /// The head a forkchoice update asked for but could not reach, because it (or an ancestor)
    /// is not yet known: the block the engine would be snap-syncing towards. Set whenever
    /// [`forkchoice_updated`](Self::forkchoice_updated) reports `SYNCING`, cleared once an update
    /// canonicalizes a head. A real op-geth EL fills this gap from its peers over devp2p; this
    /// ephemeral engine has no p2p, so the Go action harness reads it (`optest_syncTarget`) and
    /// copies the missing blocks in from the peer engine.
    sync_target: Option<B256>,
}

impl TestEngine {
    /// Construct an engine over a fresh ephemeral chain initialized from `genesis`.
    pub fn new(genesis: Genesis) -> Result<Self> {
        Ok(Self {
            chain: EphemeralChain::new(genesis)?,
            in_flight: HashMap::new(),
            current: None,
            pending: HashMap::new(),
            known_blocks: HashMap::new(),
            sync_target: None,
        })
    }

    /// Construct an engine over an already-built ephemeral chain. Tests use this to activate
    /// hardforks via the chain-spec builder rather than round-tripping them through genesis JSON.
    #[cfg(test)]
    pub(crate) fn from_chain(chain: EphemeralChain) -> Self {
        Self {
            chain,
            in_flight: HashMap::new(),
            current: None,
            pending: HashMap::new(),
            known_blocks: HashMap::new(),
            sync_target: None,
        }
    }

    /// Import a complete execution payload (`engine_newPayload`).
    ///
    /// Validates the payload's layout, executes it against its parent state with OP semantics, and
    /// verifies the post-state root. A valid block is only recorded (in `known_blocks`): as on a
    /// real EL, a processed payload is not canonical until a forkchoice update names it (or a
    /// descendant) as head, so `latest` keeps reading the current head. Returns `VALID` (with the
    /// block hash as `latestValidHash`), `INVALID` (with the parent hash), or `SYNCING` when the
    /// parent block is unknown.
    pub fn new_payload(&mut self, payload: OpExecutionData) -> Result<PayloadStatus> {
        match exec::import_payload(&self.chain, payload, &self.known_blocks)? {
            ImportOutcome::Syncing => Ok(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
            ImportOutcome::Invalid(status) => Ok(status),
            ImportOutcome::Known(block_hash) => {
                Ok(PayloadStatus::new(PayloadStatusEnum::Valid, Some(block_hash)))
            }
            ImportOutcome::Valid(executed) => {
                let block_hash = executed.recovered_block().hash();
                // Retain every validated block so a later forkchoice update can canonicalize it —
                // as a linear extension, onto an alternate fork, or when flipping back to a
                // reorged-out one.
                self.known_blocks.insert(block_hash, executed);
                Ok(PayloadStatus::new(PayloadStatusEnum::Valid, Some(block_hash)))
            }
        }
    }

    /// Import a block obtained from a peer engine during sync backfill (`optest_importBlock`).
    ///
    /// Runs the ordinary `new_payload` validate-and-record path and, when the block extends the
    /// current head, advances the head onto it, as a real EL's block-sync insertion does — without
    /// touching the safe/finalized pointers, which stay wherever the consensus client last put
    /// them. A block that is already canonical or lies off the head is only recorded.
    pub fn import_block(&mut self, payload: OpExecutionData) -> Result<PayloadStatus> {
        let status = self.new_payload(payload)?;
        if status.status != PayloadStatusEnum::Valid {
            return Ok(status);
        }
        let hash = status.latest_valid_hash.expect("valid status carries the block hash");
        let head = self.chain.latest_header().hash();
        let header = self.chain.resolve_block_header(hash, &self.known_blocks)?;
        if let Some(header) = header.filter(|header| header.parent_hash == head) {
            let advanced = self.chain.reorg_to(&header, &self.known_blocks)?;
            debug_assert!(advanced, "a just-recorded block is always reachable");
            self.evict_stale_in_flight();
        }
        Ok(status)
    }

    /// Update the forkchoice (`engine_forkchoiceUpdated`).
    ///
    /// `version` is the message version of the `engine_forkchoiceUpdated` method called. It only
    /// matters with `attributes`, which must carry exactly the fields that version and the forks
    /// active at their timestamp require.
    ///
    /// Canonicalizes `head`, reorging onto it when it is on an alternate fork (using the retained
    /// `known_blocks`): a known `head` yields `VALID`, and an unknown one `SYNCING` (never a silent
    /// `VALID`), with or without attributes. A non-zero `safe`/`finalized` that is unknown or not
    /// an ancestor of `head` is an [`Error::UnknownForkchoiceBlock`] /
    /// [`Error::NonCanonicalForkchoiceBlock`]. When `attributes` is `Some`, a new payload is opened
    /// on top of `head` and its [`PayloadId`] is returned. Attributes that are malformed for the
    /// forks active at their timestamp are an [`Error::MalformedPayloadAttributes`]; ones not later
    /// than `head`, or whose forced transactions cannot be applied, are an
    /// [`Error::InvalidPayloadAttributes`]. Either leaves the chain untouched: every check runs
    /// before anything is mutated.
    pub fn forkchoice_updated(
        &mut self,
        version: EngineApiMessageVersion,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated> {
        let head = state.head_block_hash;

        // An unknown head reports SYNCING — never a silent VALID — with or without attributes, and
        // records the head so the Go harness (which stands in for the devp2p snap sync a real EL
        // would run) knows to backfill towards it.
        let Some(head_header) = self.chain.resolve_block_header(head, &self.known_blocks)? else {
            self.sync_target = Some(head);
            return Ok(ForkchoiceUpdated::from_status(PayloadStatusEnum::Syncing));
        };
        let pointers = self.chain.forkchoice_pointers(
            &head_header,
            state.safe_block_hash,
            state.finalized_block_hash,
            &self.known_blocks,
        )?;

        // Open the requested block build *before* moving the head. Building may fail because a
        // forced transaction is invalid (e.g. a derived block whose batch carries a
        // badly-signed tx); surfacing that as an invalid-payload-attributes engine error
        // lets op-node request a deposits-only replacement instead of retrying forever.
        // Opening first means such a failure leaves the canonical chain untouched — the
        // blocks the head move would have orphaned stay queryable, which op-node relies on
        // when it consolidates the next safe attributes against the existing unsafe chain.
        // The build reads the head's state whether it is canonical or a side block.
        let opened = match attributes {
            Some(attributes) => {
                let (parent, parent_state) =
                    self.chain.block_state(head, &self.known_blocks)?.ok_or_else(|| {
                        Error::Execution(format!("no state for resolved head {head}"))
                    })?;
                Some(InFlightPayload::open(
                    &self.chain,
                    version,
                    &parent,
                    &parent_state,
                    attributes,
                )?)
            }
            None => None,
        };

        // Canonicalize the chain onto head (reorging if it is on an alternate fork), then move the
        // safe/finalized pointers.
        if !self.chain.apply_forkchoice(&head_header, pointers, &self.known_blocks)? {
            return Err(Error::Execution(format!("head {head} is known but not reachable")));
        }
        // The head is canonical now, so there is nothing left to sync towards.
        self.sync_target = None;
        // The head may have moved past (or away from) an earlier in-flight payload's parent; drop
        // it so a later get_payload for it reports UnknownPayloadId rather than re-sealing
        // a stale block.
        self.evict_stale_in_flight();

        let valid =
            ForkchoiceUpdated::from_status(PayloadStatusEnum::Valid).with_latest_valid_hash(head);
        let Some(in_flight) = opened else {
            return Ok(valid);
        };
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

    /// The head a forkchoice update last reported `SYNCING` for and has not since resolved
    /// (`optest_syncTarget`), or `None` when the engine is not behind. The Go harness reads this to
    /// drive the block backfill that stands in for a real EL's devp2p snap sync.
    pub const fn sync_target(&self) -> Option<B256> {
        self.sync_target
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
    /// for it then reports an unknown payload. Called on every head-moving path (a forkchoice
    /// update, a head-extending `import_block`) so the same `getPayload` returns
    /// `UnknownPayloadId`.
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

    /// Build the OP-enriched RPC receipts of a block by hash (`eth_getBlockReceipts`), or `None` if
    /// the block is unknown.
    pub fn rpc_receipts_by_block_hash(
        &self,
        hash: B256,
    ) -> Result<Option<Vec<OpTransactionReceipt>>> {
        self.chain.rpc_receipts_by_block_hash(hash)
    }

    /// Build the OP-enriched RPC receipt of a transaction (`eth_getTransactionReceipt`), or `None`
    /// if the transaction is unknown.
    pub fn rpc_receipt_by_tx_hash(&self, tx_hash: B256) -> Result<Option<OpTransactionReceipt>> {
        self.chain.rpc_receipt_by_tx_hash(tx_hash)
    }

    /// Execute a call request read-only at the state of block `block_hash` (`eth_call`),
    /// returning the call output. A revert surfaces as [`Error::Revert`].
    pub fn eth_call(
        &self,
        block_hash: B256,
        request: alloy_rpc_types_eth::TransactionRequest,
    ) -> Result<Bytes> {
        ethcall::call(&self.chain, block_hash, request)
    }

    /// Estimate the lowest gas limit that lets `request` succeed at the state of block
    /// `block_hash` (`eth_estimateGas`), mirroring op-geth's estimator (see `ethcall`).
    pub fn estimate_gas(
        &self,
        block_hash: B256,
        request: alloy_rpc_types_eth::TransactionRequest,
    ) -> Result<u64> {
        ethcall::estimate_gas(&self.chain, block_hash, request)
    }

    /// Park a raw transaction in the pending buffer (`eth_sendRawTransaction`), returning its hash.
    ///
    /// The transaction is decoded and indexed by sender and nonce but not executed or validated
    /// beyond decoding — it waits until [`include_next_tx`](Self::include_next_tx) drains it into a
    /// block. This backs the Go action tests' `EthClient().SendTransaction`.
    pub fn send_raw_transaction(&mut self, raw: &[u8]) -> Result<B256> {
        let tx = OpTransactionSigned::decode_2718_exact(raw)
            .map_err(|err| Error::TxDecode(err.to_string()))?;
        let sender = tx.recover_signer().map_err(|err| Error::TxDecode(err.to_string()))?;
        let nonce = tx.nonce();
        // The EIP-2718 hash is the keccak of the canonical encoding, i.e. exactly `raw`.
        let hash = keccak256(raw);
        self.pending.entry(sender).or_default().insert(nonce, Bytes::copy_from_slice(raw));
        Ok(hash)
    }

    /// The pending nonce of `address` (`eth_getTransactionCount(addr, "pending")`): the sender's
    /// nonce in the latest committed state plus the run of parked transactions that continue from
    /// it without a gap. This is what the Go tests' `PendingNonceAt` reads to pick the next nonce.
    pub fn pending_nonce(&self, address: Address) -> Result<u64> {
        let state = self.chain.state_at(self.chain.latest_header().hash())?.ok_or_else(|| {
            Error::Execution("no state for latest block to read pending nonce".to_string())
        })?;
        let mut nonce = state.account_nonce(&address)?.unwrap_or_default();
        if let Some(parked) = self.pending.get(&address) {
            while parked.contains_key(&nonce) {
                nonce += 1;
            }
        }
        Ok(nonce)
    }

    /// Include the next parked transaction from `from` in the block being built
    /// (`optest_includeNextTx`). It stays parked until the committed state passes its nonce.
    ///
    /// The eligible transaction is the one whose nonce equals `from`'s nonce in the parent state
    /// plus the number of `from`'s transactions already included in this block — exactly what
    /// `firstValidTx` selects against the geth engine. Backs the Go `ActL2IncludeTx(from)`.
    pub fn include_next_tx(&mut self, from: Address) -> Result<IncludeNextOutcome> {
        let id = self.current.ok_or(Error::NotBuildingBlock)?;
        let in_flight = self.in_flight.get(&id).ok_or(Error::NotBuildingBlock)?;
        let parent_hash = in_flight.parent_hash();
        let included = in_flight.included_count_from(from);

        let state = self
            .chain
            .state_at(parent_hash)?
            .ok_or_else(|| Error::Execution(format!("no state for parent block {parent_hash}")))?;
        let base = state.account_nonce(&from)?.unwrap_or_default();
        let want = base + included;

        let Some(parked) = self.pending.get_mut(&from) else {
            return Ok(IncludeNextOutcome::NoTx);
        };
        // The build's parent is the canonical head, so nonces below `base` are committed.
        parked.retain(|&nonce, _| nonce >= base);
        let Some(raw) = parked.get(&want).cloned() else {
            return Ok(IncludeNextOutcome::NoTx);
        };

        match self.include_tx(Some(id), raw.as_ref())? {
            IncludeTxOutcome::Included { tx_hash, gas_used } => {
                Ok(IncludeNextOutcome::Included { tx_hash, gas_used })
            }
            IncludeTxOutcome::Skipped => Ok(IncludeNextOutcome::Skipped),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testsupport::{head_only, sequence, test_engine, user_sender, user_tx};

    fn genesis_hash(engine: &TestEngine) -> B256 {
        engine.chain.genesis_hash()
    }

    #[test]
    fn imports_a_chain_of_payloads_before_any_forkchoice_update() {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let b1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let b2 = sequence(&mut seq, b1.payload.block_hash(), 4, &[user_tx(1)]);

        // b2's parent b1 is processed but not yet canonical: it must execute on b1's state.
        let mut ver = test_engine(user_sender());
        assert!(ver.new_payload(b1).unwrap().is_valid());
        let status = ver.new_payload(b2.clone()).unwrap();
        assert!(status.is_valid(), "b2 on a not yet canonical parent: {status:?}");

        let b2_hash = b2.payload.block_hash();
        assert!(ver.forkchoice_updated_as_op_node(head_only(b2_hash), None).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), b2_hash);
        assert_eq!(ver.chain.latest_header().number, 2);
    }

    #[test]
    fn extends_an_abandoned_fork_and_flips_back() {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a1_hash = a1.payload.block_hash();
        let a2 = sequence(&mut seq, a1_hash, 4, &[user_tx(1)]);
        let b2 = sequence(&mut seq, a1_hash, 5, &[user_tx(1)]);
        let b3 = sequence(&mut seq, b2.payload.block_hash(), 7, &[user_tx(2)]);
        let (a2_hash, b2_hash, b3_hash) =
            (a2.payload.block_hash(), b2.payload.block_hash(), b3.payload.block_hash());

        let mut ver = test_engine(user_sender());
        assert!(ver.new_payload(a1).unwrap().is_valid());
        assert!(ver.new_payload(a2).unwrap().is_valid());
        assert!(ver.forkchoice_updated_as_op_node(head_only(a2_hash), None).unwrap().is_valid());

        // b2 forks off the canonical chain; b3 extends that abandoned fork.
        assert!(ver.new_payload(b2).unwrap().is_valid());
        let status = ver.new_payload(b3).unwrap();
        assert!(status.is_valid(), "b3 on the side block b2: {status:?}");

        assert!(ver.forkchoice_updated_as_op_node(head_only(b3_hash), None).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), b3_hash);
        assert_eq!(ver.block_by_number(2).unwrap().unwrap().header.hash_slow(), b2_hash);

        assert!(ver.forkchoice_updated_as_op_node(head_only(a2_hash), None).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), a2_hash);
        assert!(ver.block_by_number(3).unwrap().is_none(), "b3 reorged out");
    }

    #[test]
    fn executes_a_deep_side_chain_on_its_own_state() {
        let sender = user_sender();
        let mut seq = test_engine(sender);
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a1_hash = a1.payload.block_hash();
        let a2 = sequence(&mut seq, a1_hash, 4, &[user_tx(1)]);
        // b2..b4 fork off a1, each spending the sender's next nonce, so every block only executes
        // on a state that has applied all of its side ancestors in order.
        let b2 = sequence(&mut seq, a1_hash, 5, &[user_tx(1)]);
        let b3 = sequence(&mut seq, b2.payload.block_hash(), 7, &[user_tx(2)]);
        let b4 = sequence(&mut seq, b3.payload.block_hash(), 9, &[user_tx(3)]);
        let b4_hash = b4.payload.block_hash();

        let mut ver = test_engine(sender);
        for payload in [a1, a2.clone()] {
            assert!(ver.new_payload(payload).unwrap().is_valid());
        }
        assert!(
            ver.forkchoice_updated_as_op_node(head_only(a2.payload.block_hash()), None)
                .unwrap()
                .is_valid()
        );
        for (name, payload) in [("b2", b2), ("b3", b3), ("b4", b4)] {
            let status = ver.new_payload(payload).unwrap();
            assert!(status.is_valid(), "{name} on its side parent: {status:?}");
        }

        assert!(ver.forkchoice_updated_as_op_node(head_only(b4_hash), None).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), b4_hash);
        assert_eq!(ver.chain.latest_header().number, 4);
        let state = ver.chain.state_at(b4_hash).unwrap().expect("state at b4");
        assert_eq!(state.account_nonce(&sender).unwrap(), Some(4));
        let receipts = ver.receipts_by_block_hash(b4_hash).unwrap().expect("b4 receipts");
        assert_eq!(receipts.len(), 1);
        assert!(alloy_consensus::TxReceipt::status(&receipts[0]), "b4's transaction succeeded");
    }

    #[test]
    fn builds_on_a_side_block() {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a1_hash = a1.payload.block_hash();
        let a2 = sequence(&mut seq, a1_hash, 4, &[user_tx(1)]);
        let b2 = sequence(&mut seq, a1_hash, 5, &[user_tx(1)]);
        let b2_hash = b2.payload.block_hash();

        let mut ver = test_engine(user_sender());
        for payload in [a1, a2.clone(), b2] {
            assert!(ver.new_payload(payload).unwrap().is_valid());
        }
        assert!(
            ver.forkchoice_updated_as_op_node(head_only(a2.payload.block_hash()), None)
                .unwrap()
                .is_valid()
        );

        // Building on the side block b2 reorgs onto it and opens the build on its state.
        let updated = ver
            .forkchoice_updated_as_op_node(
                head_only(b2_hash),
                Some(crate::testsupport::payload_attrs(7, vec![], false)),
            )
            .expect("fcu with attrs on a side head");
        assert!(updated.is_valid(), "{updated:?}");
        let id = updated.payload_id.expect("payload id");
        assert_eq!(ver.chain.latest_header().hash(), b2_hash);
        ver.include_tx(None, &crate::testsupport::encode(&user_tx(2)))
            .expect("include on b2's state");
        let sealed = ver.get_payload(id).expect("seal");
        assert_eq!(sealed.payload.as_v1().parent_hash, b2_hash);
        assert!(ver.new_payload(sealed).unwrap().is_valid());
    }

    /// A verifier that has imported the canonical chain a1 <- a2 (head) and the side block b2
    /// forking off a1. Returns the engine and the hashes (a1, a2, b2).
    fn forked_verifier() -> (TestEngine, B256, B256, B256) {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a1_hash = a1.payload.block_hash();
        let a2 = sequence(&mut seq, a1_hash, 4, &[user_tx(1)]);
        let b2 = sequence(&mut seq, a1_hash, 5, &[user_tx(1)]);
        let (a2_hash, b2_hash) = (a2.payload.block_hash(), b2.payload.block_hash());

        let mut ver = test_engine(user_sender());
        for payload in [a1, a2, b2] {
            assert!(ver.new_payload(payload).unwrap().is_valid());
        }
        assert!(ver.forkchoice_updated_as_op_node(head_only(a2_hash), None).unwrap().is_valid());
        (ver, a1_hash, a2_hash, b2_hash)
    }

    fn forkchoice(head: B256, safe: B256, finalized: B256) -> ForkchoiceState {
        ForkchoiceState {
            head_block_hash: head,
            safe_block_hash: safe,
            finalized_block_hash: finalized,
        }
    }

    #[test]
    fn rejects_safe_block_on_an_abandoned_fork() {
        let (mut ver, _, a2, b2) = forked_verifier();
        let err =
            ver.forkchoice_updated_as_op_node(forkchoice(a2, b2, B256::ZERO), None).unwrap_err();
        assert!(
            matches!(
                err,
                Error::NonCanonicalForkchoiceBlock { which: ForkchoicePointer::Safe, .. }
            ),
            "{err:?}"
        );
        assert!(ver.chain.safe_header().is_none(), "safe pointer moved");
    }

    #[test]
    fn rejects_safe_block_the_update_reorgs_out() {
        let (mut ver, _, a2, b2) = forked_verifier();
        let err =
            ver.forkchoice_updated_as_op_node(forkchoice(b2, a2, B256::ZERO), None).unwrap_err();
        assert!(
            matches!(
                err,
                Error::NonCanonicalForkchoiceBlock { which: ForkchoicePointer::Safe, .. }
            ),
            "{err:?}"
        );
        // Validated before anything moved: the reorg onto b2 did not happen.
        assert_eq!(ver.chain.latest_header().hash(), a2);
    }

    #[test]
    fn rejects_finalized_block_off_the_new_chain() {
        let (mut ver, _, a2, b2) = forked_verifier();
        let err =
            ver.forkchoice_updated_as_op_node(forkchoice(b2, B256::ZERO, a2), None).unwrap_err();
        assert!(
            matches!(
                err,
                Error::NonCanonicalForkchoiceBlock { which: ForkchoicePointer::Finalized, .. }
            ),
            "{err:?}"
        );
        assert_eq!(ver.chain.latest_header().hash(), a2);
    }

    #[test]
    fn rejects_unknown_finalized_block() {
        let (mut ver, _, a2, _) = forked_verifier();
        let unknown = B256::repeat_byte(0xfe);
        let err = ver.forkchoice_updated_as_op_node(forkchoice(a2, a2, unknown), None).unwrap_err();
        assert!(
            matches!(
                err,
                Error::UnknownForkchoiceBlock { which: ForkchoicePointer::Finalized, .. }
            ),
            "{err:?}"
        );
        assert!(ver.chain.safe_header().is_none(), "safe pointer moved");
    }

    #[test]
    fn accepts_safe_and_finalized_ancestors_across_a_reorg() {
        let (mut ver, a1, _, b2) = forked_verifier();
        let updated = ver.forkchoice_updated_as_op_node(forkchoice(b2, a1, a1), None).unwrap();
        assert!(updated.is_valid(), "{updated:?}");
        assert_eq!(ver.chain.latest_header().hash(), b2);
        assert_eq!(ver.chain.safe_header().map(|h| h.hash()), Some(a1));
        assert_eq!(ver.chain.finalized_header().map(|h| h.hash()), Some(a1));
    }

    #[test]
    fn attributes_on_an_unknown_head_report_syncing() {
        let (mut ver, ..) = forked_verifier();
        let unknown = B256::repeat_byte(0xab);
        let attrs = crate::testsupport::payload_attrs(9, vec![], false);
        let updated =
            ver.forkchoice_updated_as_op_node(head_only(unknown), Some(attrs)).expect("fcu");
        assert!(updated.is_syncing(), "{updated:?}");
        assert!(updated.payload_id.is_none());
        assert_eq!(ver.sync_target(), Some(unknown));
    }

    #[test]
    fn invalid_attributes_on_a_side_head_leave_the_chain_untouched() {
        let (mut ver, _, a2, b2) = forked_verifier();
        // b2 has timestamp 5, so attributes at 5 do not follow it.
        let attrs = crate::testsupport::payload_attrs(5, vec![], false);
        let err = ver.forkchoice_updated_as_op_node(head_only(b2), Some(attrs)).unwrap_err();
        assert!(matches!(err, Error::InvalidPayloadAttributes(_)), "{err:?}");
        assert_eq!(ver.chain.latest_header().hash(), a2, "head moved");
    }

    #[test]
    fn an_unappliable_forced_transaction_is_invalid_attributes() {
        let (mut ver, _, a2, b2) = forked_verifier();
        // The sender's nonce on b2 is 2; a forced transaction with nonce 7 cannot be applied.
        let forced = vec![crate::testsupport::encode(&user_tx(7))];
        let attrs = crate::testsupport::payload_attrs(7, forced, true);
        let err = ver.forkchoice_updated_as_op_node(head_only(b2), Some(attrs)).unwrap_err();
        assert!(matches!(err, Error::InvalidPayloadAttributes(_)), "{err:?}");
        assert_eq!(ver.chain.latest_header().hash(), a2, "head moved");
    }

    #[test]
    fn forkchoice_errors_take_precedence_over_attributes() {
        let (mut ver, _, a2, b2) = forked_verifier();
        let attrs = crate::testsupport::payload_attrs(7, vec![], false);
        let err = ver
            .forkchoice_updated_as_op_node(forkchoice(a2, b2, B256::ZERO), Some(attrs))
            .unwrap_err();
        assert!(matches!(err, Error::NonCanonicalForkchoiceBlock { .. }), "{err:?}");
        assert!(ver.in_flight.is_empty(), "build opened despite the forkchoice error");
    }

    #[test]
    fn importing_a_canonical_block_moves_nothing() {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a2 = sequence(&mut seq, a1.payload.block_hash(), 4, &[user_tx(1)]);
        let a2_hash = a2.payload.block_hash();

        let mut ver = test_engine(user_sender());
        assert!(ver.import_block(a1.clone()).unwrap().is_valid());
        assert!(ver.import_block(a2).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), a2_hash);
        let attrs = crate::testsupport::payload_attrs(6, vec![], false);
        let id =
            ver.forkchoice_updated_as_op_node(head_only(a2_hash), Some(attrs)).unwrap().payload_id;

        let status = ver.import_block(a1).unwrap();
        assert!(status.is_valid(), "{status:?}");
        assert_eq!(ver.chain.latest_header().hash(), a2_hash, "head moved back");
        ver.get_payload(id.expect("payload id")).expect("in-flight build survives");
    }

    #[test]
    fn importing_a_block_off_the_head_does_not_move_it() {
        let mut seq = test_engine(user_sender());
        let g = genesis_hash(&seq);
        let a1 = sequence(&mut seq, g, 2, &[user_tx(0)]);
        let a1_hash = a1.payload.block_hash();
        let a2 = sequence(&mut seq, a1_hash, 4, &[user_tx(1)]);
        let b2 = sequence(&mut seq, a1_hash, 5, &[user_tx(1)]);
        let (a2_hash, b2_hash) = (a2.payload.block_hash(), b2.payload.block_hash());

        let mut ver = test_engine(user_sender());
        for payload in [a1, a2] {
            assert!(ver.import_block(payload).unwrap().is_valid());
        }
        assert!(ver.import_block(b2).unwrap().is_valid());
        assert_eq!(ver.chain.latest_header().hash(), a2_hash, "head moved onto a side block");
        assert!(ver.forkchoice_updated_as_op_node(head_only(b2_hash), None).unwrap().is_valid());
    }
}
