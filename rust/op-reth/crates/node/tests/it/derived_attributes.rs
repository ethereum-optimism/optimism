//! Derived (`noTxPool`) attributes are built when `engine_forkchoiceUpdated` hands them to the
//! payload service. Transactions that make the payload invalid are rejected there with `-38003`,
//! so the rollup node can replace the block with a deposits-only one, and the engine receives a
//! valid payload's executed block from `engine_getPayload`.

use alloy_consensus::{Sealable, SignableTransaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, B64, B256, Bytes, Signature, TxKind};
use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated, PayloadStatus};
use jsonrpsee::{
    core::{ClientError, client::ClientT},
    rpc_params,
};
use op_alloy_consensus::{
    OpTxEnvelope, SDMGasEntry, TxDeposit, build_post_exec_tx, encode_jovian_extra_data,
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4};
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_e2e_test_utils::{Adapter, TmpDB};
use reth_node_api::NodeTypesWithDBAdapter;
use reth_node_builder::{FullNode, FullNodeTypesAdapter, Node, NodeBuilder, NodeConfig};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_forks::OpHardforks;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_optimism_payload_builder::{OpPayloadAttrs, OpPayloadBuilder, OpPayloadBuilderAttributes};
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::{HeaderProvider, providers::BlockchainProvider};
use reth_tasks::Runtime;
use std::{sync::Arc, time::Duration};

const INVALID_PAYLOAD_ATTRIBUTES: i32 = -38003;

async fn fcu(
    client: &impl ClientT,
    state: ForkchoiceState,
    attrs: &OpPayloadAttrs,
) -> Result<ForkchoiceUpdated, ClientError> {
    client.request("engine_forkchoiceUpdatedV3", rpc_params![state, attrs.0.clone()]).await
}

#[track_caller]
fn assert_invalid_attributes(result: Result<ForkchoiceUpdated, ClientError>, case: &str) {
    match result {
        Err(ClientError::Call(err)) => {
            assert_eq!(err.code(), INVALID_PAYLOAD_ATTRIBUTES, "{case}: unexpected error {err:?}");
            assert!(
                err.data().is_some_and(|data| data.get().contains("invalid payload attributes")),
                "{case}: not rejected by the payload builder: {err:?}"
            );
        }
        other => panic!("{case}: expected INVALID_PAYLOAD_ATTRIBUTES, got {other:?}"),
    }
}

/// Transactions shared by the cases: a deposit funding the sender of a 21k-gas transfer.
struct Txs {
    deposit: Bytes,
    /// Another deposit, for a later block.
    next_deposit: Bytes,
    /// Valid after `deposit`, which raises the sender's nonce to 1.
    transfer: Bytes,
    nonce_too_high: Bytes,
}

impl Txs {
    fn new(chain_id: u64) -> eyre::Result<Self> {
        let transfer = |nonce| {
            TxEip1559 {
                chain_id,
                nonce,
                gas_limit: 21_000,
                max_fee_per_gas: 1_000_000_000_000,
                to: TxKind::Call(Address::with_last_byte(1)),
                ..Default::default()
            }
            .into_signed(Signature::test_signature())
        };
        let from = transfer(1).recover_signer()?;
        let deposit = |source| TxDeposit {
            source_hash: B256::with_last_byte(source),
            from,
            to: TxKind::Call(Address::with_last_byte(1)),
            mint: 1_000_000_000_000_000_000,
            gas_limit: 21_000,
            ..Default::default()
        };
        let encode = |tx: OpTxEnvelope| -> Bytes { tx.encoded_2718().into() };
        Ok(Self {
            deposit: encode(OpTxEnvelope::Deposit(deposit(1).seal_slow())),
            next_deposit: encode(OpTxEnvelope::Deposit(deposit(2).seal_slow())),
            transfer: encode(OpTxEnvelope::Eip1559(transfer(1))),
            nonce_too_high: encode(OpTxEnvelope::Eip1559(transfer(5))),
        })
    }
}

fn post_exec(block_number: u64, entries: Vec<SDMGasEntry>) -> Bytes {
    OpTxEnvelope::PostExec(build_post_exec_tx(block_number, entries).seal_slow())
        .encoded_2718()
        .into()
}

const fn refund(index: u64, gas_refund: u64) -> SDMGasEntry {
    SDMGasEntry { index, gas_refund }
}

