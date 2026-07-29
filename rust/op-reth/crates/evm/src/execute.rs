//! Optimism block execution strategy.

/// Helper type with backwards compatible methods to obtain executor providers.
pub type OpExecutorProvider = crate::OpEvmConfig;

#[cfg(test)]
mod tests {
    use crate::{OpEvmConfig, OpRethReceiptBuilder};
    use alloc::sync::Arc;
    use alloy_consensus::{Block, BlockBody, Eip658Value, Header, SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, Signature, StorageKey, StorageValue, U256, address, b256};
    use op_alloy_consensus::TxDeposit;
    use op_revm::constants::L1_BLOCK_CONTRACT;
    use reth_chainspec::MIN_TRANSACTION_GAS;
    use reth_evm::execute::{BasicBlockExecutor, BlockExecutionOutput, Executor};
    use reth_optimism_chainspec::{OP_DEV, OpChainSpec, OpChainSpecBuilder};
    use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
    use reth_primitives_traits::{Account, RecoveredBlock};
    use reth_revm::{database::StateProviderDatabase, test_utils::StateProviderTest};
    use std::{collections::HashMap, str::FromStr};

    fn create_op_state_provider() -> StateProviderTest {
        let mut db = StateProviderTest::default();

        let l1_block_contract_account =
            Account { balance: U256::ZERO, bytecode_hash: None, nonce: 1 };

        let mut l1_block_storage = HashMap::default();
        // base fee
        l1_block_storage.insert(StorageKey::with_last_byte(1), StorageValue::from(1000000000));
        // l1 fee overhead
        l1_block_storage.insert(StorageKey::with_last_byte(5), StorageValue::from(188));
        // l1 fee scalar
        l1_block_storage.insert(StorageKey::with_last_byte(6), StorageValue::from(684000));
        // l1 free scalars post ecotone
        l1_block_storage.insert(
            StorageKey::with_last_byte(3),
            StorageValue::from_str(
                "0x0000000000000000000000000000000000001db0000d27300000000000000005",
            )
            .unwrap(),
        );

        db.insert_account(L1_BLOCK_CONTRACT, l1_block_contract_account, None, l1_block_storage);

        db
    }

    fn evm_config(chain_spec: Arc<OpChainSpec>) -> OpEvmConfig {
        OpEvmConfig::new(chain_spec, OpRethReceiptBuilder::default())
    }

    /// Build a single-block executor for `chain_spec`, preload the L1 block contract, and execute
    /// `transactions` (with their recovered `senders`) against `db`.
    fn execute_block(
        chain_spec: Arc<OpChainSpec>,
        db: &StateProviderTest,
        header: Header,
        transactions: Vec<OpTransactionSigned>,
        senders: Vec<Address>,
    ) -> BlockExecutionOutput<OpReceipt> {
        let provider = evm_config(chain_spec);
        let mut executor = BasicBlockExecutor::new(provider, StateProviderDatabase::new(db));

        // make sure the L1 block contract state is preloaded.
        executor.with_state_mut(|state| {
            state.load_cache_account(L1_BLOCK_CONTRACT).unwrap();
        });

        executor
            .execute(&RecoveredBlock::new_unhashed(
                Block { header, body: BlockBody { transactions, ..Default::default() } },
                senders,
            ))
            .expect("block execution should succeed")
    }

