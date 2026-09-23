//! Public projection execution parity tests.

pub(crate) use kona_protocol::is_projection_user_deposit as is_user_deposit;

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
