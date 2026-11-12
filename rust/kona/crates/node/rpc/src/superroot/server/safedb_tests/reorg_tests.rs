use super::{
    execution_layer::{
        Chain, InMemoryExecutionLayer, Read, TARGET, block_info, chain, engine_state_with,
        output_root, recorded_safe_heads, rollup_config_with_genesis, serve_execution_layer,
        super_root_rpc,
    },
    *,
};
use jsonrpsee::server::ServerHandle;
use std::sync::Mutex;

#[derive(Clone, Copy, Debug)]
enum Mutation {
    DatabaseResetDuringProof,
    LocalSafeReplacedDuringProof,
    None,
    ReorgAtTargetRead,
    ReorgBeforeQuery,
    ReorgDuringProof,
    RewindBelowTargetAtTargetRead,
    /// The replacement chain is canonical from the target read until the proof read.
    SwitchAwayAndBackAroundTargetRead,
}

/// Applies `mutation` to the canonical chain, the safe head database or the engine state, at its
/// point in the query.
fn mutate_at_its_read(
    mut mutation: Mutation,
    original: Chain,
    replacement: Chain,
    database: Arc<SafeDatabase>,
    engine_state: tokio::sync::watch::Sender<EngineState>,
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
            (Mutation::LocalSafeReplacedDuringProof, Read::Proof) => {
                let replaced_local_safe = block_info(&replacement[&LOCAL_SAFE]);
                engine_state.send_modify(|state| {
                    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
                        local_safe_head: Some(replaced_local_safe),
                        ..Default::default()
                    });
                });
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

fn heads_of(chain: &Chain) -> EngineSyncStateUpdate {
    EngineSyncStateUpdate {
        finalized_head: Some(block_info(&chain[&FINALIZED])),
        local_safe_head: Some(block_info(&chain[&LOCAL_SAFE])),
        safe_head: Some(block_info(&chain[&SAFE])),
        unsafe_head: Some(block_info(&chain[&(LOCAL_SAFE + 2)])),
    }
}

async fn fixture(recorded_head: u64, mutation: Mutation) -> ReorgFixture {
    let original = chain(LOCAL_SAFE + 2, u64::MAX);
    let replacement = chain(LOCAL_SAFE + 2, TARGET);
    let (directory, database) = recorded_safe_heads(&original, recorded_head);
    let (engine_state_sender, engine_state) = engine_state_with(heads_of(&original));
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
            engine_state_sender,
            recorded_head,
        ),
    );
    let config = rollup_config_with_genesis(&original[&0]);
    let (server, query_client) =
        serve_execution_layer(Arc::new(Mutex::new(execution_layer)), config.clone()).await;
    let rpc = super_root_rpc(query_client, engine_state, config, database);
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

/// A known limit: a target below its record is read by number, so an EL that switches away and
/// back around that read returns the replacement block's output when neither the local-safe
/// head nor the record changes, as here.
#[tokio::test]
async fn a_switch_away_and_back_below_the_record_returns_the_replacement_output() {
    let fixture = fixture(LOCAL_SAFE, Mutation::SwitchAwayAndBackAroundTargetRead).await;

    let response =
        fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap();

    let replacement = chain(LOCAL_SAFE + 2, TARGET);
    assert_eq!(
        response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)].output_root,
        output_root(&replacement[&TARGET])
    );
}

/// The engine can replace the local-safe head while its output is read, so the output just read
/// no longer belongs to the local-safe chain.
#[tokio::test]
async fn a_local_safe_head_replaced_during_the_read_is_rejected() {
    let fixture = fixture(LOCAL_SAFE, Mutation::LocalSafeReplacedDuringProof).await;

    let error =
        fixture.rpc.superroot_at_timestamp(U64::from(timestamp_of(LOCAL_SAFE))).await.unwrap_err();

    assert_eq!(error.code(), -32000);
    assert!(error.message().contains("changed"), "{error}");
}

/// An EL can fail `eth_getProof` for a block off its canonical chain, so a record that was
/// reorged out must be rejected before its block is proven.
#[tokio::test]
async fn a_reorged_out_exact_record_is_rejected_without_a_proof_read() {
    let original = chain(LOCAL_SAFE + 2, u64::MAX);
    let replacement = chain(LOCAL_SAFE + 2, TARGET);
    let execution_layer = Arc::new(Mutex::new(InMemoryExecutionLayer::new(
        [&original, &replacement],
        replacement.clone(),
        |_: Read, _: &mut Chain| {},
    )));
    let config = rollup_config_with_genesis(&original[&0]);
    let (server, query_client) =
        serve_execution_layer(execution_layer.clone(), config.clone()).await;
    let (_directory, database) = recorded_safe_heads(&original, TARGET);
    let (_, engine_state) = engine_state_with(heads_of(&replacement));
    let rpc = super_root_rpc(query_client, engine_state, config, database);

    let error = rpc.superroot_at_timestamp(U64::from(timestamp_of(TARGET))).await.unwrap_err();
    let _ = server.stop();

    assert_eq!(error.code(), -32000);
    assert!(!execution_layer.lock().unwrap().request_counts().contains_key("eth_getProof"));
}
