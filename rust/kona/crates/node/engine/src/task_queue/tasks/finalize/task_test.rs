use crate::{
    EngineState, EngineSyncState, EngineSyncStateUpdate, EngineTaskError,
    EngineTaskErrorSeverity::{self, Critical, Skip, Temporary},
    ExecutableEngineTask, FinalizeBlockId, FinalizeTask, FinalizeTaskError,
    test_utils::{
        RpcMock, TestEngineStateBuilder, TestRpcBlock as RpcBlock, test_block_info,
        test_engine_client, test_l2_block, test_rpc_block,
    },
};
use alloy_consensus::Header;
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_rpc_types_eth::{BlockTransactions, Header as RpcHeader};
use kona_genesis::{ChainGenesis, RollupConfig};
use kona_protocol::L2BlockInfo;
use rstest::rstest;
use std::sync::Arc;

const STALE: u64 = 5;
const FINALIZED: u64 = 10;
const ADVANCED: u64 = 15;
const SAFE: u64 = 20;
const UNSAFE: u64 = 30;

/// The L2 genesis block, which carries no L1-info deposit.
fn genesis_block() -> RpcBlock {
    RpcBlock::new(RpcHeader::new(Header::default()), BlockTransactions::Full(vec![]))
}

fn rollup_config() -> Arc<RollupConfig> {
    let genesis = BlockNumHash { hash: genesis_block().header.hash, number: 0 };
    Arc::new(RollupConfig {
        genesis: ChainGenesis { l2: genesis, ..Default::default() },
        ..Default::default()
    })
}

/// An L2 block at `number`. Blocks built from different `fork` values share a height but not a
/// hash.
fn l2_block(number: u64, fork: u8) -> RpcBlock {
    test_rpc_block(&test_l2_block(number, B256::ZERO, fork))
}

fn canonical(number: u64) -> RpcBlock {
    l2_block(number, 0)
}

fn conflicting(number: u64) -> RpcBlock {
    l2_block(number, 1)
}

fn l2_block_info(block: &RpcBlock) -> L2BlockInfo {
    L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &rollup_config().genesis)
        .unwrap()
}

fn by_hash(block: &RpcBlock) -> FinalizeBlockId {
    FinalizeBlockId::ByHash(l2_block_info(block).block_info.id())
}

fn by_number(block: &RpcBlock) -> FinalizeBlockId {
    FinalizeBlockId::ByNumber(block.header.number)
}

fn established() -> L2BlockInfo {
    l2_block_info(&canonical(FINALIZED))
}

fn uninitialized() -> L2BlockInfo {
    L2BlockInfo::default()
}

fn engine_state(finalized_head: L2BlockInfo) -> EngineState {
    TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(UNSAFE))
        .with_safe_head(test_block_info(SAFE))
        .with_finalized_head(finalized_head)
        .build()
}

fn with_finalized_head(state: &EngineState, finalized_head: L2BlockInfo) -> EngineSyncState {
    state.sync_state.apply_update(EngineSyncStateUpdate {
        finalized_head: Some(finalized_head),
        ..Default::default()
    })
}

async fn finalize(
    target: FinalizeBlockId,
    state: &mut EngineState,
    configure_engine: impl FnOnce(&RpcMock),
) -> Result<(), FinalizeTaskError> {
    let cfg = rollup_config();
    let (engine_client, l1, l2) = test_engine_client(cfg.clone());
    configure_engine(&l2);

    let result = FinalizeTask::new(Arc::new(engine_client), cfg, target).execute(state).await;

    l1.assert_finished();
    l2.assert_finished();
    result
}

const fn is_behind(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::BehindFinalizedHead { finalized: FINALIZED, target: STALE })
}

const fn is_conflict(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::ConflictsWithFinalizedHead { .. })
}

const fn is_not_canonical(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::NotCanonical { .. })
}

const fn is_number_mismatch(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::BlockNumberMismatch { requested: ADVANCED, .. })
}

const fn is_not_safe(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::BlockNotSafe)
}

const fn is_not_yet_available(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::NotYetAvailable { .. })
}

const fn is_block_not_found(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::BlockNotFound(ADVANCED))
}

const fn is_transport_error(err: &FinalizeTaskError) -> bool {
    matches!(err, FinalizeTaskError::TransportError(_))
}

