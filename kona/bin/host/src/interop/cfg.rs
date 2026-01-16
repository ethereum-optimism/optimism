//! This module contains all CLI-specific code for the interop entrypoint.

use super::{InteropHintHandler, InteropLocalInputs};
use crate::{
    DiskKeyValueStore, MemoryKeyValueStore, OfflineHostBackend, OnlineHostBackend,
    OnlineHostBackendCfg, PreimageServer, SharedKeyValueStore, SplitKeyValueStore,
    eth::rpc_provider, server::PreimageServerError,
};
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Provider, RootProvider};
use clap::Parser;
use kona_cli::cli_styles;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_preimage::{
    BidirectionalChannel, Channel, HintReader, HintWriter, OracleReader, OracleServer,
};
use kona_proof_interop::HintType;
use kona_providers_alloy::{OnlineBeaconClient, OnlineBlobProvider};
use kona_std_fpvm::{FileChannel, FileDescriptor};
use op_alloy_network::Optimism;
use serde::Serialize;
use std::{collections::HashMap, fs, path::PathBuf, str::FromStr, sync::Arc};
use tokio::{
    sync::RwLock,
    task::{self, JoinHandle},
};

/// The interop host application.
#[derive(Default, Parser, Serialize, Clone, Debug)]
#[command(styles = cli_styles())]
pub struct InteropHost {
    /// Hash of the L1 head block, marking a static, trusted cutoff point for reading data from the
    /// L1 chain.
    #[arg(long, env)]
    pub l1_head: B256,
    /// Agreed [PreState] to start from.
    ///
    /// [PreState]: kona_proof_interop::PreState
    #[arg(long, visible_alias = "l2-pre-state", value_parser = Bytes::from_str, env)]
    pub agreed_l2_pre_state: Bytes,
    /// Claimed L2 post-state to validate.
    #[arg(long, visible_alias = "l2-claim", env)]
    pub claimed_l2_post_state: B256,
    /// Claimed L2 timestamp, corresponding to the L2 post-state.
    #[arg(long, visible_alias = "l2-timestamp", env)]
    pub claimed_l2_timestamp: u64,
    /// Addresses of L2 JSON-RPC endpoints to use (eth and debug namespace required).
    #[arg(
        long,
        visible_alias = "l2s",
        requires = "l1_node_address",
        requires = "l1_beacon_address",
        value_delimiter = ',',
        env
    )]
    pub l2_node_addresses: Option<Vec<String>>,
    /// Address of L1 JSON-RPC endpoint to use (eth and debug namespace required)
    #[arg(
        long,
        visible_alias = "l1",
        requires = "l2_node_addresses",
        requires = "l1_beacon_address",
        env
    )]
    pub l1_node_address: Option<String>,
    /// Address of the L1 Beacon API endpoint to use.
    #[arg(
        long,
        visible_alias = "beacon",
        requires = "l1_node_address",
        requires = "l2_node_addresses",
        env
    )]
    pub l1_beacon_address: Option<String>,
    /// The Data Directory for preimage data storage. Optional if running in online mode,
    /// required if running in offline mode.
    #[arg(
        long,
        visible_alias = "db",
        required_unless_present_all = ["l2_node_addresses", "l1_node_address", "l1_beacon_address"],
        env
    )]
    pub data_dir: Option<PathBuf>,
    /// Run the client program natively.
    #[arg(long, conflicts_with = "server", required_unless_present = "server")]
    pub native: bool,
    /// Run in pre-image server mode without executing any client program. If not provided, the
    /// host will run the client program in the host process.
    #[arg(long, conflicts_with = "native", required_unless_present = "native")]
    pub server: bool,
    /// Path to rollup configs. If provided, the host will use this config instead of attempting to
    /// look up the configs in the superchain registry.
    /// The rollup configs should be stored as serde-JSON serialized files.
    #[arg(long, alias = "rollup-cfgs", value_delimiter = ',', env)]
    pub rollup_config_paths: Option<Vec<PathBuf>>,
    /// Path to l1 configs. If provided, the host will use this config instead of attempting to
    /// look up the configs in the superchain registry.
    /// The l1 configs should be stored as serde-JSON serialized files.
    #[arg(long, alias = "l1-cfgs", value_delimiter = ',', env)]
    pub l1_config_paths: Option<Vec<PathBuf>>,
}

