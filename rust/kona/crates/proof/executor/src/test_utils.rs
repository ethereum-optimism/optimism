//! Test utilities for the executor.

use crate::{StatelessL2Builder, TrieDBProvider};
use alloy_consensus::Header;
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{B256, Bytes, Sealable};
use alloy_provider::{Provider, RootProvider, network::primitives::BlockTransactions};
use alloy_rlp::Decodable;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::PayloadAttributes;
use alloy_transport_http::{Client, Http};
use kona_genesis::RollupConfig;
use kona_mpt::{NoopTrieHinter, TrieNode, TrieProvider};
use kona_registry::ROLLUP_CONFIGS;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use rocksdb::{DB, Options};
use serde::{Deserialize, Serialize};
use std::{path::PathBuf, sync::Arc};
use tokio::{fs, runtime::Handle, sync::Mutex};

/// Executes a [`ExecutorTestFixture`] stored at the passed `fixture_path` and asserts that the
/// produced block hash matches the expected block hash.
pub async fn run_test_fixture(fixture_path: PathBuf) {
    // First, untar the fixture.
    let fixture_dir = tempfile::tempdir().expect("Failed to create temporary directory");
    tokio::process::Command::new("tar")
        .arg("-xvf")
        .arg(fixture_path.as_path())
        .arg("-C")
        .arg(fixture_dir.path())
        .arg("--strip-components=1")
        .output()
        .await
        .expect("Failed to untar fixture");

    let mut options = Options::default();
    options.set_compression_type(rocksdb::DBCompressionType::Snappy);
    options.create_if_missing(true);
    let kv_store = DB::open(&options, fixture_dir.path().join("kv"))
        .unwrap_or_else(|e| panic!("Failed to open database at {fixture_dir:?}: {e}"));
    let provider = DiskTrieNodeProvider::new(kv_store);
    let fixture: ExecutorTestFixture =
        serde_json::from_slice(&fs::read(fixture_dir.path().join("fixture.json")).await.unwrap())
            .expect("Failed to deserialize fixture");

    let mut executor = StatelessL2Builder::new(
        &fixture.rollup_config,
        OpEvmFactory::default(),
        provider,
        NoopTrieHinter,
        fixture.parent_header.seal_slow(),
    );

    let outcome = executor.build_block(fixture.executing_payload).unwrap();

    assert_eq!(
        outcome.header.hash(),
        fixture.expected_block_hash,
        "Produced header does not match the expected header"
    );
}

/// The test fixture format for the [`StatelessL2Builder`].
#[derive(Debug, Serialize, Deserialize)]
pub struct ExecutorTestFixture {
    /// The rollup configuration for the executing chain.
    pub rollup_config: RollupConfig,
    /// The parent block header.
    pub parent_header: Header,
    /// The executing payload attributes.
    pub executing_payload: OpPayloadAttributes,
    /// The expected block hash
    pub expected_block_hash: B256,
}

/// A test fixture creator for the [`StatelessL2Builder`].
#[derive(Debug)]
pub struct ExecutorTestFixtureCreator {
    /// The RPC provider for the L2 execution layer.
    pub provider: RootProvider,
    /// The block number to create the test fixture for.
    pub block_number: u64,
    /// The key value store for the test fixture.
    pub kv_store: Arc<Mutex<rocksdb::DB>>,
    /// The data directory for the test fixture.
    pub data_dir: PathBuf,
}

impl ExecutorTestFixtureCreator {
    /// Creates a new [`ExecutorTestFixtureCreator`] with the given parameters.
    pub fn new(provider_url: &str, block_number: u64, base_fixture_directory: PathBuf) -> Self {
        let base = base_fixture_directory.join(format!("block-{block_number}"));

        let url = provider_url.parse().expect("Invalid provider URL");
        let http = Http::<Client>::new(url);
        let provider = RootProvider::new(RpcClient::new(http, false));

        let mut options = Options::default();
        options.set_compression_type(rocksdb::DBCompressionType::Snappy);
        options.create_if_missing(true);
        let db = DB::open(&options, base.join("kv").as_path())
            .unwrap_or_else(|e| panic!("Failed to open database at {base:?}: {e}"));

        Self { provider, block_number, kv_store: Arc::new(Mutex::new(db)), data_dir: base }
    }
}

