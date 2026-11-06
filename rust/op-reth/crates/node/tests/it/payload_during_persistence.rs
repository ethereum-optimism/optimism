//! A payload job must finish within the block time while a persistence batch is in flight, and
//! keep building after the batch lands. The batch is parked inside the write transaction the
//! persistence service opens before it writes anything, so the test controls when it lands.

use crate::engine_driver::{EngineDriver, StartedPayloadJob, wait_until, within_timeout};
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use jsonrpsee::{core::client::ClientT, rpc_params};
use reth_chainspec::EthChainSpec;
use reth_db::test_utils::{create_test_rw_db_with_datadir, tempdir_path};
use reth_db_api::{Database, DatabaseError, database_metrics::DatabaseMetrics};
use reth_e2e_test_utils::{transaction::TransactionTestContext, wallet::Wallet};
use reth_node_builder::{Node, NodeBuilder, NodeConfig};
use reth_node_core::args::{DatadirArgs, RpcServerArgs};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpEngineTypes, OpNode};
use reth_payload_builder::PayloadBuilderHandle;
use reth_provider::{BlockNumReader, ChainSpecProvider, DatabaseProviderFactory};
use reth_tasks::Runtime;
use rstest::rstest;
use std::{
    path::PathBuf,
    sync::{Arc, Condvar, Mutex},
    time::Duration,
};

/// The block time the payload job is held to, OP Mainnet's.
const BLOCK_TIME: Duration = Duration::from_secs(2);
/// How often a payload job rebuilds, so a transaction sent to the pool mid-job lands in the
/// job's payload well inside the block time.
const BUILDER_INTERVAL: Duration = Duration::from_millis(50);

/// Parks database write transactions while closed. The persistence service opens one at the
/// start of every batch, so a closed gate keeps a batch in flight, inside its transaction with
/// nothing committed, until the gate opens.
#[derive(Debug, Default)]
struct WriteGate {
    opened: Condvar,
    state: Mutex<WriteGateState>,
}

#[derive(Debug, Default)]
struct WriteGateState {
    closed: bool,
    waiting_writers: usize,
}

impl WriteGate {
    /// Closes the gate until the returned guard is dropped, which also happens when an assertion
    /// fails, so the persistence service never stays parked through the node's shutdown.
    fn close(self: &Arc<Self>) -> OpenOnDrop {
        self.state.lock().unwrap().closed = true;
        OpenOnDrop(Arc::clone(self))
    }

    fn open(&self) {
        self.state.lock().unwrap().closed = false;
        self.opened.notify_all();
    }

    fn waiting_writers(&self) -> usize {
        self.state.lock().unwrap().waiting_writers
    }

    fn wait_while_closed(&self) {
        let mut state = self.state.lock().unwrap();
        state.waiting_writers += 1;
        let mut state = self.opened.wait_while(state, |state| state.closed).unwrap();
        state.waiting_writers -= 1;
    }
}

struct OpenOnDrop(Arc<WriteGate>);

impl Drop for OpenOnDrop {
    fn drop(&mut self) {
        self.0.open();
    }
}

/// The node's database with every write transaction parked at the gate right after it opens.
#[derive(Clone, Debug)]
struct WriteGatedDatabase<DB> {
    gate: Arc<WriteGate>,
    inner: DB,
}

impl<DB: Database> Database for WriteGatedDatabase<DB> {
    type TX = DB::TX;
    type TXMut = DB::TXMut;

    fn tx(&self) -> Result<Self::TX, DatabaseError> {
        self.inner.tx()
    }

    fn tx_mut(&self) -> Result<Self::TXMut, DatabaseError> {
        let tx = self.inner.tx_mut()?;
        self.gate.wait_while_closed();
        Ok(tx)
    }

    fn path(&self) -> PathBuf {
        self.inner.path()
    }

    fn oldest_reader_txnid(&self) -> Option<u64> {
        self.inner.oldest_reader_txnid()
    }

    fn last_txnid(&self) -> Option<u64> {
        self.inner.last_txnid()
    }
}

impl<DB> DatabaseMetrics for WriteGatedDatabase<DB> {}

