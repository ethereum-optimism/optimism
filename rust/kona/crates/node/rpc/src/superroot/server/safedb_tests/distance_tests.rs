use super::{
    execution_layer::{
        Chain, InMemoryExecutionLayer, Read, block_info, chain, output_root,
        rollup_config_with_genesis, serve_execution_layer,
    },
    *,
};
use std::{collections::BTreeMap, sync::Mutex};

const TARGET: u64 = 3;

/// The EL requests of one query, given its block reads by hash and by number and one proof.
fn el_requests(blocks_by_hash: usize, blocks_by_number: usize) -> BTreeMap<&'static str, usize> {
    [
        ("eth_getBlockByHash", blocks_by_hash),
        ("eth_getBlockByNumber", blocks_by_number),
        ("eth_getProof", 1),
    ]
    .into_iter()
    .filter(|(_, count)| *count > 0)
    .collect()
}

/// The request count, not wall-clock time, is what this test asserts: each request is one EL
/// round trip, and a remote EL turns a count that grows with the distance into a timeout.
#[rstest]
#[case::at_the_recorded_head(0, el_requests(1, 1))]
#[case::one_below_the_recorded_head(1, el_requests(0, 3))]
#[case::a_span_batch_below_the_recorded_head(1_800, el_requests(0, 3))]
#[case::thousands_below_the_recorded_head(5_000, el_requests(0, 3))]
#[tokio::test]
async fn the_el_requests_for_a_target_do_not_grow_with_its_distance_below_the_record(
    #[case] distance: u64,
    #[case] expected_requests: BTreeMap<&'static str, usize>,
) {
    let recorded_head = TARGET + distance;
    let canonical = chain(recorded_head, u64::MAX);
    let execution_layer = Arc::new(Mutex::new(InMemoryExecutionLayer::new(
        [&canonical],
        canonical.clone(),
        |_: Read, _: &mut Chain| {},
    )));
    let config = rollup_config_with_genesis(&canonical[&0]);
    let (server, query_client) =
        serve_execution_layer(execution_layer.clone(), config.clone()).await;
    let directory = TempDir::new().unwrap();
    let database = SafeDatabase::new(directory.path()).unwrap();
    for (height, l1) in [(1, 100), (recorded_head, 110)] {
        database.safe_head_updated(block_info(&canonical[&height]), l1_block(l1).id()).unwrap();
    }
    let head = block_info(&canonical[&recorded_head]);
    let sync_state = EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
        finalized_head: Some(head),
        local_safe_head: Some(head),
        safe_head: Some(head),
        unsafe_head: Some(head),
    });
    let (_, engine_state) =
        tokio::sync::watch::channel(EngineState { sync_state, ..Default::default() });
    let rpc = SuperRootRpc::new(
        query_client,
        engine_state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        Some(Arc::new(database)),
    );

    let response = rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap();
    let _ = server.stop();

    let optimistic = &response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)];
    assert_eq!(optimistic.output_root, output_root(&canonical[&TARGET]));
    assert_eq!(response.data.unwrap().verified_required_l1, l1_block(110).id());
    assert_eq!(*execution_layer.lock().unwrap().request_counts(), expected_requests);
}
