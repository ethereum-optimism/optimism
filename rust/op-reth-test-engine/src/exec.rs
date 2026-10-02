//! The `engine_newPayload` import path.
//!
//! [`import_payload`] validates a complete payload against the OP consensus rules, executes it
//! against its parent state with OP semantics, verifies the execution results and post-state root,
//! and—on success—commits it as the new canonical head.

use std::sync::Arc;

use alloy_consensus::BlockHeader as _;
use alloy_primitives::{B256, keccak256};
use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum};
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

/// Import a complete execution payload as the new canonical head (`engine_newPayload`).
///
/// Runs the checks op-reth's engine tree runs on a new payload: the payload layout and block hash,
/// the header on its own and against its parent, the body pre-execution, then — after executing
/// the block — the receipts root, logs bloom, gas used and Jovian DA footprint, the Isthmus
/// withdrawals root, and the state root. A violated rule yields `INVALID` pointing at the parent;
/// a provider or internal execution failure is an error, not a verdict on the block. A block that
/// is already known is `VALID` without being executed again.
pub(crate) fn import_payload(
    chain: &EphemeralChain,
    payload: OpExecutionData,
) -> crate::Result<PayloadStatus> {
    let chain_spec = chain.chain_spec();
    let validator = OpExecutionPayloadValidator::new(chain_spec.clone());

    // Structural validation + payload -> block. No execution happens here.
    let sealed = match validator.ensure_well_formed_payload::<OpTransactionSigned>(payload) {
        Ok(block) => block,
        Err(err) => return Ok(invalid(None, err.to_string())),
    };
    let block_hash = sealed.hash();
    let parent_hash = sealed.header().parent_hash;
    let expected_state_root = sealed.header().state_root;

    // A block already on the chain was validated when it was first imported; re-executing and
    // re-committing it would move the head back onto it.
    if chain.sealed_header(block_hash)?.is_some() {
        return Ok(PayloadStatus::new(PayloadStatusEnum::Valid, Some(block_hash)));
    }

    // The parent must be known; without its state we cannot execute, so we report SYNCING rather
    // than guessing.
    let (Some(parent), Some(state)) =
        (chain.sealed_header(parent_hash)?, chain.state_at(parent_hash)?)
    else {
        return Ok(PayloadStatus::from_status(PayloadStatusEnum::Syncing));
    };

    let consensus = OpBeaconConsensus::new(chain_spec.clone());
    if let Err(err) = validate_pre_execution(&consensus, &sealed, &parent) {
        return Ok(invalid(Some(parent_hash), err.to_string()));
    }

    let recovered = match sealed.try_recover() {
        Ok(recovered) => recovered,
        Err(_) => {
            return Ok(invalid(Some(parent_hash), "failed to recover transaction senders".into()));
        }
    };

    let evm_config: OpEvmConfig =
        OpEvmConfig::new(chain_spec.clone(), OpRethReceiptBuilder::default());
    let executor = BasicBlockExecutor::new(evm_config, StateProviderDatabase::new(&state));
    let output: BlockExecutionOutput<OpReceipt> = match executor.execute(&recovered) {
        Ok(output) => output,
        Err(BlockExecutionError::Validation(err)) => {
            return Ok(invalid(Some(parent_hash), err.to_string()));
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
        return Ok(invalid(Some(parent_hash), err.to_string()));
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
            Err(err) => return Ok(invalid(Some(parent_hash), err.to_string())),
        }
    }

    // Verify the post-state root matches the header before accepting the block.
    let (computed_root, trie_updates) = state.state_root_with_updates(hashed_state.clone())?;
    if computed_root != expected_state_root {
        return Ok(invalid(
            Some(parent_hash),
            format!(
                "state root mismatch: computed {computed_root}, expected {expected_state_root}"
            ),
        ));
    }

    let executed = ExecutedBlock::new(
        Arc::new(recovered),
        Arc::new(output),
        ComputedTrieData::new(
            Arc::new(hashed_state.into_sorted()),
            Arc::new(trie_updates.into_sorted()),
        ),
    );
    chain.commit_block(executed);

    Ok(PayloadStatus::new(PayloadStatusEnum::Valid, Some(block_hash)))
}

