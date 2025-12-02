//! Contains the main Supervisor service runner.

use alloy_primitives::ChainId;
use alloy_provider::{RootProvider, network::Ethereum};
use alloy_rpc_client::RpcClient;
use anyhow::Result;
use futures::future;
use jsonrpsee::client_transport::ws::Url;
use kona_supervisor_core::{
    ChainProcessor, CrossSafetyCheckerJob, LogIndexer, ReorgHandler, Supervisor,
    config::Config,
    event::ChainEvent,
    l1_watcher::L1Watcher,
    rpc::{AdminError, AdminRequest, AdminRpc, SupervisorRpc},
    safety_checker::{CrossSafePromoter, CrossUnsafePromoter},
    syncnode::{Client, ClientConfig, ManagedNode, ManagedNodeClient, ManagedNodeCommand},
};
use kona_supervisor_rpc::{SupervisorAdminApiServer, SupervisorApiServer};
use kona_supervisor_storage::{ChainDb, ChainDbFactory, DerivationStorageWriter, LogStorageWriter};
use std::{collections::HashMap, sync::Arc};
use tokio::{sync::mpsc, task::JoinSet, time::Duration};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

use crate::actors::{
    ChainProcessorActor, ManagedNodeActor, MetricWorker, SupervisorActor, SupervisorRpcActor,
};

// simplify long type signature
type ManagedLogIndexer = LogIndexer<ManagedNode<ChainDb, Client>, ChainDb>;

/// The main service structure for the Kona
/// [`SupervisorService`](`kona_supervisor_core::SupervisorService`). Orchestrates the various
/// components of the supervisor.
#[derive(Debug)]
pub struct Service {
    config: Arc<Config>,

    supervisor: Arc<Supervisor<ManagedNode<ChainDb, Client>>>,
    database_factory: Arc<ChainDbFactory>,
    managed_nodes: HashMap<ChainId, Arc<ManagedNode<ChainDb, Client>>>,
    log_indexers: HashMap<ChainId, Arc<ManagedLogIndexer>>,

    // channels
    chain_event_senders: HashMap<ChainId, mpsc::Sender<ChainEvent>>,
    chain_event_receivers: HashMap<ChainId, mpsc::Receiver<ChainEvent>>,
    managed_node_senders: HashMap<ChainId, mpsc::Sender<ManagedNodeCommand>>,
    managed_node_receivers: HashMap<ChainId, mpsc::Receiver<ManagedNodeCommand>>,
    admin_receiver: Option<mpsc::Receiver<AdminRequest>>,

    cancel_token: CancellationToken,
    join_set: JoinSet<Result<(), anyhow::Error>>,
}

impl Service {
    /// Creates a new Supervisor service instance.
    pub fn new(cfg: Config) -> Self {
        let config = Arc::new(cfg);
        let database_factory = Arc::new(ChainDbFactory::new(config.datadir.clone()).with_metrics());
        let supervisor = Arc::new(Supervisor::new(config.clone(), database_factory.clone()));

        Self {
            config,

            supervisor,
            database_factory,
            managed_nodes: HashMap::new(),
            log_indexers: HashMap::new(),

            chain_event_senders: HashMap::new(),
            chain_event_receivers: HashMap::new(),
            managed_node_senders: HashMap::new(),
            managed_node_receivers: HashMap::new(),
            admin_receiver: None,

            cancel_token: CancellationToken::new(),
            join_set: JoinSet::new(),
        }
    }

    /// Initialises the Supervisor service.
    pub async fn initialise(&mut self) -> Result<()> {
        // create sender and receiver channels for each chain
        for chain_id in self.config.rollup_config_set.rollups.keys() {
            let (chain_tx, chain_rx) = mpsc::channel::<ChainEvent>(1000);
            self.chain_event_senders.insert(*chain_id, chain_tx);
            self.chain_event_receivers.insert(*chain_id, chain_rx);

            let (managed_node_tx, managed_node_rx) = mpsc::channel::<ManagedNodeCommand>(1000);
            self.managed_node_senders.insert(*chain_id, managed_node_tx);
            self.managed_node_receivers.insert(*chain_id, managed_node_rx);
        }

        self.init_database().await?;
        self.init_chain_processor().await?;
        self.init_managed_nodes().await?;
        self.init_l1_watcher()?;
        self.init_cross_safety_checker().await?;

        // todo: run metric worker only if metrics are enabled
        self.init_rpc_server().await?;
        self.init_metric_reporter().await;
        Ok(())
    }

