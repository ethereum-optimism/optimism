use super::{
    execution_layer::{
        Chain, InMemoryExecutionLayer, Read, block_info, chain, output_root,
        rollup_config_with_genesis, serve_execution_layer,
    },
    *,
};
use jsonrpsee::server::ServerHandle;
use std::sync::Mutex;

const TARGET: u64 = 3;

#[derive(Clone, Copy, Debug)]
enum Mutation {
    DatabaseResetDuringProof,
    None,
    ReorgAtTargetRead,
    ReorgBeforeQuery,
    ReorgDuringProof,
    RewindBelowTargetAtTargetRead,
    /// The replacement chain is canonical from the target read until the proof read.
    SwitchAwayAndBackAroundTargetRead,
}

/// Applies `mutation` to the canonical chain, or to the safe head database, at its point in the
/// query.
fn mutate_at_its_read(
    mut mutation: Mutation,
    original: Chain,
    replacement: Chain,
    database: Arc<SafeDatabase>,
    recorded_head: u64,
) -> impl FnMut(Read, &mut Chain) + Send + 'static {
    move |read, canonical| {
        mutation = match (mutation, read) {
            (Mutation::DatabaseResetDuringProof, Read::Proof) => {
                database.clear().unwrap();
                for (height, l1) in [(1, 100), (recorded_head, 200)] {
                    database
                        .safe_head_updated(block_info(&replacement[&height]), l1_block(l1).id())
                        .unwrap();
                }
                Mutation::None
            }
            (Mutation::ReorgAtTargetRead, Read::BlockWithTransactions) |
            (Mutation::ReorgDuringProof, Read::Proof) => {
                canonical.clone_from(&replacement);
                Mutation::None
            }
            (Mutation::RewindBelowTargetAtTargetRead, Read::BlockWithTransactions) => {
                canonical.retain(|&number, _| number < TARGET);
                Mutation::None
            }
            (Mutation::SwitchAwayAndBackAroundTargetRead, Read::BlockWithTransactions) => {
                canonical.clone_from(&replacement);
                Mutation::SwitchAwayAndBackAroundTargetRead
            }
            (Mutation::SwitchAwayAndBackAroundTargetRead, Read::Proof) => {
                canonical.clone_from(&original);
                Mutation::None
            }
            (pending, _) => pending,
        };
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

async fn fixture(recorded_head: u64, mutation: Mutation) -> ReorgFixture {
    let original = chain(LOCAL_SAFE + 2, u64::MAX);
    let replacement = chain(LOCAL_SAFE + 2, TARGET);
    let directory = TempDir::new().unwrap();
    let database = Arc::new(SafeDatabase::new(directory.path()).unwrap());
    for (height, l1) in [(1, 100), (recorded_head, 110)] {
        database.safe_head_updated(block_info(&original[&height]), l1_block(l1).id()).unwrap();
    }
    let canonical = if matches!(mutation, Mutation::ReorgBeforeQuery) {
        replacement.clone()
    } else {
        original.clone()
    };
    let execution_layer = InMemoryExecutionLayer::new(
        [&original, &replacement],
        canonical,
        mutate_at_its_read(
            mutation,
            original.clone(),
            replacement.clone(),
            database.clone(),
            recorded_head,
        ),
    );
    let config = rollup_config_with_genesis(&original[&0]);
    let (server, query_client) =
        serve_execution_layer(Arc::new(Mutex::new(execution_layer)), config.clone()).await;
    let sync_state = EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
        finalized_head: Some(block_info(&original[&FINALIZED])),
        local_safe_head: Some(block_info(&original[&LOCAL_SAFE])),
        safe_head: Some(block_info(&original[&SAFE])),
        unsafe_head: Some(block_info(&original[&(LOCAL_SAFE + 2)])),
    });
    let (_, engine_state) =
        tokio::sync::watch::channel(EngineState { sync_state, ..Default::default() });
    let rpc = SuperRootRpc::new(
        query_client,
        engine_state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        Some(database),
    );
    ReorgFixture {
        _directory: directory,
        expected_output: output_root(&original[&TARGET]),
        rpc,
        server,
    }
}

#[rstest]
#[case::stale_record(TARGET, Mutation::ReorgBeforeQuery)]
#[case::sparse_stale_record(LOCAL_SAFE, Mutation::ReorgBeforeQuery)]
#[case::reorg_at_the_target_read(LOCAL_SAFE, Mutation::ReorgAtTargetRead)]
#[case::rewind_below_the_target_at_its_read(LOCAL_SAFE, Mutation::RewindBelowTargetAtTargetRead)]
#[case::reorg_during_proof(TARGET, Mutation::ReorgDuringProof)]
#[case::database_reset_during_proof(TARGET, Mutation::DatabaseResetDuringProof)]
#[tokio::test]
async fn inconsistent_history_is_never_returned_as_verified(
    #[case] recorded_head: u64,
    #[case] mutation: Mutation,
) {
    let fixture = fixture(recorded_head, mutation).await;

    let error =
        fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap_err();

    assert_eq!(error.code(), -32000);
    assert!(error.message().contains("changed"), "{error}");
}

/// A target that is its own record is read by hash, so a chain that switches away and back
/// around that read cannot substitute the replacement block at its height.
#[rstest]
#[case::exact_record(TARGET, Mutation::None)]
#[case::sparse_record(LOCAL_SAFE, Mutation::None)]
#[case::exact_record_switched_away_and_back(TARGET, Mutation::SwitchAwayAndBackAroundTargetRead)]
#[tokio::test]
async fn a_record_canonical_at_the_end_of_the_query_returns_the_recorded_chain_output(
    #[case] recorded_head: u64,
    #[case] mutation: Mutation,
) {
    let fixture = fixture(recorded_head, mutation).await;

    let response =
        fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap();

    assert_eq!(
        response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)].output_root,
        fixture.expected_output
    );
    assert_eq!(response.data.unwrap().verified_required_l1, l1_block(110).id());
}
