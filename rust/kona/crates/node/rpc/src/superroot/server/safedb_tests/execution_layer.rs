use super::*;
use crate::test_utils::rpc_block;
use alloy_primitives::keccak256;
use alloy_rpc_types_engine::JwtSecret;
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle},
    types::ErrorObjectOwned,
};
use kona_engine::{EngineClient, EngineQueryClient, test_utils::RpcMock};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_rpc_types::Transaction;
use serde_json::Value;
use std::{collections::BTreeMap, sync::Mutex};

/// Blocks keyed by number.
pub(super) type Chain = BTreeMap<u64, Block<Transaction>>;

pub(super) fn block_info(block: &Block<Transaction>) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: block.header.hash,
            number: block.header.number,
            parent_hash: block.header.parent_hash,
            timestamp: block.header.timestamp,
        },
        ..Default::default()
    }
}

pub(super) fn storage_root(block: &Block<Transaction>) -> B256 {
    keccak256(block.header.state_root)
}

/// The pre-Isthmus output root of `block`, with the message-passer storage root that
/// [`InMemoryExecutionLayer`] proves for it.
pub(super) fn output_root(block: &Block<Transaction>) -> B256 {
    OutputRoot::from_parts(block.header.state_root, storage_root(block), block.header.hash).hash()
}

/// A chain from genesis to `head` whose state roots differ from `fork_at` onward, so two chains
/// with different `fork_at` share their blocks below it.
pub(super) fn chain(head: u64, fork_at: u64) -> Chain {
    let mut blocks = BTreeMap::new();
    let mut parent_hash = B256::ZERO;
    for number in 0..=head {
        let branch = if number >= fork_at { "replacement" } else { "original" };
        let block = rpc_block(BlockOutput {
            block: L2BlockInfo {
                block_info: BlockInfo {
                    number,
                    parent_hash,
                    timestamp: timestamp_of(number),
                    ..Default::default()
                },
                ..Default::default()
            },
            output: OutputRoot {
                block_hash: B256::ZERO,
                bridge_storage_root: B256::ZERO,
                state_root: keccak256(format!("{branch} state at {number}")),
            },
        });
        parent_hash = block.header.hash;
        blocks.insert(number, block);
    }
    blocks
}

pub(super) fn rollup_config_with_genesis(genesis: &Block<Transaction>) -> Arc<RollupConfig> {
    let mut config = (*rollup_config()).clone();
    config.genesis.l2 = block_info(genesis).block_info.id();
    Arc::new(config)
}

/// A read a super-root query makes of the EL, as told apart on the wire.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum Read {
    BlockByHash,
    BlockByNumber(u64),
    Proof,
}

/// An EL served from memory. It serves every block it was given by hash and the canonical chain
/// by number, counts requests by method and block reads that asked for transactions, and calls
/// `before_read` before serving each read, so a test can change the canonical chain at a chosen
/// point of a query.
pub(super) struct InMemoryExecutionLayer<BeforeRead> {
    before_read: BeforeRead,
    blocks: BTreeMap<B256, Block<Transaction>>,
    canonical: Chain,
    request_counts: BTreeMap<&'static str, usize>,
    reads_with_transactions: usize,
}

impl<BeforeRead> InMemoryExecutionLayer<BeforeRead>
where
    BeforeRead: FnMut(Read, &mut Chain) + Send + 'static,
{
    pub(super) fn new<'chain>(
        known: impl IntoIterator<Item = &'chain Chain>,
        canonical: Chain,
        before_read: BeforeRead,
    ) -> Self {
        Self {
            before_read,
            blocks: known
                .into_iter()
                .flat_map(|chain| chain.values())
                .map(|block| (block.header.hash, block.clone()))
                .collect(),
            canonical,
            request_counts: BTreeMap::new(),
            reads_with_transactions: 0,
        }
    }

    pub(super) const fn request_counts(&self) -> &BTreeMap<&'static str, usize> {
        &self.request_counts
    }

    /// The number of block reads that asked for full transactions rather than only hashes.
    pub(super) const fn reads_with_transactions(&self) -> usize {
        self.reads_with_transactions
    }

    fn block_by_number(&mut self, number: u64, full: bool) -> Option<Block<Transaction>> {
        self.record_block_read("eth_getBlockByNumber", Read::BlockByNumber(number), full);
        self.canonical.get(&number).cloned()
    }

    fn block_by_hash(&mut self, hash: B256, full: bool) -> Option<Block<Transaction>> {
        self.record_block_read("eth_getBlockByHash", Read::BlockByHash, full);
        self.blocks.get(&hash).cloned()
    }

    fn proof(&mut self, hash: B256) -> Result<EIP1186AccountProofResponse, ErrorObjectOwned> {
        self.record("eth_getProof", Read::Proof);
        let block = self.blocks.get(&hash).ok_or_else(|| {
            ErrorObjectOwned::owned(-32000, format!("unknown block {hash}"), None::<()>)
        })?;
        Ok(EIP1186AccountProofResponse { storage_hash: storage_root(block), ..Default::default() })
    }

    fn record_block_read(&mut self, method: &'static str, read: Read, full: bool) {
        self.reads_with_transactions += usize::from(full);
        self.record(method, read);
    }

    fn record(&mut self, method: &'static str, read: Read) {
        *self.request_counts.entry(method).or_default() += 1;
        (self.before_read)(read, &mut self.canonical);
    }
}

/// Serves `execution_layer` and returns a query client connected to it.
pub(super) async fn serve_execution_layer<BeforeRead>(
    execution_layer: Arc<Mutex<InMemoryExecutionLayer<BeforeRead>>>,
    config: Arc<RollupConfig>,
) -> (ServerHandle, EngineQueryClient)
where
    BeforeRead: FnMut(Read, &mut Chain) + Send + 'static,
{
    let mut module = RpcModule::new(execution_layer);
    module
        .register_method("eth_getBlockByNumber", |params, execution_layer, _| {
            let (number, full): (U64, bool) = params.parse()?;
            Ok::<_, ErrorObjectOwned>(
                execution_layer.lock().unwrap().block_by_number(number.to::<u64>(), full),
            )
        })
        .unwrap();
    module
        .register_method("eth_getBlockByHash", |params, execution_layer, _| {
            let (hash, full): (B256, bool) = params.parse()?;
            Ok::<_, ErrorObjectOwned>(execution_layer.lock().unwrap().block_by_hash(hash, full))
        })
        .unwrap();
    module
        .register_method("eth_getProof", |params, execution_layer, _| {
            let (_address, _slots, hash): (Value, Value, B256) = params.parse()?;
            execution_layer.lock().unwrap().proof(hash)
        })
        .unwrap();
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let address = server.local_addr().unwrap();
    let server = server.start(module);
    let engine = EngineClient::new(
        RpcMock::default().provider(),
        EngineClient::rpc_client(
            format!("http://{address}").parse().unwrap(),
            JwtSecret::from_hex("00".repeat(32)).unwrap(),
        ),
        config,
    );
    (server, engine.query_client())
}