    async fn init_database(&self) -> Result<()> {
        info!(target: "supervisor::service", "Initialising databases for all chains...");

        for (chain_id, config) in self.config.rollup_config_set.rollups.iter() {
            // Initialise the database for each chain.
            let db = self.database_factory.get_or_create_db(*chain_id)?;
            let interop_time = config.interop_time;
            let derived_pair = config.genesis.get_derived_pair();
            if config.is_interop(derived_pair.derived.timestamp) {
                info!(target: "supervisor::service", chain_id, interop_time, %derived_pair, "Initialising database for interop activation block");
                db.initialise_log_storage(derived_pair.derived)?;
                db.initialise_derivation_storage(derived_pair)?;
            }
            info!(target: "supervisor::service", chain_id, "Database initialized successfully");
        }
        Ok(())
    }

    async fn init_managed_node(&mut self, config: &ClientConfig) -> Result<()> {
        info!(target: "supervisor::service", node = %config.url, "Initialising managed node...");
        let url = Url::parse(&self.config.l1_rpc).map_err(|err| {
            error!(target: "supervisor::service", %err, "Failed to parse L1 RPC URL");
            anyhow::anyhow!("failed to parse L1 RPC URL: {err}")
        })?;
        let provider = RootProvider::<Ethereum>::new_http(url);
        let client = Arc::new(Client::new(config.clone()));

        let chain_id = client.chain_id().await.map_err(|err| {
            error!(target: "supervisor::service", %err, "Failed to get chain ID from client");
            anyhow::anyhow!("failed to get chain ID from client: {err}")
        })?;

        let db = self.database_factory.get_db(chain_id)?;

        let chain_event_sender = self
            .chain_event_senders
            .get(&chain_id)
            .ok_or(anyhow::anyhow!("no chain event sender found for chain {chain_id}"))?
            .clone();

        let managed_node =
            ManagedNode::<ChainDb, Client>::new(client.clone(), db, provider, chain_event_sender);

        if self.managed_nodes.contains_key(&chain_id) {
            warn!(target: "supervisor::service", %chain_id, "Managed node for chain already exists, skipping initialization");
            return Ok(());
        }

        let managed_node = Arc::new(managed_node);
        // add the managed node to the supervisor service
        // also checks if the chain ID is supported
        self.supervisor.add_managed_node(chain_id, managed_node.clone()).await?;

        // set the managed node in the log indexer
        let log_indexer = self
            .log_indexers
            .get(&chain_id)
            .ok_or(anyhow::anyhow!("no log indexer found for chain {chain_id}"))?
            .clone();
        log_indexer.set_block_provider(managed_node.clone()).await;

        self.managed_nodes.insert(chain_id, managed_node.clone());
        info!(target: "supervisor::service",
             chain_id,
            "Managed node for chain initialized successfully",
        );

        // start managed node actor
        let managed_node_receiver = self
            .managed_node_receivers
            .remove(&chain_id)
            .ok_or(anyhow::anyhow!("no managed node receiver found for chain {chain_id}"))?;

        let cancel_token = self.cancel_token.clone();
        self.join_set.spawn(async move {
            if let Err(err) =
                ManagedNodeActor::new(client, managed_node, managed_node_receiver, cancel_token)
                    .start()
                    .await
            {
                Err(anyhow::anyhow!(err))
            } else {
                Ok(())
            }
        });
        Ok(())
    }

    async fn init_managed_nodes(&mut self) -> Result<()> {
        let configs = self.config.l2_consensus_nodes_config.clone();
        for config in configs.iter() {
            self.init_managed_node(config).await?;
        }
        Ok(())
    }

    async fn init_chain_processor(&mut self) -> Result<()> {
        info!(target: "supervisor::service", "Initialising chain processors for all chains...");

        for (chain_id, _) in self.config.rollup_config_set.rollups.iter() {
            let db = self.database_factory.get_db(*chain_id)?;

            let managed_node_sender = self
                .managed_node_senders
                .get(chain_id)
                .ok_or(anyhow::anyhow!("no managed node sender found for chain {chain_id}"))?
                .clone();

            let log_indexer = Arc::new(LogIndexer::new(*chain_id, None, db.clone()));
            self.log_indexers.insert(*chain_id, log_indexer.clone());

            // initialise chain processor for the chain.
            let mut processor = ChainProcessor::new(
                self.config.clone(),
                *chain_id,
                log_indexer,
                db,
                managed_node_sender,
            );

            // todo: enable metrics only if configured
            processor = processor.with_metrics();

            // Start the chain processor actor.
            let chain_event_receiver = self
                .chain_event_receivers
                .remove(chain_id)
                .ok_or(anyhow::anyhow!("no chain event receiver found for chain {chain_id}"))?;

            let cancel_token = self.cancel_token.clone();
            self.join_set.spawn(async move {
                if let Err(err) =
                    ChainProcessorActor::new(processor, cancel_token, chain_event_receiver)
                        .start()
                        .await
                {
                    Err(anyhow::anyhow!(err))
                } else {
                    Ok(())
                }
            });
        }
        Ok(())
    }