/// Waits for the job's best payload to hold the block's L1 info transaction and the pool
/// transaction.
async fn wait_for_pool_transaction_built(
    builder: &PayloadBuilderHandle<OpEngineTypes>,
    job: &StartedPayloadJob,
) {
    within_timeout("the job never rebuilt with the pool transaction", async {
        loop {
            if let Some(Ok(best)) = builder.best_payload(job.payload_id).await &&
                best.block().body().transactions.len() == 2
            {
                return;
            }
            tokio::time::sleep(Duration::from_millis(10)).await;
        }
    })
    .await;
}

#[rstest]
#[case::reth_2_5_default(7)]
#[case::reth_2_6_default(50)]
fn payload_job_completes_while_persistence_batch_is_in_flight(#[case] persistence_threshold: u64) {
    super::crash_backtrace::install();
    tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .unwrap()
        .block_on(run_scenario(persistence_threshold))
        .unwrap();
}

/// Builds one block more than the threshold, which makes reth persist them all as one batch,
/// and parks that batch inside its write transaction. A payload job started then must resolve
/// within the block time with the batch still parked. A second job is started with the batch
/// parked and the batch is released while the job runs: the job must still resolve within the
/// block time, and the payload it resolves to, rebuilt with a pool transaction after the batch
/// landed, must be valid, which checks the state the builder read across the batch's landing.
async fn run_scenario(persistence_threshold: u64) -> eyre::Result<()> {
    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
    let mut config = NodeConfig::test()
        .map_chain(chain_spec)
        .with_unused_ports()
        .with_datadir_args(DatadirArgs { datadir: tempdir_path().into(), ..Default::default() })
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    config.engine.persistence_threshold = persistence_threshold;
    // Nothing stays in memory after a batch, so the batch is the whole chain built so far.
    config.engine.memory_block_buffer_target = Some(0);
    // Every block of the batch reaches the database, so the disk height tracks the batch.
    config.engine.num_state_masking_blocks = 0;
    config.builder.interval = BUILDER_INTERVAL;

    let gate = Arc::new(WriteGate::default());
    let database = WriteGatedDatabase {
        gate: gate.clone(),
        inner: create_test_rw_db_with_datadir(config.datadir().data_dir()),
    };
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .with_database(database)
        .with_launch_context(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let disk_height = || node.provider.database_provider_ro().unwrap().last_block_number().unwrap();
    // The height the in-memory state has handed to the database and dropped from its overlay.
    let overlay_persisted_height = || {
        node.provider.canonical_in_memory_state().get_persisted_num_hash().map_or(0, |b| b.number)
    };
    let rpc_client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();
    let mut driver = EngineDriver {
        client: node.auth_server_handle().http_client(),
        engine: node.add_ons_handle.beacon_engine_handle.clone(),
        head: node.provider.chain_spec().genesis_hash(),
        wallet,
    };

    for number in 1..=persistence_threshold {
        driver.build_block(number).await?;
    }
    assert_eq!(disk_height(), 0, "no batch is due before the threshold is exceeded");

    let open_on_drop = gate.close();
    let batch_tip = persistence_threshold + 1;
    driver.build_block(batch_tip).await?;
    wait_until("the batch never reached the database", || gate.waiting_writers() > 0).await;

    let job_duration = driver.build_block(batch_tip + 1).await?;
    assert!(
        job_duration <= BLOCK_TIME,
        "the payload job took {job_duration:?} with a {batch_tip}-block persistence batch parked, more than the block time {BLOCK_TIME:?}"
    );
    assert_eq!(disk_height(), 0, "the job resolved with nothing of the batch committed");

    let job = driver.start_payload_job(batch_tip + 2).await?;
    drop(open_on_drop);
    wait_until("the released batch never reached the disk", || disk_height() == batch_tip).await;
    let pool_sender = driver.spare_signer();
    let pool_transaction =
        TransactionTestContext::transfer_tx_bytes(driver.wallet.chain_id, pool_sender).await;
    let _: B256 =
        rpc_client.request("eth_sendRawTransaction", rpc_params![pool_transaction]).await?;
    wait_for_pool_transaction_built(&node.payload_builder_handle, &job).await;
    let job_duration = driver.finish_payload_job(job).await?;
    assert!(
        job_duration <= BLOCK_TIME,
        "the payload job took {job_duration:?} across the landing of a {batch_tip}-block persistence batch, more than the block time {BLOCK_TIME:?}"
    );
    wait_until("the landed batch never left the in-memory state", || {
        overlay_persisted_height() == batch_tip
    })
    .await;

    driver.build_block(batch_tip + 3).await?;
    Ok(())
}
