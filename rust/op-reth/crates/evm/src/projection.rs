//! Public projection execution parity tests.

pub(crate) use kona_protocol::is_projection_user_deposit as is_user_deposit;

use alloy_evm::block::{BlockExecutionError, BlockValidationError};
use alloy_op_evm::block::OpBlockExecutionError;

/// Returns whether `err` is the projection execution rule rejecting a block: a sequencer
/// transaction that did not execute successfully (`ProjectionSequencerTxFailed`) or that the EVM
/// rejected as invalid (`ProjectionSequencerTxInvalid`: nonce, intrinsic gas, calldata floor, ...).
pub fn is_projection_sequencer_tx_failure(err: &BlockExecutionError) -> bool {
    matches!(
        err,
        BlockExecutionError::Validation(BlockValidationError::Other(inner))
            if matches!(
                inner.downcast_ref::<OpBlockExecutionError>(),
                Some(
                    OpBlockExecutionError::ProjectionSequencerTxFailed { .. } |
                        OpBlockExecutionError::ProjectionSequencerTxInvalid { .. }
                )
            )
    )
}

/// Checks `err` against an invalid outcome of the shared executor vectors
/// (`op-private-interop/projection/testdata/execution.json`): the expected error variant and,
/// for the projection rule, the failing transaction's index.
#[cfg(test)]
pub(crate) fn assert_vector_error(
    err: &BlockExecutionError,
    expected: &serde_json::Value,
    what: &str,
) {
    let want = expected["error"].as_str().unwrap();
    let index = expected["tx_index"].as_u64().unwrap();
    match (want, err) {
        ("InvalidTx", BlockExecutionError::Validation(BlockValidationError::InvalidTx { .. })) => {}
        (_, BlockExecutionError::Validation(BlockValidationError::Other(inner))) => {
            match (want, inner.downcast_ref::<OpBlockExecutionError>()) {
                (
                    "ProjectionSequencerTxFailed",
                    Some(OpBlockExecutionError::ProjectionSequencerTxFailed { tx_index }),
                ) |
                (
                    "ProjectionSequencerTxInvalid",
                    Some(OpBlockExecutionError::ProjectionSequencerTxInvalid { tx_index, .. }),
                ) => assert_eq!(*tx_index, index, "{what}: tx index"),
                _ => panic!("{what}: expected {want}, got {inner}"),
            }
            assert!(is_projection_sequencer_tx_failure(err), "{what}");
        }
        _ => panic!("{what}: expected {want}, got {err}"),
    }
}

#[cfg(test)]
mod tests {
    use crate::{OpBlockExecutionCtx, OpEvmConfig, PostExecMode};
    use alloc::sync::Arc;
    use alloy_consensus::{Sealable, transaction::Recovered};
    use alloy_evm::{
        Evm, EvmEnv, EvmFactory,
        block::{BlockExecutor, BlockExecutorFactory, TxResult},
    };
    use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
    use op_alloy_consensus::{TxDeposit, UserDepositSource};
    use op_revm::OpSpecId;
    use reth_optimism_chainspec::{OpChainSpec, project_genesis_from};
    use reth_optimism_primitives::OpTransactionSigned;
    use revm::{
        Database,
        context::{BlockEnv, CfgEnv},
        database::{InMemoryDB, State},
        state::{AccountInfo, Bytecode},
    };

