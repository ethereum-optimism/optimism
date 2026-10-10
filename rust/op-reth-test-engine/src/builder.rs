//! Stateful in-flight payload building: `forkchoice_updated`-with-attributes →
//! [`include_tx`](InFlightPayload::include_tx)\* → [`get_payload`](InFlightPayload::get_payload).
//!
//! An [`InFlightPayload`] is opened by a forkchoice update that carries payload attributes. It
//! fixes the parent, the next-block environment, and the attributes' transactions — the deposits,
//! followed on a block derived from a batch by the batch's user transactions — then accepts pool
//! transactions one at a time before being sealed into an execution payload.
//!
//! reth's block builder ties the executor, EVM state, and parent header to a single borrowed
//! lifetime, so it cannot be parked across RPC round-trips. Instead the in-flight payload keeps the
//! ordered transaction list and re-runs [`EphemeralChain::assemble_block`] on demand; block
//! assembly is deterministic, so re-execution yields the same block the eventual `get_payload`
//! seals. The semantics mirror `op-e2e/actions/helpers/engineapi`'s `L2EngineAPI`/`BlockProcessor`
//! (forced transactions applied at block start, `no_tx_pool` → force-empty, the gas-limit checks in
//! `CheckTxWithinGasLimit`).

use alloy_consensus::Transaction as _;
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload, OpPayloadAttributes};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_node::engine::ensure_well_formed_payload_attributes;
use reth_optimism_payload_builder::{OpPayloadAttrs, OpPayloadBuilderAttributes};
use reth_optimism_primitives::OpTransactionSigned;
use reth_payload_primitives::{
    BuildNextEnv, EngineApiMessageVersion, InvalidPayloadAttributesError,
};
use reth_primitives_traits::SealedHeader;

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

/// A payload being built on top of a fixed parent.
#[derive(Debug)]
pub(crate) struct InFlightPayload {
    id: PayloadId,
    parent_hash: B256,
    next_env: OpNextBlockEnvAttributes,
    /// The payload attributes' transactions in block order: the deposits, then on a block
    /// derived from a batch the batch's user transactions. Executed before any pool transaction.
    forced_txs: Vec<OpTransactionSigned>,
    /// Pool transactions added via `include_tx`, in inclusion order.
    pool_txs: Vec<OpTransactionSigned>,
    /// Reset to `attributes.no_tx_pool` when the block is opened; consulted (and mutable) by
    /// `include_tx`. When set, `include_tx` is a no-op.
    force_empty: bool,
    gas_limit: u64,
    /// Gas used by all currently-included transactions (`forced_txs` and `pool_txs`).
    cumulative_gas: u64,
    /// The payload `get_payload` sealed. Once set the build is closed: nothing more can be
    /// included, and later `get_payload` calls return this same payload.
    sealed: Option<OpExecutionData>,
}