/// The consensus checks that need no execution: the header on its own (including its
/// Holocene/Jovian `extraData` encoding) and against `parent`, and the body against the header.
fn validate_pre_execution(
    consensus: &OpBeaconConsensus<OpChainSpec>,
    block: &SealedBlock<OpBlock>,
    parent: &SealedHeader,
) -> Result<(), ConsensusError> {
    consensus.validate_header(block.sealed_header())?;
    consensus.validate_header_against_parent(block.sealed_header(), parent)?;
    Consensus::<OpBlock>::validate_block_pre_execution(consensus, block)
}

const fn invalid(latest_valid_hash: Option<B256>, error: String) -> PayloadStatus {
    PayloadStatus::new(PayloadStatusEnum::Invalid { validation_error: error }, latest_valid_hash)
}

#[cfg(test)]
mod tests {
    use super::*;

    use alloy_consensus::{SignableTransaction, TxReceipt, proofs::calculate_transaction_root};
    use alloy_eips::eip1559::BaseFeeParams;
    use alloy_primitives::{B64, Signature, U256};
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
        let status = import_payload(&chain, payload_of(&block)).expect("import payload");

        assert!(status.is_valid(), "expected VALID, got {status:?}");
        assert_eq!(status.latest_valid_hash, Some(block_hash));

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

        let status = import_payload(&chain, payload_of(&block)).expect("import payload");

        assert!(status.is_invalid(), "expected INVALID, got {status:?}");
        // INVALID points at the last valid block (the parent), not the rejected block.
        assert_eq!(status.latest_valid_hash, Some(parent_hash));
        // The corrupt block was not committed.
        assert!(chain.block_by_number(1).expect("query").is_none());
    }

    #[test]
    fn reimporting_a_known_block_is_valid_and_moves_nothing() {
        let chain = test_chain(user_sender());
        let b1 = block_on_genesis(&chain, &[deposit_tx(depositor())]);
        let b1_hash = b1.header.hash_slow();
        assert!(import_payload(&chain, payload_of(&b1)).unwrap().is_valid());

        let env = next_env(4, b1.header.extra_data.clone());
        let b2 = chain.assemble_block(b1_hash, env, &[]).expect("assemble b2");
        let b2 = b2.block.clone_sealed_block().into_block();
        let b2_hash = b2.header.hash_slow();
        assert!(import_payload(&chain, payload_of(&b2)).unwrap().is_valid());
        assert_eq!(chain.latest_header().hash(), b2_hash);

        // Re-importing b1 reports it VALID without re-committing it over its descendant.
        let status = import_payload(&chain, payload_of(&b1)).expect("re-import b1");
        assert!(status.is_valid(), "expected VALID, got {status:?}");
        assert_eq!(status.latest_valid_hash, Some(b1_hash));
        assert_eq!(chain.latest_header().hash(), b2_hash, "re-import moved the head");
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

        let status = import_payload(&chain, payload_of(&block)).expect("import payload");
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
        assert_tampered_block_rejected("Extra data is not the correct length", |header| {
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

        let status = import_payload(&chain, payload_of(&block)).expect("import payload");
        assert!(status.is_invalid(), "expected INVALID, got {status:?}");
        assert_eq!(status.latest_valid_hash, Some(chain.genesis_hash()));
        assert!(chain.block_by_number(1).expect("query").is_none(), "rejected block committed");
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
        let updated = engine.forkchoice_updated_auto(
            ForkchoiceState {
                head_block_hash: B256::repeat_byte(0xaa),
                safe_block_hash: B256::ZERO,
                finalized_block_hash: B256::ZERO,
            },
            None,
        );
        assert!(updated.unwrap().is_syncing());

        // A known head (genesis) is VALID.
        let updated =
            engine.forkchoice_updated_auto(crate::testsupport::fcu(genesis_hash), None).unwrap();
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
            .forkchoice_updated_auto(
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
