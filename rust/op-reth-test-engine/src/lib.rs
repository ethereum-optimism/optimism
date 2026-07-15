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

use alloy_consensus::{BlockHeader as _, Header, Transaction as _, transaction::SignerRecoverable};
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
    /// A forkchoice-update-with-attributes could not open a block from the given attributes — the
    /// attributes are malformed or a forced transaction they carry cannot be applied. Mirrors
    /// op-geth returning `eth.InvalidPayloadAttributes` from `startBlock`; op-node maps this to a
    /// payload error and (under Holocene, for a derived block) requests a deposits-only replacement
    /// rather than retrying or resetting forever.
    #[error("invalid payload attributes: {0}")]
    InvalidPayloadAttributes(String),
    /// A non-zero `safe`/`finalized` forkchoice block is unknown to the chain.
    #[error("forkchoice {which} block {hash} is unknown")]
    UnknownForkchoiceBlock {
        /// Which forkchoice pointer was unknown: `"safe"` or `"finalized"`.
        which: &'static str,
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
    /// An `eth_call`/`eth_estimateGas` execution reverted. Carries the revert output so the RPC
    /// layer can attach it as error data (geth revert errors do the same).
    #[error("execution reverted")]
    Revert(Bytes),
    /// An unsupported operation was requested.
    #[error("unsupported: {0}")]
    Unsupported(&'static str),
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
    /// [`include_next_tx`](Self::include_next_tx) drains. This is deliberately *not* a transaction
    /// pool: parked transactions are never auto-included, gossiped, replaced, revalidated, or
    /// reorg-reinjected — the buffer only exists so the Go action tests' `SendTransaction` /
    /// `PendingNonceAt` / `ActL2IncludeTx(from)` sequence keeps working unchanged over the socket
    /// (the concrete `*ethclient.Client` they use cannot be intercepted Go-side).
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
        match exec::import_payload(&self.chain, payload)? {
            ImportOutcome::Syncing => Ok(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
            ImportOutcome::Invalid(status) => Ok(status),
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
    /// Runs the ordinary `new_payload` validate-and-record path and then advances the canonical
    /// head onto the block, as a real EL's block-sync insertion does — without touching the
    /// safe/finalized pointers, which stay wherever the consensus client last put them.
    pub fn import_block(&mut self, payload: OpExecutionData) -> Result<PayloadStatus> {
        let status = self.new_payload(payload)?;
        if status.status == PayloadStatusEnum::Valid {
            let head = status.latest_valid_hash.expect("valid status carries the block hash");
            let advanced =
                self.chain.advance_forkchoice(head, B256::ZERO, B256::ZERO, &self.known_blocks)?;
            debug_assert!(advanced, "a just-recorded block is always reachable");
            self.evict_stale_in_flight();
        }
        Ok(status)
    }

    /// Update the forkchoice (`engine_forkchoiceUpdated`).
    ///
    /// Canonicalizes `head`, reorging onto it when it is on an alternate fork (using the retained
    /// `known_blocks`): a resolvable `head` yields `VALID`, an unknown or unreachable `head` yields
    /// `SYNCING` (never a silent `VALID`), and an unknown non-zero `safe`/`finalized` is an
    /// [`Error::UnknownForkchoiceBlock`]. When `attributes` is `Some`, a new payload is opened on
    /// top of `head` and its [`PayloadId`] is returned; the deposits are applied immediately, so
    /// invalid attributes error here (mirroring op-geth's `startBlock`).
    pub fn forkchoice_updated(
        &mut self,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated> {
        let head = state.head_block_hash;

        // Open the requested block build *before* moving the head. Building may fail because a
        // forced transaction is invalid (e.g. a derived block whose batch carries a
        // badly-signed tx); surfacing that as an invalid-payload-attributes engine error
        // lets op-node request a deposits-only replacement instead of retrying forever.
        // Opening first means such a failure leaves the canonical chain untouched — the
        // blocks the head move would have orphaned stay queryable, which op-node relies on
        // when it consolidates the next safe attributes against the existing unsafe chain.
        // The parent state an open reads is available whether the head is the current tip
        // or a still-canonical ancestor.
        let opened = match attributes {
            Some(attributes) => Some(
                InFlightPayload::open(&self.chain, head, attributes)
                    .map_err(|err| Error::InvalidPayloadAttributes(err.to_string()))?,
            ),
            None => None,
        };

        // Canonicalize the chain onto head (reorging if it is on an alternate fork), then move the
        // safe/finalized pointers. An unknown or unreachable head reports SYNCING — never a silent
        // VALID — and an unknown non-zero safe/finalized pointer errors.
        let advanced = self.chain.advance_forkchoice(
            head,
            state.safe_block_hash,
            state.finalized_block_hash,
            &self.known_blocks,
        )?;
        if !advanced {
            // Record the head we could not reach so the Go harness (which stands in for the
            // devp2p snap sync a real EL would run) knows to backfill towards it.
            self.sync_target = Some(head);
            return Ok(ForkchoiceUpdated::from_status(PayloadStatusEnum::Syncing));
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
    /// block is being built, mirroring `L2EngineAPI.RemainingBlockGas`.
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

    /// Seal and return the block being built (`engine_getPayload`). The payload is left in flight,
    /// so it can be fetched again; feed the result to [`new_payload`](Self::new_payload) to commit
    /// it.
    pub fn get_payload(&self, id: PayloadId) -> Result<OpExecutionData> {
        self.in_flight.get(&id).ok_or(Error::UnknownPayloadId(id))?.get_payload(&self.chain)
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
    /// (`optest_includeNextTx`), draining it from the buffer on success.
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

        let Some(raw) = self.pending.get(&from).and_then(|parked| parked.get(&want)).cloned()
        else {
            return Ok(IncludeNextOutcome::NoTx);
        };

        match self.include_tx(Some(id), raw.as_ref())? {
            IncludeTxOutcome::Included { tx_hash, gas_used } => {
                if let Some(parked) = self.pending.get_mut(&from) {
                    parked.remove(&want);
                }
                Ok(IncludeNextOutcome::Included { tx_hash, gas_used })
            }
            IncludeTxOutcome::Skipped => Ok(IncludeNextOutcome::Skipped),
        }
    }
}
