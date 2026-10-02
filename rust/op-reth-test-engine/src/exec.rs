//! The `engine_newPayload` import path.
//!
//! [`import_payload`] validates a complete payload against the OP consensus rules and executes it
//! against its parent state with OP semantics, verifying the execution results and post-state root.
//! It does *not* touch the canonical chain: the caller
//! ([`new_payload`](crate::TestEngine::new_payload)) records the returned block, and only a later
//! forkchoice update canonicalizes it. A parent that is not canonical yet is executed on through
//! the side blocks the caller passes in.

use std::{collections::HashMap, sync::Arc};

use alloy_consensus::BlockHeader as _;
use alloy_primitives::{B256, keccak256};
use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum};
use op_alloy_consensus::{decode_holocene_extra_data, decode_jovian_extra_data};
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_chain_state::ExecutedBlock;
use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator};
use reth_evm::execute::{BasicBlockExecutor, BlockExecutionError, Executor};
use reth_execution_types::BlockExecutionOutput;
use reth_optimism_chainspec::{OpChainSpec, OpHardforks};
use reth_optimism_consensus::{OpBeaconConsensus, OpConsensusError, isthmus};
use reth_optimism_evm::{OpEvmConfig, OpRethReceiptBuilder};
use reth_optimism_payload_builder::OpExecutionPayloadValidator;
use reth_optimism_primitives::{
    L2_TO_L1_MESSAGE_PASSER_ADDRESS, OpBlock, OpPrimitives, OpReceipt, OpTransactionSigned,
};
use reth_primitives_traits::{SealedBlock, SealedHeader};
use reth_revm::database::StateProviderDatabase;
use reth_storage_api::{HashedPostStateProvider, StateRootProvider};
use reth_trie::ComputedTrieData;

use crate::{Error, chain::EphemeralChain};

/// The result of validating and executing a payload, before it is committed to any chain.
#[derive(Debug)]
pub(crate) enum ImportOutcome {
    /// The payload validated and executed cleanly. The executed block is ready for the caller to
    /// commit as a linear head extension or to buffer as an alternate-fork block.
    Valid(ExecutedBlock<OpPrimitives>),
    /// The block is already known, on the chain or among the side blocks; it was validated when
    /// first imported and is not executed again.
    Known(B256),
    /// The payload violates a consensus rule or does not execute to its declared state root.
    /// Carries the `INVALID` status (with `latestValidHash`) to return verbatim.
    Invalid(PayloadStatus),
    /// The parent block is unknown, so the payload cannot be executed. The caller reports
    /// `SYNCING`.
    Syncing,
}

