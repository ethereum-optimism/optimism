use super::*;
use crate::test_utils::rpc_block;
use alloy_rpc_types_engine::JwtSecret;
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle},
    types::ErrorObjectOwned,
};
use kona_engine::{EngineClient, test_utils::RpcMock};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_rpc_types::Transaction;
use serde_json::Value;
use std::{collections::BTreeMap, sync::Mutex};

#[derive(Clone, Copy, Debug)]
enum Mutation {
    DatabaseResetDuringProof,
    None,
    ReorgBeforeQuery,
    ReorgDuringProof,
}

struct ExecutionState {
    blocks: BTreeMap<B256, Block<Transaction>>,
    canonical: BTreeMap<u64, Block<Transaction>>,
    database: Arc<SafeDatabase>,
    mutation: Mutation,
    recorded_head: u64,
    replacement: BTreeMap<u64, Block<Transaction>>,
}

impl ExecutionState {
    fn proof(&mut self, block_id: Value) -> EIP1186AccountProofResponse {
        match self.mutation {
            Mutation::DatabaseResetDuringProof => {
                self.database.clear().unwrap();
                for (height, l1) in [(1, 100), (self.recorded_head, 200)] {
                    self.database
                        .safe_head_updated(
                            block_info(&self.replacement[&height]),
                            l1_block(l1).id(),
                        )
                        .unwrap();
                }
            }
            Mutation::ReorgDuringProof => self.canonical = self.replacement.clone(),
            Mutation::None | Mutation::ReorgBeforeQuery => {}
        }
        self.mutation = Mutation::None;
        let block = if let Some(hash) = block_id.get("blockHash") {
            &self.blocks[&serde_json::from_value::<B256>(hash.clone()).unwrap()]
        } else {
            let number = serde_json::from_value::<U64>(block_id).unwrap().to::<u64>();
            &self.canonical[&number]
        };
        EIP1186AccountProofResponse { storage_hash: storage_root(block), ..Default::default() }
    }
}

struct ReorgFixture {
    _directory: TempDir,
    expected_output: B256,
    rpc: SuperRootRpc,
    server: ServerHandle,
}

impl Drop for ReorgFixture {
    fn drop(&mut self) {
        let _ = self.server.stop();
    }
}

fn block_info(block: &Block<Transaction>) -> L2BlockInfo {
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

fn storage_root(block: &Block<Transaction>) -> B256 {
    B256::with_last_byte(block.header.state_root.0[31] + 0x20)
}

fn chain(fork_at: u64) -> BTreeMap<u64, Block<Transaction>> {
    let mut blocks = BTreeMap::new();
    let mut parent_hash = B256::ZERO;
    for number in 0..=LOCAL_SAFE + 2 {
        let state_root =
            B256::with_last_byte(if number >= fork_at { 0xa0 } else { 0x40 } + number as u8);
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
                state_root,
            },
        });
        parent_hash = block.header.hash;
        blocks.insert(number, block);
    }
    blocks
}

async fn fixture(recorded_head: u64, mutation: Mutation) -> ReorgFixture {
    let original = chain(u64::MAX);
    let replacement = chain(3);
    let directory = TempDir::new().unwrap();
    let database = Arc::new(SafeDatabase::new(directory.path()).unwrap());
    for (height, l1) in [(1, 100), (recorded_head, 110)] {
        database.safe_head_updated(block_info(&original[&height]), l1_block(l1).id()).unwrap();
    }
    let expected_output = OutputRoot::from_parts(
        original[&3].header.state_root,
        storage_root(&original[&3]),
        original[&3].header.hash,
    )
    .hash();
    let canonical = if matches!(mutation, Mutation::ReorgBeforeQuery) {
        replacement.clone()
    } else {
        original.clone()
    };
    let state = ExecutionState {
        blocks: original
            .values()
            .chain(replacement.values())
            .map(|block| (block.header.hash, block.clone()))
            .collect(),
        canonical,
        database: database.clone(),
        mutation,
        recorded_head,
        replacement,
    };
    let mut module = RpcModule::new(Mutex::new(state));
    module
        .register_method("eth_getBlockByNumber", |params, state, _| {
            let (number, _full): (U64, bool) = params.parse()?;
            Ok::<_, ErrorObjectOwned>(
                state.lock().unwrap().canonical.get(&number.to::<u64>()).cloned(),
            )
        })
        .unwrap();
    module
        .register_method("eth_getBlockByHash", |params, state, _| {
            let (hash, _full): (B256, bool) = params.parse()?;
            Ok::<_, ErrorObjectOwned>(state.lock().unwrap().blocks.get(&hash).cloned())
        })
        .unwrap();
    module
        .register_method("eth_getProof", |params, state, _| {
            let (_address, _slots, block_id): (Value, Value, Value) = params.parse()?;
            Ok::<_, ErrorObjectOwned>(state.lock().unwrap().proof(block_id))
        })
        .unwrap();
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let address = server.local_addr().unwrap();
    let server = server.start(module);
    let mut config = (*rollup_config()).clone();
    config.genesis.l2 = block_info(&original[&0]).block_info.id();
    let config = Arc::new(config);
    let engine = EngineClient::new(
        RpcMock::default().provider(),
        EngineClient::rpc_client(
            format!("http://{address}").parse().unwrap(),
            JwtSecret::from_hex("00".repeat(32)).unwrap(),
        ),
        config.clone(),
    );
    let sync_state = EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
        finalized_head: Some(block_info(&original[&FINALIZED])),
        local_safe_head: Some(block_info(&original[&LOCAL_SAFE])),
        safe_head: Some(block_info(&original[&SAFE])),
        unsafe_head: Some(block_info(&original[&(LOCAL_SAFE + 2)])),
    });
    let (_, engine_state) =
        tokio::sync::watch::channel(EngineState { sync_state, ..Default::default() });
    let rpc = SuperRootRpc::new(
        engine.query_client(),
        engine_state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        Some(database),
    );
    ReorgFixture { _directory: directory, expected_output, rpc, server }
}

#[rstest]
#[case::stale_record(3, Mutation::ReorgBeforeQuery)]
#[case::sparse_stale_record(5, Mutation::ReorgBeforeQuery)]
#[case::reorg_during_proof(3, Mutation::ReorgDuringProof)]
#[case::database_reset_during_proof(3, Mutation::DatabaseResetDuringProof)]
#[tokio::test]
async fn inconsistent_history_is_never_returned_as_verified(
    #[case] recorded_head: u64,
    #[case] mutation: Mutation,
) {
    let fixture = fixture(recorded_head, mutation).await;

    let error = fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(3))).await.unwrap_err();

    assert_eq!(error.code(), -32000);
    assert!(error.message().contains("changed"), "{error}");
}

#[rstest]
#[case::exact_record(3)]
#[case::sparse_record(5)]
#[tokio::test]
async fn unchanged_history_returns_the_recorded_chain_output(#[case] recorded_head: u64) {
    let fixture = fixture(recorded_head, Mutation::None).await;

    let response = fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(3))).await.unwrap();

    assert_eq!(
        response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)].output_root,
        fixture.expected_output
    );
    assert_eq!(response.data.unwrap().verified_required_l1, l1_block(110).id());
}