type Provider = BlockchainProvider<NodeTypesWithDBAdapter<OpNode, TmpDB>>;
type TestNode = FullNode<
    Adapter<OpNode>,
    <OpNode as Node<FullNodeTypesAdapter<OpNode, TmpDB, Provider>>>::AddOns,
>;

async fn launch(chain_spec: Arc<OpChainSpec>) -> eyre::Result<TestNode> {
    let mut config = NodeConfig::test().map_chain(chain_spec).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let op_node = OpNode::default();
    Ok(NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node)
}

/// Launches a node on `chain_spec`, checks that every invalid case is rejected at FCU, then builds
/// `valid` and a sibling of it, making both canonical in turn, and finally a deposits-only child.
async fn run(
    chain_spec: Arc<OpChainSpec>,
    invalid: Vec<(&str, Vec<Bytes>)>,
    valid: Vec<Bytes>,
    deposits_only: Vec<Bytes>,
) -> eyre::Result<()> {
    let jovian = chain_spec.is_jovian_active_at_timestamp(chain_spec.genesis_header().timestamp);
    let genesis = chain_spec.genesis_header().clone();
    let node = launch(chain_spec.clone()).await?;
    let client = node.auth_server_handle().http_client();
    let memory = node.provider.canonical_in_memory_state();
    let genesis_head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    let derived = |transactions: Vec<Bytes>, prev_randao: u8, timestamp: u64| {
        let mut attrs = optimism_payload_attributes(timestamp);
        attrs.0.payload_attributes.prev_randao = B256::with_last_byte(prev_randao);
        attrs.0.no_tx_pool = Some(true);
        if jovian {
            attrs.0.eip_1559_params = Some(B64::ZERO);
            attrs.0.min_base_fee = Some(0);
        }
        attrs.0.transactions = Some(transactions);
        attrs
    };

    for (case, transactions) in invalid {
        assert_invalid_attributes(
            fcu(&client, genesis_head, &derived(transactions, 0, genesis.timestamp + 2)).await,
            case,
        );
    }

    let mut canonical = genesis_head;
    // The second build replaces the first block, so it builds on a canonical ancestor of the tip.
    let builds = [(valid.clone(), 1, false), (valid, 2, false), (deposits_only, 3, true)];
    for (transactions, prev_randao, on_tip) in builds {
        let (parent, timestamp) = if on_tip {
            (canonical, genesis.timestamp + 4)
        } else {
            (genesis_head, genesis.timestamp + 2)
        };
        let attrs = derived(transactions.clone(), prev_randao, timestamp);
        let updated = fcu(&client, parent, &attrs).await?;
        assert!(updated.payload_status.is_valid(), "derived attributes {prev_randao}: {updated:?}");
        let id = updated.payload_id.expect("valid derived attributes return a payload id");

        let (block, new_payload) = if jovian {
            let payload: OpExecutionPayloadEnvelopeV4 =
                client.request("engine_getPayloadV5", rpc_params![id]).await?;
            let block = payload.execution_payload.payload_inner.payload_inner.payload_inner.clone();
            let params = rpc_params![
                payload.execution_payload,
                Vec::<B256>::new(),
                payload.parent_beacon_block_root,
                payload.execution_requests
            ];
            (block, ("engine_newPayloadV4", params))
        } else {
            let payload: OpExecutionPayloadEnvelopeV3 =
                client.request("engine_getPayloadV3", rpc_params![id]).await?;
            let block = payload.execution_payload.payload_inner.payload_inner.clone();
            let params = rpc_params![
                payload.execution_payload,
                Vec::<B256>::new(),
                payload.parent_beacon_block_root
            ];
            (block, ("engine_newPayloadV3", params))
        };
        assert_eq!(block.transactions, transactions, "the block keeps every derived transaction");
        assert_eq!(block.parent_hash, parent.head_block_hash);

        // getPayload hands the executed block to the engine, so newPayload does not execute it
        // again. The engine tracks it as the pending block while its parent is the tip.
        if parent == canonical {
            tokio::time::timeout(Duration::from_secs(10), async {
                while memory.pending_block_num_hash().map(|b| b.hash) != Some(block.block_hash) {
                    tokio::time::sleep(Duration::from_millis(10)).await;
                }
            })
            .await
            .expect("the engine never received the executed block");
        }

        let (method, params) = new_payload;
        let status: PayloadStatus = client.request(method, params).await?;
        assert!(status.is_valid(), "newPayload of the built block {prev_randao}: {status:?}");
        canonical = ForkchoiceState::same_hash(block.block_hash);
        let updated: ForkchoiceUpdated =
            client.request("engine_forkchoiceUpdatedV3", rpc_params![canonical]).await?;
        assert!(updated.payload_status.is_valid(), "FCU to the built block: {updated:?}");
    }

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn invalid_derived_transactions_are_rejected_at_fcu() -> eyre::Result<()> {
    // A Jovian parent must carry Jovian extra data, or no child of it passes validation.
    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    genesis.extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism(), 0)?;
    genesis.base_fee_per_gas = Some(1_000_000_000);
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build(),
    );
    let Txs { deposit, next_deposit, transfer, nonce_too_high } = Txs::new(chain_spec.chain_id())?;
    let invalid = vec![
        ("nonce too high", vec![deposit.clone(), nonce_too_high]),
        ("sender cannot pay", vec![transfer.clone()]),
        (
            "two 0x7D",
            vec![
                deposit.clone(),
                transfer.clone(),
                post_exec(1, vec![refund(1, 1)]),
                post_exec(1, vec![refund(1, 1)]),
            ],
        ),
        (
            "0x7D anchored to the next block",
            vec![deposit.clone(), transfer.clone(), post_exec(2, vec![refund(1, 1)])],
        ),
        ("empty refund entries", vec![deposit.clone(), transfer.clone(), post_exec(1, vec![])]),
        (
            "refund targets a deposit",
            vec![deposit.clone(), transfer.clone(), post_exec(1, vec![refund(0, 1)])],
        ),
        (
            "refund above evmGasUsed",
            vec![deposit.clone(), transfer.clone(), post_exec(1, vec![refund(1, 21_001)])],
        ),
        (
            "refund entry past the last tx",
            vec![deposit.clone(), transfer.clone(), post_exec(1, vec![refund(1, 1), refund(5, 1)])],
        ),
        (
            "0x7D is not the last tx",
            vec![deposit.clone(), post_exec(1, vec![refund(1, 1)]), transfer.clone()],
        ),
    ];
    let valid = vec![deposit, transfer, post_exec(1, vec![refund(1, 1)])];
    run(chain_spec, invalid, valid, vec![next_deposit]).await
}

