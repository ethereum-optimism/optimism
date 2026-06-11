//! Tests for [`FinalizeTask::execute`].

use crate::{
    EngineTaskExt, FinalizeBlockId, FinalizeTask, FinalizeTaskError,
    test_utils::{TestEngineStateBuilder, test_engine_client_builder},
};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::b256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;

/// When the engine receives a `ByHash` finalize request for a block hash it doesn't have,
/// [`FinalizeTask`] must fail with [`FinalizeTaskError::BlockNotFound`] rather than silently
/// finalize whatever it happens to have at the same height.
///
/// Reproduces the bug at [`crate::FinalizeTask`]: with the old `block_number: u64` field, the task
/// looked up the block by number, so an upstream-driven `(N, H_b)` request would silently finalize
/// `H_a` if the engine's canonical chain disagrees. EL finalization is irreversible, so this is
/// unrecoverable.
///
/// Baseline (Phase 1, by-number lookup still in place): the task finds the engine's stale block at
/// number `N` and returns a different error (or `Ok`), never [`FinalizeTaskError::BlockNotFound`].
/// Post-fix (Phase 2, by-hash lookup): the lookup of `H_b` returns `None` and the task fails
/// loudly.
#[tokio::test]
async fn finalize_task_by_hash_errors_when_engine_lacks_hash() {
    const N: u64 = 10;
    let hash_a = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let hash_b = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");

    // The engine has H_a at height N. Register the block under its number key so a by-number
    // lookup finds it; do *not* register it under any hash key so a by-hash lookup of H_b returns
    // None.
    let block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> =
        alloy_rpc_types_eth::Block {
            header: alloy_rpc_types_eth::Header {
                hash: hash_a,
                inner: alloy_consensus::Header {
                    number: N,
                    parent_hash: b256!(
                        "0202020202020202020202020202020202020202020202020202020202020202"
                    ),
                    timestamp: N * 2,
                    ..Default::default()
                },
                ..Default::default()
            },
            ..Default::default()
        };

    let cfg = Arc::new(RollupConfig::default());
    let engine_client = test_engine_client_builder()
        .with_config(cfg.clone())
        .with_l2_block(BlockId::Number(N.into()), block)
        .build();

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
}
