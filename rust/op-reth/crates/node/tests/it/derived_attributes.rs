//! Derived (`noTxPool`) attributes are built inside `engine_forkchoiceUpdated`. Transactions that
//! make the payload invalid are rejected there with `-38003`, so the rollup node can replace the
//! block with a deposits-only one, and a valid payload is built exactly once. A build that fails
//! for a node-local reason leaves the attributes to the payload service instead.

use alloy_consensus::{Sealable, SignableTransaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, B64, B256, Bytes, Signature, TxKind};
use alloy_rpc_types_engine::{
    ClientCode, ClientVersionV1, ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use jsonrpsee::{
    core::{ClientError, client::ClientT},
    rpc_params,
};
use op_alloy_consensus::{
    OpTxEnvelope, SDMGasEntry, TxDeposit, build_post_exec_tx, encode_jovian_extra_data,
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4};
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_node_api::{
    AddOnsContext, EngineApiValidator, FullNodeComponents, PayloadBuilderError, PayloadValidator,
};
use reth_node_builder::{
    Node, NodeBuilder, NodeConfig,
    rpc::{EngineApiBuilder, PayloadValidatorBuilder},
};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_forks::OpHardforks;
use reth_optimism_node::{
    OpBuiltPayload, OpEngineTypes, OpEngineValidatorBuilder, OpNode,
    utils::optimism_payload_attributes,
};
use reth_optimism_payload_builder::{OpPayloadAttrs, OpPayloadBuilder, OpPayloadBuilderAttributes};
use reth_optimism_primitives::{OpBlock, OpTransactionSigned};
use reth_optimism_rpc::{
    OpEngineApi,
    engine::{
        DerivedPayloadBuilder, DerivedPayloads, OP_ENGINE_CAPABILITIES,
        builds_at_forkchoice_updated,
    },
};
use reth_payload_builder::PayloadStore;
use reth_primitives_traits::SealedHeader;
use reth_provider::{HeaderProvider, ProviderError};
use reth_rpc_api::IntoEngineApiRpcModule;
use reth_rpc_engine_api::{EngineApi, EngineCapabilities};
use reth_tasks::Runtime;
use std::{
    sync::{Arc, Mutex, mpsc},
    time::Duration,
};

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
            assert_eq!(err.code(), INVALID_PAYLOAD_ATTRIBUTES, "{case}: unexpected error {err:?}")
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

/// Launches a node on `chain_spec`, checks that every invalid case is rejected at FCU, then builds
/// `valid` and a sibling of it, making both canonical in turn, and finally a deposits-only child.
async fn run(
    chain_spec: Arc<OpChainSpec>,
    invalid: Vec<(&str, Vec<Bytes>)>,
    valid: Vec<Bytes>,
    deposits_only: Vec<Bytes>,
) -> eyre::Result<()> {
    let latest_forks =
        chain_spec.is_jovian_active_at_timestamp(chain_spec.genesis_header().timestamp);
    let genesis = chain_spec.genesis_header().clone();
    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);

    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let client = node.auth_server_handle().http_client();
    let memory = node.provider.canonical_in_memory_state();
    let genesis_head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    let derived = |transactions: Vec<Bytes>, prev_randao: u8, timestamp: u64| {
        let mut attrs = optimism_payload_attributes(timestamp);
        attrs.0.payload_attributes.prev_randao = B256::with_last_byte(prev_randao);
        attrs.0.no_tx_pool = Some(true);
        if latest_forks {
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
        let job = node.payload_builder_handle.payload_timestamp(id).await;
        if builds_at_forkchoice_updated(&attrs) {
            assert!(job.is_none(), "the payload was built at FCU, so no payload job may build it");
        } else {
            assert!(job.is_some(), "the payload service builds these attributes");
        }

        let (block, new_payload) = if latest_forks {
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

/// How the derived builder of [`FailingEngineApi`] fails.
#[derive(Clone)]
enum Failure {
    /// The parent's state is unavailable.
    MissingState,
    Panic,
    /// Signals `entered`, then waits for `release` before failing like `MissingState`.
    Held {
        entered: mpsc::Sender<()>,
        release: Arc<Mutex<mpsc::Receiver<()>>>,
    },
}

impl DerivedPayloadBuilder<OpEngineTypes> for Failure {
    fn build_derived_payload(
        &self,
        parent: SealedHeader,
        _attributes: OpPayloadAttrs,
        _id: PayloadId,
    ) -> Result<OpBuiltPayload, PayloadBuilderError> {
        match self {
            Self::MissingState => {}
            Self::Panic => panic!("derived build panicked"),
            Self::Held { entered, release } => {
                entered.send(()).unwrap();
                release.lock().unwrap().recv().unwrap();
            }
        }
        Err(ProviderError::StateForHashNotFound(parent.hash()).into())
    }
}

/// [`reth_optimism_node::OpEngineApiBuilder`], with a derived builder that fails.
#[derive(Clone)]
struct FailingEngineApi(Failure);

type Validator<N> = <OpEngineValidatorBuilder as PayloadValidatorBuilder<N>>::Validator;

impl<N> EngineApiBuilder<N> for FailingEngineApi
where
    N: FullNodeComponents<Types = OpNode>,
    OpEngineValidatorBuilder: PayloadValidatorBuilder<N>,
    Validator<N>: EngineApiValidator<OpEngineTypes>
        + PayloadValidator<OpEngineTypes, Block = OpBlock>
        + Clone,
    OpEngineApi<N::Provider, OpEngineTypes, N::Pool, Validator<N>, OpChainSpec>:
        IntoEngineApiRpcModule + Send + Sync + 'static,
{
    type EngineApi = OpEngineApi<N::Provider, OpEngineTypes, N::Pool, Validator<N>, OpChainSpec>;

    async fn build_engine_api(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::EngineApi> {
        let validator = OpEngineValidatorBuilder::default().build(ctx).await?;
        let inner = EngineApi::new(
            ctx.node.provider().clone(),
            ctx.config.chain.clone(),
            ctx.beacon_engine_handle.clone(),
            PayloadStore::new(ctx.node.payload_builder_handle().clone()),
            ctx.node.pool().clone(),
            ctx.node.task_executor().clone(),
            ClientVersionV1 {
                code: ClientCode::RH,
                name: "op-reth".to_string(),
                version: String::new(),
                commit: String::new(),
            },
            EngineCapabilities::new(OP_ENGINE_CAPABILITIES.iter().copied()),
            validator.clone(),
            ctx.config.engine.accept_execution_requests_hash,
            ctx.node.network().clone(),
        );
        let payloads = DerivedPayloads::new(
            self.0,
            ctx.node.payload_builder_handle().clone(),
            ctx.node.task_executor().clone(),
        );
        Ok(OpEngineApi::new(inner, ctx.node.provider().clone(), validator, payloads))
    }
}

/// Derived attributes on an Ecotone chain, whose payload the payload service can build.
fn ecotone_derived() -> eyre::Result<(Arc<OpChainSpec>, OpPayloadAttrs)> {
    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().ecotone_activated().build());
    let Txs { deposit, transfer, .. } = Txs::new(chain_spec.chain_id())?;
    let mut attrs = optimism_payload_attributes(chain_spec.genesis_header().timestamp + 2);
    attrs.0.no_tx_pool = Some(true);
    attrs.0.transactions = Some(vec![deposit, transfer]);
    Ok((chain_spec, attrs))
}

/// `debug_executePayload` gives the fault proof program the witness of a derived block up to its
/// invalid transaction, where the program stops executing it.
#[tokio::test(flavor = "multi_thread")]
async fn witness_covers_a_derived_block_up_to_its_invalid_transaction() -> eyre::Result<()> {
    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().ecotone_activated().build());
    let Txs { deposit, nonce_too_high, .. } = Txs::new(chain_spec.chain_id())?;
    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;

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

/// Derived attributes that reth rejects, or that come with an unknown head, get reth's own
/// response, the one it gives the same attributes without `noTxPool`.
#[tokio::test(flavor = "multi_thread")]
async fn derived_attributes_get_reths_response_when_not_built() -> eyre::Result<()> {
    let (chain_spec, attrs) = ecotone_derived()?;
    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let client = node.auth_server_handle().http_client();
    let genesis = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    let mut not_after_parent = attrs.clone();
    not_after_parent.0.payload_attributes.timestamp = chain_spec.genesis_header().timestamp;
    let mut no_beacon_root = attrs.clone();
    no_beacon_root.0.payload_attributes.parent_beacon_block_root = None;
    let unknown_head = ForkchoiceState::same_hash(B256::repeat_byte(0xab));
    let cases = [
        ("timestamp not after the parent", genesis, not_after_parent),
        ("no parent beacon block root", genesis, no_beacon_root),
        ("unknown head", unknown_head, attrs),
    ];

    for (case, state, derived) in cases {
        assert!(builds_at_forkchoice_updated(&derived), "{case}");
        let mut sequenced = derived.clone();
        sequenced.0.no_tx_pool = None;
        let expected = format!("{:?}", fcu(&client, state, &sequenced).await);
        let actual = format!("{:?}", fcu(&client, state, &derived).await);
        assert_eq!(actual, expected, "{case}");
    }
    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn node_local_build_failures_fall_back_to_the_payload_service() -> eyre::Result<()> {
    let (chain_spec, attrs) = ecotone_derived()?;
    assert!(builds_at_forkchoice_updated(&attrs));
    let head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    for failure in [Failure::MissingState, Failure::Panic] {
        let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
        config.network.discovery.discv5_port = Some(0);
        config.network.discovery.discv5_port_ipv6 = Some(0);
        let op_node = OpNode::default();
        let node = NodeBuilder::new(config)
            .testing_node(Runtime::test())
            .with_types::<OpNode>()
            .with_components(op_node.components())
            .with_add_ons(op_node.add_ons().with_engine_api(FailingEngineApi(failure)))
            .launch()
            .await?
            .node;
        let client = node.auth_server_handle().http_client();

        let updated = fcu(&client, head, &attrs).await?;
        assert!(updated.payload_status.is_valid(), "{updated:?}");
        let id = updated.payload_id.expect("the payload service builds the attributes");
        assert!(node.payload_builder_handle.payload_timestamp(id).await.is_some());

        let payload: OpExecutionPayloadEnvelopeV3 =
            client.request("engine_getPayloadV3", rpc_params![id]).await?;
        assert_eq!(
            Some(&payload.execution_payload.payload_inner.payload_inner.transactions),
            attrs.0.transactions.as_ref(),
        );
        let status: PayloadStatus = client
            .request(
                "engine_newPayloadV3",
                rpc_params![
                    payload.execution_payload,
                    Vec::<B256>::new(),
                    payload.parent_beacon_block_root
                ],
            )
            .await?;
        assert!(status.is_valid(), "{status:?}");
    }
    Ok(())
}

/// A forkchoice update waits until a derived build and its fallback are done, so the fallback
/// cannot overwrite it.
#[tokio::test(flavor = "multi_thread")]
async fn forkchoice_updates_wait_for_a_derived_build() -> eyre::Result<()> {
    let (chain_spec, attrs) = ecotone_derived()?;
    let (entered_tx, entered) = mpsc::channel();
    let (release, release_rx) = mpsc::channel();
    let failure = Failure::Held { entered: entered_tx, release: Arc::new(Mutex::new(release_rx)) };

    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons().with_engine_api(FailingEngineApi(failure)))
        .launch()
        .await?
        .node;
    let client = node.auth_server_handle().http_client();
    let head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    let derived = tokio::spawn({
        let client = client.clone();
        async move { fcu(&client, head, &attrs).await }
    });
    tokio::task::spawn_blocking(move || entered.recv()).await??;
    let mut plain = tokio::spawn({
        let client = client.clone();
        async move {
            client
                .request::<ForkchoiceUpdated, _>("engine_forkchoiceUpdatedV3", rpc_params![head])
                .await
        }
    });
    assert!(
        tokio::time::timeout(Duration::from_millis(500), &mut plain).await.is_err(),
        "a forkchoice update must not run while a derived build is in progress",
    );

    release.send(())?;
    let updated = derived.await??;
    assert!(plain.await??.payload_status.is_valid());
    // Resolving the payload job lets its build finish before the node shuts down.
    let id = updated.payload_id.expect("the payload service builds the attributes");
    let _: OpExecutionPayloadEnvelopeV3 =
        client.request("engine_getPayloadV3", rpc_params![id]).await?;
    Ok(())
}
