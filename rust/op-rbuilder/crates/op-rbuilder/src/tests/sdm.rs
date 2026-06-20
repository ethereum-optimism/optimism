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
use alloy_primitives::Bytes;
use alloy_provider::Provider;
use alloy_rpc_types_eth::Block;
use macros::rb_test;
use op_alloy_consensus::{OpTxEnvelope, PostExecPayload};
use op_alloy_rpc_types::Transaction;
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpec;
use revm::bytecode::opcode;
use std::sync::Arc;

/// Runtime bytecode `SSTORE(slot 0, 1); STOP` — stores `1` into slot 0 on every call.
/// Calling the deployed contract twice in one block makes the second call write a slot the
/// first already warmed, which is exactly the access pattern that produces an SDM gas
/// refund entry.
fn store_slot_zero_runtime() -> [u8; 5] {
    [
        opcode::PUSH1,
        0x01,
        opcode::PUSH0,
        opcode::SSTORE,
        opcode::STOP,
    ]
}

/// Wraps `runtime` in a minimal constructor that copies it into memory and returns it,
/// yielding init code for a `CREATE` transaction.
fn deploy(runtime: &[u8]) -> Bytes {
    let size = u8::try_from(runtime.len()).expect("runtime length fits in one byte");
    let mut code = vec![
        opcode::PUSH1,
        size,         // size of the runtime to copy and return
        opcode::DUP1, //   keep a second copy of size for RETURN below
        opcode::PUSH1,
        0,                // [patched] offset of the runtime within this code
        opcode::PUSH0,    // destination offset 0 in memory
        opcode::CODECOPY, // memory[0..size] = code[offset..offset + size]
        opcode::PUSH0,    // return offset 0
        opcode::RETURN,   // return memory[0..size] as the deployed runtime
    ];
    // The runtime is appended right after the constructor, so its offset is the
    // constructor length — patch the placeholder rather than hard-coding it.
    code[4] = u8::try_from(code.len()).expect("constructor length fits in one byte");
    code.extend_from_slice(runtime);
    Bytes::from(code)
}

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
        .with_input(deploy(&store_slot_zero_runtime()))
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
