use super::{
    execution_layer::{
        Chain, InMemoryExecutionLayer, Read, TARGET, block_info, chain, engine_state_with,
        output_root, recorded_safe_heads, rollup_config_with_genesis, serve_execution_layer,
        super_root_rpc,
    },
    *,
};
use std::{collections::BTreeMap, sync::Mutex};

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
#[case::at_the_recorded_head(0, el_requests(1, 2))]
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
    let (_directory, database) = recorded_safe_heads(&canonical, recorded_head);
    let head = block_info(&canonical[&recorded_head]);
    let (_, engine_state) = engine_state_with(EngineSyncStateUpdate {
        finalized_head: Some(head),
        local_safe_head: Some(head),
        safe_head: Some(head),
        unsafe_head: Some(head),
    });
    let rpc = super_root_rpc(query_client, engine_state, config, database);

    let response = rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap();
    let _ = server.stop();

    let optimistic = &response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)];
    assert_eq!(optimistic.output_root, output_root(&canonical[&TARGET]));
    assert_eq!(response.data.unwrap().verified_required_l1, l1_block(110).id());
    assert_eq!(*execution_layer.lock().unwrap().request_counts(), expected_requests);
}
