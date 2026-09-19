//! Real OP payload jobs must not stop Engine API traffic while persistence hands off its overlay.

use alloy_genesis::Genesis;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceState, PayloadId, PayloadStatus};
use jsonrpsee::{core::client::ClientT, rpc_params};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
use reth_basic_payload_builder::{
    BuildArguments, BuildOutcome, HeaderForPayload, MissingPayloadBehaviour, PayloadBuilder,
    PayloadConfig,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{transaction::TransactionTestContext, wallet::Wallet};
use reth_node_api::PayloadAttributes;
use reth_node_builder::{
    BuilderContext, FullNodeTypes, Node, NodeBuilder, NodeConfig,
    components::{BasicPayloadServiceBuilder, PayloadBuilderBuilder},
};
use reth_node_metrics::recorder::install_prometheus_recorder;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_payload_builder::{PayloadBuilderError, PayloadKind};
use reth_provider::{BlockNumReader, ChainSpecProvider, DatabaseProviderFactory};
use reth_tasks::Runtime;
use reth_transaction_pool::TransactionPool;
use std::{future::Future, process::Command, sync::Arc, time::Duration};
use tokio::sync::{mpsc, oneshot};

const GUARD: Duration = Duration::from_secs(20);
const ABANDONED_TIMESTAMP: u64 = 5;

/// Dropping this also releases the worker, including when an assertion fails.
#[derive(Debug)]
struct BuildGate {
    id: PayloadId,
    release: oneshot::Sender<()>,
    finished: oneshot::Receiver<bool>,
}

impl BuildGate {
    async fn release(self) {
        let _ = self.release.send(());
        assert!(guard("cancelled worker did not finish", self.finished).await.unwrap());
    }
}

/// Decorates the real builder, without replacing any build result or lease management.
#[derive(Clone, Debug)]
struct GatedBuilder<B> {
    inner: B,
    entered: mpsc::UnboundedSender<BuildGate>,
}

impl<B: PayloadBuilder> PayloadBuilder for GatedBuilder<B> {
    type Attributes = B::Attributes;
    type BuiltPayload = B::BuiltPayload;

    fn try_build(
        &self,
        args: BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> Result<BuildOutcome<Self::BuiltPayload>, PayloadBuilderError> {
        // Let the first attempt produce a real best payload. Hold its replacement attempt so
        // normal getPayload can return that best payload while the detached worker still owns
        // the BasicPayloadJob lease. The abandoned job instead stalls on its first attempt.
        let finished = (args.best_payload.is_some() ||
            args.config.attributes.timestamp() == ABANDONED_TIMESTAMP)
            .then(|| {
                let (release, wait) = oneshot::channel();
                let (finished, done) = oneshot::channel();
                let gate = BuildGate { id: args.config.payload_id(), release, finished: done };
                if self.entered.send(gate).is_ok() {
                    let _ = wait.blocking_recv();
                }
                finished
            });
        let cancelled = args.cancel.is_cancelled();
        let result = self.inner.try_build(args);
        if let Some(finished) = finished {
            let _ = finished.send(cancelled);
        }
        result
    }

    fn on_missing_payload(
        &self,
        args: BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> MissingPayloadBehaviour<Self::BuiltPayload> {
        self.inner.on_missing_payload(args)
    }

    fn build_empty_payload(
        &self,
        config: PayloadConfig<Self::Attributes, HeaderForPayload<Self::BuiltPayload>>,
    ) -> Result<Self::BuiltPayload, PayloadBuilderError> {
        self.inner.build_empty_payload(config)
    }
}

struct GatedBuilderFactory<B> {
    inner: B,
    entered: mpsc::UnboundedSender<BuildGate>,
}

impl<N, Pool, Evm, B> PayloadBuilderBuilder<N, Pool, Evm> for GatedBuilderFactory<B>
where
    N: FullNodeTypes,
    Pool: TransactionPool,
    Evm: Send,
    B: PayloadBuilderBuilder<N, Pool, Evm>,
{
    type PayloadBuilder = GatedBuilder<B::PayloadBuilder>;

    async fn build_payload_builder(
        self,
        ctx: &BuilderContext<N>,
        pool: Pool,
        evm: Evm,
    ) -> eyre::Result<Self::PayloadBuilder> {
        Ok(GatedBuilder {
            inner: self.inner.build_payload_builder(ctx, pool, evm).await?,
            entered: self.entered,
        })
    }
}

async fn guard<T>(message: &str, future: impl Future<Output = T>) -> T {
    tokio::time::timeout(GUARD, future).await.unwrap_or_else(|_| panic!("{message}"))
}

async fn wait_until(message: &str, mut ready: impl FnMut() -> bool) {
    guard(message, async {
        while !ready() {
            tokio::task::yield_now().await;
        }
    })
    .await;
}

fn persistence_completions() -> u64 {
    // This histogram is recorded by EngineApiTreeHandler::finish_persistence, not the disk
    // writer. Once observed, the engine must pass through its active-build handoff branch
    // before it can receive another request. Disk height alone cannot establish that ordering.
    install_prometheus_recorder()
        .handle()
        .render()
        .lines()
        .find_map(|line| {
            line.strip_prefix("reth_consensus_engine_beacon_persistence_duration_count ")
                .map(|count| count.parse().unwrap())
        })
        .unwrap_or_default()
}

#[test]
fn pending_handoff_keeps_engine_responsive_and_drains() {
    // Metrics handles are cached globally upstream. A fresh process makes the rendezvous
    // specific to this node, even under cargo test with other node tests running concurrently.
    const CHILD: &str = "OP_RETH_HANDOFF_TEST_CHILD";
    if std::env::var_os(CHILD).is_none() {
        let output = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "payload_handoff::pending_handoff_keeps_engine_responsive_and_drains",
                "--nocapture",
            ])
            .env(CHILD, "1")
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "handoff regression failed: {}\n{}\n{}",
            output.status,
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr),
        );
        return;
    }

    super::crash_backtrace::install();
    // Also bound startup/teardown: a blocked OS worker can outlive Tokio's async timeout.
    let (_watchdog, watchdog_rx) = std::sync::mpsc::channel::<()>();
    std::thread::spawn(move || {
        if matches!(
            watchdog_rx.recv_timeout(Duration::from_secs(180)),
            Err(std::sync::mpsc::RecvTimeoutError::Timeout)
        ) {
            eprintln!("handoff regression exceeded its process hang guard");
            std::process::exit(1);
        }
    });
    tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .unwrap()
        .block_on(run_handoff_scenario())
        .unwrap();
}

