//! SDM operator-gate coverage for the public op-rbuilder binary.
//!
//! The public builder installs the stock null refund policy. Lagoon activation and the admin RPC
//! remain wired for downstream policies, but toggling the operator opt-in must never make this
//! shipped binary emit a trailing PostExec (`0x7D`) transaction.

use crate::{
    sdm_admin::SdmAdminApiClient,
    tests::{BlockTransactionsExt, LocalInstance, default_node_config},
};
use alloy_provider::Provider;
use alloy_rpc_types_eth::Block;
use macros::rb_test;
use op_alloy_consensus::{OpTxEnvelope, PostExecPayload};
use op_alloy_rpc_types::Transaction;
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpec;
use std::sync::Arc;

/// The test-framework chain spec with the SDM protocol gate (Lagoon) active at genesis.
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
async fn null_policy_is_inert_across_operator_opt_in_toggles(
    rbuilder: LocalInstance,
) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let client = rbuilder.rpc_client().await?;

    let status = client.sdm_status(None).await?;
    assert!(status.protocol_active, "Lagoon must be active at genesis");
    assert!(
        !status.operator_sdm_opt_in,
        "operator opt-in must start disabled"
    );

    for enabled in [false, true, false] {
        client.set_operator_sdm_opt_in(enabled).await?;
        let status = client.sdm_status(None).await?;
        assert_eq!(status.operator_sdm_opt_in, enabled);
        assert_eq!(
            status.effective, enabled,
            "the shared gate plumbing must remain live"
        );

        let pending = driver.create_transaction().send().await?;
        let block = driver.build_new_block().await?;
        assert!(
            block.includes(pending.tx_hash()),
            "normal transaction must make progress"
        );
        assert!(
            post_exec_txs(&block).is_empty(),
            "public null policy emitted a PostExec tx with operator opt-in {enabled}"
        );

        driver
            .provider()
            .get_transaction_receipt(*pending.tx_hash())
            .await?
            .expect("normal transaction receipt must exist");
    }

    Ok(())
}