impl ExecutorTestFixtureCreator {
    /// Create a static test fixture with the configuration provided.
    pub async fn create_static_fixture(self) {
        let chain_id = self.provider.get_chain_id().await.expect("Failed to get chain ID");
        let rollup_config = ROLLUP_CONFIGS.get(&chain_id).expect("Rollup config not found");

        let executing_block = self
            .provider
            .get_block_by_number(self.block_number.into())
            .await
            .expect("Failed to get parent block")
            .expect("Block not found");
        let parent_block = self
            .provider
            .get_block_by_number((self.block_number - 1).into())
            .await
            .expect("Failed to get parent block")
            .expect("Block not found");

        let executing_header = executing_block.header;
        let parent_header = parent_block.header.inner.seal_slow();

        let encoded_executing_transactions = match executing_block.transactions {
            BlockTransactions::Hashes(transactions) => {
                let mut encoded_transactions = Vec::with_capacity(transactions.len());
                for tx_hash in transactions {
                    let tx = self
                        .provider
                        .client()
                        .request::<&[B256; 1], Bytes>("debug_getRawTransaction", &[tx_hash])
                        .await
                        .expect("Block not found");
                    encoded_transactions.push(tx);
                }
                encoded_transactions
            }
            _ => panic!("Only BlockTransactions::Hashes are supported."),
        };

        let payload_attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: executing_header.timestamp,
                parent_beacon_block_root: executing_header.parent_beacon_block_root,
                prev_randao: executing_header.mix_hash,
                withdrawals: Default::default(),
                suggested_fee_recipient: executing_header.beneficiary,
            },
            gas_limit: Some(executing_header.gas_limit),
            transactions: Some(encoded_executing_transactions),
            no_tx_pool: None,
            eip_1559_params: rollup_config.is_holocene_active(executing_header.timestamp).then(
                || {
                    executing_header.extra_data[1..9]
                        .try_into()
                        .expect("Invalid header format for Holocene")
                },
            ),
            min_base_fee: rollup_config.is_jovian_active(executing_header.timestamp).then(|| {
                // The min base fee is the bytes 9-17 of the extra data.
                executing_header.extra_data[9..17]
                    .try_into()
                    .map(u64::from_be_bytes)
                    .expect("Invalid header format for Jovian")
            }),
        };

        let fixture_path = self.data_dir.join("fixture.json");
        let fixture = ExecutorTestFixture {
            rollup_config: rollup_config.clone(),
            parent_header: parent_header.inner().clone(),
            executing_payload: payload_attrs.clone(),
            expected_block_hash: executing_header.hash_slow(),
        };

        let mut executor = StatelessL2Builder::new(
            rollup_config,
            OpEvmFactory::default(),
            self,
            NoopTrieHinter,
            parent_header,
        );
        let outcome = executor.build_block(payload_attrs).expect("Failed to execute block");

        assert_eq!(
            outcome.header.inner(),
            &executing_header.inner,
            "Produced header does not match the expected header"
        );
        fs::write(fixture_path.as_path(), serde_json::to_vec(&fixture).unwrap()).await.unwrap();

        // Tar the fixture.
        let data_dir = fixture_path.parent().unwrap();
        tokio::process::Command::new("tar")
            .arg("-czf")
            .arg(data_dir.with_extension("tar.gz").file_name().unwrap())
            .arg(data_dir.file_name().unwrap())
            .current_dir(data_dir.parent().unwrap())
            .output()
            .await
            .expect("Failed to tar fixture");

        // Remove the leftover directory.
        fs::remove_dir_all(data_dir).await.expect("Failed to remove temporary directory");
    }
}

impl TrieProvider for ExecutorTestFixtureCreator {
    type Error = TestTrieNodeProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        // Fetch the preimage from the L2 chain provider.
        let preimage: Bytes = tokio::task::block_in_place(move || {
            Handle::current().block_on(async {
                let preimage: Bytes = self
                    .provider
                    .client()
                    .request("debug_dbGet", &[key])
                    .await
                    .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?;

                self.kv_store
                    .lock()
                    .await
                    .put(key, preimage.clone())
                    .map_err(|_| TestTrieNodeProviderError::KVStore)?;

                Ok(preimage)
            })
        })?;

