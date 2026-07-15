//! Stateful in-flight payload building: `forkchoice_updated`-with-attributes →
//! [`include_tx`](InFlightPayload::include_tx)\* → [`get_payload`](InFlightPayload::get_payload).
//!
//! An [`InFlightPayload`] is opened by a forkchoice update that carries payload attributes. It
//! fixes the parent, the next-block environment, and the sequencer (deposit) transactions, then
//! accepts pool transactions one at a time before being sealed into an execution payload.
//!
//! reth's block builder ties the executor, EVM state, and parent header to a single borrowed
//! lifetime, so it cannot be parked across RPC round-trips. Instead the in-flight payload keeps the
//! ordered transaction list and re-runs [`EphemeralChain::assemble_block`] on demand; block
//! assembly is deterministic, so re-execution yields the same block the eventual `get_payload`
//! seals. The semantics mirror `op-e2e/actions/helpers/engineapi`'s `L2EngineAPI`/`BlockProcessor`
//! (deposits applied at block start, `no_tx_pool` → force-empty, the gas-limit checks in
//! `CheckTxWithinGasLimit`).

use alloy_consensus::{Transaction as _, transaction::SignerRecoverable};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{Address, B256, keccak256};
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload, OpPayloadAttributes};
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_payload_builder::OpPayloadBuilderAttributes;
use reth_optimism_primitives::OpTransactionSigned;
use reth_payload_primitives::BuildNextEnv;

use crate::{Error, chain::EphemeralChain};

/// Engine API message version fed into the payload-id derivation. The id is opaque to consumers —
/// only its consistency between the opening forkchoice update and the later `get_payload` matters —
/// so a fixed version suffices.
const PAYLOAD_VERSION: u8 = 3;

/// The outcome of an [`include_tx`](crate::TestEngine::include_tx) call.
#[derive(Debug)]
pub enum IncludeTxOutcome {
    /// The transaction was executed and appended to the block.
    Included {
        /// The included transaction's hash.
        tx_hash: B256,
        /// Gas the transaction consumed.
        gas_used: u64,
    },
    /// Force-empty is set, so the transaction was silently dropped. Mirrors `L2EngineAPI.IncludeTx`
    /// returning `(nil, nil)` when `l2ForceEmpty` is true.
    Skipped,
}

/// The outcome of an [`include_next_tx`](crate::TestEngine::include_next_tx) call — including the
/// next parked transaction from a given sender.
#[derive(Debug)]
pub enum IncludeNextOutcome {
    /// A parked transaction was found and executed into the block.
    Included {
        /// The included transaction's hash.
        tx_hash: B256,
        /// Gas the transaction consumed.
        gas_used: u64,
    },
    /// Force-empty is set, so nothing was included (mirrors `ActL2IncludeTx`'s force-empty skip).
    Skipped,
    /// No parked transaction from the sender was valid for inclusion next (its next expected nonce
    /// is not present in the buffer). Mirrors `firstValidTx` finding no pending transaction.
    NoTx,
}

/// A payload being built on top of a fixed parent.
#[derive(Debug)]
pub(crate) struct InFlightPayload {
    id: PayloadId,
    parent_hash: B256,
    next_env: OpNextBlockEnvAttributes,
    /// Sequencer (deposit) transactions from the payload attributes; always included first.
    deposits: Vec<OpTransactionSigned>,
    /// Pool transactions added via `include_tx`, in inclusion order.
    pool_txs: Vec<OpTransactionSigned>,
    /// Reset to `attributes.no_tx_pool` when the block is opened; consulted (and mutable) by
    /// `include_tx`. When set, `include_tx` is a no-op.
    force_empty: bool,
    gas_limit: u64,
    /// Gas used by all currently-included transactions (`deposits` + `pool_txs`).
    cumulative_gas: u64,
}

