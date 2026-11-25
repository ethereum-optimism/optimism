//! Tests for the [`DenyTask`].

use crate::{
    DenyTask, DenyTaskError, EngineState, EngineSyncStateUpdate, EngineTaskError,
    EngineTaskErrorSeverity, ExecutableEngineTask, SafeLabel,
    test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block, test_rpc_block},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

const FINALIZED: u64 = 5;
const SAFE: u64 = 8;
const LOCAL_SAFE: u64 = 15;
const UNSAFE: u64 = 20;

/// The engine's chain.
struct Chain {
    blocks: Vec<op_alloy_consensus::OpBlock>,
    genesis: kona_genesis::ChainGenesis,
}

impl Chain {
    fn new() -> Self {
        let mut blocks: Vec<op_alloy_consensus::OpBlock> = Vec::new();
        for number in 0..=UNSAFE {
            let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
            blocks.push(test_l2_block(number, parent, 0));
        }
        let genesis = kona_genesis::ChainGenesis {
            l2: BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
            ..Default::default()
        };
        Self { blocks, genesis }
    }

    fn info(&self, number: u64) -> L2BlockInfo {
        L2BlockInfo::from_block_and_genesis(&self.blocks[number as usize], &self.genesis).unwrap()
    }

    fn id(&self, number: u64) -> BlockNumHash {
        self.info(number).block_info.id()
    }

    fn rpc_block(&self, number: u64) -> crate::test_utils::TestRpcBlock {
        test_rpc_block(&self.blocks[number as usize])
    }

    fn engine_state(&self) -> EngineState {
        TestEngineStateBuilder::new()
            .with_safe_label(SafeLabel::Verified { activation_timestamp: 0 })
            .with_unsafe_head(self.info(UNSAFE))
            .with_local_safe_head(self.info(LOCAL_SAFE))
            .with_safe_head(self.info(SAFE))
            .with_finalized_head(self.info(FINALIZED))
            .build()
    }

    fn cfg(&self) -> Arc<RollupConfig> {
        Arc::new(RollupConfig { block_time: 2, genesis: self.genesis, ..Default::default() })
    }
}

/// A denied block the engine derived moves the local-safe head to its parent, so derivation
/// replaces it as it derives it again. The unsafe head stays.
#[tokio::test]
async fn denied_local_safe_block_moves_local_safe_to_its_parent() {
    let chain = Chain::new();
    let mut state = chain.engine_state();
    let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(chain.info(11)),
        ..Default::default()
    });
    let cfg = chain.cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_block_header_by_number(12, Some(chain.rpc_block(12)));
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([chain.id(11).hash, true]),
        chain.rpc_block(11),
    );
    l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let task = DenyTask::new(Arc::new(client), cfg, vec![chain.id(17), chain.id(12)], vec![]);

    let err = task.execute(&mut state).await.unwrap_err();

    assert!(
        matches!(err, DenyTaskError::Rewound { denied, local_safe } if denied == chain.id(12) && local_safe == chain.id(11)),
        "{err:?}"
    );
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Rederive);
    assert_eq!(state.sync_state.local_safe_head(), chain.info(11));
    assert_eq!(state.sync_state.unsafe_head(), chain.info(UNSAFE));
    assert_eq!(state.sync_state.safe_head(), chain.info(SAFE));
    l1.assert_finished();
    l2.assert_finished();
}

/// A denied block the engine does not have at its height, or has not derived yet, changes
/// nothing: it was replaced already, or derivation replaces it when it gets there.
#[tokio::test]
async fn denied_blocks_the_engine_has_not_derived_change_nothing() {
    let chain = Chain::new();
    let mut state = chain.engine_state();
    let state_before = state;
    let cfg = chain.cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_block_header_by_number(12, Some(chain.rpc_block(12)));
    let replaced = BlockNumHash { number: 12, hash: B256::repeat_byte(0xee) };
    let task =
        DenyTask::new(Arc::new(client), cfg, vec![replaced, chain.id(LOCAL_SAFE + 1)], vec![]);

    task.execute(&mut state).await.unwrap();

    assert_eq!(state, state_before);
    l1.assert_finished();
    l2.assert_finished();
}