        // Decode the preimage into a trie node.
        TrieNode::decode(&mut preimage.as_ref()).map_err(TestTrieNodeProviderError::Rlp)
    }
}

impl TrieDBProvider for ExecutorTestFixtureCreator {
    fn bytecode_by_hash(&self, hash: B256) -> Result<Bytes, Self::Error> {
        // geth hashdb scheme code hash key prefix
        const CODE_PREFIX: u8 = b'c';

        // Fetch the preimage from the L2 chain provider.
        let preimage: Bytes = tokio::task::block_in_place(move || {
            Handle::current().block_on(async {
                // Attempt to fetch the code from the L2 chain provider.
                let code_hash = [&[CODE_PREFIX], hash.as_slice()].concat();
                let code = self
                    .provider
                    .client()
                    .request::<&[Bytes; 1], Bytes>("debug_dbGet", &[code_hash.into()])
                    .await;

                // Check if the first attempt to fetch the code failed. If it did, try fetching the
                // code hash preimage without the geth hashdb scheme prefix.
                let code = match code {
                    Ok(code) => code,
                    Err(_) => self
                        .provider
                        .client()
                        .request::<&[B256; 1], Bytes>("debug_dbGet", &[hash])
                        .await
                        .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?,
                };

                self.kv_store
                    .lock()
                    .await
                    .put(hash, code.clone())
                    .map_err(|_| TestTrieNodeProviderError::KVStore)?;

                Ok(code)
            })
        })?;

        Ok(preimage)
    }

    fn header_by_hash(&self, hash: B256) -> Result<Header, Self::Error> {
        let encoded_header: Bytes = tokio::task::block_in_place(move || {
            Handle::current().block_on(async {
                let preimage: Bytes = self
                    .provider
                    .client()
                    .request("debug_getRawHeader", &[hash])
                    .await
                    .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?;

                self.kv_store
                    .lock()
                    .await
                    .put(hash, preimage.clone())
                    .map_err(|_| TestTrieNodeProviderError::KVStore)?;

                Ok(preimage)
            })
        })?;

        // Decode the Header.
        Header::decode(&mut encoded_header.as_ref()).map_err(TestTrieNodeProviderError::Rlp)
    }
}

/// A simple [`TrieDBProvider`] that reads data from a disk-based key-value store.
#[derive(Debug)]
pub struct DiskTrieNodeProvider {
    kv_store: DB,
}

impl DiskTrieNodeProvider {
    /// Creates a new [`DiskTrieNodeProvider`] with the given [`rocksdb`] K/V store.
    pub const fn new(kv_store: DB) -> Self {
        Self { kv_store }
    }
}

impl TrieProvider for DiskTrieNodeProvider {
    type Error = TestTrieNodeProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        TrieNode::decode(
            &mut self
                .kv_store
                .get(key)
                .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?
                .ok_or(TestTrieNodeProviderError::PreimageNotFound)?
                .as_slice(),
        )
        .map_err(TestTrieNodeProviderError::Rlp)
    }
}

impl TrieDBProvider for DiskTrieNodeProvider {
    fn bytecode_by_hash(&self, code_hash: B256) -> Result<Bytes, Self::Error> {
        self.kv_store
            .get(code_hash)
            .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?
            .map(Bytes::from)
            .ok_or(TestTrieNodeProviderError::PreimageNotFound)
    }

    fn header_by_hash(&self, hash: B256) -> Result<Header, Self::Error> {
        Header::decode(
            &mut self
                .kv_store
                .get(hash)
                .map_err(|_| TestTrieNodeProviderError::PreimageNotFound)?
                .ok_or(TestTrieNodeProviderError::PreimageNotFound)?
                .as_slice(),
        )
        .map_err(TestTrieNodeProviderError::Rlp)
    }
}

/// An error type for the [`DiskTrieNodeProvider`] and [`ExecutorTestFixtureCreator`].
#[derive(Debug, thiserror::Error)]
pub enum TestTrieNodeProviderError {
    /// The preimage was not found in the key-value store.
    #[error("Preimage not found")]
    PreimageNotFound,
    /// Failed to decode the RLP-encoded data.
    #[error("Failed to decode RLP: {0}")]
    Rlp(alloy_rlp::Error),
    /// Failed to write back to the key-value store.
    #[error("Failed to write back to key value store")]
    KVStore,
}