impl InFlightPayload {
    /// Open a new in-flight payload on top of `parent_hash` from the given attributes.
    ///
    /// Decodes and applies the deposit transactions (an invalid deposit fails here, mirroring
    /// `L2EngineAPI.startBlock`), derives the next-block environment (Holocene/Jovian `extraData`,
    /// gas limit), and seeds the cumulative gas from a deposits-only assembly.
    pub(crate) fn open(
        chain: &EphemeralChain,
        parent_hash: B256,
        attributes: OpPayloadAttributes,
    ) -> crate::Result<Self> {
        let builder_attrs = OpPayloadBuilderAttributes::<OpTransactionSigned>::try_new(
            parent_hash,
            attributes,
            PAYLOAD_VERSION,
        )
        .map_err(|err| Error::Execution(format!("invalid payload attributes: {err}")))?;

        let parent = chain
            .sealed_header(parent_hash)?
            .ok_or_else(|| Error::Execution(format!("parent block {parent_hash} is unknown")))?;
        let next_env = OpNextBlockEnvAttributes::build_next_env(
            &builder_attrs,
            &parent,
            chain.chain_spec().as_ref(),
        )
        .map_err(|err| Error::Execution(format!("build next block env: {err}")))?;

        let id = builder_attrs.id;
        let force_empty = builder_attrs.no_tx_pool;
        let gas_limit = next_env.gas_limit;
        let deposits: Vec<OpTransactionSigned> =
            builder_attrs.transactions.into_iter().map(|tx| tx.into_value()).collect();

        // Assemble the deposits-only block once: this validates the deposits (a bad one errors, as
        // in op-geth's startBlock) and gives the starting cumulative gas.
        let cumulative_gas =
            chain.assemble_block(parent_hash, next_env.clone(), &deposits)?.gas_used;

        Ok(Self {
            id,
            parent_hash,
            next_env,
            deposits,
            pool_txs: Vec::new(),
            force_empty,
            gas_limit,
            cumulative_gas,
        })
    }

    /// The payload id assigned when this block was opened.
    pub(crate) const fn id(&self) -> PayloadId {
        self.id
    }

    /// The parent hash this block is being built on top of.
    pub(crate) const fn parent_hash(&self) -> B256 {
        self.parent_hash
    }

    /// How many pool transactions from `from` have already been included in this block. Combined
    /// with the sender's nonce in the parent state, this gives the nonce of the next transaction
    /// from `from` eligible for inclusion — the parking buffer's lookup key. Mirrors
    /// `L2EngineAPI.PendingIndices`.
    pub(crate) fn included_count_from(&self, from: Address) -> u64 {
        self.pool_txs
            .iter()
            .filter(|tx| tx.recover_signer().is_ok_and(|signer| signer == from))
            .count() as u64
    }

    /// Whether force-empty is set. Mirrors `L2EngineAPI.ForcedEmpty`.
    pub(crate) const fn forced_empty(&self) -> bool {
        self.force_empty
    }

    /// Set the force-empty flag. Mirrors `L2EngineAPI.SetForceEmpty`.
    pub(crate) const fn set_force_empty(&mut self, value: bool) {
        self.force_empty = value;
    }

    /// Gas remaining in the block. Mirrors `L2EngineAPI.RemainingBlockGas` (the block gas pool).
    pub(crate) const fn remaining_block_gas(&self) -> u64 {
        self.gas_limit.saturating_sub(self.cumulative_gas)
    }

    /// The deposits followed by the pool transactions, in block order.
    fn all_txs(&self) -> Vec<OpTransactionSigned> {
        let mut txs = self.deposits.clone();
        txs.extend(self.pool_txs.iter().cloned());
        txs
    }