/// A target above the finalized head, pinned to the engine's canonical block at its height.
fn pinned_target() -> FinalizeBlockId {
    by_hash(&canonical(ADVANCED))
}

/// A target above the finalized head, identified by number only.
fn numbered_target() -> FinalizeBlockId {
    by_number(&canonical(ADVANCED))
}

/// What the engine answers for a block it cannot serve.
#[derive(Clone, Copy, Debug)]
enum Unavailable {
    ErrorResponse,
    Null,
}

fn expect_unavailable_block(l2: &RpcMock, number: u64, answer: Unavailable) {
    match answer {
        Unavailable::ErrorResponse => l2.expect_error_code("eth_getBlockByNumber", -32000),
        Unavailable::Null => l2.expect_block_by_number(number, None),
    }
}

/// Targets behind the finalized head, a pinned hash that conflicts with it, and targets above the
/// safe head are skipped before asking the execution layer for the block: the test engine expects
/// no request at all. A reset can lower the safe head below a target that was safe when it was
/// requested, and a later request finalizes it.
#[rstest]
#[case::stale_by_hash(by_hash(&canonical(STALE)), is_behind)]
#[case::stale_by_number(by_number(&canonical(STALE)), is_behind)]
#[case::conflicting_by_hash(by_hash(&conflicting(FINALIZED)), is_conflict)]
#[case::above_safe_by_hash(by_hash(&canonical(UNSAFE)), is_not_safe)]
#[case::above_safe_by_number(by_number(&canonical(UNSAFE)), is_not_safe)]
#[tokio::test]
async fn finalize_task_skips_target_without_lookup(
    #[case] target: FinalizeBlockId,
    #[case] is_expected_error: fn(&FinalizeTaskError) -> bool,
) {
    let mut state = engine_state(established());
    let original_state = state;

    let err = finalize(target, &mut state, |_| {}).await.unwrap_err();

    assert!(is_expected_error(&err), "{err:?}");
    assert_eq!(err.severity(), Skip);
    assert_eq!(state, original_state);
}

/// A target above the safe head is refused before asking the execution layer for the block: the
/// test engine expects no request at all.
#[tokio::test]
async fn finalize_task_refuses_target_above_safe_head_without_lookup() {
    let mut state = engine_state(established());
    let original_state = state;

    let err = finalize(by_hash(&canonical(SAFE + 5)), &mut state, |_| {}).await.unwrap_err();

    assert!(matches!(err, FinalizeTaskError::BlockNotSafe), "{err:?}");
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Critical);
    assert_eq!(state, original_state);
}

/// A target whose block differs from the engine's canonical block is skipped. A pinned hash is
/// never replaced by the canonical block at its height, even above the finalized head, where a
/// delegate on another fork names a block the engine does not consider canonical.
#[rstest]
#[case::conflicting_by_number(by_number(&conflicting(FINALIZED)), conflicting(FINALIZED), is_conflict)]
#[case::other_fork_by_hash(by_hash(&conflicting(ADVANCED)), canonical(ADVANCED), is_not_canonical)]
#[tokio::test]
async fn finalize_task_skips_target_off_the_canonical_chain(
    #[case] target: FinalizeBlockId,
    #[case] served_block: RpcBlock,
    #[case] is_expected_error: fn(&FinalizeTaskError) -> bool,
) {
    let mut state = engine_state(established());
    let original_state = state;

    let err = finalize(target, &mut state, |l2| {
        l2.expect_block_by_number(target.number(), Some(served_block))
    })
    .await
    .unwrap_err();

    assert!(is_expected_error(&err), "{err:?}");
    assert_eq!(err.severity(), Skip);
    assert_eq!(state, original_state);
}