    #[test]
    fn op_deposit_fields_pre_canyon() {
        let header = Header {
            timestamp: 1,
            number: 1,
            gas_limit: 1_000_000,
            gas_used: 42_000,
            receipts_root: b256!(
                "0x83465d1e7d01578c0d609be33570f91242f013e9e295b0879905346abbd63731"
            ),
            ..Default::default()
        };

        let mut db = create_op_state_provider();

        let addr = Address::ZERO;
        let account = Account { balance: U256::MAX, ..Account::default() };
        db.insert_account(addr, account, None, HashMap::default());

        let chain_spec =
            Arc::new(OpChainSpecBuilder::optimism_mainnet().regolith_activated().build());

        let tx: OpTransactionSigned = TxEip1559 {
            chain_id: chain_spec.chain.id(),
            nonce: 0,
            gas_limit: MIN_TRANSACTION_GAS,
            to: addr.into(),
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into();

        let tx_deposit: OpTransactionSigned = TxDeposit {
            from: addr,
            to: addr.into(),
            gas_limit: MIN_TRANSACTION_GAS,
            ..Default::default()
        }
        .into();

        // Attempt to execute a block with one deposit and one non-deposit transaction
        let output = execute_block(chain_spec, &db, header, vec![tx, tx_deposit], vec![addr, addr]);

        let receipts = &output.receipts;
        let tx_receipt = &receipts[0];
        let deposit_receipt = &receipts[1];

        assert!(!matches!(tx_receipt, OpReceipt::Deposit(_)));
        // deposit_nonce is present only in deposit transactions
        let OpReceipt::Deposit(deposit_receipt) = deposit_receipt else {
            panic!("expected deposit")
        };
        assert!(deposit_receipt.deposit_nonce.is_some());
        // deposit_receipt_version is not present in pre canyon transactions
        assert!(deposit_receipt.deposit_receipt_version.is_none());
    }

    #[test]
    fn op_deposit_fields_post_canyon() {
        // ensure_create2_deployer will fail if timestamp is set to less than 2
        let header = Header {
            timestamp: 2,
            number: 1,
            gas_limit: 1_000_000,
            gas_used: 42_000,
            receipts_root: b256!(
                "0xfffc85c4004fd03c7bfbe5491fae98a7473126c099ac11e8286fd0013f15f908"
            ),
            ..Default::default()
        };

        let mut db = create_op_state_provider();
        let addr = Address::ZERO;
        let account = Account { balance: U256::MAX, ..Account::default() };

        db.insert_account(addr, account, None, HashMap::default());

        let chain_spec =
            Arc::new(OpChainSpecBuilder::optimism_mainnet().canyon_activated().build());

        let tx: OpTransactionSigned = TxEip1559 {
            chain_id: chain_spec.chain.id(),
            nonce: 0,
            gas_limit: MIN_TRANSACTION_GAS,
            to: addr.into(),
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into();

        let tx_deposit: OpTransactionSigned = TxDeposit {
            from: addr,
            to: addr.into(),
            gas_limit: MIN_TRANSACTION_GAS,
            ..Default::default()
        }
        .into();

        // attempt to execute an empty block with parent beacon block root, this should not fail
        let output = execute_block(chain_spec, &db, header, vec![tx, tx_deposit], vec![addr, addr]);

        let receipts = &output.receipts;
        let tx_receipt = &receipts[0];
        let deposit_receipt = &receipts[1];

        // deposit_receipt_version is set to 1 for post canyon deposit transactions
        assert!(!matches!(tx_receipt, OpReceipt::Deposit(_)));
        let OpReceipt::Deposit(deposit_receipt) = deposit_receipt else {
            panic!("expected deposit")
        };
        assert_eq!(deposit_receipt.deposit_receipt_version, Some(1));

        // deposit_nonce is present only in deposit transactions
        assert!(deposit_receipt.deposit_nonce.is_some());
    }

    /// Test that demonstrates constructing, executing, and verifying a transaction
    /// on the `OP_DEV` network without constructing a full node.
    ///
    /// This test uses prefunded accounts from the dev genesis (derived from mnemonic
    /// "test test test test test test test test test test test junk").
    #[test]
    fn op_dev_transaction_execution() {
        // First prefunded account from OP_DEV genesis
        // Derived from "test test test test test test test test test test test junk"
        let sender = address!("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266");
        let recipient = address!("70997970C51812dc3A010C7d01b50e0d17dc79C8");

        // Use OP_DEV chain specification
        let chain_spec = OP_DEV.clone();

        // Create state provider with prefunded accounts and L1 block contract
        let mut db = create_op_state_provider();

        // Add sender account with balance from OP_DEV genesis (1,000,000 ETH)
        let sender_balance = U256::from_str("0xD3C21BCECCEDA1000000").unwrap();
        let sender_account = Account { balance: sender_balance, nonce: 0, bytecode_hash: None };
        db.insert_account(sender, sender_account, None, HashMap::default());

        // Add recipient account with zero balance so the post-state assertion is exact
        let recipient_account = Account { balance: U256::ZERO, nonce: 0, bytecode_hash: None };
        db.insert_account(recipient, recipient_account, None, HashMap::default());

        // Create EIP-1559 transfer transaction
        let transfer_value = U256::from(1_000_000_000_000_000_000u128); // 1 ETH
        let tx: OpTransactionSigned = TxEip1559 {
            chain_id: chain_spec.chain.id(),
            nonce: 0,
            gas_limit: MIN_TRANSACTION_GAS,
            max_fee_per_gas: 20_000_000_000,         // 20 gwei
            max_priority_fee_per_gas: 1_000_000_000, // 1 gwei
            to: recipient.into(),
            value: transfer_value,
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into();

        // Block header for execution (OP_DEV has all hardforks active, so we need
        // parent_beacon_block_root for Cancun compatibility)
        let header = Header {
            timestamp: 2,
            number: 1,
            gas_limit: 30_000_000,
            gas_used: MIN_TRANSACTION_GAS,
            base_fee_per_gas: Some(1_000_000_000), // 1 gwei base fee
            parent_beacon_block_root: Some(b256!(
                "0x0000000000000000000000000000000000000000000000000000000000000001"
            )),
            ..Default::default()
        };

        // Execute block with single transfer transaction
        let output = execute_block(chain_spec, &db, header, vec![tx], vec![sender]);

        // Verify execution results
        assert_eq!(output.receipts.len(), 1, "Should have exactly one receipt");

        let receipt = &output.receipts[0];

        // Verify transaction succeeded
        let OpReceipt::Eip1559(eip1559_receipt) = receipt else {
            panic!("Expected EIP-1559 receipt, got {receipt:?}");
        };
        assert_eq!(eip1559_receipt.status, Eip658Value::Eip658(true), "Transaction should succeed");

        // Verify gas was consumed (21000 for simple transfer)
        assert_eq!(
            eip1559_receipt.cumulative_gas_used, MIN_TRANSACTION_GAS,
            "Gas used should match minimum transaction gas"
        );

        // Verify the post-state: the transfer moved funds and the sender nonce was bumped.
        let sender_post = output
            .state
            .account(&sender)
            .and_then(|acc| acc.info.as_ref())
            .expect("sender account should be present in post-state");
        assert_eq!(sender_post.nonce, 1, "Sender nonce should be incremented to 1");

        let recipient_post = output
            .state
            .account(&recipient)
            .and_then(|acc| acc.info.as_ref())
            .expect("recipient account should be present in post-state");
        assert_eq!(
            recipient_post.balance, transfer_value,
            "Recipient balance should equal the transferred value"
        );
    }
}
