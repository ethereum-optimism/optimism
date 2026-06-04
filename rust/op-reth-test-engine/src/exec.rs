//! The `engine_newPayload` import path.
//!
//! [`import_payload`] validates a complete payload, executes it against its parent state with OP
//! semantics, verifies the post-state root, and—on success—commits it as the new canonical head.

use std::sync::Arc;

use alloy_primitives::B256;
use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum};
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_chain_state::ExecutedBlock;
use reth_evm::execute::{BasicBlockExecutor, Executor};
use reth_execution_types::BlockExecutionOutput;
use reth_optimism_evm::{OpEvmConfig, OpRethReceiptBuilder};
use reth_optimism_payload_builder::OpExecutionPayloadValidator;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_revm::database::StateProviderDatabase;
use reth_storage_api::{HashedPostStateProvider, StateRootProvider};
use reth_trie::ComputedTrieData;

use crate::chain::EphemeralChain;

/// Import a complete execution payload as the new canonical head (`engine_newPayload`).
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

    let recovered = match sealed.try_recover() {
        Ok(recovered) => recovered,
        Err(_) => return Ok(invalid(None, "failed to recover transaction senders".to_string())),
    };

    // The parent must be known; without its state we cannot execute, so we report SYNCING rather
    // than guessing.
    let Some(state) = chain.state_at(parent_hash)? else {
        return Ok(PayloadStatus::from_status(PayloadStatusEnum::Syncing));
    };

    let evm_config: OpEvmConfig = OpEvmConfig::new(chain_spec, OpRethReceiptBuilder::default());
    let executor = BasicBlockExecutor::new(evm_config, StateProviderDatabase::new(&state));
    let output: BlockExecutionOutput<OpReceipt> = match executor.execute(&recovered) {
        Ok(output) => output,
        Err(err) => return Ok(invalid(Some(parent_hash), err.to_string())),
    };

    // Verify the post-state root matches the header before accepting the block.
    let hashed_state = state.hashed_post_state(&output.state);
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

const fn invalid(latest_valid_hash: Option<B256>, error: String) -> PayloadStatus {
    PayloadStatus::new(PayloadStatusEnum::Invalid { validation_error: error }, latest_valid_hash)
}

#[cfg(test)]
mod tests {
    use super::*;

    use std::collections::BTreeMap;

    use alloy_consensus::{
        SignableTransaction, TxEip1559, TxReceipt, transaction::SignerRecoverable,
    };
    use alloy_eips::eip1559::BaseFeeParams;
    use alloy_genesis::{ChainConfig, Genesis, GenesisAccount};
    use alloy_primitives::{Address, B64, Bytes, Signature, TxKind, U256, b256};
    use op_alloy_consensus::{TxDeposit, encode_jovian_extra_data};
    use op_alloy_rpc_types_engine::OpExecutionPayload;
    use op_revm::constants::{
        ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT, L1_BASE_FEE_SLOT,
        L1_BLOCK_CONTRACT,
    };
    use reth_chainspec::{Chain, EthereumHardfork, ForkCondition};
    use reth_evm::{
        ConfigureEvm,
        execute::{BlockBuilder, BlockBuilderOutcome},
    };
    use reth_optimism_chainspec::OpChainSpecBuilder;
    use reth_optimism_evm::OpNextBlockEnvAttributes;
    use reth_optimism_primitives::{OpBlock, OpReceipt};
    use reth_primitives_traits::RecoveredBlock;
    use reth_revm::db::State;
    use reth_storage_api::StateProvider;

    use crate::Error;

    /// A synthetic OP chain id — not OP Mainnet (10), whose genesis hash op-reth pins to the
    /// registry value rather than recomputing from the (here synthetic) genesis state.
    const CHAIN_ID: u64 = 901;
    const GAS_LIMIT: u64 = 30_000_000;
    /// A user-tx gas price the funded sender can always cover.
    const MAX_FEE: u128 = 10_000_000_000; // 10 gwei
    /// Sender balance — 1 ETH is realistic and avoids the overflow risk of `U256::MAX`.
    const FUNDED_BALANCE: U256 = U256::from_limbs([1_000_000_000_000_000_000, 0, 0, 0]);

    fn slot(slot: U256) -> B256 {
        B256::from(slot.to_be_bytes::<32>())
    }

    fn value(v: u64) -> B256 {
        B256::from(U256::from(v).to_be_bytes::<32>())
    }

