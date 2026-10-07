use crate::{
    EngineTaskExt, FinalizeBlockId, FinalizeTask, FinalizeTaskError,
    test_utils::{TestEngineStateBuilder, test_engine_client},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::b256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;

/// When the engine receives a `ByHash` finalize request for a block hash it doesn't have,
/// [`FinalizeTask`] must fail with [`FinalizeTaskError::BlockNotFound`] rather than silently
/// finalize whatever it happens to have at the same height.
///
/// Assert the exact by-hash RPC lookup, so a by-number lookup cannot silently finalize a
/// different canonical block at the same height.
#[tokio::test]
async fn finalize_task_by_hash_errors_when_engine_lacks_hash() {
    const N: u64 = 10;
    let hash_a = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let hash_b = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");

    let cfg = Arc::new(RollupConfig::default());
    let (engine_client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([hash_b, true]),
        Option::<alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>>::None,
    );

    // Place the safe head at N so the sanity check (`safe_head.number >= block_id.number`) passes
    // and `execute()` reaches the lookup we care about.
    let safe_head = L2BlockInfo {
        block_info: BlockInfo {
            number: N,
            hash: hash_a,
            parent_hash: Default::default(),
            timestamp: N * 2,
        },
        l1_origin: BlockNumHash::default(),
        seq_num: 0,
    };
    let mut state =
        TestEngineStateBuilder::new().with_unsafe_head(safe_head).with_safe_head(safe_head).build();

    let task = FinalizeTask::new(
        Arc::new(engine_client),
        cfg,
        FinalizeBlockId::ByHash(BlockNumHash { number: N, hash: hash_b }),
    );

    let result = task.execute(&mut state).await;

    assert!(
        matches!(result, Err(FinalizeTaskError::BlockNotFound(n)) if n == N),
        "expected BlockNotFound({N}) — got {result:?}. The by-hash lookup must fail loudly when \
         the engine lacks the requested hash; instead, the task either succeeded (finalizing the \
         wrong block) or surfaced a different error."
    );
    l1.assert_finished();
    l2.assert_finished();
}