    /// Both projection execution rules are switched on by the projection genesis only.
    #[test]
    fn only_projection_genesis_enables_execution_rules() {
        let private: alloy_genesis::Genesis = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
        ))
        .unwrap();
        let plain = OpEvmConfig::optimism(Arc::new(OpChainSpec::from_genesis(private.clone())));
        assert!(plain.executor_factory.deposit_noop().is_none());
        assert!(!plain.executor_factory.requires_sequencer_tx_success());
        let projected = OpEvmConfig::optimism(Arc::new(OpChainSpec::from_genesis(
            project_genesis_from(&private).unwrap(),
        )));
        assert!(projected.executor_factory.deposit_noop().is_some());
        assert!(projected.executor_factory.requires_sequencer_tx_success());
    }

    /// Every case of the shared op-reth/Kona executor vectors
    /// (`op-private-interop/projection/testdata/execution.json`, spec-sound-profile §E.4), in both
    /// modes, through the real trigger in [`OpEvmConfig::optimism`]: the private genesis for
    /// `execution`, its projection for `projection`. Blocks run from the vector prestate in an
    /// `InMemoryDB`.
    #[test]
    fn projection_execution_vectors() {
        use alloy_consensus::{Header, TxReceipt, transaction::SignerRecoverable};
        use alloy_eips::Decodable2718;
        use alloy_genesis::GenesisAccount;
        use reth_evm::ConfigureEvm;
        use revm::database::states::bundle_state::BundleRetention;
        use serde_json::Value;
        use std::collections::BTreeMap;

        fn b256(v: &Value) -> B256 {
            v.as_str().unwrap().parse().unwrap()
        }

        let private: alloy_genesis::Genesis = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
        ))
        .unwrap();
        let vectors: Value = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/projection/testdata/execution.json"
        ))
        .unwrap();
        let env = &vectors["env"];
        let u64_of = |key: &str| env[key].as_u64().unwrap();
        let prestate: BTreeMap<Address, AccountInfo> = env["prestate"]
            .as_object()
            .unwrap()
            .iter()
            .map(|(address, account)| {
                let code: Bytes = account["code"].as_str().unwrap().parse().unwrap();
                let code = Bytecode::new_raw(code);
                let info = AccountInfo {
                    nonce: account["nonce"].as_u64().unwrap(),
                    balance: account["balance"].as_str().unwrap().parse().unwrap(),
                    code_hash: code.hash_slow(),
                    code: Some(code),
                    ..Default::default()
                };
                (address.parse().unwrap(), info)
            })
            .collect();
        let cases = vectors["cases"].as_array().unwrap();
        assert!(cases.len() >= 9);

        for case in cases {
            let name = case["name"].as_str().unwrap();
            let txs: Vec<_> = case["transactions"]
                .as_array()
                .unwrap()
                .iter()
                .map(|raw| {
                    let raw: Bytes = raw.as_str().unwrap().parse().unwrap();
                    OpTransactionSigned::decode_2718(&mut raw.as_ref())
                        .unwrap()
                        .try_into_recovered()
                        .unwrap()
                })
                .collect();
            for (mode, projection) in [("execution", false), ("projection", true)] {
                let expected = &case[mode];
                let genesis = if projection {
                    project_genesis_from(&private).unwrap()
                } else {
                    private.clone()
                };
                let config = OpEvmConfig::optimism(Arc::new(OpChainSpec::from_genesis(genesis)));
                let header = Header {
                    number: u64_of("number"),
                    timestamp: u64_of("timestamp"),
                    gas_limit: u64_of("gas_limit"),
                    base_fee_per_gas: Some(u64_of("base_fee")),
                    beneficiary: env["coinbase"].as_str().unwrap().parse().unwrap(),
                    mix_hash: b256(&env["prev_randao"]),
                    ..Default::default()
                };
                let mut evm_env = config.evm_env(&header).unwrap();
                evm_env.cfg_env.chain_id = u64_of("chain_id");

                let mut db = State::builder()
                    .with_database(InMemoryDB::default())
                    .with_bundle_update()
                    .build();
                for (address, info) in &prestate {
                    db.insert_account(*address, info.clone());
                }
                let factory = &config.executor_factory;
                let evm = factory.evm_factory().create_evm(&mut db, evm_env);
                let executor = factory.create_executor(
                    evm,
                    OpBlockExecutionCtx {
                        parent_beacon_block_root: Some(b256(&env["parent_beacon_block_root"])),
                        ..Default::default()
                    },
                );
                let result = executor.execute_block(txs.iter());

                if !expected["valid"].as_bool().unwrap() {
                    let err =
                        result.err().unwrap_or_else(|| panic!("{name}/{mode}: must be invalid"));
                    super::assert_vector_error(&err, expected, &format!("{name}/{mode}"));
                    continue;
                }

                let result = result.unwrap_or_else(|e| panic!("{name}/{mode}: {e}"));
                let statuses: Vec<u64> =
                    result.receipts.iter().map(|r| u64::from(r.status())).collect();
                let want: Vec<u64> = expected["statuses"]
                    .as_array()
                    .unwrap()
                    .iter()
                    .map(|s| s.as_u64().unwrap())
                    .collect();
                assert_eq!(statuses, want, "{name}/{mode}: statuses");
                assert_eq!(
                    result.gas_used,
                    expected["gas_used"].as_u64().unwrap(),
                    "{name}/{mode}: gas used"
                );

                // Post-state = prestate overlaid with the bundle. The state root is computed as a
                // genesis alloc root, which needs no trie dependency here.
                db.merge_transitions(BundleRetention::PlainState);
                let bundle = db.take_bundle();
                let mut post: BTreeMap<Address, GenesisAccount> = prestate
                    .iter()
                    .map(|(address, info)| {
                        let account = GenesisAccount::default()
                            .with_nonce(Some(info.nonce))
                            .with_balance(info.balance)
                            .with_code(info.code.as_ref().map(|c| c.original_bytes()));
                        (*address, account)
                    })
                    .collect();
                for (address, account) in bundle.state() {
                    let Some(info) = account.info.as_ref() else {
                        post.remove(address);
                        continue;
                    };
                    let code = info
                        .code
                        .clone()
                        .or_else(|| bundle.contracts.get(&info.code_hash).cloned());
                    let storage: BTreeMap<B256, B256> = account
                        .storage
                        .iter()
                        .filter(|(_, slot)| !slot.present_value.is_zero())
                        .map(|(key, slot)| (B256::from(*key), B256::from(slot.present_value)))
                        .collect();
                    let entry = GenesisAccount::default()
                        .with_nonce(Some(info.nonce))
                        .with_balance(info.balance)
                        .with_code(code.map(|c| c.original_bytes()).filter(|c| !c.is_empty()))
                        .with_storage((!storage.is_empty()).then_some(storage));
                    post.insert(*address, entry);
                }
                let root = OpChainSpec::from_genesis(
                    alloy_genesis::Genesis::default().extend_accounts(post),
                )
                .genesis_header()
                .state_root;
                assert_eq!(root, b256(&expected["state_root"]), "{name}/{mode}: state root");
            }
        }
    }

    #[test]
    fn projection_build_and_import_share_deposit_policy() {
        use crate::{ConfigurePostExecEvm, OpNextBlockEnvAttributes};
        use alloy_consensus::{Block, BlockBody};
        use reth_evm::execute::BlockBuilder;
        use reth_primitives_traits::{SealedBlock, SealedHeader};

        let private: alloy_genesis::Genesis = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
        ))
        .unwrap();
        let config = OpEvmConfig::optimism(Arc::new(OpChainSpec::from_genesis(
            project_genesis_from(&private).unwrap(),
        )));
        let parent = SealedHeader::seal_slow(config.chain_spec().genesis_header().clone());
        let sender = Address::with_last_byte(0xaa);
        let tx: OpTransactionSigned = TxDeposit {
            from: sender,
            to: TxKind::Call(sender),
            mint: 1000,
            gas_limit: 100_000,
            ..Default::default()
        }
        .seal_slow()
        .into();
        let tx = Recovered::new_unchecked(tx, sender);
        let attributes = OpNextBlockEnvAttributes {
            timestamp: parent.timestamp + 2,
            suggested_fee_recipient: Address::ZERO,
            prev_randao: B256::ZERO,
            gas_limit: parent.gas_limit,
            parent_beacon_block_root: Some(B256::ZERO),
            extra_data: parent.extra_data.clone(),
        };
        let mut db = State::builder().with_database(InMemoryDB::default()).build();
        let mut builder = config
            .post_exec_builder_for_next_block(&mut db, &parent, attributes, PostExecMode::Produce)
            .unwrap();
        assert_eq!(builder.execute_transaction(tx.clone()).unwrap().tx_gas_used(), 0);
        drop(builder);
        assert_eq!(db.basic(sender).unwrap().unwrap_or_default().balance, U256::ZERO);

        let mut header = parent.clone_header();
        header.number += 1;
        header.timestamp += 2;
        let block = SealedBlock::new_unhashed(Block { header, body: BlockBody::default() });
        let mut db = State::builder().with_database(InMemoryDB::default()).build();
        let mut executor =
            config.post_exec_executor_for_block(&mut db, &block, PostExecMode::Disabled).unwrap();
        assert_eq!(executor.execute_transaction(&tx).unwrap().tx_gas_used(), 0);
        drop(executor);
        assert_eq!(db.basic(sender).unwrap().unwrap_or_default().balance, U256::ZERO);
    }

    #[test]
    fn projection_deposits_have_no_execution_effects() {
        let private: alloy_genesis::Genesis = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
        ))
        .unwrap();
        for projection in [false, true] {
            for create in [false, true] {
                for mode in [
                    PostExecMode::Disabled,
                    PostExecMode::Produce,
                    PostExecMode::Verify(op_alloy_consensus::PostExecPayload {
                        version: op_alloy_consensus::POST_EXEC_PAYLOAD_VERSION,
                        block_number: 1,
                        gas_refund_entries: Vec::new(),
                    }),
                ] {
                    let genesis = if projection {
                        project_genesis_from(&private).unwrap()
                    } else {
                        private.clone()
                    };
                    let config =
                        OpEvmConfig::optimism(Arc::new(OpChainSpec::from_genesis(genesis)));
                    let sender = Address::with_last_byte(0xaa);
                    let target =
                        if create { sender.create(7) } else { Address::with_last_byte(0xbb) };
                    // SSTORE(0, 42), LOG0, STOP: exercises application state and logs as well as
                    // value.
                    let code = Bytecode::new_raw(Bytes::from_static(&[
                        0x60, 42, 0x60, 0, 0x55, 0x60, 0, 0x60, 0, 0xa0, 0,
                    ]));
                    let mut db = State::builder().with_database(InMemoryDB::default()).build();
                    db.insert_account(
                        sender,
                        AccountInfo { balance: U256::from(300), nonce: 7, ..Default::default() },
                    );
                    if !create {
                        db.insert_account(
                            target,
                            AccountInfo {
                                code_hash: code.hash_slow(),
                                code: Some(code.clone()),
                                ..Default::default()
                            },
                        );
                    }
                    let factory = &config.executor_factory;
                    let evm = factory.evm_factory().create_evm(
                        &mut db,
                        EvmEnv {
                            cfg_env: CfgEnv::new_with_spec(OpSpecId::JOVIAN),
                            block_env: BlockEnv {
                                number: U256::from(1),
                                timestamp: U256::from(2_000_000_000u64),
                                gas_limit: 1_000_000,
                                ..Default::default()
                            },
                        },
                    );
                    let mut executor = factory.create_executor(
                        evm,
                        OpBlockExecutionCtx { post_exec_mode: mode, ..Default::default() },
                    );
                    let tx: OpTransactionSigned = TxDeposit {
                        source_hash: UserDepositSource::new(B256::ZERO, 0).source_hash(),
                        from: sender,
                        to: if create { TxKind::Create } else { TxKind::Call(target) },
                        input: if create { code.original_bytes() } else { Bytes::new() },
                        mint: 1_000,
                        value: U256::from(500),
                        gas_limit: 100_000,
                        ..Default::default()
                    }
                    .seal_slow()
                    .into();
                    let tx = Recovered::new_unchecked(tx, sender);
                    let result = executor.execute_transaction_without_commit(&tx).unwrap();
                    assert!(result.result().result.is_success());
                    if projection {
                        assert!(result.result().state.is_empty());
                        assert_eq!(result.evm_gas_used, 0);
                        assert_eq!(result.canonical_gas_used, 0);
                        assert!(result.result().result.logs().is_empty());
                    } else {
                        assert!(result.evm_gas_used > 0);
                        assert_eq!(result.result().result.logs().len(), 1);
                    }
                    executor.commit_transaction(result);
                    let sender_info = executor.evm.db_mut().basic(sender).unwrap().unwrap();
                    let target_info =
                        executor.evm.db_mut().basic(target).unwrap().unwrap_or_default();
                    assert_eq!(sender_info.balance, U256::from(if projection { 300 } else { 800 }));
                    assert_eq!(sender_info.nonce, if projection { 7 } else { 8 });
                    assert_eq!(target_info.balance, U256::from(if projection { 0 } else { 500 }));
                    assert_eq!(
                        executor.evm.db_mut().storage(target, U256::ZERO).unwrap(),
                        U256::from(if projection { 0 } else { 42 })
                    );
                }
            }
        }
    }
}