#[tokio::test(flavor = "multi_thread")]
async fn invalid_derived_transactions_are_rejected_at_fcu_before_holocene() -> eyre::Result<()> {
    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().ecotone_activated().build());
    let Txs { deposit, next_deposit, transfer, nonce_too_high } = Txs::new(chain_spec.chain_id())?;
    let invalid = vec![
        ("nonce too high", vec![deposit.clone(), nonce_too_high]),
        ("sender cannot pay", vec![transfer.clone()]),
        (
            "0x7D before SDM",
            vec![deposit.clone(), transfer.clone(), post_exec(1, vec![refund(1, 1)])],
        ),
    ];
    run(chain_spec, invalid, vec![deposit, transfer], vec![next_deposit]).await
}

/// `debug_executePayload` still gives the fault proof program a witness for a derived block with
/// an invalid transaction, which no longer builds.
#[tokio::test(flavor = "multi_thread")]
async fn witness_covers_a_derived_block_up_to_its_invalid_transaction() -> eyre::Result<()> {
    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().ecotone_activated().build());
    let Txs { deposit, nonce_too_high, .. } = Txs::new(chain_spec.chain_id())?;
    let node = launch(chain_spec.clone()).await?;

    let parent = node.provider.sealed_header_by_hash(chain_spec.genesis_hash())?.unwrap();
    let mut attrs = optimism_payload_attributes(parent.timestamp + 2);
    attrs.0.no_tx_pool = Some(true);
    attrs.0.transactions = Some(vec![deposit, nonce_too_high]);
    let builder =
        OpPayloadBuilder::<_, _, _, (), OpPayloadBuilderAttributes<OpTransactionSigned>>::new(
            node.pool.clone(),
            node.provider.clone(),
            node.evm_config,
        );

    let witness = builder.payload_witness(parent, attrs.0)?;
    assert!(!witness.state.is_empty(), "the deposit's state is in the witness");
    Ok(())
}