impl InFlightPayload {
    /// Open a new in-flight payload on top of `parent_hash` from the given attributes.
    ///
    /// Validates the attributes against `version`, the message version of the
    /// `engine_forkchoiceUpdated` method that carried them, the active forks and the parent (an
    /// [`Error::MalformedPayloadAttributes`] or [`Error::InvalidPayloadAttributes`] otherwise),
    /// derives the next-block environment (Holocene/Jovian `extraData`, gas limit), and decodes the
    /// attributes' transactions, which must not put a deposit after a non-deposit (an
    /// [`Error::InvalidPayloadAttributes`] otherwise). It then applies them (an invalid one fails
    /// here, mirroring `L2EngineAPI.startBlock`), seeding the cumulative gas from that assembly.
    pub(crate) fn open(
        chain: &EphemeralChain,
        version: EngineApiMessageVersion,
        parent_hash: B256,
        attributes: OpPayloadAttributes,
    ) -> crate::Result<Self> {
        let parent = chain
            .sealed_header(parent_hash)?
            .ok_or_else(|| Error::Execution(format!("parent block {parent_hash} is unknown")))?;
        let attributes = OpPayloadAttrs(attributes);
        validate_attributes(&chain.chain_spec(), version, &parent, &attributes)?;

        let builder_attrs = OpPayloadBuilderAttributes::<OpTransactionSigned>::try_new(
            parent_hash,
            attributes.0,
            PAYLOAD_VERSION,
        )
        .map_err(|err| Error::InvalidPayloadAttributes(err.to_string()))?;
        let next_env = OpNextBlockEnvAttributes::build_next_env(
            &builder_attrs,
            &parent,
            chain.chain_spec().as_ref(),
        )
        .map_err(|err| Error::InvalidPayloadAttributes(format!("build next block env: {err}")))?;

        let id = builder_attrs.id;
        let force_empty = builder_attrs.no_tx_pool;
        let gas_limit = next_env.gas_limit;
        let forced_txs: Vec<_> =
            builder_attrs.transactions.into_iter().map(|tx| tx.into_value()).collect();
        check_deposits_lead(&forced_txs)?;

        // Assemble the forced transactions once: this validates them (a bad one errors, as in
        // op-geth's startBlock) and gives the starting cumulative gas.
        let cumulative_gas =
            chain.assemble_block(parent_hash, next_env.clone(), &forced_txs)?.gas_used;

        Ok(Self {
            id,
            parent_hash,
            next_env,
            forced_txs,
            pool_txs: Vec::new(),
            force_empty,
            gas_limit,
            cumulative_gas,
            sealed: None,
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

    /// Whether force-empty is set. Mirrors `L2EngineAPI.ForcedEmpty`.
    pub(crate) const fn forced_empty(&self) -> bool {
        self.force_empty
    }

    /// Set the force-empty flag. Mirrors `L2EngineAPI.SetForceEmpty`.
    pub(crate) const fn set_force_empty(&mut self, value: bool) {
        self.force_empty = value;
    }

    /// Gas remaining in the block, or `0` once it is sealed. Mirrors
    /// `L2EngineAPI.RemainingBlockGas` (the block gas pool, gone after `endBlock`).
    pub(crate) const fn remaining_block_gas(&self) -> u64 {
        if self.sealed.is_some() {
            return 0;
        }
        self.gas_limit.saturating_sub(self.cumulative_gas)
    }

    /// The forced transactions, then the pool transactions, in block order.
    fn block_txs(&self) -> impl Iterator<Item = &OpTransactionSigned> {
        self.forced_txs.iter().chain(&self.pool_txs)
    }

    /// Decode, gas-check, and execute a pool transaction, appending it on success.
    ///
    /// Mirrors `L2EngineAPI.IncludeTx` + `BlockProcessor.CheckTxWithinGasLimit`: a sealed block is
    /// no longer being built ([`Error::NotBuildingBlock`]); a force-empty block drops the
    /// transaction ([`IncludeTxOutcome::Skipped`]); a declared gas limit above the block gas limit
    /// or above the remaining gas is rejected with [`Error::ExceedsGasLimit`] /
    /// [`Error::UsesTooMuchGas`]; a transaction that fails execution errors without being appended.
    pub(crate) fn include_tx(
        &mut self,
        chain: &EphemeralChain,
        raw: &[u8],
    ) -> crate::Result<IncludeTxOutcome> {
        if self.sealed.is_some() {
            return Err(Error::NotBuildingBlock);
        }
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
        let tx_hash = tx.tx_hash();

        // Re-run the whole list with the candidate appended. On any execution error the candidate
        // is not retained, so the in-flight payload is unchanged.
        let txs = self.block_txs().chain([&tx]);
        let built = chain.assemble_block(self.parent_hash, self.next_env.clone(), txs)?;

        let gas_used = built.gas_used - self.cumulative_gas;
        self.pool_txs.push(tx);
        self.cumulative_gas = built.gas_used;
        Ok(IncludeTxOutcome::Included { tx_hash, gas_used })
    }

    /// Seal the block and return it as execution-payload data ready for `new_payload`.
    ///
    /// Mirrors `L2EngineAPI.getPayload` → `endBlock`: sealing closes the build. Fetching the
    /// payload again returns the block sealed first, as reth's payload store does, so a retried
    /// `getPayload` is safe.
    pub(crate) fn get_payload(&mut self, chain: &EphemeralChain) -> crate::Result<OpExecutionData> {
        if let Some(sealed) = &self.sealed {
            return Ok(sealed.clone());
        }
        let built =
            chain.assemble_block(self.parent_hash, self.next_env.clone(), self.block_txs())?;
        let block = built.block.clone_sealed_block().into_block();
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
        Ok(self.sealed.insert(OpExecutionData::new(payload, sidecar)).clone())
    }
}

/// Check that no deposit among the payload attributes' transactions follows a non-deposit: the
/// deposits lead the block. A violation is an [`Error::InvalidPayloadAttributes`].
fn check_deposits_lead(txs: &[OpTransactionSigned]) -> crate::Result<()> {
    let leading = txs.iter().take_while(|tx| tx.is_deposit()).count();
    if let Some(offset) = txs[leading..].iter().position(|tx| tx.is_deposit()) {
        return Err(Error::InvalidPayloadAttributes(format!(
            "deposit transaction {} follows a non-deposit transaction",
            leading + offset
        )));
    }
    Ok(())
}

/// Validate `attributes`, carried by an `engine_forkchoiceUpdated` call of message version
/// `version`, for a build on `parent`: attributes that op-reth's
/// [`ensure_well_formed_payload_attributes`] rejects (a field `version` or the forks active at
/// their timestamp require is missing, or one they do not support is present) are an
/// [`Error::MalformedPayloadAttributes`], and ones whose timestamp is not after the parent's an
/// [`Error::InvalidPayloadAttributes`].
fn validate_attributes(
    chain_spec: &OpChainSpec,
    version: EngineApiMessageVersion,
    parent: &SealedHeader,
    attributes: &OpPayloadAttrs,
) -> crate::Result<()> {
    ensure_well_formed_payload_attributes(chain_spec, version, attributes)
        .map_err(Error::MalformedPayloadAttributes)?;
    if attributes.payload_attributes.timestamp <= parent.timestamp {
        return Err(Error::InvalidPayloadAttributes(
            InvalidPayloadAttributesError::InvalidTimestamp.to_string(),
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::IncludeTxOutcome;
    use crate::{
        Error, TestEngine,
        testsupport::{
            GAS_LIMIT, deposit_tx, depositor, encode, fcu, isthmus_test_engine, payload_attrs,
            test_engine, user_sender, user_tx, user_tx_with_gas,
        },
    };
    use alloy_primitives::b64;
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use reth_payload_primitives::{
        EngineApiMessageVersion, EngineObjectValidationError, VersionSpecificValidationError,
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
        forced_txs: Vec<alloy_primitives::Bytes>,
        user_txs: &[reth_optimism_primitives::OpTransactionSigned],
    ) -> B256 {
        let updated = engine
            .forkchoice_updated_auto(fcu(parent), Some(payload_attrs(timestamp, forced_txs, false)))
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

        let updated = engine.forkchoice_updated_auto(fcu(head), None).expect("fcu advance");
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

    /// Open a build on genesis with `attrs` and return the error rejecting them.
    fn attributes_rejection(engine: &mut TestEngine, attrs: OpPayloadAttributes) -> Error {
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        engine.forkchoice_updated_auto(fcu(genesis), Some(attrs)).expect_err("attributes accepted")
    }

    /// Assert `attrs` cannot open a block on genesis.
    fn assert_attributes_rejected(engine: &mut TestEngine, attrs: OpPayloadAttributes) {
        let err = attributes_rejection(engine, attrs);
        assert!(matches!(err, Error::InvalidPayloadAttributes(_)), "{err:?}");
    }

    /// Assert `attrs` fail the engine API's checks with an invalid-params error, the class of a
    /// missing or out-of-range OP field.
    fn assert_invalid_params(engine: &mut TestEngine, attrs: OpPayloadAttributes) {
        let err = attributes_rejection(engine, attrs);
        assert!(
            matches!(
                err,
                Error::MalformedPayloadAttributes(EngineObjectValidationError::InvalidParams(_))
            ),
            "{err:?}"
        );
    }

    /// Assert `attrs` fail the engine API's version-specific field checks.
    fn assert_version_field_rejected(engine: &mut TestEngine, attrs: OpPayloadAttributes) {
        let err = attributes_rejection(engine, attrs);
        assert!(
            matches!(
                err,
                Error::MalformedPayloadAttributes(EngineObjectValidationError::PayloadAttributes(
                    _
                ))
            ),
            "{err:?}"
        );
    }

    #[test]
    fn rejects_attributes_not_after_parent() {
        let mut engine = test_engine(user_sender());
        // Genesis has timestamp 0.
        assert_attributes_rejected(&mut engine, payload_attrs(0, vec![], false));
    }

    #[test]
    fn rejects_attributes_missing_fork_fields() {
        let mut engine = test_engine(user_sender());

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.payload_attributes.withdrawals = None;
        assert_version_field_rejected(&mut engine, attrs);

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.payload_attributes.parent_beacon_block_root = None;
        assert_version_field_rejected(&mut engine, attrs);

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.gas_limit = None;
        assert_invalid_params(&mut engine, attrs);

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.eip_1559_params = None;
        assert_invalid_params(&mut engine, attrs);

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.min_base_fee = None;
        assert_invalid_params(&mut engine, attrs);
    }

    #[test]
    fn rejects_attributes_through_a_method_version_the_fork_does_not_use() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let mut through_v2 = |attrs| {
            engine
                .forkchoice_updated(EngineApiMessageVersion::V2, fcu(genesis), Some(attrs))
                .expect_err("Ecotone attributes accepted through V2")
        };

        // V2 carries no parent beacon block root...
        let err = through_v2(payload_attrs(2, vec![], false));
        assert!(
            matches!(
                err,
                Error::MalformedPayloadAttributes(EngineObjectValidationError::PayloadAttributes(
                    VersionSpecificValidationError::ParentBeaconBlockRootNotSupportedBeforeV3
                ))
            ),
            "{err:?}"
        );

        // ...and serves no timestamp from Ecotone (Cancun) on.
        let mut attrs = payload_attrs(2, vec![], false);
        attrs.payload_attributes.parent_beacon_block_root = None;
        let err = through_v2(attrs);
        assert!(
            matches!(
                err,
                Error::MalformedPayloadAttributes(EngineObjectValidationError::UnsupportedFork)
            ),
            "{err:?}"
        );
    }

    #[test]
    fn rejects_invalid_holocene_eip1559_params() {
        let mut engine = test_engine(user_sender());
        // Denominator (first four bytes) zero with a non-zero elasticity, and vice versa.
        for params in [b64!("0000000000000008"), b64!("0000000800000000")] {
            let mut attrs = payload_attrs(2, vec![], false);
            attrs.eip_1559_params = Some(params);
            assert_invalid_params(&mut engine, attrs);
        }
    }

    #[test]
    fn rejects_min_base_fee_before_jovian() {
        let mut engine = isthmus_test_engine(user_sender());
        // payload_attrs sets a min base fee, which only Jovian attributes may carry.
        assert_invalid_params(&mut engine, payload_attrs(2, vec![], false));

        let mut attrs = payload_attrs(2, vec![], false);
        attrs.min_base_fee = None;
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let updated =
            engine.forkchoice_updated_auto(fcu(genesis), Some(attrs)).expect("fcu with attrs");
        assert!(updated.payload_id.is_some(), "pre-Jovian attributes without min base fee open");
    }

    #[test]
    fn builds_derived_attributes_with_batch_transactions() {
        // Attributes derived from a batch carry the deposits followed by the batch's user
        // transactions, with no_tx_pool set; all of them are forced into the block in order.
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let forced =
            vec![encode(&deposit_tx(depositor())), encode(&user_tx(0)), encode(&user_tx(1))];
        let updated = engine
            .forkchoice_updated_auto(fcu(genesis), Some(payload_attrs(2, forced, true)))
            .unwrap();
        let id = updated.payload_id.expect("payload id");
        assert_eq!(engine.remaining_block_gas(None), GAS_LIMIT - 3 * 21_000);

        let data = engine.get_payload(id).expect("get payload");
        let block_hash = data.payload.block_hash();
        assert!(engine.new_payload(data).unwrap().is_valid());

        let receipts = engine.receipts_by_block_hash(block_hash).unwrap().expect("receipts");
        assert_eq!(receipts.len(), 3, "deposit + two batch transactions");
        assert!(matches!(receipts[0], OpReceipt::Deposit(_)), "deposit first");
        assert!(receipts[1..].iter().all(|r| !matches!(r, OpReceipt::Deposit(_)) && r.status()));
    }

    #[test]
    fn rejects_a_deposit_after_a_forced_user_transaction() {
        let mut engine = test_engine(user_sender());
        let late_deposit = deposit_tx(alloy_primitives::Address::with_last_byte(0xdf));
        let forced =
            vec![encode(&deposit_tx(depositor())), encode(&user_tx(0)), encode(&late_deposit)];
        assert_attributes_rejected(&mut engine, payload_attrs(2, forced, true));
    }

    #[test]
    fn include_tx_reports_gas_and_updates_remaining() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // Open an empty block (no forced transactions) so remaining gas starts at the full gas
        // limit.
        let updated = engine
            .forkchoice_updated_auto(fcu(genesis), Some(payload_attrs(2, vec![], false)))
            .unwrap();
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
        let updated = engine.forkchoice_updated_auto(fcu(genesis), Some(attrs)).unwrap();
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
        let updated = engine
            .forkchoice_updated_auto(fcu(genesis), Some(payload_attrs(2, vec![], true)))
            .unwrap();
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
    fn sealed_build_is_closed() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let updated = engine
            .forkchoice_updated_auto(fcu(genesis), Some(payload_attrs(2, vec![], false)))
            .unwrap();
        let id = updated.payload_id.expect("payload id");
        engine.include_tx(None, &encode(&user_tx(0))).unwrap();

        let sealed = engine.get_payload(id).expect("get payload");

        // Sealing ends the build, as op-geth's endBlock does: nothing more can be included and no
        // block gas is left to report.
        let err = engine.include_tx(None, &encode(&user_tx(1))).unwrap_err();
        assert!(matches!(err, Error::NotBuildingBlock), "{err:?}");
        assert_eq!(engine.remaining_block_gas(None), 0);

        // Fetching the payload again returns the block already sealed, so a retried getPayload
        // can't produce a different block.
        let again = engine.get_payload(id).expect("get payload again");
        assert_eq!(again.payload.block_hash(), sealed.payload.block_hash());
    }

    #[test]
    fn committed_payload_id_is_evicted() {
        // Mirrors TestL2SequencerAPI: once a payload is committed and the head advances past its
        // parent, re-sealing it (get_payload) must report UnknownPayloadId — op-node maps that code
        // to BuildErrCodeUnknownPayload.
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        let updated = engine
            .forkchoice_updated_auto(fcu(genesis), Some(payload_attrs(2, vec![], false)))
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
    fn forkchoice_away_from_parent_evicts_build() {
        let mut engine = test_engine(user_sender());
        let genesis = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        let block1 = build_block(&mut engine, genesis, 2, vec![], &[user_tx(0)]);

        let updated = engine
            .forkchoice_updated_auto(fcu(block1), Some(payload_attrs(4, vec![], false)))
            .expect("fcu with attrs");
        let id = updated.payload_id.expect("payload id returned");

        // Resetting the head below the build's parent strands the build.
        assert!(engine.forkchoice_updated_auto(fcu(genesis), None).unwrap().is_valid());
        let err = engine.get_payload(id).unwrap_err();
        assert!(matches!(err, Error::UnknownPayloadId(evicted) if evicted == id), "{err:?}");
        let err = engine.include_tx(None, &encode(&user_tx(1))).unwrap_err();
        assert!(matches!(err, Error::NotBuildingBlock), "{err:?}");
    }

    #[test]
    fn user_sender_is_funded() {
        // Guards the test setup: the canonical test signature recovers to the funded account.
        let sender = user_tx(0).recover_signer().unwrap();
        assert_eq!(sender, user_sender());
    }
}