    /// Decode, gas-check, and execute a pool transaction, appending it on success.
    ///
    /// Mirrors `L2EngineAPI.IncludeTx` + `BlockProcessor.CheckTxWithinGasLimit`: a force-empty
    /// block drops the transaction ([`IncludeTxOutcome::Skipped`]); a declared gas limit above the
    /// block gas limit or above the remaining gas is rejected with [`Error::ExceedsGasLimit`] /
    /// [`Error::UsesTooMuchGas`]; a transaction that fails execution errors without being appended.
    pub(crate) fn include_tx(
        &mut self,
        chain: &EphemeralChain,
        raw: &[u8],
    ) -> crate::Result<IncludeTxOutcome> {
        if self.force_empty {
            return Ok(IncludeTxOutcome::Skipped);
        }

        let tx = OpTransactionSigned::decode_2718_exact(raw)
            .map_err(|err| Error::TxDecode(err.to_string()))?;
        let tx_gas = tx.gas_limit();
        if tx_gas > self.gas_limit {
            return Err(Error::ExceedsGasLimit { tx_gas, block_gas_limit: self.gas_limit });
        }
        let remaining = self.remaining_block_gas();
        if tx_gas > remaining {
            return Err(Error::UsesTooMuchGas { tx_gas, remaining });
        }
        // The EIP-2718 transaction hash is the keccak of its canonical encoding, which is exactly
        // the `raw` bytes `decode_2718_exact` just consumed in full.
        let tx_hash = keccak256(raw);

        // Re-run the whole list with the candidate appended. On any execution error the candidate
        // is not retained, so the in-flight payload is unchanged.
        let mut txs = self.all_txs();
        txs.push(tx.clone());
        let built = chain.assemble_block(self.parent_hash, self.next_env.clone(), &txs)?;

        let gas_used = built.gas_used - self.cumulative_gas;
        self.pool_txs.push(tx);
        self.cumulative_gas = built.gas_used;
        Ok(IncludeTxOutcome::Included { tx_hash, gas_used })
    }

    /// Seal the block and return it as execution-payload data ready for `new_payload`.
    ///
    /// Mirrors `L2EngineAPI.getPayload` → `endBlock`. Idempotent: the in-flight payload is not
    /// consumed, so the sealed payload can be fetched more than once.
    pub(crate) fn get_payload(&self, chain: &EphemeralChain) -> crate::Result<OpExecutionData> {
        let built =
            chain.assemble_block(self.parent_hash, self.next_env.clone(), &self.all_txs())?;
        let block = built.block.clone_sealed_block().into_block();
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
        Ok(OpExecutionData::new(payload, sidecar))
    }
}

#[cfg(test)]
mod tests {
    use super::IncludeTxOutcome;
    use crate::{
        Error, TestEngine,
        testsupport::{
            GAS_LIMIT, deposit_tx, depositor, encode, fcu, payload_attrs, test_engine, user_sender,
            user_tx, user_tx_with_gas,
        },
    };

    use alloy_consensus::{BlockHeader, TxReceipt, transaction::SignerRecoverable};
    use alloy_primitives::B256;
    use reth_optimism_primitives::OpReceipt;

    /// Open a payload on `parent`, include `user_txs`, seal it, import it, and advance the
    /// forkchoice — the full sequencer flow. Returns the new head hash.
    fn build_block(
        engine: &mut TestEngine,
        parent: B256,
        timestamp: u64,
        deposits: Vec<alloy_primitives::Bytes>,
        user_txs: &[reth_optimism_primitives::OpTransactionSigned],
    ) -> B256 {
        let updated = engine
            .forkchoice_updated(fcu(parent), Some(payload_attrs(timestamp, deposits, false)))
            .expect("fcu with attrs");
        assert!(updated.is_valid(), "fcu(attrs) valid: {updated:?}");
        let id = updated.payload_id.expect("payload id returned");

        for tx in user_txs {
            match engine.include_tx(None, &encode(tx)).expect("include tx") {
                IncludeTxOutcome::Included { .. } => {}
                IncludeTxOutcome::Skipped => panic!("tx unexpectedly skipped"),
            }
        }

        let data = engine.get_payload(id).expect("get payload");
        let status = engine.new_payload(data).expect("new payload");
        assert!(status.is_valid(), "newPayload valid: {status:?}");
        let head = status.latest_valid_hash.expect("latest valid hash");

        let updated = engine.forkchoice_updated(fcu(head), None).expect("fcu advance");
        assert!(updated.is_valid(), "fcu advance valid: {updated:?}");
        head
    }