/// A canonical target above the finalized head becomes the finalized head through one forkchoice
/// update. The canonical block at the finalized height is already finalized, so it needs no update.
#[rstest]
#[case::advancing_by_hash(established(), by_hash, canonical(ADVANCED), 1)]
#[case::advancing_by_number(established(), by_number, canonical(ADVANCED), 1)]
#[case::identical_by_number(established(), by_number, canonical(FINALIZED), 0)]
#[case::uninitialized_adopts_genesis(uninitialized(), by_hash, genesis_block(), 1)]
#[tokio::test]
async fn finalize_task_finalizes_canonical_target(
    #[case] finalized_head: L2BlockInfo,
    #[case] identify: fn(&RpcBlock) -> FinalizeBlockId,
    #[case] served_block: RpcBlock,
    #[case] forkchoice_updates: usize,
) {
    let mut state = engine_state(finalized_head);
    let target = identify(&served_block);
    let expected_sync_state = with_finalized_head(&state, l2_block_info(&served_block));

    finalize(target, &mut state, |l2| {
        l2.expect_block_by_number(target.number(), Some(served_block));
        for _ in 0..forkchoice_updates {
            l2.expect_forkchoice_updated(
                expected_sync_state.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        }
    })
    .await
    .unwrap();

    assert_eq!(state.sync_state, expected_sync_state);
}

/// A target pinned to the finalized head is already finalized, so it needs no lookup.
#[tokio::test]
async fn finalize_task_skips_lookup_for_finalized_head() {
    let mut state = engine_state(established());
    let original_state = state;

    finalize(by_hash(&canonical(FINALIZED)), &mut state, |_| {}).await.unwrap();

    assert_eq!(state, original_state);
}

/// The engine serving a block at a height other than the requested one stops the node instead of
/// finalizing anything. A misplaced block could otherwise bypass the guards against moving
/// finality backward.
#[rstest]
#[case::stale_block_by_number(by_number(&canonical(ADVANCED)), canonical(STALE))]
#[case::higher_block_by_number(by_number(&canonical(ADVANCED)), canonical(SAFE))]
#[case::stale_block_by_hash(by_hash(&canonical(ADVANCED)), canonical(STALE))]
#[tokio::test]
async fn finalize_task_stops_on_block_at_unrequested_height(
    #[case] target: FinalizeBlockId,
    #[case] served_block: RpcBlock,
) {
    let mut state = engine_state(established());
    let original_state = state;

    let err = finalize(target, &mut state, |l2| {
        l2.expect_block_by_number(target.number(), Some(served_block))
    })
    .await
    .unwrap_err();

    assert!(is_number_mismatch(&err), "{err:?}");
    assert_eq!(err.severity(), Critical);
    assert_eq!(state, original_state);
}

/// While the execution layer catches up, it has no block at a pinned target's height yet, and may
/// answer null or an error. Either way the target is skipped, so it neither stops the node nor
/// holds the queue until its upstream resubmits it. A by-number target is below the safe head, so
/// the engine must have it: its absence stops the node and an error response is retried.
#[rstest]
#[case::pinned_null(pinned_target(), Unavailable::Null, is_not_yet_available, Skip)]
#[case::pinned_error_response(
    pinned_target(),
    Unavailable::ErrorResponse,
    is_not_yet_available,
    Skip
)]
#[case::by_number_null(numbered_target(), Unavailable::Null, is_block_not_found, Critical)]
#[case::by_number_error_response(
    numbered_target(),
    Unavailable::ErrorResponse,
    is_transport_error,
    Temporary
)]
#[tokio::test]
async fn finalize_task_handles_block_the_engine_cannot_serve(
    #[case] target: FinalizeBlockId,
    #[case] answer: Unavailable,
    #[case] is_expected_error: fn(&FinalizeTaskError) -> bool,
    #[case] severity: EngineTaskErrorSeverity,
) {
    let mut state = engine_state(established());
    let original_state = state;

    let err =
        finalize(target, &mut state, |l2| expect_unavailable_block(l2, target.number(), answer))
            .await
            .unwrap_err();

    assert!(is_expected_error(&err), "{err:?}");
    assert_eq!(err.severity(), severity, "{err:?}");
    assert_eq!(state, original_state);
}

/// A pinned target skipped while the engine catches up finalizes once its upstream resubmits it
/// and the engine has the block.
#[tokio::test]
async fn resubmitted_pinned_target_finalizes_once_engine_has_it() {
    let mut state = engine_state(established());
    let served_block = canonical(ADVANCED);
    let target = by_hash(&served_block);
    let expected_sync_state = with_finalized_head(&state, l2_block_info(&served_block));

    let err = finalize(target, &mut state, |l2| l2.expect_block_by_number(ADVANCED, None))
        .await
        .unwrap_err();
    assert!(matches!(err, FinalizeTaskError::NotYetAvailable { .. }), "{err:?}");

    finalize(target, &mut state, |l2| {
        l2.expect_block_by_number(ADVANCED, Some(served_block));
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
    })
    .await
    .unwrap();

    assert_eq!(state.sync_state, expected_sync_state);
}
