//! This module contains all CLI-specific code for the interop entrypoint.

use super::{InteropHintHandler, InteropLocalInputs};
use crate::{
    OfflineHostBackend, OnlineHostBackend, OnlineHostBackendCfg, PreimageServer,
    SharedKeyValueStore,
    eth::rpc_provider,
    kv::{DataFormat, create_key_value_store},
    server::PreimageServerError,
};
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Provider, RootProvider};
use alloy_rlp::Decodable;
use clap::Parser;
use kona_cli::cli_styles;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_preimage::{
    BidirectionalChannel, Channel, HintReader, HintWriter, OracleReader, OracleServer,
    VerifyingPreimageFetcher,
};
use kona_proof_interop::{HintType, INVALID_TRANSITION, PreState};
use kona_providers_alloy::{OnlineBeaconClient, OnlineBlobProvider};
use kona_registry::DEPENDENCY_SETS;
use kona_std_fpvm::{FileChannel, FileDescriptor};
use op_alloy_network::Optimism;
use serde::Serialize;
use std::{
    collections::{BTreeMap, HashMap},
    path::PathBuf,
    str::FromStr,
    sync::Arc,
};
use tokio::task::{self, JoinHandle};

/// The interop host application.
#[derive(Default, Parser, Serialize, Clone, Debug)]
#[command(styles = cli_styles())]
pub struct InteropHost {
    /// Hash of the L1 head block, marking a static, trusted cutoff point for reading data from the
    /// L1 chain.
    #[arg(long, env)]
    pub l1_head: B256,
    /// Agreed [`PreState`] to start from.
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
    /// The default format for preimage data storage on disk. If the data directory already
    /// contains a `kvformat` marker file, that format is used instead of this value.
    #[arg(long, default_value = "directory", env)]
    pub data_format: DataFormat,
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
    /// Path to l1 config. If provided, the host will use this config instead of attempting to
    /// look up the config in the superchain registry.
    /// The l1 config should be stored as serde-JSON serialized files.
    #[arg(long, alias = "l1-cfg")]
    pub l1_config_path: Option<PathBuf>,
    /// Path to the dependency set config. If provided, the host will use this config for the
    /// dependency set. The config should be stored as a serde-JSON serialized file.
    #[arg(long, alias = "depset-cfg", env)]
    pub dependency_set_path: Option<PathBuf>,
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
    /// No l1 config found.
    #[error("No l1 config found")]
    NoL1Config,
    /// Task failed to execute to completion.
    #[error("Join error: {0}")]
    ExecutionError(#[from] tokio::task::JoinError),
    /// A RPC error.
    #[error("Rpc Error: {0}")]
    RpcError(#[from] alloy_transport::RpcError<alloy_transport::TransportErrorKind>),
    /// A blob provider error.
    #[error("Blob provider error: {0}")]
    BlobProviderError(#[from] kona_derive::BlobProviderError),
    /// An error when no provider found for chain ID.
    #[error("No provider found for chain ID: {0}")]
    RootProviderError(u64),
    /// Neither the superchain-registry nor `--depset-cfg` has a dependency set for the chain
    /// whose set the client loads.
    #[error(
        "No dependency set for chain {chain_id}, the first chain of the agreed pre-state: it is \
         not in the superchain-registry and --depset-cfg was not provided. Supply the \
         dependency-set JSON file the super roots are derived with (op-supernode \
         --dependency-set, or op-node --interop.dependency-set)."
    )]
    MissingDependencySet {
        /// The first chain of the agreed pre-state.
        chain_id: u64,
    },
    /// The agreed pre-state does not decode.
    #[error("Failed decoding the agreed pre-state: {0}")]
    InvalidAgreedPreState(alloy_rlp::Error),
    /// The dependency-set file cannot be read.
    #[error("Failed reading dependency set {path:?}: {source}")]
    UnreadableDependencySet {
        /// The dependency-set file.
        path: PathBuf,
        /// The IO error.
        source: std::io::Error,
    },
    /// The dependency-set file does not parse.
    #[error("Failed deserializing dependency set {path:?}: {source}")]
    InvalidDependencySet {
        /// The dependency-set file.
        path: PathBuf,
        /// The parse error.
        source: serde_json::Error,
    },
    /// Any other error.
    #[error("Error: {0}")]
    Other(&'static str),
}

impl InteropHost {
    /// Starts the [`InteropHost`] application.
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

    /// Refuses to start with a `--depset-cfg` that cannot be read or parsed, even when the
    /// registry's set wins.
    fn require_readable_dependency_set(&self) -> Result<(), InteropHostError> {
        self.read_dependency_set().transpose()?;
        Ok(())
    }

    /// Starts the preimage server, communicating with the client over the provided channels.
    ///
    /// Refuses to start when a `--rollup-config-paths` file cannot be read or parsed, when the
    /// host cannot resolve the dependency set the client loads (see [`Self::dependency_set`]), or
    /// when a configured `--depset-cfg` cannot be read or parsed.
    pub async fn start_server<C>(
        &self,
        hint: C,
        preimage: C,
    ) -> Result<JoinHandle<Result<(), InteropHostError>>, InteropHostError>
    where
        C: Channel + Send + Sync + 'static,
    {
        self.read_rollup_configs().transpose()?;
        self.dependency_set()?;
        self.require_readable_dependency_set()?;

        let kv_store = self.create_key_value_store()?;

        let task_handle = if self.is_offline() {
            task::spawn(async {
                PreimageServer::new(
                    OracleServer::new(preimage),
                    HintReader::new(hint),
                    Arc::new(VerifyingPreimageFetcher::new(OfflineHostBackend::new(kv_store))),
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
            );

            task::spawn(async {
                PreimageServer::new(
                    OracleServer::new(preimage),
                    HintReader::new(hint),
                    Arc::new(VerifyingPreimageFetcher::new(backend)),
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

    /// Reads the [`RollupConfig`]s from the file system and returns a map of L2 chain ID ->
    /// [`RollupConfig`]s. Uses `BTreeMap` to ensure deterministic JSON serialization order when
    /// these configs are served as preimages to the cannon VM.
    pub fn read_rollup_configs(
        &self,
    ) -> Option<Result<BTreeMap<u64, RollupConfig>, InteropHostError>> {
        let rollup_config_paths = self.rollup_config_paths.as_ref()?;

        Some(rollup_config_paths.iter().try_fold(BTreeMap::default(), |mut acc, path| {
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
    pub fn read_l1_config(&self) -> Result<L1ChainConfig, InteropHostError> {
        let path = self.l1_config_path.as_ref().ok_or_else(|| InteropHostError::NoL1Config)?;

        // Read the serialized config from the file system.
        let ser_config = std::fs::read_to_string(path)?;

        // Deserialize the config and return it.
        serde_json::from_str(&ser_config)
            .map_err(|_| InteropHostError::Other("failed to parse L1 config"))
    }

    /// Reads the [`DependencySet`] from the file system.
    pub fn read_dependency_set(&self) -> Option<Result<DependencySet, InteropHostError>> {
        let path = self.dependency_set_path.as_ref()?;

        Some(
            std::fs::read_to_string(path)
                .map_err(|source| InteropHostError::UnreadableDependencySet {
                    path: path.clone(),
                    source,
                })
                .and_then(|s| {
                    serde_json::from_str(&s).map_err(|source| {
                        InteropHostError::InvalidDependencySet { path: path.clone(), source }
                    })
                }),
        )
    }

    /// Returns the [`DependencySet`] the client loads in `BootInfo::load`: the
    /// superchain-registry's set for the first chain of the agreed pre-state, otherwise the set
    /// the host serves from `--depset-cfg`. Returns `None` for an `INVALID_TRANSITION`
    /// pre-state, which the client resolves without loading a set.
    pub fn dependency_set(&self) -> Result<Option<DependencySet>, InteropHostError> {
        if self.agreed_l2_pre_state == INVALID_TRANSITION {
            return Ok(None);
        }
        let pre_state = PreState::decode(&mut self.agreed_l2_pre_state.as_ref())
            .map_err(InteropHostError::InvalidAgreedPreState)?;
        let chain_id = pre_state
            .output_roots()
            .first()
            .ok_or(InteropHostError::Other("the agreed pre-state has no output roots"))?
            .chain_id;
        if let Some(dependency_set) = DEPENDENCY_SETS.get(&chain_id) {
            return Ok(Some(dependency_set.clone()));
        }
        self.read_dependency_set()
            .ok_or(InteropHostError::MissingDependencySet { chain_id })?
            .map(Some)
    }

    /// Creates the key-value store for the host backend.
    ///
    /// If the data directory contains a `kvformat` marker file, the recorded format is used to
    /// ensure compatibility with existing data. Otherwise, `--data-format` is used as the default.
    fn create_key_value_store(&self) -> Result<SharedKeyValueStore, InteropHostError> {
        let local_kv_store = InteropLocalInputs::new(self.clone());
        Ok(create_key_value_store(local_kv_store, self.data_dir.as_deref(), self.data_format)?)
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
        .await?;

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

    /// `L2PayloadWitness` bulk-populates the store via `debug_executePayload`.
    /// `L2BlockData` re-executes a replaced optimistic block and bulk-populates its preimages.
    const HIGH_LEVEL_HINTS: &'static [HintType] =
        &[HintType::L2PayloadWitness, HintType::L2BlockData];
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
    /// Returns the L2 [`RootProvider`] for the given chain ID.
    pub fn l2(&self, chain_id: &u64) -> Result<&RootProvider<Optimism>, InteropHostError> {
        self.l2s.get(chain_id).ok_or_else(|| InteropHostError::RootProviderError(*chain_id))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;
    use kona_interop::{MESSAGE_EXPIRY_WINDOW, OutputRootWithChain, SuperRoot};
    use kona_proof_interop::{OptimisticBlock, TransitionState};

    const OP_SEPOLIA: u64 = 11155420;

    /// RLP-encodes a super-root pre-state over `chain_ids`. [`SuperRoot::new`] sorts by chain id,
    /// so the first chain is the smallest id.
    fn super_root_pre_state(chain_ids: &[u64]) -> Bytes {
        let output_roots =
            chain_ids.iter().map(|id| OutputRootWithChain::new(*id, B256::ZERO)).collect();
        alloy_rlp::encode(PreState::SuperRoot(SuperRoot::new(1, output_roots))).into()
    }

    /// Writes a dependency set for chain 901 to a file in `dir`.
    fn write_dependency_set(dir: &tempfile::TempDir) -> PathBuf {
        let file = dir.path().join("depset.json");
        std::fs::write(&file, r#"{"dependencies":{"901":{}}}"#).unwrap();
        file
    }

    #[test]
    fn the_registry_set_of_the_first_pre_state_chain_wins() {
        let dir = tempfile::tempdir().unwrap();

        // The client takes the embedded set before the one the host serves, so the host must too.
        for dependency_set_path in [None, Some(write_dependency_set(&dir))] {
            let host = InteropHost {
                agreed_l2_pre_state: super_root_pre_state(&[OP_SEPOLIA]),
                dependency_set_path,
                ..Default::default()
            };
            assert_eq!(host.dependency_set().unwrap(), Some(DEPENDENCY_SETS[&OP_SEPOLIA].clone()));
        }
    }

    #[test]
    fn a_first_chain_outside_the_registry_uses_the_configured_set() {
        let dir = tempfile::tempdir().unwrap();
        let host = InteropHost {
            agreed_l2_pre_state: super_root_pre_state(&[901, OP_SEPOLIA]),
            dependency_set_path: Some(write_dependency_set(&dir)),
            ..Default::default()
        };

        // The lookup keys on the first pre-state chain (901), not on OP Sepolia.
        let dependency_set = host.dependency_set().unwrap().unwrap();
        assert_eq!(dependency_set.dependencies.keys().copied().collect::<Vec<_>>(), [901]);
    }

    #[test]
    fn a_first_chain_outside_the_registry_without_a_set_is_an_error() {
        let host =
            InteropHost { agreed_l2_pre_state: super_root_pre_state(&[901]), ..Default::default() };

        assert!(matches!(
            host.dependency_set(),
            Err(InteropHostError::MissingDependencySet { chain_id: 901 })
        ));
    }

    #[test]
    fn an_invalid_transition_needs_no_dependency_set() {
        let host = InteropHost { agreed_l2_pre_state: INVALID_TRANSITION, ..Default::default() };

        assert_eq!(host.dependency_set().unwrap(), None);
    }

    #[test]
    fn a_transition_state_keys_on_its_first_chain() {
        let dir = tempfile::tempdir().unwrap();
        let super_root = SuperRoot::new(
            1,
            vec![
                OutputRootWithChain::new(901, B256::ZERO),
                OutputRootWithChain::new(OP_SEPOLIA, B256::ZERO),
            ],
        );
        // Step 1: the active chain is OP Sepolia, but the client keys on the first chain (901).
        let pre_state = PreState::TransitionState(TransitionState::new(
            super_root,
            vec![OptimisticBlock::new(B256::ZERO, B256::ZERO)],
            1,
        ));
        let host = InteropHost {
            agreed_l2_pre_state: alloy_rlp::encode(pre_state).into(),
            dependency_set_path: Some(write_dependency_set(&dir)),
            ..Default::default()
        };

        let dependency_set = host.dependency_set().unwrap().unwrap();
        assert_eq!(dependency_set.dependencies.keys().copied().collect::<Vec<_>>(), [901]);
    }

    #[test]
    fn an_undecodable_pre_state_is_an_error() {
        let host =
            InteropHost { agreed_l2_pre_state: Bytes::from_static(&[0xff]), ..Default::default() };

        assert!(matches!(host.dependency_set(), Err(InteropHostError::InvalidAgreedPreState(_))));
    }

    #[tokio::test]
    async fn start_server_refuses_a_missing_dependency_set() {
        let host = InteropHost {
            server: true,
            agreed_l2_pre_state: super_root_pre_state(&[901]),
            ..Default::default()
        };
        let hint = BidirectionalChannel::new().unwrap();
        let preimage = BidirectionalChannel::new().unwrap();

        let err = host.start_server(hint.host, preimage.host).await.unwrap_err();

        assert!(matches!(err, InteropHostError::MissingDependencySet { chain_id: 901 }));
    }

    #[tokio::test]
    async fn start_server_refuses_an_unreadable_rollup_config() {
        let dir = tempfile::tempdir().unwrap();
        let host = InteropHost {
            server: true,
            agreed_l2_pre_state: super_root_pre_state(&[OP_SEPOLIA]),
            rollup_config_paths: Some(vec![dir.path().join("missing.json")]),
            ..Default::default()
        };
        let hint = BidirectionalChannel::new().unwrap();
        let preimage = BidirectionalChannel::new().unwrap();

        let err = host.start_server(hint.host, preimage.host).await.unwrap_err();

        assert!(matches!(err, InteropHostError::IOError(_)));
    }

    #[tokio::test]
    async fn start_server_refuses_an_unreadable_set_when_the_registry_wins() {
        let dir = tempfile::tempdir().unwrap();
        let host = InteropHost {
            server: true,
            agreed_l2_pre_state: super_root_pre_state(&[OP_SEPOLIA]),
            dependency_set_path: Some(dir.path().join("missing.json")),
            ..Default::default()
        };
        let hint = BidirectionalChannel::new().unwrap();
        let preimage = BidirectionalChannel::new().unwrap();

        let err = host.start_server(hint.host, preimage.host).await.unwrap_err();

        assert!(matches!(err, InteropHostError::UnreadableDependencySet { .. }));
    }

    #[test]
    fn test_require_readable_dependency_set() {
        let dir = tempfile::tempdir().unwrap();
        let host = |dependency_set_path| InteropHost { dependency_set_path, ..Default::default() };

        assert!(host(None).require_readable_dependency_set().is_ok());

        let missing = dir.path().join("missing.json");
        assert!(matches!(
            host(Some(missing)).require_readable_dependency_set(),
            Err(InteropHostError::UnreadableDependencySet { .. })
        ));

        let too_long = dir.path().join("depset.json");
        std::fs::write(&too_long, r#"{"dependencies":{},"overrideMessageExpiryWindow":604801}"#)
            .unwrap();
        let err = host(Some(too_long)).require_readable_dependency_set().unwrap_err();
        assert!(matches!(err, InteropHostError::InvalidDependencySet { .. }));
        assert!(err.to_string().contains("overrideMessageExpiryWindow 604801s exceeds"));

        let at_limit = dir.path().join("depset-at-limit.json");
        std::fs::write(
            &at_limit,
            format!(
                r#"{{"dependencies":{{}},"overrideMessageExpiryWindow":{MESSAGE_EXPIRY_WINDOW}}}"#
            ),
        )
        .unwrap();
        assert!(host(Some(at_limit)).require_readable_dependency_set().is_ok());
    }

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