/// Validate and execute a complete execution payload (`engine_newPayload`) without committing it.
///
/// Runs the checks op-reth's engine tree runs on a new payload: the payload layout and block hash,
/// the header on its own and against its parent, the body pre-execution, then — after executing
/// the block — the receipts root, logs bloom, gas used and Jovian DA footprint, the Isthmus
/// withdrawals root, and the state root. On success the executed block is returned for the caller
/// to canonicalize; a violated rule yields [`ImportOutcome::Invalid`] pointing at the parent, and
/// an unknown parent [`ImportOutcome::Syncing`]. A provider or internal execution failure is an
/// error, not a verdict on the block.
pub(crate) fn import_payload(
    chain: &EphemeralChain,
    payload: OpExecutionData,
    side_blocks: &HashMap<B256, ExecutedBlock<OpPrimitives>>,
) -> crate::Result<ImportOutcome> {
    let chain_spec = chain.chain_spec();
    let validator = OpExecutionPayloadValidator::new(chain_spec.clone());

    // Structural validation + payload -> block. No execution happens here.
    let sealed = match validator.ensure_well_formed_payload::<OpTransactionSigned>(payload) {
        Ok(block) => block,
        Err(err) => return Ok(ImportOutcome::Invalid(invalid(None, err.to_string()))),
    };
    let block_hash = sealed.hash();
    let parent_hash = sealed.header().parent_hash;
    let expected_state_root = sealed.header().state_root;

    if side_blocks.contains_key(&block_hash) || chain.sealed_header(block_hash)?.is_some() {
        return Ok(ImportOutcome::Known(block_hash));
    }

    // The parent must be known; without its state we cannot execute, so we report SYNCING rather
    // than guessing.
    let Some((parent, state)) = chain.block_state(parent_hash, side_blocks)? else {
        return Ok(ImportOutcome::Syncing);
    };

    let consensus = OpBeaconConsensus::new(chain_spec.clone());
    if let Err(err) = validate_pre_execution(&consensus, &chain_spec, &sealed, &parent) {
        return Ok(ImportOutcome::Invalid(invalid(Some(parent_hash), err.to_string())));
    }

    let recovered = match sealed.try_recover() {
        Ok(recovered) => recovered,
        Err(_) => {
            return Ok(ImportOutcome::Invalid(invalid(
                Some(parent_hash),
                "failed to recover transaction senders".to_string(),
            )));
        }
    };

    let evm_config: OpEvmConfig =
        OpEvmConfig::new(chain_spec.clone(), OpRethReceiptBuilder::default());
    let executor = BasicBlockExecutor::new(evm_config, StateProviderDatabase::new(&state));
    let output: BlockExecutionOutput<OpReceipt> = match executor.execute(&recovered) {
        Ok(output) => output,
        Err(BlockExecutionError::Validation(err)) => {
            return Ok(ImportOutcome::Invalid(invalid(Some(parent_hash), err.to_string())));
        }
        Err(err) => return Err(Error::Execution(err.to_string())),
    };

    if let Err(err) = FullConsensus::<OpPrimitives>::validate_block_post_execution(
        &consensus,
        &recovered,
        &output.result,
        None,
        None,
    ) {
        return Ok(ImportOutcome::Invalid(invalid(Some(parent_hash), err.to_string())));
    }

    let hashed_state = state.hashed_post_state(&output.state)?;

    // Isthmus repurposes the header's withdrawals root as the L2ToL1MessagePasser storage root.
    if chain_spec.is_isthmus_active_at_timestamp(recovered.timestamp()) {
        let storage_updates = hashed_state
            .storages
            .get(&keccak256(L2_TO_L1_MESSAGE_PASSER_ADDRESS))
            .cloned()
            .unwrap_or_default();
        match isthmus::verify_withdrawals_root_prehashed(
            storage_updates,
            &state,
            recovered.header(),
        ) {
            Ok(()) => {}
            Err(OpConsensusError::L2WithdrawalsRootCalculationFail(err)) => return Err(err.into()),
            Err(err) => {
                return Ok(ImportOutcome::Invalid(invalid(Some(parent_hash), err.to_string())));
            }
        }
    }

    // Verify the post-state root matches the header before accepting the block.
    let (computed_root, trie_updates) = state.state_root_with_updates(hashed_state.clone())?;
    if computed_root != expected_state_root {
        return Ok(ImportOutcome::Invalid(invalid(
            Some(parent_hash),
            format!(
                "state root mismatch: computed {computed_root}, expected {expected_state_root}"
            ),
        )));
    }

    let executed = ExecutedBlock::new(
        Arc::new(recovered),
        Arc::new(output),
        ComputedTrieData::new(
            Arc::new(hashed_state.into_sorted()),
            Arc::new(trie_updates.into_sorted()),
        ),
    );
    Ok(ImportOutcome::Valid(executed))
}

/// The consensus checks that need no execution: the header on its own and against `parent`, the
/// body against the header, and the Holocene/Jovian `extraData` encoding.
fn validate_pre_execution(
    consensus: &OpBeaconConsensus<OpChainSpec>,
    chain_spec: &OpChainSpec,
    block: &SealedBlock<OpBlock>,
    parent: &SealedHeader,
) -> Result<(), ConsensusError> {
    consensus.validate_header(block.sealed_header())?;
    consensus.validate_header_against_parent(block.sealed_header(), parent)?;
    Consensus::<OpBlock>::validate_block_pre_execution(consensus, block)?;
    validate_extra_data(chain_spec, block.header())
}

/// From Holocene the block's `extraData` carries its EIP-1559 parameters, and from Jovian also the
/// minimum base fee; a block whose `extraData` does not decode in the active format is invalid.
fn validate_extra_data(
    chain_spec: &OpChainSpec,
    header: &alloy_consensus::Header,
) -> Result<(), ConsensusError> {
    let decoded = if chain_spec.is_jovian_active_at_timestamp(header.timestamp) {
        decode_jovian_extra_data(&header.extra_data).map(|_| ())
    } else if chain_spec.is_holocene_active_at_timestamp(header.timestamp) {
        decode_holocene_extra_data(&header.extra_data).map(|_| ())
    } else {
        Ok(())
    };
    decoded.map_err(|err| ConsensusError::msg(format!("invalid extra data: {err}")))
}