/// An error that can occur when handling interop hosts
#[derive(Debug, thiserror::Error)]
pub enum InteropHostError {
    /// An error when handling preimage requests.
    #[error("Error handling preimage request: {0}")]
    PreimageServerError(#[from] PreimageServerError),
    /// An IO error.
    #[error("IO error: {0}")]
    IOError(#[from] std::io::Error),
    /// A JSON parse error.
    #[error("Failed deserializing RollupConfig: {0}")]
    ParseError(#[from] serde_json::Error),
    /// Impossible to find the L1 chain config for the given chain ID.
    #[error("L1 chain config not found for chain ID: {0}")]
    L1ChainConfigNotFound(u64),
    /// Task failed to execute to completion.
    #[error("Join error: {0}")]
    ExecutionError(#[from] tokio::task::JoinError),
    /// A RPC error.
    #[error("Rpc Error: {0}")]
    RpcError(#[from] alloy_transport::RpcError<alloy_transport::TransportErrorKind>),
    /// An error when no provider found for chain ID.
    #[error("No provider found for chain ID: {0}")]
    RootProviderError(u64),
    /// Any other error.
    #[error("Error: {0}")]
    Other(&'static str),
}

impl InteropHost {
    /// Starts the [InteropHost] application.
    pub async fn start(self) -> Result<(), InteropHostError> {
        if self.server {
            let hint = FileChannel::new(FileDescriptor::HintRead, FileDescriptor::HintWrite);
            let preimage =
                FileChannel::new(FileDescriptor::PreimageRead, FileDescriptor::PreimageWrite);

            self.start_server(hint, preimage).await?.await?
        } else {
            self.start_native().await
        }
    }

    /// Starts the preimage server, communicating with the client over the provided channels.
    async fn start_server<C>(
        &self,
        hint: C,
        preimage: C,
    ) -> Result<JoinHandle<Result<(), InteropHostError>>, InteropHostError>
    where
        C: Channel + Send + Sync + 'static,
    {
        let kv_store = self.create_key_value_store()?;

        let task_handle = if self.is_offline() {
            task::spawn(async {
                PreimageServer::new(
                    OracleServer::new(preimage),
                    HintReader::new(hint),
                    Arc::new(OfflineHostBackend::new(kv_store)),
                )
                .start()
                .await
                .map_err(InteropHostError::from)
            })
        } else {
            let providers = self.create_providers().await?;
            let backend = OnlineHostBackend::new(
                self.clone(),
                kv_store.clone(),
                providers,
                InteropHintHandler,
            )
            .with_proactive_hint(HintType::L2BlockData);

            task::spawn(async {
                PreimageServer::new(
                    OracleServer::new(preimage),
                    HintReader::new(hint),
                    Arc::new(backend),
                )
                .start()
                .await
                .map_err(InteropHostError::from)
            })
        };

        Ok(task_handle)
    }

    /// Starts the host in native mode, running both the client and preimage server in the same
    /// process.
    async fn start_native(&self) -> Result<(), InteropHostError> {
        let hint = BidirectionalChannel::new()?;
        let preimage = BidirectionalChannel::new()?;

        let server_task = self.start_server(hint.host, preimage.host).await?;
        let client_task = task::spawn(kona_client::interop::run(
            OracleReader::new(preimage.client),
            HintWriter::new(hint.client),
        ));

        let (_, client_result) = tokio::try_join!(server_task, client_task)?;

        // Bubble up the exit status of the client program if execution completes.
        std::process::exit(client_result.is_err() as i32)
    }

    /// Returns `true` if the host is running in offline mode.
    pub const fn is_offline(&self) -> bool {
        self.l1_node_address.is_none() &&
            self.l2_node_addresses.is_none() &&
            self.l1_beacon_address.is_none() &&
            self.data_dir.is_some()
    }

    /// Reads the [RollupConfig]s from the file system and returns a map of L2 chain ID ->
    /// [RollupConfig]s.
    pub fn read_rollup_configs(
        &self,
    ) -> Option<Result<HashMap<u64, RollupConfig>, InteropHostError>> {
        let rollup_config_paths = self.rollup_config_paths.as_ref()?;

        Some(rollup_config_paths.iter().try_fold(HashMap::default(), |mut acc, path| {
            // Read the serialized config from the file system.
            let ser_config = std::fs::read_to_string(path)?;

            // Deserialize the config and return it.
            let cfg: RollupConfig = serde_json::from_str(&ser_config)?;

            acc.insert(cfg.l2_chain_id.id(), cfg);
            Ok(acc)
        }))
    }

    /// Reads the [`L1ChainConfig`]s from the file system and returns a map of L1 chain ID ->
    /// [`L1ChainConfig`]s.
    pub fn read_l1_configs(&self) -> Option<Result<HashMap<u64, L1ChainConfig>, InteropHostError>> {
        let l1_config_paths = self.l1_config_paths.as_ref()?;

        Some(l1_config_paths.iter().try_fold(HashMap::default(), |mut acc, path| {
            // Read the serialized config from the file system.
            let ser_config = fs::read_to_string(path)?;

            // Deserialize the config and return it.
            let cfg: L1ChainConfig = serde_json::from_str(&ser_config)?;

            acc.insert(cfg.chain_id, cfg);
            Ok(acc)
        }))
    }

    /// Creates the key-value store for the host backend.
    fn create_key_value_store(&self) -> Result<SharedKeyValueStore, InteropHostError> {
        let local_kv_store = InteropLocalInputs::new(self.clone());

        let kv_store: SharedKeyValueStore = if let Some(ref data_dir) = self.data_dir {
            let disk_kv_store = DiskKeyValueStore::new(data_dir.clone());
            let split_kv_store = SplitKeyValueStore::new(local_kv_store, disk_kv_store);
            Arc::new(RwLock::new(split_kv_store))
        } else {
            let mem_kv_store = MemoryKeyValueStore::new();
            let split_kv_store = SplitKeyValueStore::new(local_kv_store, mem_kv_store);
            Arc::new(RwLock::new(split_kv_store))
        };

        Ok(kv_store)
    }

    /// Creates the providers required for the preimage server backend.
    async fn create_providers(&self) -> Result<InteropProviders, InteropHostError> {
        let l1_provider = rpc_provider(
            self.l1_node_address.as_ref().ok_or(InteropHostError::Other("Provider must be set"))?,
        )
        .await;

        let blob_provider = OnlineBlobProvider::init(OnlineBeaconClient::new_http(
            self.l1_beacon_address
                .clone()
                .ok_or(InteropHostError::Other("Beacon API URL must be set"))?,
        ))
        .await;

        // Resolve all chain IDs to their corresponding providers.
        let l2_node_addresses = self
            .l2_node_addresses
            .as_ref()
            .ok_or(InteropHostError::Other("L2 node addresses must be set"))?;
        let mut l2_providers = HashMap::default();
        for l2_node_address in l2_node_addresses {
            let l2_provider = rpc_provider::<Optimism>(l2_node_address).await;
            let chain_id = l2_provider.get_chain_id().await?;
            l2_providers.insert(chain_id, l2_provider);
        }

        Ok(InteropProviders { l1: l1_provider, blobs: blob_provider, l2s: l2_providers })
    }
}

impl OnlineHostBackendCfg for InteropHost {
    type HintType = HintType;
    type Providers = InteropProviders;
}

/// The providers required for the single chain host.
#[derive(Debug, Clone)]
pub struct InteropProviders {
    /// The L1 EL provider.
    pub l1: RootProvider,
    /// The L1 beacon node provider.
    pub blobs: OnlineBlobProvider<OnlineBeaconClient>,
    /// The L2 EL providers, keyed by chain ID.
    pub l2s: HashMap<u64, RootProvider<Optimism>>,
}

impl InteropProviders {
    /// Returns the L2 [RootProvider] for the given chain ID.
    pub fn l2(&self, chain_id: &u64) -> Result<&RootProvider<Optimism>, InteropHostError> {
        self.l2s.get(chain_id).ok_or_else(|| InteropHostError::RootProviderError(*chain_id))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;

    #[test]
    fn test_parse_interop_host_cli() {
        let hash = b256!("ffd7db0f9d5cdeb49c4c9eba649d4dc6d852d64671e65488e57f58584992ac68");
        let host = InteropHost::parse_from([
            "interop-host",
            "--l1-head",
            "ffd7db0f9d5cdeb49c4c9eba649d4dc6d852d64671e65488e57f58584992ac68",
            "--l2-pre-state",
            "ffd7db0f9d5cdeb49c4c9eba649d4dc6d852d64671e65488e57f58584992ac68",
            "--claimed-l2-post-state",
            &hash.to_string(),
            "--claimed-l2-timestamp",
            "0",
            "--native",
            "--l2-node-addresses",
            "http://localhost:8545",
            "--l1-node-address",
            "http://localhost:8546",
            "--l1-beacon-address",
            "http://localhost:8547",
        ]);
        assert_eq!(host.l1_head, hash);
        assert_eq!(host.agreed_l2_pre_state, Bytes::from(hash.0));
        assert_eq!(host.claimed_l2_post_state, hash);
        assert_eq!(host.claimed_l2_timestamp, 0);
        assert!(host.native);
    }

    #[test]
    fn test_parse_interop_hex_bytes() {
        let hash = b256!("ffd7db0f9d5cdeb49c4c9eba649d4dc6d852d64671e65488e57f58584992ac68");
        let host = InteropHost::parse_from([
            "interop-host",
            "--l1-head",
            "ffd7db0f9d5cdeb49c4c9eba649d4dc6d852d64671e65488e57f58584992ac68",
            "--l2-pre-state",
            "ff",
            "--claimed-l2-post-state",
            &hash.to_string(),
            "--claimed-l2-timestamp",
            "0",
            "--native",
            "--l2-node-addresses",
            "http://localhost:8545",
            "--l1-node-address",
            "http://localhost:8546",
            "--l1-beacon-address",
            "http://localhost:8547",
        ]);
        assert_eq!(host.l1_head, hash);
        assert_eq!(host.agreed_l2_pre_state, Bytes::from([0xff]));
        assert_eq!(host.claimed_l2_post_state, hash);
        assert_eq!(host.claimed_l2_timestamp, 0);
        assert!(host.native);
    }
}