    #[test]
    fn sequencer_flow_builds_chain() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // Block 1: one forced deposit plus a user tx.
        let block1 = build_block(
            &mut engine,
            genesis,
            2,
            vec![encode(&deposit_tx(depositor()))],
            &[user_tx(0)],
        );
        // Block 2 builds on block 1 with a second user tx (no deposit).
        let block2 = build_block(&mut engine, block1, 4, vec![], &[user_tx(1)]);

        // A valid chain of two blocks on top of genesis.
        let h1 = engine.block_by_number(1).unwrap().expect("block 1");
        let h2 = engine.block_by_number(2).unwrap().expect("block 2");
        assert_eq!(h1.header.number(), 1);
        assert_eq!(h1.header.parent_hash, genesis);
        assert_eq!(h1.header.hash_slow(), block1);
        assert_eq!(h2.header.number(), 2);
        assert_eq!(h2.header.parent_hash, block1);
        assert_eq!(h2.header.hash_slow(), block2);
        assert_ne!(block1, block2);

        // Block 1's committed receipts: deposit first (with OP-specific fields), then the user tx.
        let receipts =
            engine.receipts_by_block_hash(block1).unwrap().expect("block 1 receipts present");
        assert_eq!(receipts.len(), 2, "deposit + user tx");
        let OpReceipt::Deposit(deposit) = &receipts[0] else {
            panic!("first receipt should be a deposit, got {:?}", receipts[0]);
        };
        assert!(deposit.deposit_nonce.is_some(), "deposit nonce present");
        assert_eq!(deposit.deposit_receipt_version, Some(1), "post-Canyon receipt version");
        assert!(!matches!(receipts[1], OpReceipt::Deposit(_)), "user tx is not a deposit");
        assert!(receipts[1].status(), "user tx succeeded");
    }

    #[test]
    fn forkchoice_reorgs_to_alternate_fork_and_back() {
        use alloy_rpc_types_engine::ForkchoiceState;

        // Drive a block on `parent` without moving safe/finalized, so a reorg can later reset the
        // head below them freely.
        fn head_only(head: B256) -> ForkchoiceState {
            ForkchoiceState {
                head_block_hash: head,
                safe_block_hash: B256::ZERO,
                finalized_block_hash: B256::ZERO,
            }
        }
        fn build(
            engine: &mut TestEngine,
            parent: B256,
            timestamp: u64,
            user_tx: &reth_optimism_primitives::OpTransactionSigned,
        ) -> B256 {
            let updated = engine
                .forkchoice_updated(
                    head_only(parent),
                    Some(payload_attrs(timestamp, vec![], false)),
                )
                .expect("fcu with attrs");
            let id = updated.payload_id.expect("payload id");
            engine.include_tx(None, &encode(user_tx)).expect("include tx");
            let data = engine.get_payload(id).expect("get payload");
            let head = engine.new_payload(data).expect("new payload").latest_valid_hash.unwrap();
            engine.forkchoice_updated(head_only(head), None).expect("fcu advance");
            head
        }

        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // Canonical chain genesis -> a1 -> a2 -> a3.
        let a1 = build(&mut engine, genesis, 2, &user_tx(0));
        let a2 = build(&mut engine, a1, 4, &user_tx(1));
        let a3 = build(&mut engine, a2, 6, &user_tx(2));
        assert_eq!(engine.chain.latest_header().hash(), a3);

        // Build a sibling of a2 on a1 (a different timestamp yields a different hash), reorging a2
        // and a3 out — the shape of op-node's rewind / invalid-payload replacement.
        let b2 = build(&mut engine, a1, 5, &user_tx(1));
        assert_ne!(b2, a2);
        assert_eq!(engine.chain.latest_header().hash(), b2);
        assert_eq!(engine.chain.latest_header().number, 2);
        assert_eq!(engine.block_by_number(2).unwrap().unwrap().header.hash_slow(), b2);
        assert!(engine.block_by_number(3).unwrap().is_none(), "a3 reorged out");
        // The shared ancestor is untouched.
        assert_eq!(engine.block_by_number(1).unwrap().unwrap().header.hash_slow(), a1);

        // Flip back to the original tip a3: a full reorg onto the abandoned fork, re-materialized
        // from the retained known_blocks.
        let updated = engine.forkchoice_updated(head_only(a3), None).expect("fcu back to a3");
        assert!(updated.is_valid(), "reorg back to a3: {updated:?}");
        assert_eq!(engine.chain.latest_header().hash(), a3);
        assert_eq!(engine.chain.latest_header().number, 3);
        assert_eq!(engine.block_by_number(3).unwrap().unwrap().header.hash_slow(), a3);
        assert_eq!(engine.block_by_number(2).unwrap().unwrap().header.hash_slow(), a2);
        assert_eq!(engine.block_by_number(1).unwrap().unwrap().header.hash_slow(), a1);
    }

    #[test]
    fn committed_payload_id_is_evicted() {
        // Mirrors TestL2SequencerAPI: once a payload is committed and the head advances past its
        // parent, re-sealing it (get_payload) must report UnknownPayloadId — op-node maps that code
        // to BuildErrCodeUnknownPayload.
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        let updated = engine
            .forkchoice_updated(fcu(genesis), Some(payload_attrs(2, vec![], false)))
            .expect("fcu with attrs");
        let id = updated.payload_id.expect("payload id returned");

        // Sealing before the commit succeeds.
        let data = engine.get_payload(id).expect("get payload before commit");
        let status = engine.new_payload(data).expect("new payload");
        assert!(status.is_valid(), "newPayload valid: {status:?}");

        // The build job is gone once its parent is no longer the head.
        let err = engine.get_payload(id).unwrap_err();
        assert!(
            matches!(err, Error::UnknownPayloadId(evicted) if evicted == id),
            "expected UnknownPayloadId({id}), got {err:?}"
        );
    }

    #[test]
    fn include_tx_reports_gas_and_updates_remaining() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // Open an empty block (no deposits) so remaining gas starts at the full gas limit.
        let updated =
            engine.forkchoice_updated(fcu(genesis), Some(payload_attrs(2, vec![], false))).unwrap();
        assert!(updated.is_valid());
        assert_eq!(engine.remaining_block_gas(None), GAS_LIMIT);

        let outcome = engine.include_tx(None, &encode(&user_tx(0))).unwrap();
        let IncludeTxOutcome::Included { gas_used, .. } = outcome else {
            panic!("expected inclusion, got {outcome:?}");
        };
        // A basic value transfer costs the 21000 intrinsic gas.
        assert_eq!(gas_used, 21_000);
        assert_eq!(engine.remaining_block_gas(None), GAS_LIMIT - 21_000);
    }

    #[test]
    fn include_tx_enforces_gas_limits() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // A tight block gas limit makes the two rejection paths reachable.
        let block_gas = 100_000;
        let mut attrs = payload_attrs(2, vec![], false);
        attrs.gas_limit = Some(block_gas);
        let updated = engine.forkchoice_updated(fcu(genesis), Some(attrs)).unwrap();
        assert!(updated.is_valid());

        // Declared gas above the block gas limit is rejected outright.
        let err =
            engine.include_tx(None, &encode(&user_tx_with_gas(0, block_gas + 1))).unwrap_err();
        assert!(matches!(err, Error::ExceedsGasLimit { .. }), "{err:?}");

        // Include a normal tx (21000 used), leaving 79000 gas.
        engine.include_tx(None, &encode(&user_tx(0))).unwrap();

        // A tx whose declared gas exceeds the remaining gas is rejected as UsesTooMuchGas — and the
        // in-flight payload is left unchanged.
        let err = engine.include_tx(None, &encode(&user_tx_with_gas(1, 90_000))).unwrap_err();
        assert!(matches!(err, Error::UsesTooMuchGas { .. }), "{err:?}");
        assert_eq!(engine.remaining_block_gas(None), block_gas - 21_000);
    }

    #[test]
    fn force_empty_skips_inclusion() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // no_tx_pool sets force-empty at block start.
        let updated =
            engine.forkchoice_updated(fcu(genesis), Some(payload_attrs(2, vec![], true))).unwrap();
        assert!(updated.is_valid());
        assert!(engine.forced_empty(None));

        // include_tx is a silent no-op under force-empty (mirrors op-geth returning nil,nil).
        let outcome = engine.include_tx(None, &encode(&user_tx(0))).unwrap();
        assert!(matches!(outcome, IncludeTxOutcome::Skipped), "{outcome:?}");
        assert_eq!(engine.remaining_block_gas(None), GAS_LIMIT);

        // Clearing force-empty lets the tx in.
        engine.set_force_empty(None, false).unwrap();
        assert!(!engine.forced_empty(None));
        let outcome = engine.include_tx(None, &encode(&user_tx(0))).unwrap();
        assert!(matches!(outcome, IncludeTxOutcome::Included { .. }), "{outcome:?}");
    }

    #[test]
    fn builder_calls_error_when_not_building() {
        let mut engine = test_engine(user_sender());
        // No block is being built.
        assert_eq!(engine.remaining_block_gas(None), 0);
        assert!(!engine.forced_empty(None));
        let err = engine.include_tx(None, &encode(&user_tx(0))).unwrap_err();
        assert!(matches!(err, Error::NotBuildingBlock), "{err:?}");
        let err = engine.get_payload(alloy_rpc_types_engine::PayloadId::new([0; 8])).unwrap_err();
        assert!(matches!(err, Error::UnknownPayloadId(_)), "{err:?}");
    }

    #[test]
    fn parking_buffer_drains_in_nonce_order() {
        use super::IncludeNextOutcome;

        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let sender = user_sender();

        // Open a block so include_next_tx has an in-flight payload to target.
        let updated =
            engine.forkchoice_updated(fcu(genesis), Some(payload_attrs(2, vec![], false))).unwrap();
        assert!(updated.is_valid());

        // Park two user txs out of nonce order — the buffer is nonce-keyed, so order is irrelevant.
        engine.send_raw_transaction(&encode(&user_tx(1))).unwrap();
        engine.send_raw_transaction(&encode(&user_tx(0))).unwrap();

        // Pending nonce = base (0) + the contiguous parked run (0,1) = 2.
        assert_eq!(engine.pending_nonce(sender).unwrap(), 2);

        // Draining includes nonce 0 first, then nonce 1, then reports nothing left.
        assert!(matches!(
            engine.include_next_tx(sender).unwrap(),
            IncludeNextOutcome::Included { .. }
        ));
        assert!(matches!(
            engine.include_next_tx(sender).unwrap(),
            IncludeNextOutcome::Included { .. }
        ));
        assert!(matches!(engine.include_next_tx(sender).unwrap(), IncludeNextOutcome::NoTx));

        // Both parked txs were executed into the block: two 21000-gas transfers consumed 42000.
        assert_eq!(engine.remaining_block_gas(None), GAS_LIMIT - 42_000);
    }

    #[test]
    fn include_next_tx_needs_a_block() {
        let mut engine = test_engine(user_sender());
        engine.send_raw_transaction(&encode(&user_tx(0))).unwrap();
        // No in-flight payload: mirrors ErrNotBuildingBlock.
        assert!(matches!(
            engine.include_next_tx(user_sender()),
            Err(crate::Error::NotBuildingBlock)
        ));
    }

    #[test]
    fn parked_tx_skipped_under_force_empty() {
        use super::IncludeNextOutcome;
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        // no_tx_pool → force-empty at open.
        let updated =
            engine.forkchoice_updated(fcu(genesis), Some(payload_attrs(2, vec![], true))).unwrap();
        assert!(updated.is_valid());
        engine.send_raw_transaction(&encode(&user_tx(0))).unwrap();
        // The parked tx stays parked; inclusion is skipped, not consumed.
        assert!(matches!(
            engine.include_next_tx(user_sender()).unwrap(),
            IncludeNextOutcome::Skipped
        ));
        assert_eq!(engine.pending_nonce(user_sender()).unwrap(), 1, "still parked");
    }

    #[test]
    fn user_sender_is_funded() {
        // Guards the test setup: the canonical test signature recovers to the funded account.
        let sender = user_tx(0).recover_signer().unwrap();
        assert_eq!(sender, user_sender());
    }
}