const fn invalid(latest_valid_hash: Option<B256>, error: String) -> PayloadStatus {
    PayloadStatus::new(PayloadStatusEnum::Invalid { validation_error: error }, latest_valid_hash)
}

#[cfg(test)]
mod tests {
    use super::*;

    use alloy_consensus::{SignableTransaction, TxReceipt, proofs::calculate_transaction_root};
    use alloy_eips::eip1559::BaseFeeParams;
    use alloy_primitives::{B64, Bloom, Signature, U256};
    use alloy_rpc_types_engine::ForkchoiceState;
    use op_alloy_consensus::{OpTxEnvelope, encode_holocene_extra_data};
    use reth_optimism_primitives::{OpBlock, OpReceipt};
    use reth_storage_api::StateProvider;

    use crate::{
        Error, ForkchoicePointer,
        testsupport::{
            FUNDED_BALANCE, deposit_tx, depositor, next_env, payload_of, test_chain, test_engine,
            user_sender, user_tx,
        },
    };

    /// Assemble block 1 (timestamp 2) on genesis from `txs`, without committing it.
    fn block_on_genesis(chain: &EphemeralChain, txs: &[OpTransactionSigned]) -> OpBlock {
        let genesis = chain.latest_header();
        let env = next_env(2, genesis.extra_data.clone());
        let built = chain.assemble_block(genesis.hash(), env, txs).expect("assemble block");
        built.block.clone_sealed_block().into_block()
    }

    #[test]
    fn import_block_with_deposit() {
        let chain = test_chain(user_sender());

        // Produce a valid block (deposit first, then the user tx) and round-trip it as a payload.
        let block = block_on_genesis(&chain, &[deposit_tx(depositor()), user_tx(0)]);
        let block_hash = block.header.hash_slow();
        let outcome =
            import_payload(&chain, payload_of(&block), &HashMap::new()).expect("import payload");

        let ImportOutcome::Valid(executed) = outcome else {
            panic!("expected VALID, got {outcome:?}");
        };
        assert_eq!(executed.recovered_block().hash(), block_hash);
        // import_payload does not touch the chain; committing is the caller's job.
        chain.commit_block(executed);

        let receipts =
            chain.receipts_by_block_hash(block_hash).expect("query").expect("receipts present");
        assert_eq!(receipts.len(), 2, "deposit + user tx");

        // The deposit receipt carries OP-specific fields; post-Canyon it has receipt version 1.
        let OpReceipt::Deposit(deposit) = &receipts[0] else {
            panic!("first receipt should be a deposit, got {:?}", receipts[0]);
        };
        assert!(deposit.deposit_nonce.is_some(), "deposit nonce present");
        assert_eq!(deposit.deposit_receipt_version, Some(1), "post-Canyon receipt version");

        // The user tx is a normal (non-deposit) receipt and succeeded.
        assert!(!matches!(receipts[1], OpReceipt::Deposit(_)), "user tx is not a deposit");
        assert!(receipts[1].status(), "user tx succeeded");

        // The sender paid execution gas *plus* the L1 data fee. We derive the exact gas charge
        // (gas used × base fee, with zero priority fee) from the block and receipts; the balance
        // drops by strictly more than that, and the excess is the L1 fee. This asserts the L1 fee
        // is charged, not its exact value.
        let user_gas = receipts[1].cumulative_gas_used() - receipts[0].cumulative_gas_used();
        let base_fee = block.header.base_fee_per_gas.expect("post-1559 base fee");
        let gas_charge = U256::from(user_gas) * U256::from(base_fee);

        let state = chain.state_at(block_hash).expect("query").expect("state at head");
        let balance = state.account_balance(&user_sender()).expect("balance").unwrap_or_default();
        let charged = FUNDED_BALANCE - balance;
        assert!(
            charged > gas_charge,
            "charged {charged} should exceed pure gas {gas_charge} by the L1 fee",
        );
    }