    /// An EIP-1559 user transaction signed with the canonical test signature. Its sender is
    /// recovered from the signature, so callers fund that address in genesis.
    fn user_tx(nonce: u64) -> OpTransactionSigned {
        TxEip1559 {
            chain_id: CHAIN_ID,
            nonce,
            gas_limit: 21_000,
            max_fee_per_gas: MAX_FEE,
            max_priority_fee_per_gas: 0,
            to: TxKind::Call(Address::ZERO),
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into()
    }

    /// A distinct depositor address so the deposit's nonce bump doesn't collide with the user tx.
    fn depositor() -> Address {
        Address::with_last_byte(0xde)
    }

    /// A bare deposit (type 0x7E) from `from`.
    fn deposit_tx(from: Address) -> OpTransactionSigned {
        TxDeposit { from, to: TxKind::Call(from), gas_limit: 21_000, ..Default::default() }.into()
    }

    /// The `L1Block` predeploy storage that drives the L1-cost function, keyed by the slots
    /// op-revm reads (operator-fee and DA-footprint scalars default to zero).
    fn l1_block_storage() -> BTreeMap<B256, B256> {
        BTreeMap::from([
            (slot(L1_BASE_FEE_SLOT), value(1_000_000_000)),
            (slot(ECOTONE_L1_BLOB_BASE_FEE_SLOT), value(1)),
            // Base-fee / blob-base-fee scalars packed at their byte offsets (value taken from
            // op-reth's own evm execute tests, so the L1 cost is non-zero).
            (
                slot(ECOTONE_L1_FEE_SCALARS_SLOT),
                b256!("0x0000000000000000000000000000000000001db0000d27300000000000000005"),
            ),
        ])
    }

    /// EIP-1559 parameters carried in the block's `extra_data`, encoded for the Jovian/Karst
    /// header format. Zero params fall back to the canyon defaults.
    fn eip1559_extra_data() -> Bytes {
        encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism_canyon(), 1)
            .expect("encode extra data")
    }

    /// An ephemeral OP Mainnet chain with Karst active at genesis, the `L1Block` predeploy seeded,
    /// and `funded` given a spendable balance.
    fn test_chain(funded: Address) -> EphemeralChain {
        let alloc = BTreeMap::from([
            (
                L1_BLOCK_CONTRACT,
                GenesisAccount {
                    nonce: Some(1),
                    storage: Some(l1_block_storage()),
                    ..Default::default()
                },
            ),
            (funded, GenesisAccount { balance: FUNDED_BALANCE, ..Default::default() }),
        ]);

        let genesis = Genesis {
            config: ChainConfig { chain_id: CHAIN_ID, ..Default::default() },
            gas_limit: GAS_LIMIT,
            base_fee_per_gas: Some(1_000_000_000),
            excess_blob_gas: Some(0),
            blob_gas_used: Some(0),
            extra_data: eip1559_extra_data(),
            alloc,
            ..Default::default()
        };

        // `karst_activated()` activates the OP hardforks but not the Ethereum forks they ride;
        // Isthmus rides Prague, which the builder omits, so add it for a consistent spec.
        // See ethereum-optimism/optimism#21239.
        let spec = OpChainSpecBuilder::optimism_mainnet()
            .chain(Chain::from_id(CHAIN_ID))
            .karst_activated()
            .with_fork(EthereumHardfork::Prague, ForkCondition::Timestamp(0))
            .genesis(genesis)
            .build();
        EphemeralChain::from_chain_spec(Arc::new(spec)).expect("build ephemeral chain")
    }

    /// Next-block attributes on top of genesis. Timestamp >= 2 dodges the create2-deployer
    /// activation guard; the parent's `extra_data` carries the EIP-1559 params forward.
    fn next_attrs(chain: &EphemeralChain) -> OpNextBlockEnvAttributes {
        OpNextBlockEnvAttributes {
            timestamp: 2,
            suggested_fee_recipient: Address::ZERO,
            prev_randao: B256::ZERO,
            gas_limit: GAS_LIMIT,
            parent_beacon_block_root: Some(B256::ZERO),
            extra_data: chain.tip_header().extra_data.clone(),
        }
    }

    /// Assemble a block from the chain tip plus `attributes` and `txs` in one shot, computing all
    /// roots. Does not commit; the caller round-trips it through [`import_payload`].
    fn build_block(
        chain: &EphemeralChain,
        attributes: OpNextBlockEnvAttributes,
        txs: Vec<OpTransactionSigned>,
    ) -> crate::Result<RecoveredBlock<OpBlock>> {
        let evm_config: OpEvmConfig =
            OpEvmConfig::new(chain.chain_spec(), OpRethReceiptBuilder::default());
        let parent = chain.tip_header();
        let state = chain.latest_state()?;
        let mut db = State::builder()
            .with_database(StateProviderDatabase::new(&state))
            .with_bundle_update()
            .build();
        // Assembling an OP block reads the DA-footprint scalar from the L1Block predeploy; a cold
        // cache there panics, so preload it.
        db.load_cache_account(L1_BLOCK_CONTRACT).map_err(exec_err)?;

        let mut builder =
            evm_config.builder_for_next_block(&mut db, &parent, attributes).map_err(exec_err)?;
        builder.apply_pre_execution_changes().map_err(exec_err)?;
        for tx in txs {
            let recovered = SignerRecoverable::try_into_recovered(tx).map_err(|_| {
                Error::Execution("failed to recover transaction sender".to_string())
            })?;
            builder.execute_transaction(recovered).map_err(exec_err)?;
        }
        let BlockBuilderOutcome { block, .. } = builder.finish(&state, None).map_err(exec_err)?;
        Ok(block)
    }