/// A denied block at or below the finalized head is not replaced.
#[tokio::test]
async fn denied_finalized_block_is_not_replaced() {
    let chain = Chain::new();
    let mut state = chain.engine_state();
    let state_before = state;
    let cfg = chain.cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_block_header_by_number(FINALIZED, Some(chain.rpc_block(FINALIZED)));
    let task = DenyTask::new(Arc::new(client), cfg, vec![chain.id(FINALIZED)], vec![]);

    let err = task.execute(&mut state).await.unwrap_err();

    assert!(matches!(err, DenyTaskError::Finalized { .. }), "{err:?}");
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Skip);
    assert_eq!(state, state_before);
    l1.assert_finished();
    l2.assert_finished();
}

/// A block the engine replaced on a denial a reorg voided moves the local-safe head below it, so
/// derivation derives the height again without the denial; where the engine has the denied
/// block, or has not derived the height yet, nothing moves. With a denied block below, the
/// lower height goes first.
#[rstest::rstest]
#[case::replaced(12, None, Some(12))]
#[case::the_engine_has_the_denied_block(12, Some(12), None)]
#[case::not_derived_yet(LOCAL_SAFE + 1, None, None)]
#[case::a_denied_block_below_goes_first(13, Some(11), Some(11))]
#[tokio::test]
async fn a_replacement_on_a_voided_denial_is_derived_again(
    #[case] reorged_height: u64,
    #[case] denied_height: Option<u64>,
    #[case] rewound_below: Option<u64>,
) {
    let chain = Chain::new();
    let mut state = chain.engine_state();
    let state_before = state;
    let cfg = chain.cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    // The engine's chain holds a replacement, not the block denied at the reorged height, unless
    // that height is the denied one too.
    let voided = if denied_height == Some(reorged_height) {
        chain.id(reorged_height)
    } else {
        BlockNumHash { number: reorged_height, hash: B256::repeat_byte(0xdd) }
    };
    let denied: Vec<_> = denied_height
        .into_iter()
        .filter(|height| *height != reorged_height)
        .map(|height| chain.id(height))
        .collect();
    let mut lookups: Vec<u64> = denied.iter().map(|block| block.number).collect();
    if reorged_height <= LOCAL_SAFE {
        lookups.push(reorged_height);
    }
    lookups.sort_unstable();
    for number in lookups {
        l2.expect_block_header_by_number(number, Some(chain.rpc_block(number)));
        if Some(number) == rewound_below {
            break;
        }
    }
    if let Some(number) = rewound_below {
        let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(chain.info(number - 1)),
            ..Default::default()
        });
        l2.expect_params(
            "eth_getBlockByHash",
            serde_json::json!([chain.id(number - 1).hash, true]),
            chain.rpc_block(number - 1),
        );
        l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
    }
    let task = DenyTask::new(Arc::new(client), cfg, denied.clone(), vec![voided]);

    let result = task.execute(&mut state).await;

    match rewound_below {
        Some(number) if denied.iter().any(|block| block.number == number) => assert!(
            matches!(result, Err(DenyTaskError::Rewound { denied, local_safe }) if denied == chain.id(number) && local_safe == chain.id(number - 1)),
            "{result:?}"
        ),
        Some(number) => assert!(
            matches!(result, Err(DenyTaskError::Restored { replacement, local_safe }) if replacement == chain.id(number) && local_safe == chain.id(number - 1)),
            "{result:?}"
        ),
        None => {
            result.unwrap();
            assert_eq!(state, state_before);
        }
    }
    l1.assert_finished();
    l2.assert_finished();
}
