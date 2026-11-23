//! `eth_sendRawTransactionConditional`: the method exists only on a node that enables it, rejects a
//! condition that the next block cannot meet or that costs more than the limit, includes a
//! transaction whose condition holds, and on a node with a sequencer endpoint forwards the
//! transaction to the sequencer.

use crate::lagoon::{TRANSFER_GAS, chain_spec, http_client, launch_node, signed_transaction};
use alloy_consensus::{BlockHeader, EMPTY_ROOT_HASH, TxReceipt};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, keccak256};
use alloy_rpc_types_eth::erc4337::{AccountStorage, TransactionConditional};
use jsonrpsee::{
    core::client::Error as ClientError,
    types::{
        ErrorObjectOwned,
        error::{INVALID_PARAMS_CODE, METHOD_NOT_FOUND_CODE},
    },
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{NodeHelperType, wallet::Wallet};
use reth_optimism_node::{OpNode, args::RollupArgs};
use reth_optimism_txpool::conditional::MaybeConditionalTransaction;
use reth_provider::ReceiptProvider;
use reth_rpc_api::clients::L2EthApiExtClient;
use reth_transaction_pool::TransactionPool;
use std::iter;

/// An account with no storage: every slot reads zero and its storage root is empty.
const NO_STORAGE: Address = Address::repeat_byte(0x42);

/// Rollup arguments that enable `eth_sendRawTransactionConditional`.
fn conditional_enabled() -> RollupArgs {
    RollupArgs { enable_tx_conditional: true, ..Default::default() }
}

/// A signed transfer from the funded test wallet.
async fn transfer(chain_id: u64) -> Bytes {
    let wallet = Wallet::default().with_chain_id(chain_id);
    signed_transaction(
        &wallet,
        0,
        TRANSFER_GAS,
        20_000_000_000,
        TxKind::Call(Address::repeat_byte(0x99)),
        Bytes::new(),
    )
    .await
}

/// A condition on the storage of [`NO_STORAGE`].
fn known_storage(storage: AccountStorage) -> TransactionConditional {
    TransactionConditional {
        known_accounts: iter::once((NO_STORAGE, storage)).collect(),
        ..Default::default()
    }
}

fn call_error(error: ClientError) -> ErrorObjectOwned {
    let ClientError::Call(error) = error else {
        panic!("expected a JSON-RPC error response, got {error:?}");
    };
    error
}

/// Asserts that `node` included `hash` and that the transaction succeeded.
fn assert_included(node: &NodeHelperType<OpNode>, hash: B256) -> eyre::Result<()> {
    let receipt = node.inner.provider.receipt_by_hash(hash)?.expect("the transaction is included");
    assert!(receipt.status(), "the transaction succeeds");
    Ok(())
}

/// Without `--rollup.enable-tx-conditional` the method does not exist.
#[tokio::test]
async fn method_not_found_when_disabled() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = chain_spec()?;
    let node = launch_node(&chain_spec, RollupArgs::default()).await?;

    let error = L2EthApiExtClient::send_raw_transaction_conditional(
        &http_client(&node),
        transfer(chain_spec.chain_id()).await,
        TransactionConditional::default(),
    )
    .await
    .expect_err("the method must not be served");
    assert_eq!(call_error(error).code(), METHOD_NOT_FOUND_CODE);

    Ok(())
}