    fn exec_err(err: impl core::fmt::Display) -> Error {
        Error::Execution(err.to_string())
    }

    #[test]
    fn import_block_with_deposit() {
        let user = user_tx(0);
        let sender = user.recover_signer().expect("recover user sender");
        let chain = test_chain(sender);

        // Produce a valid block (deposit first, then the user tx) and round-trip it as a payload.
        let built = build_block(&chain, next_attrs(&chain), vec![deposit_tx(depositor()), user])
            .expect("build block");
        let block_hash = built.hash();
        let block = built.clone_sealed_block().into_block();
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
        let status =
            import_payload(&chain, OpExecutionData::new(payload, sidecar)).expect("import payload");

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
        let balance = state.account_balance(&sender).expect("balance").unwrap_or_default();
        let charged = FUNDED_BALANCE - balance;
        assert!(
            charged > gas_charge,
            "charged {charged} should exceed pure gas {gas_charge} by the L1 fee",
        );
    }

    #[test]
    fn import_invalid_block() {
        let user = user_tx(0);
        let sender = user.recover_signer().expect("recover user sender");
        let chain = test_chain(sender);

        let built = build_block(&chain, next_attrs(&chain), vec![deposit_tx(depositor()), user])
            .expect("build");
        // Corrupt the state root; `from_block_slow` re-derives a matching block hash, so the layout
        // check passes and only execution catches the mismatch.
        let mut block = built.clone_sealed_block().into_block();
        let parent_hash = block.header.parent_hash;
        block.header.state_root = B256::repeat_byte(0xff);

        let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
        let status =
            import_payload(&chain, OpExecutionData::new(payload, sidecar)).expect("import payload");

        assert!(status.is_invalid(), "expected INVALID, got {status:?}");
        // INVALID points at the last valid block (the parent), not the rejected block.
        assert_eq!(status.latest_valid_hash, Some(parent_hash));
        // The corrupt block was not committed.
        assert!(chain.block_by_number(1).expect("query").is_none());
    }

    /// A genesis-only engine (forkchoice needs no hardforks, so a default genesis suffices).
    fn simple_engine() -> crate::TestEngine {
        let genesis = Genesis::default().extend_accounts([(
            Address::with_last_byte(0x42),
            GenesisAccount { balance: FUNDED_BALANCE, ..Default::default() },
        )]);
        crate::TestEngine::new(genesis).expect("engine")
    }

    fn fcu(head: B256, safe: B256, finalized: B256) -> alloy_rpc_types_engine::ForkchoiceState {
        alloy_rpc_types_engine::ForkchoiceState {
            head_block_hash: head,
            safe_block_hash: safe,
            finalized_block_hash: finalized,
        }
    }

    #[test]
    fn fcu_unknown_head_is_syncing() {
        let engine = simple_engine();
        let genesis_hash = engine.header_by_number(0).unwrap().unwrap().hash_slow();

        // An unknown head must report SYNCING — never a silent VALID.
        let updated =
            engine.forkchoice_updated(fcu(B256::repeat_byte(0xaa), B256::ZERO, B256::ZERO), None);
        assert!(updated.unwrap().is_syncing());

        // A known head (genesis) is VALID.
        let updated =
            engine.forkchoice_updated(fcu(genesis_hash, genesis_hash, genesis_hash), None).unwrap();
        assert!(updated.is_valid());
        assert_eq!(updated.payload_status.latest_valid_hash, Some(genesis_hash));
    }

    #[test]
    fn fcu_unknown_safe_is_error() {
        let engine = simple_engine();
        let genesis_hash = engine.header_by_number(0).unwrap().unwrap().hash_slow();
        // A known head with an unknown (non-zero) safe block is a forkchoice error, not silently
        // ignored.
        let err = engine
            .forkchoice_updated(fcu(genesis_hash, B256::repeat_byte(0xbb), B256::ZERO), None)
            .unwrap_err();
        assert!(matches!(err, Error::UnknownForkchoiceBlock { which: "safe", .. }), "{err:?}");
    }
}