    #[test]
    fn import_invalid_block() {
        let chain = test_chain(user_sender());

        // Corrupt the state root; `payload_of` re-derives a matching block hash, so the layout
        // check passes and only execution catches the mismatch.
        let mut block = block_on_genesis(&chain, &[deposit_tx(depositor()), user_tx(0)]);
        let parent_hash = block.header.parent_hash;
        block.header.state_root = B256::repeat_byte(0xff);

        let outcome =
            import_payload(&chain, payload_of(&block), &HashMap::new()).expect("import payload");

        let ImportOutcome::Invalid(status) = outcome else {
            panic!("expected INVALID, got {outcome:?}");
        };
        assert!(status.is_invalid(), "expected INVALID, got {status:?}");
        // INVALID points at the last valid block (the parent), not the rejected block.
        assert_eq!(status.latest_valid_hash, Some(parent_hash));
        // The corrupt block was not committed.
        assert!(chain.block_by_number(1).expect("query").is_none());
    }

    /// Import `block` and commit it, as `new_payload` does for a linear head extension.
    fn import_and_commit(chain: &EphemeralChain, block: &OpBlock) {
        let outcome =
            import_payload(chain, payload_of(block), &HashMap::new()).expect("import payload");
        let ImportOutcome::Valid(executed) = outcome else {
            panic!("expected VALID, got {outcome:?}");
        };
        chain.commit_block(executed);
    }

    #[test]
    fn reimporting_a_known_block_is_not_executed_again() {
        let chain = test_chain(user_sender());
        let b1 = block_on_genesis(&chain, &[deposit_tx(depositor())]);
        let b1_hash = b1.header.hash_slow();
        import_and_commit(&chain, &b1);

        let env = next_env(4, b1.header.extra_data.clone());
        let b2 = chain.assemble_block(b1_hash, env, &[]).expect("assemble b2");
        import_and_commit(&chain, &b2.block.clone_sealed_block().into_block());

        let outcome =
            import_payload(&chain, payload_of(&b1), &HashMap::new()).expect("re-import b1");
        assert!(matches!(outcome, ImportOutcome::Known(hash) if hash == b1_hash), "{outcome:?}");
    }

    #[test]
    fn reimporting_a_known_side_block_is_not_executed_again() {
        let chain = test_chain(user_sender());
        let b1 = block_on_genesis(&chain, &[deposit_tx(depositor())]);
        let b1_hash = b1.header.hash_slow();
        let ImportOutcome::Valid(executed) =
            import_payload(&chain, payload_of(&b1), &HashMap::new()).expect("import b1")
        else {
            panic!("b1 valid");
        };
        let side = HashMap::from([(b1_hash, executed)]);

        let outcome = import_payload(&chain, payload_of(&b1), &side).expect("re-import b1");
        assert!(matches!(outcome, ImportOutcome::Known(hash) if hash == b1_hash), "{outcome:?}");
    }

    /// Import a deposit-only block 1 whose header `tamper` corrupts (the block hash is recomputed),
    /// and assert it is rejected as INVALID for `reason`, pointing at genesis, and is not
    /// committed.
    ///
    /// A deposit pays no base fee and the test genesis has no system contracts, so none of the
    /// tampered fields feeds into the post-state: only consensus validation can catch them.
    fn assert_tampered_block_rejected(
        reason: &str,
        tamper: impl FnOnce(&mut alloy_consensus::Header),
    ) {
        let chain = test_chain(user_sender());
        let mut block = block_on_genesis(&chain, &[deposit_tx(depositor())]);
        tamper(&mut block.header);

        let outcome =
            import_payload(&chain, payload_of(&block), &HashMap::new()).expect("import payload");
        let ImportOutcome::Invalid(status) = outcome else {
            panic!("expected INVALID, got {outcome:?}");
        };
        let PayloadStatusEnum::Invalid { validation_error } = &status.status else {
            panic!("expected INVALID, got {status:?}");
        };
        assert!(validation_error.contains(reason), "unexpected rejection: {validation_error}");
        assert_eq!(status.latest_valid_hash, Some(chain.genesis_hash()));
        assert!(chain.block_by_number(1).expect("query").is_none(), "rejected block committed");
    }

    #[test]
    fn rejects_receipts_root_mismatch() {
        assert_tampered_block_rejected("receipt root mismatch", |header| {
            header.receipts_root = B256::repeat_byte(0x11)
        });
    }

    #[test]
    fn rejects_logs_bloom_mismatch() {
        assert_tampered_block_rejected("header bloom filter mismatch", |header| {
            header.logs_bloom = Bloom::repeat_byte(0x01);
        });
    }

