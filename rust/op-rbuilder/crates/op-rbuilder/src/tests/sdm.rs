//! SDM PostExec (`0x7D`) production tests.
//!
//! These run against a chain spec with Interop (Lagoon) active at genesis, so the SDM
//! protocol gate is on and the operator opt-in (`admin_setSdmPostExecOptIn`) decides
//! whether the builder produces the trailing PostExec tx. The driver round-trips every
//! built payload through `newPayload` on the same node, which re-executes it with the
//! post-exec mode derived from the block's own transactions — so a block whose state
//! includes refund settlement but lacks the trailing `0x7D` (or carries a spurious one)
//! fails validation with a state-root mismatch and the test errors out.

use crate::{
    sdm_admin::SdmAdminApiClient,
    tests::{BlockTransactionsExt, LocalInstance, default_node_config},
};
use alloy_primitives::{Bytes, hex};
use alloy_provider::Provider;
use alloy_rpc_types_eth::Block;
use macros::rb_test;
use op_alloy_consensus::{OpTxEnvelope, PostExecPayload};
use op_alloy_rpc_types::Transaction;
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpec;
use std::sync::Arc;

/// Init code deploying a 5-byte runtime (`PUSH1 1, PUSH0, SSTORE, STOP`) that stores `1`
/// into slot 0 on every call. Two calls in the same block make the second call touch a
/// slot the first already warmed, which is exactly what generates an SDM gas refund entry.
const STORE_SLOT_ZERO_INIT_CODE: [u8; 14] = hex!("60058060095f395ff360015f5500");

/// The test-framework chain spec with the SDM protocol gate (Interop/Lagoon) active at
/// genesis. Everything else matches [`default_node_config`].
fn sdm_node_config() -> NodeConfig<OpChainSpec> {
    let mut genesis: serde_json::Value =
        serde_json::from_str(include_str!("./framework/artifacts/genesis.json.tmpl"))
            .expect("invalid genesis JSON");
    genesis["config"]["lagoonTime"] = 0.into();
    let chain_spec =
        OpChainSpec::from_genesis(serde_json::from_value(genesis).expect("invalid genesis"));

    NodeConfig::<OpChainSpec> {
        chain: Arc::new(chain_spec),
        ..default_node_config()
    }
}

/// Returns `(tx position, decoded payload)` for every PostExec tx in the block.
fn post_exec_txs(block: &Block<Transaction>) -> Vec<(usize, PostExecPayload)> {
    block
        .transactions
        .as_transactions()
        .expect("block must be requested with full transactions")
        .iter()
        .enumerate()
        .filter_map(|(position, tx)| match tx.inner.inner.inner() {
            OpTxEnvelope::PostExec(sealed) => Some((position, sealed.inner().payload.clone())),
            _ => None,
        })
        .collect()
}

#[rb_test(config = sdm_node_config())]
async fn post_exec_tx_follows_operator_opt_in(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let client = rbuilder.rpc_client().await?;

    let status = client.sdm_status(None).await?;
    assert!(
        status.protocol_active,
        "chain spec must have the SDM protocol gate active at genesis"
    );
    assert!(
        !status.post_exec_opt_in,
        "operator opt-in must start disabled on boot"
    );

    // Deploy the repeated-slot contract.
    let deploy = driver
        .create_transaction()
        .with_create()
        .with_input(Bytes::from_static(&STORE_SLOT_ZERO_INIT_CODE))
        .send()
        .await?;
    driver.build_new_block().await?;
    let receipt = driver
        .provider()
        .get_transaction_receipt(*deploy.tx_hash())
        .await?
        .expect("deploy receipt must exist");
    let contract = receipt
        .inner
        .contract_address
        .expect("deploy receipt must carry the contract address");

    // Not opted in: the repeated-slot workload must not produce a PostExec tx even though
    // the protocol gate is active.
    let first = driver.create_transaction().with_to(contract).send().await?;
    let second = driver.create_transaction().with_to(contract).send().await?;
    let block = driver.build_new_block().await?;
    assert!(block.includes(first.tx_hash()) && block.includes(second.tx_hash()));
    assert!(
        post_exec_txs(&block).is_empty(),
        "no PostExec tx may be produced without the operator opt-in"
    );

    // Opted in: the same workload must produce exactly one PostExec tx, in the final
    // position, anchored to this block, with refund entries pointing at earlier txs.
    client.set_sdm_post_exec_opt_in(true).await?;
    assert!(client.sdm_status(None).await?.effective);

    let first = driver.create_transaction().with_to(contract).send().await?;
    let second = driver.create_transaction().with_to(contract).send().await?;
    let block = driver.build_new_block().await?;
    assert!(block.includes(first.tx_hash()) && block.includes(second.tx_hash()));

    let post_exec = post_exec_txs(&block);
    let tx_count = block.transactions.len();
    assert_eq!(
        post_exec.len(),
        1,
        "opted-in builder must produce exactly one PostExec tx, found {post_exec:?}"
    );
    let (position, payload) = &post_exec[0];
    assert_eq!(
        *position,
        tx_count - 1,
        "the PostExec tx must be the last transaction in the block"
    );
    assert_eq!(
        payload.block_number, block.header.number,
        "the PostExec payload must be anchored to its block"
    );
    assert!(
        !payload.gas_refund_entries.is_empty(),
        "the repeated-slot workload must generate refund entries"
    );
    for entry in &payload.gas_refund_entries {
        assert!(
            (entry.index as usize) < *position,
            "refund entries must reference transactions before the PostExec tx"
        );
    }

    // Opting back out stops production again.
    client.set_sdm_post_exec_opt_in(false).await?;
    let first = driver.create_transaction().with_to(contract).send().await?;
    let block = driver.build_new_block().await?;
    assert!(block.includes(first.tx_hash()));
    assert!(
        post_exec_txs(&block).is_empty(),
        "no PostExec tx may be produced after opting back out"
    );

    Ok(())
}