    fn init_l1_watcher(&mut self) -> Result<()> {
        info!(target: "supervisor::service", "Initialising L1 watcher...");

        let l1_rpc_url = Url::parse(&self.config.l1_rpc).map_err(|err| {
            error!(target: "supervisor::service", %err, "Failed to parse L1 RPC URL");
            anyhow::anyhow!("failed to parse L1 RPC URL: {err}")
        })?;
        let l1_rpc = RpcClient::new_http(l1_rpc_url);

        let chain_dbs_map: HashMap<ChainId, Arc<ChainDb>> = self
            .config
            .rollup_config_set
            .rollups
            .keys()
            .map(|chain_id| {
                self.database_factory.get_db(*chain_id)
                    .map(|db| (*chain_id, db)) // <-- FIX: remove Arc::new(db)
                    .map_err(|err| {
                        error!(target: "supervisor::service", %err, "Failed to get database for chain {chain_id}");
                        anyhow::anyhow!("failed to get database for chain {chain_id}: {err}")
                })
            })
            .collect::<Result<HashMap<ChainId, Arc<ChainDb>>>>()?;

        let database_factory = self.database_factory.clone();
        let cancel_token = self.cancel_token.clone();
        let event_senders = self.chain_event_senders.clone();
        self.join_set.spawn(async move {
            let reorg_handler = ReorgHandler::new(l1_rpc.clone(), chain_dbs_map).with_metrics();

            // Start the L1 watcher streaming loop.
            let l1_watcher = L1Watcher::new(
                l1_rpc.clone(),
                database_factory,
                event_senders,
                cancel_token,
                reorg_handler,
            );

            l1_watcher.run().await;
            Ok(())
        });
        Ok(())
    }

    async fn init_cross_safety_checker(&mut self) -> Result<()> {
        info!(target: "supervisor::service", "Initialising cross safety checker...");

        for (&chain_id, config) in &self.config.rollup_config_set.rollups {
            let db = Arc::clone(&self.database_factory);
            let cancel = self.cancel_token.clone();

            let chain_event_sender = self
                .chain_event_senders
                .get(&chain_id)
                .ok_or(anyhow::anyhow!("no chain event sender found for chain {chain_id}"))?
                .clone();

            let cross_safe_job = CrossSafetyCheckerJob::new(
                chain_id,
                db.clone(),
                cancel.clone(),
                Duration::from_secs(config.block_time),
                CrossSafePromoter,
                chain_event_sender.clone(),
                self.config.clone(),
            );

            self.join_set.spawn(async move {
                cross_safe_job.run().await;
                Ok(())
            });

            let cross_unsafe_job = CrossSafetyCheckerJob::new(
                chain_id,
                db,
                cancel,
                Duration::from_secs(config.block_time),
                CrossUnsafePromoter,
                chain_event_sender,
                self.config.clone(),
            );

            self.join_set.spawn(async move {
                cross_unsafe_job.run().await;
                Ok(())
            });
        }
        Ok(())
    }

    async fn init_metric_reporter(&mut self) {
        // Initialize the metric reporter actor.
        let database_factory = self.database_factory.clone();
        let cancel_token = self.cancel_token.clone();
        self.join_set.spawn(async move {
            if let Err(err) =
                MetricWorker::new(Duration::from_secs(30), vec![database_factory], cancel_token)
                    .start()
                    .await
            {
                Err(anyhow::anyhow!(err))
            } else {
                Ok(())
            }
        });
    }