/// A condition that the next block cannot meet, or that costs more than the limit, is rejected for
/// the reason it fails, and the transaction does not enter the pool.
#[tokio::test]
async fn invalid_condition_is_rejected() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = chain_spec()?;
    let mut node = launch_node(&chain_spec, conditional_enabled()).await?;
    let latest = node.advance_block().await?.block().header().clone();
    let client = http_client(&node);
    let tx = transfer(chain_spec.chain_id()).await;

    let cases = [
        (
            "block number max at the latest block",
            TransactionConditional {
                block_number_max: Some(latest.number()),
                ..Default::default()
            },
            "invalid conditional parameters",
        ),
        (
            "timestamp max at the latest block",
            TransactionConditional {
                timestamp_max: Some(latest.timestamp()),
                ..Default::default()
            },
            "invalid conditional parameters",
        ),
        (
            // One for the account and one per slot: 5001, above the limit of 5000.
            "cost above the limit",
            known_storage(AccountStorage::Slots(
                (0..5000u64).map(|slot| (U256::from(slot), B256::ZERO)).collect(),
            )),
            "conditional cost exceeded maximum allowed",
        ),
        (
            "storage slot mismatch",
            known_storage(AccountStorage::Slots(
                iter::once((U256::ZERO, B256::with_last_byte(1))).collect(),
            )),
            "storage value mismatch",
        ),
        (
            "storage root mismatch",
            known_storage(AccountStorage::RootHash(B256::repeat_byte(1))),
            "storage root mismatch",
        ),
    ];
    for (case, condition, message) in cases {
        let error =
            L2EthApiExtClient::send_raw_transaction_conditional(&client, tx.clone(), condition)
                .await
                .expect_err(case);
        let error = call_error(error);
        assert_eq!((error.code(), error.message()), (INVALID_PARAMS_CODE, message), "{case}");
    }
    assert!(node.inner.pool.is_empty(), "a rejected transaction must not enter the pool");

    Ok(())
}

/// A transaction whose condition holds enters the pool with its condition and is included in the
/// next block.
#[tokio::test]
async fn condition_that_holds_is_included() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = chain_spec()?;
    let mut node = launch_node(&chain_spec, conditional_enabled()).await?;
    let latest = node.advance_block().await?.block().header().clone();
    let tx = transfer(chain_spec.chain_id()).await;
    let condition = TransactionConditional {
        known_accounts: [
            (NO_STORAGE, AccountStorage::Slots(iter::once((U256::ZERO, B256::ZERO)).collect())),
            // A second account with no storage: a condition names each account once.
            (Address::repeat_byte(0x43), AccountStorage::RootHash(EMPTY_ROOT_HASH)),
        ]
        .into_iter()
        .collect(),
        block_number_min: Some(0),
        // The max bounds are the next block, which includes the transaction. The test node builds
        // each block one second after its parent.
        block_number_max: Some(latest.number() + 1),
        timestamp_min: Some(0),
        timestamp_max: Some(latest.timestamp() + 1),
    };

    let hash = L2EthApiExtClient::send_raw_transaction_conditional(
        &http_client(&node),
        tx.clone(),
        condition.clone(),
    )
    .await?;
    assert_eq!(hash, keccak256(&tx));
    let pooled = node.inner.pool.get(&hash).expect("the transaction enters the pool");
    assert_eq!(pooled.transaction.conditional(), Some(&condition));

    node.advance_block().await?;
    assert_included(&node, hash)
}

/// A node with a sequencer endpoint forwards the transaction and its condition to the sequencer,
/// which includes it.
#[tokio::test]
async fn forwarded_to_the_sequencer() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = chain_spec()?;
    let mut sequencer = launch_node(&chain_spec, conditional_enabled()).await?;
    let forwarder = launch_node(
        &chain_spec,
        RollupArgs { sequencer: Some(sequencer.rpc_url().to_string()), ..conditional_enabled() },
    )
    .await?;
    let tx = transfer(chain_spec.chain_id()).await;
    let condition = TransactionConditional { timestamp_min: Some(0), ..Default::default() };

    let hash = L2EthApiExtClient::send_raw_transaction_conditional(
        &http_client(&forwarder),
        tx.clone(),
        condition.clone(),
    )
    .await?;
    assert_eq!(hash, keccak256(&tx));
    let pooled = sequencer.inner.pool.get(&hash).expect("the sequencer pools the transaction");
    assert_eq!(pooled.transaction.conditional(), Some(&condition));

    sequencer.advance_block().await?;
    assert_included(&sequencer, hash)
}
