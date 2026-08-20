use alloy_genesis::Genesis;
use reth_e2e_test_utils::setup_engine;
use reth_node_api::TreeConfig;
use reth_node_metrics::recorder::install_prometheus_recorder;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{
    OpNode,
    utils::{advance_chain, optimism_payload_attributes},
};
use reth_provider::BlockNumReader;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Tests that the sparse trie pipeline can be shared with the payload builder.
///
/// Mirrors reth's `test_share_sparse_trie_with_payload_builder`
/// (`crates/ethereum/node/tests/e2e/eth.rs`): enables both
/// `share_execution_cache_with_payload_builder` and `share_sparse_trie_with_payload_builder`, then
/// advances multiple blocks. Each FCU spawns a state root task that the payload builder uses for
/// incremental state root computation instead of the blocking trie walk in `BlockBuilder::finish`.
///
/// The test validates that all blocks are successfully built and their state roots are accepted by
/// the engine (`newPayload` returns VALID): a root that disagreed with the engine's own computation
/// would fail the payload submission inside `advance_chain`.
///
/// It also asserts that the shared trie produced the root, rather than only that the chain
/// advanced. Upstream's test stops at the block count, which a builder that silently ignores the
/// handle satisfies just as well.
///
/// Calls `setup_engine` directly rather than op-reth's `setup()`, which hardcodes
/// `Default::default()` in the `TreeConfig` slot and so cannot reach the flags.
#[tokio::test]
async fn test_share_sparse_trie_with_payload_builder() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let tree_config = TreeConfig::default()
        .with_share_execution_cache_with_payload_builder(true)
        .with_share_sparse_trie_with_payload_builder(true);

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json")).unwrap();
    let (mut nodes, wallet) = setup_engine::<OpNode>(
        1,
        Arc::new(
            OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
        ),
        false,
        tree_config,
        optimism_payload_attributes,
    )
    .await?;

    let mut node = nodes.pop().unwrap();
    let wallet = Arc::new(Mutex::new(wallet));

    let num_blocks = 5;
    advance_chain(num_blocks, &mut node, wallet).await?;

    let best_block = node.inner.provider.best_block_number()?;
    assert_eq!(best_block, num_blocks as u64, "Expected {num_blocks} blocks, got {best_block}");

    let from_shared_trie = counter("state_root_from_shared_trie_total");
    let fallbacks = counter("state_root_fallback_total");
    assert!(
        from_shared_trie >= num_blocks as u64,
        "expected at least {num_blocks} roots from the shared trie, got {from_shared_trie} \
         ({fallbacks} fallbacks)"
    );

    Ok(())
}

/// Reads an `optimism_payload_builder` counter out of the process-wide Prometheus recorder, which
/// the node installs while launching (`with_prometheus_server`). Absent counters read as zero: a
/// metric only appears once it has been touched.
///
/// The recorder is global to the test binary, so this is a running total across every node the
/// binary has launched. Assert lower bounds against it, never equality.
fn counter(name: &str) -> u64 {
    let rendered = install_prometheus_recorder().handle().render();
    let key = format!("reth_optimism_payload_builder_{name} ");
    rendered
        .lines()
        .find_map(|line| line.strip_prefix(&key))
        .and_then(|value| value.trim().parse().ok())
        .unwrap_or(0)
}
