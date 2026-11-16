//! `debug_traceBlock` over raw blocks containing an SDM post-exec (0x7d) transaction.
//!
//! A raw block replays through the block executor like a canonical one, so its traces carry the
//! payload's refund settled and a block the executor rejects does not trace ([#485]). Tracing must
//! also not fail whole-block on the 0x7d tx itself ([premium#163]): the block executor and
//! `OpEvm::transact_raw` both give it the consensus-defined no-op result.
//!
//! [#485]: https://github.com/ethereum-optimism/protocol-team/issues/485
//! [premium#163]: https://github.com/ethereum-optimism/optimism-premium/issues/163

use crate::lagoon::{TRANSFER_GAS, default_trace_gas, http_client, launch_node};
use alloy_consensus::{
    Block, BlockBody, EMPTY_ROOT_HASH, Header, Sealable, SignableTransaction, TxEip1559,
};
use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256};
use alloy_rpc_types_eth::TransactionRequest;
use alloy_rpc_types_trace::{common::TraceResult, geth::GethDebugTracingOptions};
use jsonrpsee::core::ClientError;
use op_alloy_consensus::{OpTxEnvelope, SDMGasEntry, TxDeposit, build_post_exec_tx};
use reth_chainspec::EthChainSpec;
use reth_evm::execute::BlockValidationError;
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::args::RollupArgs;
use reth_rpc_api::clients::DebugApiClient;
use std::sync::Arc;

/// `debug_traceBlock` over a raw block that ends in a post-exec (0x7d) transaction returns one
/// trace per transaction, including the post-exec one, with the payload's refund settled.
///
/// The refund takes the transfer below the EIP-7623 floor: the producer never refunds that far,
/// but a verifier applies whatever refund the payload carries.
#[tokio::test]
async fn test_debug_trace_block_with_post_exec_tx() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().lagoon_activated().build());
    let node = launch_node(&chain_spec, RollupArgs::default()).await?;
    let client = http_client(&node);
    let block = raw_sdm_block(&chain_spec, 30_000_000)?;

    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block(
        &client,
        alloy_rlp::encode(&block).into(),
        Some(GethDebugTracingOptions::default()),
    )
    .await?;

    assert_eq!(traces.len(), 3, "one trace per transaction, including the post-exec tx");
    for (index, trace) in traces.iter().enumerate() {
        assert!(
            matches!(trace, TraceResult::Success { .. }),
            "transaction {index} must trace successfully, got {trace:?}",
        );
    }
    assert_eq!(default_trace_gas(&traces[1]), TRANSFER_GAS - 1, "debug_traceBlock gas is settled");

    Ok(())
}

/// A raw block the block executor rejects does not trace: with the block budget exhausted by the
/// deposit, replaying the transfer fails block gas accounting and the error names the shortfall.
#[tokio::test]
async fn test_debug_trace_block_rejects_exhausted_budget() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().lagoon_activated().build());
    let node = launch_node(&chain_spec, RollupArgs::default()).await?;
    let client = http_client(&node);
    let block = raw_sdm_block(&chain_spec, TRANSFER_GAS)?;

    let error = DebugApiClient::<TransactionRequest>::debug_trace_block(
        &client,
        alloy_rlp::encode(&block).into(),
        Some(GethDebugTracingOptions::default()),
    )
    .await
    .expect_err("debug_traceBlock must reject the transfer");

    let ClientError::Call(error) = error else {
        panic!("debug_traceBlock must fail with a call error: {error:?}");
    };
    let rejection = BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
        transaction_gas_limit: TRANSFER_GAS,
        block_available_gas: 0,
    };
    assert_eq!(
        error.message(),
        rejection.to_string(),
        "debug_traceBlock error must be the executor's rejection",
    );

    Ok(())
}

/// Builds block 1 on genesis state: a funding deposit, a refunded transfer and the post-exec
/// transaction carrying that refund, under the given block gas limit.
///
/// `debug_traceBlock` replays a caller-supplied raw block on its parent's state, so the block only
/// has to decode and recover senders; consensus fields like the state root are never validated.
fn raw_sdm_block(
    chain_spec: &OpChainSpec,
    block_gas_limit: u64,
) -> eyre::Result<Block<OpTxEnvelope>> {
    let genesis = chain_spec.genesis_header();
    let header = Header {
        parent_hash: chain_spec.genesis_hash(),
        number: 1,
        timestamp: genesis.timestamp + 2,
        gas_limit: block_gas_limit,
        base_fee_per_gas: genesis.base_fee_per_gas,
        // Post-Ecotone replay runs the EIP-4788 pre-block system call, which requires a parent
        // beacon block root; the blob-gas and withdrawals fields must be present for the later
        // optionals to round-trip through RLP.
        withdrawals_root: Some(EMPTY_ROOT_HASH),
        blob_gas_used: Some(0),
        excess_blob_gas: Some(0),
        parent_beacon_block_root: Some(B256::ZERO),
        ..Default::default()
    };

    // Fund the regular transaction's recovered sender with a preceding deposit. A valid version-1
    // post-exec payload must be non-empty and may only target regular transactions.
    let regular = TxEip1559 {
        chain_id: chain_spec.chain_id(),
        // The preceding deposit increments the depositor nonce after Regolith.
        nonce: 1,
        gas_limit: TRANSFER_GAS,
        max_fee_per_gas: u128::from(header.base_fee_per_gas.unwrap_or_default()),
        to: TxKind::Call(Address::repeat_byte(0x99)),
        ..Default::default()
    }
    .into_signed(Signature::test_signature());
    let signer = regular.recover_signer()?;
    let deposit = TxDeposit {
        source_hash: B256::with_last_byte(1),
        from: signer,
        to: TxKind::Call(Address::repeat_byte(0x99)),
        mint: 1_000_000_000_000_000_000,
        value: U256::ZERO,
        gas_limit: 21_000,
        is_system_transaction: false,
        input: Bytes::new(),
    };
    let transactions = vec![
        OpTxEnvelope::Deposit(deposit.seal_slow()),
        OpTxEnvelope::Eip1559(regular),
        OpTxEnvelope::PostExec(
            build_post_exec_tx(1, vec![SDMGasEntry { index: 1, gas_refund: 1 }]).seal_slow(),
        ),
    ];
    Ok(Block::new(header, BlockBody { transactions, ommers: vec![], withdrawals: None }))
}