    async fn init_rpc_server(&mut self) -> Result<()> {
        let supervisor_rpc = SupervisorRpc::new(self.supervisor.clone());

        let mut rpc_module = supervisor_rpc.into_rpc();

        if self.config.enable_admin_api {
            info!(target: "supervisor::service", "Enabling Supervisor Admin API");

            let (admin_tx, admin_rx) = mpsc::channel::<AdminRequest>(100);
            let admin_rpc = AdminRpc::new(admin_tx);
            rpc_module
                .merge(admin_rpc.into_rpc())
                .map_err(|err| anyhow::anyhow!("failed to merge Admin RPC module: {err}"))?;
            self.admin_receiver = Some(admin_rx);
        }

        let rpc_addr = self.config.rpc_addr;
        let cancel_token = self.cancel_token.clone();
        self.join_set.spawn(async move {
            if let Err(err) =
                SupervisorRpcActor::new(rpc_addr, rpc_module, cancel_token).start().await
            {
                Err(anyhow::anyhow!(err))
            } else {
                Ok(())
            }
        });
        Ok(())
    }

    async fn handle_admin_request(&mut self, req: AdminRequest) {
        match req {
            AdminRequest::AddL2Rpc { cfg, resp } => {
                let result = match self.init_managed_node(&cfg).await {
                    Ok(()) => Ok(()),
                    Err(e) => {
                        tracing::error!(target: "supervisor::service", %e, "admin add_l2_rpc failed");
                        Err(AdminError::ServiceError(e.to_string()))
                    }
                };

                let _ = resp.send(result);
            }
        }
    }

    /// Runs the Supervisor service.
    /// This function will typically run indefinitely until interrupted.
    pub async fn run(&mut self) -> Result<()> {
        self.initialise().await?;

        // todo: refactor this to only run the tasks completion loop
        // and handle admin requests elsewhere
        loop {
            tokio::select! {
                // Admin requests (if admin_receiver was initialized)
                maybe_req = async {
                    if let Some(rx) = self.admin_receiver.as_mut() {
                        rx.recv().await
                    } else {
                        // if no receiver present, never produce a value
                        future::pending::<Option<AdminRequest>>().await
                    }
                } => {
                    if let Some(req) = maybe_req {
                        self.handle_admin_request(req).await;
                    }
                }

                // Supervisor task completions / failures
                opt = self.join_set.join_next() => {
                    match opt {
                        Some(Ok(Ok(_))) => {
                            info!(target: "supervisor::service", "Task completed successfully.");
                        }
                        Some(Ok(Err(err))) => {
                            error!(target: "supervisor::service", %err, "A task encountered an error.");
                            self.cancel_token.cancel();
                            return Err(anyhow::anyhow!("A service task failed: {err}"));
                        }
                        Some(Err(err)) => {
                            error!(target: "supervisor::service", %err, "A task encountered an error.");
                            self.cancel_token.cancel();
                            return Err(anyhow::anyhow!("A service task failed: {err}"));
                        }
                        None => break, // all tasks finished
                    }
                }
            }
        }
        Ok(())
    }

    pub async fn shutdown(mut self) -> Result<()> {
        self.cancel_token.cancel(); // Signal cancellation to all tasks

        // Wait for all tasks to finish.
        while let Some(res) = self.join_set.join_next().await {
            match res {
                Ok(Ok(_)) => {
                    info!(target: "supervisor::service", "Task completed successfully during shutdown.");
                }
                Ok(Err(err)) => {
                    error!(target: "supervisor::service", %err, "A task encountered an error during shutdown.");
                }
                Err(err) => {
                    error!(target: "supervisor::service", %err, "A task encountered an error during shutdown.");
                }
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use std::{net::SocketAddr, path::PathBuf};

    use kona_interop::DependencySet;
    use kona_supervisor_core::config::RollupConfigSet;

    use super::*;

    fn make_test_config(enable_admin: bool) -> Config {
        let mut cfg = Config::new(
            "http://localhost:8545".to_string(),
            vec![],
            PathBuf::from("/tmp/kona-supervisor"),
            SocketAddr::from(([127, 0, 0, 1], 8545)),
            false,
            DependencySet {
                dependencies: Default::default(),
                override_message_expiry_window: None,
            },
            RollupConfigSet { rollups: HashMap::new() },
        );
        cfg.enable_admin_api = enable_admin;
        cfg
    }

    #[tokio::test]
    async fn test_init_rpc_server_enables_admin_receiver_when_flag_set() {
        let cfg = Arc::new(make_test_config(true));
        let mut svc = Service::new((*cfg).clone());

        svc.config = cfg.clone();
        svc.init_rpc_server().await.expect("init_rpc_server failed");
        assert!(svc.admin_receiver.is_some(), "admin_receiver must be set when admin enabled");
    }
}