    #[test]
    fn rejects_da_footprint_mismatch() {
        // The test chain runs Karst, so Jovian is active and the header's blob gas used carries
        // the block's DA footprint.
        assert_tampered_block_rejected("blob gas used mismatch", |header| {
            header.blob_gas_used = header.blob_gas_used.map(|used| used + 1);
        });
    }

    #[test]
    fn rejects_gas_used_mismatch() {
        assert_tampered_block_rejected("block gas used mismatch", |header| header.gas_used += 1);
    }

    #[test]
    fn rejects_wrong_base_fee() {
        assert_tampered_block_rejected("block base fee mismatch", |header| {
            header.base_fee_per_gas = header.base_fee_per_gas.map(|fee| fee + 1);
        });
    }

    #[test]
    fn rejects_malformed_extra_data() {
        // Karst implies Jovian, whose extra data carries a min base fee; the Holocene encoding
        // (version 0, no min base fee) is malformed here.
        assert_tampered_block_rejected("invalid extra data", |header| {
            header.extra_data =
                encode_holocene_extra_data(B64::ZERO, BaseFeeParams::optimism_canyon())
                    .expect("encode holocene extra data");
        });
    }

    #[test]
    fn rejects_withdrawals_root_mismatch() {
        // Isthmus repurposes the withdrawals root as the L2ToL1MessagePasser storage root.
        assert_tampered_block_rejected("L2 withdrawals root mismatch", |header| {
            header.withdrawals_root = Some(B256::repeat_byte(0x22));
        });
    }

    #[test]
    fn rejects_unrecoverable_sender_at_the_parent() {
        let chain = test_chain(user_sender());
        let mut block = block_on_genesis(&chain, &[deposit_tx(depositor()), user_tx(0)]);
        let OpTxEnvelope::Eip1559(signed) = &block.body.transactions[1] else {
            panic!("user tx is EIP-1559");
        };
        // A zero signature recovers no sender.
        let unsigned = signed.tx().clone();
        block.body.transactions[1] =
            unsigned.into_signed(Signature::new(U256::ZERO, U256::ZERO, false)).into();
        block.header.transactions_root = calculate_transaction_root(&block.body.transactions);

        let outcome =
            import_payload(&chain, payload_of(&block), &HashMap::new()).expect("import payload");
        let ImportOutcome::Invalid(status) = outcome else {
            panic!("expected INVALID, got {outcome:?}");
        };
        assert_eq!(status.latest_valid_hash, Some(chain.genesis_hash()));
    }

    #[test]
    fn rejects_non_consecutive_number() {
        assert_tampered_block_rejected("does not match parent block number", |header| {
            header.number = 2;
        });
    }

    #[test]
    fn rejects_timestamp_not_after_parent() {
        assert_tampered_block_rejected("in the past compared to the parent", |header| {
            header.timestamp = 0;
        });
    }

    #[test]
    fn fcu_unknown_head_is_syncing() {
        let mut engine = test_engine(user_sender());
        let genesis_hash = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // An unknown head must report SYNCING — never a silent VALID.
        let updated = engine.forkchoice_updated_as_op_node(
            ForkchoiceState {
                head_block_hash: B256::repeat_byte(0xaa),
                safe_block_hash: B256::ZERO,
                finalized_block_hash: B256::ZERO,
            },
            None,
        );
        assert!(updated.unwrap().is_syncing());

        // A known head (genesis) is VALID.
        let updated = engine
            .forkchoice_updated_as_op_node(crate::testsupport::fcu(genesis_hash), None)
            .unwrap();
        assert!(updated.is_valid());
        assert_eq!(updated.payload_status.latest_valid_hash, Some(genesis_hash));
    }

    #[test]
    fn fcu_unknown_safe_is_error() {
        let mut engine = test_engine(user_sender());
        let genesis_hash = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        // A known head with an unknown (non-zero) safe block is a forkchoice error, not silently
        // ignored.
        let err = engine
            .forkchoice_updated_as_op_node(
                ForkchoiceState {
                    head_block_hash: genesis_hash,
                    safe_block_hash: B256::repeat_byte(0xbb),
                    finalized_block_hash: B256::ZERO,
                },
                None,
            )
            .unwrap_err();
        assert!(
            matches!(err, Error::UnknownForkchoiceBlock { which: ForkchoicePointer::Safe, .. }),
            "{err:?}"
        );
    }
}