async fn run_handoff_scenario() -> eyre::Result<()> {
    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
    let mut config = NodeConfig::test().map_chain(chain_spec).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    config.engine.persistence_threshold = 0;
    config.engine.memory_block_buffer_target = Some(0);
    config.engine.num_state_masking_blocks = 0;
    config.engine.suppress_persistence_during_build = false;
    config.builder.interval = Duration::from_millis(1);
    config.builder.deadline = Duration::from_secs(300);
    // Four overlapping replacement attempts and one abandoned first attempt.
    config.builder.max_payload_tasks = 8;

    let (entered, mut gates) = mpsc::unbounded_channel();
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components().payload(BasicPayloadServiceBuilder::new(
            GatedBuilderFactory { inner: op_node.payload_builder(), entered },
        )))
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let engine = &node.add_ons_handle.beacon_engine_handle;
    let client = node.auth_server_handle().http_client();
    let memory = node.provider.canonical_in_memory_state();
    let disk_height = || node.provider.database_provider_ro().unwrap().last_block_number().unwrap();
    let handed_off_height = || memory.get_persisted_num_hash().map_or(0, |block| block.number);
    let mut head = node.provider.chain_spec().genesis_hash();
    let mut held = Vec::new();

    for number in 1..=4 {
        let mut attrs = optimism_payload_attributes(number);
        attrs.0.transactions = Some(vec![
            TransactionTestContext::optimism_l1_block_info_tx(
                wallet.chain_id,
                wallet.inner.clone(),
                number - 1,
            )
            .await,
        ]);
        let fcu = guard(
            "FCU blocked behind a payload worker's persisted handoff",
            engine.fork_choice_updated(ForkchoiceState::same_hash(head), Some(attrs)),
        )
        .await?;
        assert!(fcu.payload_status.is_valid());
        let id = fcu.payload_id.unwrap();
        let gate = guard("replacement build did not enter its gate", gates.recv()).await.unwrap();
        assert_eq!(gate.id, id);
        held.push(gate);

        // Exercise the normal authenticated getPayload RPC, not a fabricated build outcome.
        let payload: OpExecutionPayloadEnvelopeV3 = guard(
            "getPayload waited for the replacement worker instead of returning the best payload",
            client.request("engine_getPayloadV3", rpc_params![id]),
        )
        .await?;
        assert_eq!(payload.execution_payload.payload_inner.payload_inner.block_number, number);
        head = payload.execution_payload.payload_inner.payload_inner.block_hash;
        let status: PayloadStatus = guard(
            "newPayload blocked behind a payload worker's persisted handoff",
            client.request(
                "engine_newPayloadV3",
                rpc_params![
                    payload.execution_payload,
                    Vec::<B256>::new(),
                    payload.parent_beacon_block_root
                ],
            ),
        )
        .await?;
        assert!(status.is_valid());
        let fcu = guard(
            "canonical FCU blocked behind a payload worker's persisted handoff",
            engine.fork_choice_updated(ForkchoiceState::same_hash(head), None),
        )
        .await?;
        assert!(fcu.payload_status.is_valid());

        if number == 1 {
            wait_until("engine did not observe persistence completion", || {
                persistence_completions() > 0
            })
            .await;
        }
        assert_eq!(disk_height(), 1, "a pending handoff must retain its persistence frontier");
        assert_eq!(handed_off_height(), 0, "live worker leases must protect the old overlay");
    }

    // Abandon a first attempt with no best payload. Dropping resolve_kind's response receiver
    // cancels the real service job; it cannot cancel the detached blocking worker's lease.
    let attrs = optimism_payload_attributes(ABANDONED_TIMESTAMP);
    let id = guard(
        "FCU could not start the abandoned build",
        engine.fork_choice_updated(ForkchoiceState::same_hash(head), Some(attrs)),
    )
    .await?
    .payload_id
    .unwrap();
    let abandoned = guard("abandoned worker did not enter its gate", gates.recv()).await.unwrap();
    assert_eq!(abandoned.id, id);
    drop(node.payload_builder_handle.resolve_kind(id, PayloadKind::Earliest));
    assert!(
        guard(
            "cancelled job remained in the payload service",
            node.payload_builder_handle.payload_timestamp(id)
        )
        .await
        .is_none()
    );
    abandoned.release().await;

    // Drain newer workers first. The oldest detached worker must still protect the handoff.
    while held.len() > 1 {
        held.pop().unwrap().release().await;
    }
    assert_eq!(disk_height(), 1);
    assert_eq!(handed_off_height(), 0);
    held.pop().unwrap().release().await;
    wait_until("finite payload workload never released persistence", || {
        disk_height() == 4 && handed_off_height() == 4
    })
    .await;

    // A new build after the drain must use the caught-up provider and persist normally too.
    let mut attrs = optimism_payload_attributes(6);
    attrs.0.transactions = Some(vec![
        TransactionTestContext::optimism_l1_block_info_tx(wallet.chain_id, wallet.inner.clone(), 4)
            .await,
    ]);
    let id = guard(
        "FCU did not recover after leases drained",
        engine.fork_choice_updated(ForkchoiceState::same_hash(head), Some(attrs)),
    )
    .await?
    .payload_id
    .unwrap();
    let gate = guard("post-handoff worker did not enter its gate", gates.recv()).await.unwrap();
    assert_eq!(gate.id, id);
    let payload: OpExecutionPayloadEnvelopeV3 = guard(
        "post-handoff getPayload stalled",
        client.request("engine_getPayloadV3", rpc_params![id]),
    )
    .await?;
    gate.release().await;
    assert_eq!(payload.execution_payload.payload_inner.payload_inner.block_number, 5);
    head = payload.execution_payload.payload_inner.payload_inner.block_hash;
    let status: PayloadStatus = guard(
        "post-handoff newPayload stalled",
        client.request(
            "engine_newPayloadV3",
            rpc_params![
                payload.execution_payload,
                Vec::<B256>::new(),
                payload.parent_beacon_block_root
            ],
        ),
    )
    .await?;
    assert!(status.is_valid());
    assert!(
        guard(
            "post-handoff canonical FCU stalled",
            engine.fork_choice_updated(ForkchoiceState::same_hash(head), None),
        )
        .await?
        .payload_status
        .is_valid()
    );
    wait_until("post-handoff build never persisted", || {
        disk_height() == 5 && handed_off_height() == 5
    })
    .await;
    Ok(())
}
