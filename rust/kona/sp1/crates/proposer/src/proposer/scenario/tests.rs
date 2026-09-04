use std::{
    num::{NonZeroU64, NonZeroUsize},
    sync::{
        Arc, Mutex as StdMutex,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    time::Duration,
};

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, U256};
use async_trait::async_trait;
use kona_sp1_host_utils::metrics::MetricsListen;
use kona_sp1_super_range_executor::{
    BlockId as SuperBlockId, SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1,
};
use tokio::sync::{Notify, oneshot};

use super::{NamedBarrier, ScenarioControl, ScenarioError, world::*};
use crate::{
    ZK_GAME_TYPE,
    config::{
        PrestatePrograms, ProofProviderConfig, ProofProviderKind, ProposalSafety, ProposerConfig,
        RangeSplitCount,
    },
    contract::{GameStatus, ProposalStatus, ZKGameArgs},
    ports::{
        ActionExecutor, AnchorRoot, BondState, ClaimPreflight, FactoryGame, GameClaim,
        GameCreationReceipt, GameIdentity, GameLifecycle, GameStanding, GameValidity, L1BlockRef,
        L1View, NonceState, ProofEngine, ProofInputs, ProposalHorizon, QueryTime,
        SuperRootAtTimestamp, SuperRootSource, WithdrawalState,
    },
    prover::ProofKeys,
    proving::GameProofInputs,
    signer::NUM_CONFIRMATIONS,
    superroot::SuperRootAt,
};

use crate::proposer::{
    CompactGameSummary, InFlightCreation, OperationSummary, Proposer, ProvingPurpose,
    SyncDisposition, TaskClass, TaskCompletionOutcome, TaskFailureClass, TaskId, TaskSuccess,
};

const HEAD_NUMBER: u64 = 1;
const HEAD_TIMESTAMP: u64 = 1_000;

#[derive(Default)]
struct ScenarioL1View {
    latest_head_calls: AtomicU64,
    latest_head_notify: Notify,
    fail_latest_head: AtomicBool,
    fail_respected_game_type: AtomicBool,
    fail_latest_l1_timestamp_on: AtomicU64,
    latest_l1_timestamp_calls: AtomicU64,
    release_barrier: StdMutex<Option<NamedBarrier>>,
    task_finished: StdMutex<Option<oneshot::Receiver<()>>>,
}

impl ScenarioL1View {
    fn new() -> Self {
        Self::default()
    }

    async fn wait_for_cycles(&self, expected: u64) {
        loop {
            let notified = self.latest_head_notify.notified();
            if self.latest_head_calls.load(Ordering::Relaxed) >= expected {
                return;
            }
            notified.await;
        }
    }
}

#[async_trait]
impl L1View for ScenarioL1View {
    async fn latest_head(&self) -> anyhow::Result<Option<L1BlockRef>> {
        self.latest_head_calls.fetch_add(1, Ordering::Relaxed);
        self.latest_head_notify.notify_waiters();
        if self.fail_latest_head.load(Ordering::Relaxed) {
            anyhow::bail!("latest head unavailable")
        }
        Ok(Some(L1BlockRef { hash: B256::ZERO, number: HEAD_NUMBER, timestamp: HEAD_TIMESTAMP }))
    }

    async fn block_ref(&self, number: u64) -> anyhow::Result<Option<L1BlockRef>> {
        Ok(Some(L1BlockRef { hash: B256::ZERO, number, timestamp: HEAD_TIMESTAMP }))
    }

    async fn registered_game_args(&self, _block: BlockId) -> anyhow::Result<ZKGameArgs> {
        Ok(ZKGameArgs {
            absolute_prestate: B256::ZERO,
            verifier: Address::ZERO,
            max_challenge_duration: 3_600,
            max_prove_duration: 3_600,
            challenger_bond: U256::ZERO,
            anchor_state_registry: Address::ZERO,
            weth: Address::ZERO,
        })
    }

    async fn anchor_root(&self, _registry: Address, _block: BlockId) -> anyhow::Result<AnchorRoot> {
        Ok(AnchorRoot { root: B256::left_padding_from(&[1]), sequence_number: U256::ZERO })
    }

    async fn latest_game_index(&self, _block: BlockId) -> anyhow::Result<Option<U256>> {
        Ok(None)
    }

    async fn registered_anchor_game(&self, _block: BlockId) -> anyhow::Result<Address> {
        Ok(Address::ZERO)
    }

    async fn factory_game(&self, _index: U256, _block: BlockId) -> anyhow::Result<FactoryGame> {
        unreachable!("scenario fixture does not discover factory games")
    }

    async fn game_claim(&self, _game: Address, _block: BlockId) -> anyhow::Result<GameClaim> {
        unreachable!("scenario fixture does not discover factory games")
    }

    async fn game_identity(&self, _game: Address, _block: BlockId) -> anyhow::Result<GameIdentity> {
        unreachable!("scenario fixture does not discover factory games")
    }

    async fn game_validity(&self, _game: Address, _block: BlockId) -> anyhow::Result<GameValidity> {
        unreachable!("scenario fixture does not discover factory games")
    }

    async fn game_lifecycle(
        &self,
        _game: Address,
        _registry: Address,
        _block: BlockId,
    ) -> anyhow::Result<GameLifecycle> {
        unreachable!("scenario fixture does not refresh factory games")
    }

    async fn parent_game_status(&self, _parent_index: u32, _block: BlockId) -> anyhow::Result<u8> {
        Ok(GameStatus::DefenderWins as u8)
    }

    async fn bond_state(
        &self,
        _game: Address,
        _weth: Address,
        _proposer: Address,
        _block: BlockId,
    ) -> anyhow::Result<BondState> {
        Ok(BondState {
            credit: U256::ZERO,
            withdrawal_amount: U256::ZERO,
            withdrawal_timestamp: U256::ZERO,
            delay: U256::ZERO,
        })
    }

    async fn init_bond(&self) -> anyhow::Result<U256> {
        Ok(U256::ZERO)
    }

    async fn game_status(&self, _game: Address) -> anyhow::Result<u8> {
        Ok(GameStatus::InProgress as u8)
    }

    async fn claim_preflight(
        &self,
        _game: Address,
        _weth: Address,
        _proposer: Address,
    ) -> ClaimPreflight {
        ClaimPreflight {
            credit: Ok(U256::ZERO),
            withdrawal: Ok(WithdrawalState { amount: U256::ZERO, timestamp: U256::ZERO }),
        }
    }

    async fn weth_delay(&self, _weth: Address) -> anyhow::Result<U256> {
        Ok(U256::ZERO)
    }

    async fn game_by_uuid(
        &self,
        _root_claim: B256,
        _extra_data: Vec<u8>,
    ) -> anyhow::Result<Address> {
        Ok(Address::ZERO)
    }

    async fn game_creator(&self, _game: Address) -> anyhow::Result<Address> {
        Ok(Address::ZERO)
    }

    async fn nonce_state(&self, _proposer: Address) -> anyhow::Result<NonceState> {
        Ok(NonceState { pending: 0, latest: 0 })
    }

    async fn respected_game_type(&self, _block: BlockId) -> anyhow::Result<u32> {
        let release_barrier = self.release_barrier.lock().unwrap().take();
        let task_finished = self.task_finished.lock().unwrap().take();
        if let Some(release_barrier) = release_barrier {
            release_barrier.release();
        }
        if let Some(task_finished) = task_finished {
            let _ = task_finished.await;
        }
        if self.fail_respected_game_type.load(Ordering::Relaxed) {
            anyhow::bail!("respected game type unavailable")
        }
        Ok(ZK_GAME_TYPE)
    }

    async fn parent_standing(
        &self,
        _game: Address,
        _registry: Address,
    ) -> anyhow::Result<GameStanding> {
        Ok(GameStanding { blacklisted: false, retired: false })
    }

    async fn game_standing(
        &self,
        _game: Address,
        _registry: Address,
    ) -> anyhow::Result<GameStanding> {
        Ok(GameStanding { blacklisted: false, retired: false })
    }

    async fn proof_status(&self, _game: Address) -> anyhow::Result<u8> {
        Ok(ProposalStatus::Challenged as u8)
    }

    async fn proof_inputs(&self, _game: Address) -> anyhow::Result<ProofInputs> {
        Ok(ProofInputs::default())
    }

    async fn anchor_state_registry(&self, _game: Address) -> anyhow::Result<Address> {
        Ok(Address::ZERO)
    }

    async fn latest_l1_timestamp(&self) -> anyhow::Result<u64> {
        let call = self.latest_l1_timestamp_calls.fetch_add(1, Ordering::Relaxed) + 1;
        if self.fail_latest_l1_timestamp_on.load(Ordering::Relaxed) == call {
            anyhow::bail!("latest timestamp unavailable")
        }
        Ok(HEAD_TIMESTAMP)
    }
}

struct FixedQueryTime;

impl QueryTime for FixedQueryTime {
    fn unix_timestamp(&self) -> anyhow::Result<u64> {
        Ok(3_600)
    }
}

struct FixedSuperRootSource;

#[async_trait]
impl SuperRootSource for FixedSuperRootSource {
    async fn proposal_horizon(&self, _timestamp: u64) -> anyhow::Result<ProposalHorizon> {
        Ok(ProposalHorizon { safe_timestamp: 3_600, finalized_timestamp: 3_600 })
    }

    async fn super_root_at_timestamp(
        &self,
        timestamp: u64,
    ) -> anyhow::Result<SuperRootAtTimestamp> {
        let root = B256::left_padding_from(&timestamp.to_be_bytes());
        Ok(SuperRootAtTimestamp {
            response: SuperRootAtTimestampResponse {
                current_l1: SuperBlockId::default(),
                current_safe_timestamp: timestamp,
                current_local_safe_timestamp: timestamp,
                current_finalized_timestamp: timestamp,
                optimistic_at_timestamp: Default::default(),
                chain_ids: Vec::new(),
                data: Some(SuperRootResponseData {
                    verified_required_l1: SuperBlockId::default(),
                    super_v1: SuperV1 { timestamp, chains: Vec::new() },
                    super_root: root,
                }),
            },
            root: Some(SuperRootAt { proof_bytes: vec![1], super_root: root }),
        })
    }
}

struct NoopProofEngine;

#[async_trait]
impl ProofEngine for NoopProofEngine {
    async fn prove(
        &self,
        _game_address: Address,
        _keys: Option<Arc<ProofKeys>>,
        _game: GameProofInputs,
        _responses: Vec<SuperRootAtTimestampResponse>,
    ) -> anyhow::Result<Vec<u8>> {
        Ok(vec![1])
    }

    fn clear(&self, _game_address: Address) {}
}

struct NoopActionExecutor;

#[async_trait]
impl ActionExecutor for NoopActionExecutor {
    async fn create_game(
        &self,
        _root_claim: B256,
        _extra_data: Vec<u8>,
        _init_bond: U256,
    ) -> anyhow::Result<GameCreationReceipt> {
        Ok(GameCreationReceipt {
            game_address: Address::left_padding_from(&[0xc1]),
            transaction_hash: B256::left_padding_from(&[0xc2]),
        })
    }

    async fn prove_game(&self, _game: Address, _proof: Vec<u8>) -> anyhow::Result<B256> {
        Ok(B256::left_padding_from(&[0xc3]))
    }

    async fn resolve_game(&self, _game: Address) -> anyhow::Result<B256> {
        Ok(B256::ZERO)
    }

    async fn claim_credit(&self, _game: Address, _recipient: Address) -> anyhow::Result<B256> {
        Ok(B256::ZERO)
    }
}

fn test_config(fetch_interval: u64) -> ProposerConfig {
    ProposerConfig {
        l1_rpc: "http://127.0.0.1:1".parse().unwrap(),
        superroot_rpcs: vec!["http://127.0.0.1:1".parse().unwrap()],
        factory_address: Address::ZERO,
        prestates_url: "file:///nonexistent".parse().unwrap(),
        proposal_interval_seconds: 3_600,
        proposal_safety: ProposalSafety::Finalized,
        fetch_interval,
        metrics_listen: MetricsListen::Disabled,
        sync_l1_confirmations: 0,
        tx_confirmation_timeout: 60,
        max_fee_per_gas: None,
        max_priority_fee_per_gas: None,
        proof_provider: ProofProviderKind::Mock,
        l1_beacon_rpc: "http://127.0.0.1:1".parse().unwrap(),
        l2_rpcs: vec!["http://127.0.0.1:1".parse().unwrap()],
        rollup_config_paths: None,
        l1_config_path: None,
        dependency_set_path: None,
        range_split_count: RangeSplitCount::one(),
        max_concurrent_range_proofs: NonZeroUsize::MIN,
        max_concurrent_defense_tasks: NonZeroU64::new(8).unwrap(),
        fast_finality_mode: false,
        fast_finality_proving_limit: NonZeroU64::MIN,
        proof_provider_config: ProofProviderConfig {
            timeout: 14_400,
            network_calls_timeout: 15,
            auction_timeout: 60,
            range_proof_strategy: sp1_sdk::network::FulfillmentStrategy::Reserved,
            agg_proof_strategy: sp1_sdk::network::FulfillmentStrategy::Reserved,
            range_cycle_limit: 1,
            range_gas_limit: 1,
            agg_cycle_limit: 1,
            agg_gas_limit: 1,
            max_price_per_pgu: Some(NonZeroU64::MIN),
            min_auction_period: 1,
        },
    }
}

async fn proposer_with(config: ProposerConfig, l1_view: Arc<ScenarioL1View>) -> Arc<Proposer> {
    let prestates = Arc::new(crate::proposer::PrestateCache::new(config.prestates_url.clone()));
    prestates
        .insert_for_tests(
            B256::ZERO,
            PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
        )
        .await;
    let proposer = Arc::new(
        Proposer::new_with_dependencies(
            config,
            Address::ZERO,
            l1_view,
            Arc::new(FixedQueryTime),
            Arc::new(FixedSuperRootSource),
            Arc::new(NoopProofEngine),
            Arc::new(NoopActionExecutor),
            prestates,
        )
        .await
        .unwrap(),
    );
    proposer.validate_and_init().await.unwrap();
    proposer
}

fn challenged_game(index: u64, deadline: u64) -> crate::proposer::Game {
    crate::proposer::Game {
        index: U256::from(index),
        address: Address::left_padding_from(&index.to_be_bytes()),
        parent_index: u32::MAX,
        l2_sequence_number: index * 3_600,
        status: GameStatus::InProgress,
        proposal_status: ProposalStatus::Challenged,
        deadline,
        should_attempt_to_resolve: false,
        should_attempt_to_claim_bond: false,
        absolute_prestate: B256::ZERO,
        creator: Address::ZERO,
        weth: Address::ZERO,
        anchor_state_registry: Address::ZERO,
    }
}

async fn insert_task(
    proposer: &Proposer,
    operation: OperationSummary,
    handle: tokio::task::JoinHandle<anyhow::Result<TaskSuccess>>,
) -> TaskId {
    let task_id = allocate_task_id(proposer);
    insert_allocated_task(proposer, task_id, operation, handle).await;
    task_id
}

fn allocate_task_id(proposer: &Proposer) -> TaskId {
    TaskId::allocate(&proposer.next_task_id)
}

async fn insert_allocated_task(
    proposer: &Proposer,
    task_id: TaskId,
    operation: OperationSummary,
    handle: tokio::task::JoinHandle<anyhow::Result<TaskSuccess>>,
) {
    proposer.tasks.lock().await.insert(task_id, (handle, operation));
}

struct ParkedTask {
    task_id: TaskId,
    barrier: NamedBarrier,
}

impl ParkedTask {
    async fn insert(proposer: &Proposer, operation: OperationSummary, barrier_name: &str) -> Self {
        let task_id = allocate_task_id(proposer);
        let barrier = NamedBarrier::new(barrier_name);
        let task_barrier = barrier.clone();
        insert_allocated_task(
            proposer,
            task_id,
            operation,
            tokio::spawn(async move {
                task_barrier.park(task_id).await;
                Ok(TaskSuccess::Completed)
            }),
        )
        .await;
        barrier.wait_until_reached().await;
        Self { task_id, barrier }
    }

    async fn record(&self, control: &mut ScenarioControl) {
        control.record_parked(self.task_id, &self.barrier).await.unwrap();
    }

    fn release(&self) {
        self.barrier.release();
    }
}

#[tokio::test(start_paused = true)]
async fn manual_tick_runs_without_fetch_interval_delay() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(86_400), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let before = tokio::time::Instant::now();

    let result = control.tick().await.unwrap();

    assert_eq!(tokio::time::Instant::now(), before);
    assert_eq!(result.snapshot.sync_disposition, SyncDisposition::Advanced);
    assert!(
        result.scheduled.iter().any(|scheduled| {
            matches!(scheduled.operation, OperationSummary::ProposeGame { .. })
        })
    );
    assert!(matches!(result.scheduled[0].operation, OperationSummary::ProposeGame { .. }));
    assert!(matches!(result.scheduled[1].operation, OperationSummary::ResolutionSweep));
    assert!(matches!(result.scheduled[2].operation, OperationSummary::ClaimSweep));
    assert!(result.scheduled.iter().all(|scheduled| {
        result.snapshot.active_tasks.iter().all(|active| active.task_id != scheduled.task_id)
    }));
    let tasks = proposer.tasks.lock().await;
    for scheduled in result.scheduled {
        assert_eq!(
            tasks.get(&scheduled.task_id).map(|(_, operation)| operation),
            Some(&scheduled.operation)
        );
    }
}

#[tokio::test(start_paused = true)]
async fn run_starts_immediately_then_waits_for_fetch_interval() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(600), view.clone()).await;
    let runner = tokio::spawn(proposer.run());

    view.wait_for_cycles(1).await;
    assert_eq!(view.latest_head_calls.load(Ordering::Relaxed), 1);

    tokio::time::advance(Duration::from_secs(600)).await;
    view.wait_for_cycles(2).await;
    assert_eq!(view.latest_head_calls.load(Ordering::Relaxed), 2);

    runner.abort();
    assert!(runner.await.unwrap_err().is_cancelled());
}

#[tokio::test]
async fn finished_tasks_are_replaced_in_the_same_tick() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    proposer.sync_state().await.unwrap();
    let (release_tx, release_rx) = oneshot::channel();
    let (done_tx, done_rx) = oneshot::channel();
    let completed = insert_task(
        &proposer,
        OperationSummary::ResolutionSweep,
        tokio::spawn(async move {
            let _ = release_rx.await;
            let _ = done_tx.send(());
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    let _ = release_tx.send(());
    let _ = done_rx.await;

    let second = control.tick().await.unwrap();
    assert_eq!(second.snapshot.sync_disposition, SyncDisposition::UnchangedConfirmedHead);
    assert_eq!(
        second.snapshot.last_successful_pinned_l1,
        Some(L1BlockRef { hash: B256::ZERO, number: 1, timestamp: 1_000 })
    );
    assert_eq!(
        second.completions.iter().map(|completion| completion.task_id).collect::<Vec<_>>(),
        vec![completed]
    );
    assert!(!second.snapshot.active_tasks.iter().any(|active| active.task_id == completed));
    let replacement = second
        .scheduled
        .iter()
        .find(|scheduled| matches!(scheduled.operation, OperationSummary::ResolutionSweep))
        .expect("the reaped resolution sweep must be replaced in the same cycle");
    assert!(completed < replacement.task_id);
}

#[tokio::test]
async fn failed_sync_leaves_existing_tasks_untouched() {
    let view = Arc::new(ScenarioL1View::new());
    view.fail_latest_head.store(true, Ordering::Relaxed);
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let (done_tx, done_rx) = oneshot::channel();
    let completed = insert_task(
        &proposer,
        OperationSummary::ResolutionSweep,
        tokio::spawn(async move {
            let _ = done_tx.send(());
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    let _ = done_rx.await;
    let next_id_before = proposer.next_task_id.load(Ordering::Relaxed);

    assert!(matches!(control.tick().await, Err(ScenarioError::Cycle(_))));

    assert!(proposer.tasks.lock().await.contains_key(&completed));
    assert_eq!(proposer.next_task_id.load(Ordering::Relaxed), next_id_before);
}

#[tokio::test]
async fn failed_creation_planning_does_not_stop_other_work() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    proposer.state.write().await.games.insert(U256::ONE, challenged_game(1, 5_000));
    view.fail_respected_game_type.store(true, Ordering::Relaxed);
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));

    let result = control.tick().await.unwrap();
    assert!(result.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { purpose: ProvingPurpose::Defense, .. }
    )));
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| { matches!(scheduled.operation, OperationSummary::ResolutionSweep) })
    );
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ClaimSweep))
    );
    assert!(
        !result.scheduled.iter().any(|scheduled| {
            matches!(scheduled.operation, OperationSummary::ProposeGame { .. })
        })
    );
}

#[tokio::test]
async fn failed_defense_planning_does_not_stop_resolution_or_claims() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    {
        let mut state = proposer.state.write().await;
        state.games.insert(U256::ONE, challenged_game(1, 5_000));
        state.games.insert(U256::from(2), challenged_game(2, 6_000));
        state.games.insert(U256::from(3), challenged_game(3, 7_000));
    }
    view.fail_respected_game_type.store(true, Ordering::Relaxed);
    view.fail_latest_l1_timestamp_on.store(2, Ordering::Relaxed);
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));

    let result = control.tick().await.unwrap();
    assert_eq!(
        result
            .scheduled
            .iter()
            .filter_map(|scheduled| match scheduled.operation {
                OperationSummary::ProveGame { address, .. } => Some(address),
                _ => None,
            })
            .collect::<Vec<_>>(),
        vec![challenged_game(1, 5_000).address]
    );
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| { matches!(scheduled.operation, OperationSummary::ResolutionSweep) })
    );
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ClaimSweep))
    );
}

#[tokio::test]
async fn snapshots_are_sorted_and_immutable() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    view.fail_respected_game_type.store(true, Ordering::Relaxed);
    {
        let mut state = proposer.state.write().await;
        state.anchor_game = Some(challenged_game(9, 9_000));
        state.canonical_head_index = Some(U256::from(9));
        state.canonical_head_sequence_number = Some(32_400);
    }
    {
        let mut pending = proposer.pending_games.write().await;
        pending.insert(
            U256::from(7),
            CompactGameSummary {
                factory_index: U256::from(7),
                address: Address::left_padding_from(&[7]),
                parent_index: 6,
                sequence_number: 25_200,
            },
        );
        pending.insert(
            U256::from(3),
            CompactGameSummary {
                factory_index: U256::from(3),
                address: Address::left_padding_from(&[3]),
                parent_index: 2,
                sequence_number: 10_800,
            },
        );
    }
    let claim = ParkedTask::insert(&proposer, OperationSummary::ClaimSweep, "claim sweep").await;
    let resolution =
        ParkedTask::insert(&proposer, OperationSummary::ResolutionSweep, "resolution sweep").await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    claim.record(&mut control).await;
    resolution.record(&mut control).await;

    let result = control.tick().await.unwrap();
    assert_eq!(
        result.snapshot.last_successful_pinned_l1,
        Some(L1BlockRef { hash: B256::ZERO, number: 1, timestamp: 1_000 })
    );
    assert_eq!(result.snapshot.sync_disposition, SyncDisposition::UnchangedConfirmedHead);
    assert_eq!(
        result.snapshot.anchor,
        Some(CompactGameSummary {
            factory_index: U256::from(9),
            address: challenged_game(9, 9_000).address,
            parent_index: u32::MAX,
            sequence_number: 32_400,
        })
    );
    assert_eq!(result.snapshot.canonical_head_index, Some(U256::from(9)));
    assert_eq!(result.snapshot.canonical_head_sequence_number, Some(32_400));
    assert_eq!(
        result.snapshot.pending_games.iter().map(|game| game.factory_index).collect::<Vec<_>>(),
        vec![U256::from(3), U256::from(7)]
    );
    assert_eq!(
        result.snapshot.active_tasks.iter().map(|task| task.task_id).collect::<Vec<_>>(),
        vec![resolution.task_id, claim.task_id]
    );
    let snapshot = result.snapshot.clone();
    proposer.pending_games.write().await.clear();
    proposer.state.write().await.anchor_game = None;
    assert_eq!(result.snapshot, snapshot);
    claim.release();
    resolution.release();
    control.settle(&[claim.task_id, resolution.task_id]).await.unwrap();
}

#[tokio::test]
async fn new_create_and_follow_up_check_have_distinct_task_summaries() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let proposed = control.tick().await.unwrap();
    assert!(proposed.scheduled.iter().any(|scheduled| {
        matches!(
            scheduled.operation,
            OperationSummary::ProposeGame { sequence_number: 3_600, parent_game_index: u32::MAX }
        )
    }));
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    proposer.sync_state().await.unwrap();
    *proposer.in_flight_creation.lock().await = Some(InFlightCreation {
        root_claim: B256::left_padding_from(&[9]),
        extra_data: vec![1, 2, 3],
        sequence_number: 7_200,
        parent_game_index: 4,
    });
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let reconciled = control.tick().await.unwrap();
    assert_eq!(
        reconciled.snapshot.in_flight_creation,
        Some(crate::proposer::InFlightCreationSummary {
            sequence_number: 7_200,
            parent_game_index: 4,
        })
    );
    let creation = reconciled
        .scheduled
        .iter()
        .find(|scheduled| matches!(scheduled.operation, OperationSummary::ReconcileCreation { .. }))
        .unwrap();
    assert!(matches!(
        creation.operation,
        OperationSummary::ReconcileCreation { sequence_number: 7_200, parent_game_index: 4 }
    ));
}

#[tokio::test]
async fn settle_reports_each_task_outcome() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let operation = || OperationSummary::ResolutionSweep;
    let success =
        insert_task(&proposer, operation(), tokio::spawn(async { Ok(TaskSuccess::Completed) }))
            .await;
    let failed =
        insert_task(&proposer, operation(), tokio::spawn(async { anyhow::bail!("worker failed") }))
            .await;
    let terminal = insert_task(
        &proposer,
        OperationSummary::ProveGame {
            factory_index: U256::ONE,
            address: Address::left_padding_from(&[1]),
            purpose: ProvingPurpose::Defense,
        },
        tokio::spawn(async { Ok(TaskSuccess::TerminallyUnprovable) }),
    )
    .await;
    let panicked = insert_task(
        &proposer,
        operation(),
        tokio::spawn(async {
            panic!("worker panic");
            #[allow(unreachable_code)]
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;

    let completions = control.settle(&[panicked, terminal, failed, success]).await.unwrap();
    assert_eq!(
        completions.iter().map(|completion| completion.task_id).collect::<Vec<_>>(),
        vec![success, failed, terminal, panicked]
    );
    assert_eq!(completions[0].outcome, TaskCompletionOutcome::Success);
    assert_eq!(
        completions[1].outcome,
        TaskCompletionOutcome::Failed(TaskFailureClass::ReturnedError("worker failed".into()))
    );
    assert_eq!(completions[2].outcome, TaskCompletionOutcome::TerminallyUnprovable);
    assert!(matches!(
        completions[3].outcome,
        TaskCompletionOutcome::Failed(TaskFailureClass::Panicked)
    ));
}

#[tokio::test]
async fn settle_rejects_unknown_and_finalized_task_ids() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let success = insert_task(
        &proposer,
        OperationSummary::ResolutionSweep,
        tokio::spawn(async { Ok(TaskSuccess::Completed) }),
    )
    .await;
    control.settle(&[success]).await.unwrap();

    assert_eq!(
        control.settle(&[success]).await.unwrap_err(),
        ScenarioError::AlreadyFinalized { task_id: success }
    );
    let unknown =
        TaskId(NonZeroU64::new(proposer.next_task_id.load(Ordering::Relaxed) + 10).unwrap());
    assert_eq!(
        control.settle(&[unknown]).await.unwrap_err(),
        ScenarioError::UnknownTask { task_id: unknown }
    );
}

#[tokio::test(start_paused = true)]
async fn settlement_timeout_preserves_unfinished_tasks() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let operation = || OperationSummary::ResolutionSweep;
    let completed_before_timeout =
        insert_task(&proposer, operation(), tokio::spawn(async { Ok(TaskSuccess::Completed) }))
            .await;
    let blocked = ParkedTask::insert(&proposer, operation(), "partial settlement barrier").await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_millis(10));
    assert_eq!(
        control.settle(&[completed_before_timeout, blocked.task_id]).await.unwrap_err(),
        ScenarioError::SettlementWatchdog {
            task_ids: vec![completed_before_timeout, blocked.task_id],
            completions: vec![crate::proposer::TaskCompletion {
                task_id: completed_before_timeout,
                class: TaskClass::Resolution,
                target: crate::proposer::OperationTarget::AllGames,
                outcome: TaskCompletionOutcome::Success,
            }],
        }
    );
    assert_eq!(
        control.settle(&[completed_before_timeout]).await.unwrap_err(),
        ScenarioError::AlreadyFinalized { task_id: completed_before_timeout }
    );
    blocked.release();
    assert_eq!(control.settle(&[blocked.task_id]).await.unwrap()[0].task_id, blocked.task_id);
}

#[tokio::test]
async fn task_finishing_during_tick_remains_settleable() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    let task_id = allocate_task_id(&proposer);
    let barrier = NamedBarrier::new("completion barrier");
    let (done_tx, done_rx) = oneshot::channel();
    *view.release_barrier.lock().unwrap() = Some(barrier.clone());
    *view.task_finished.lock().unwrap() = Some(done_rx);
    let task_barrier = barrier.clone();
    insert_allocated_task(
        &proposer,
        task_id,
        OperationSummary::ResolutionSweep,
        tokio::spawn(async move {
            task_barrier.park(task_id).await;
            let _ = done_tx.send(());
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    barrier.wait_until_reached().await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    control.record_parked(task_id, &barrier).await.unwrap();

    let result = control.tick().await.unwrap();

    assert!(result.completions.is_empty());
    assert!(proposer.tasks.lock().await.get(&task_id).unwrap().0.is_finished());
    assert_eq!(control.settle(&[task_id]).await.unwrap()[0].task_id, task_id);
    let scheduled = result.scheduled.iter().map(|item| item.task_id).collect::<Vec<_>>();
    let completions = control.settle(&scheduled).await.unwrap();
    assert!(
        completions
            .iter()
            .all(|completion| matches!(&completion.outcome, TaskCompletionOutcome::Success))
    );
}

#[tokio::test]
async fn settlement_rejects_tasks_finalized_by_a_later_tick() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let (done_tx, done_rx) = oneshot::channel();
    let reaped = insert_task(
        &proposer,
        OperationSummary::ResolutionSweep,
        tokio::spawn(async move {
            let _ = done_tx.send(());
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    let _ = done_rx.await;
    assert!(control.tick().await.unwrap().completions.iter().any(|item| item.task_id == reaped));
    assert_eq!(
        control.settle(&[reaped]).await.unwrap_err(),
        ScenarioError::AlreadyFinalized { task_id: reaped }
    );
}

#[tokio::test]
async fn defense_tasks_respect_the_concurrency_limit() {
    let view = Arc::new(ScenarioL1View::new());
    let mut config = test_config(30);
    config.max_concurrent_defense_tasks = NonZeroU64::new(2).unwrap();
    let proposer = proposer_with(config, view).await;
    proposer.sync_state().await.unwrap();
    {
        let mut state = proposer.state.write().await;
        for (index, deadline) in [(1, 5_000), (2, 6_000), (3, 7_000)] {
            state.games.insert(U256::from(index), challenged_game(index, deadline));
        }
    }
    let game_one = challenged_game(1, 5_000);
    let active = ParkedTask::insert(
        &proposer,
        OperationSummary::ProveGame {
            factory_index: game_one.index,
            address: game_one.address,
            purpose: ProvingPurpose::Defense,
        },
        "active defense",
    )
    .await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    active.record(&mut control).await;

    let result = control.tick().await.unwrap();
    let proving = result
        .scheduled
        .iter()
        .filter_map(|scheduled| match scheduled.operation {
            OperationSummary::ProveGame { address, .. } => Some(address),
            _ => None,
        })
        .collect::<Vec<_>>();

    assert_eq!(proving, vec![challenged_game(2, 6_000).address]);
    assert_eq!(
        proposer
            .tasks
            .lock()
            .await
            .values()
            .filter(|(_, operation)| operation.class() == TaskClass::Proving)
            .count(),
        2
    );
    active.release();
    control.settle(&[active.task_id]).await.unwrap();
}

#[tokio::test]
async fn active_create_and_sweeps_are_not_duplicated() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    proposer.sync_state().await.unwrap();
    let creation = ParkedTask::insert(
        &proposer,
        OperationSummary::ReconcileCreation { sequence_number: 9, parent_game_index: 8 },
        "creation",
    )
    .await;
    let resolution =
        ParkedTask::insert(&proposer, OperationSummary::ResolutionSweep, "resolution").await;
    let claim = ParkedTask::insert(&proposer, OperationSummary::ClaimSweep, "claim").await;
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));
    creation.record(&mut control).await;
    resolution.record(&mut control).await;
    claim.record(&mut control).await;

    let result = control.tick().await.unwrap();

    assert!(!result.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProposeGame { .. } | OperationSummary::ReconcileCreation { .. }
    )));
    assert!(
        !result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ResolutionSweep))
    );
    assert!(
        !result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ClaimSweep))
    );
    creation.release();
    resolution.release();
    claim.release();
    control.settle(&[creation.task_id, resolution.task_id, claim.task_id]).await.unwrap();
}

#[tokio::test]
async fn tick_rejects_running_tasks_without_a_reached_barrier() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let running =
        ParkedTask::insert(&proposer, OperationSummary::ResolutionSweep, "resolution barrier")
            .await;
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));

    assert_eq!(
        control.tick().await.unwrap_err(),
        ScenarioError::RunningTask { task_id: running.task_id }
    );

    running.release();
    control.settle(&[running.task_id]).await.unwrap();
}

#[tokio::test]
async fn tick_accepts_tasks_parked_at_a_reached_barrier() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let running =
        ParkedTask::insert(&proposer, OperationSummary::ResolutionSweep, "resolution barrier")
            .await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));

    running.record(&mut control).await;
    let parked_tick = control.tick().await.unwrap();
    let scheduled_ids =
        parked_tick.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>();
    let completions = control.settle(&scheduled_ids).await.unwrap();
    assert!(
        completions
            .iter()
            .all(|completion| matches!(&completion.outcome, TaskCompletionOutcome::Success))
    );
    running.release();
    assert_eq!(control.settle(&[running.task_id]).await.unwrap()[0].task_id, running.task_id);
}

#[tokio::test]
async fn parked_task_must_reach_its_reported_barrier() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let running =
        ParkedTask::insert(&proposer, OperationSummary::ResolutionSweep, "resolution barrier")
            .await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));

    let mismatched = NamedBarrier::new("different task barrier");
    let mismatched_task =
        TaskId(NonZeroU64::new(proposer.next_task_id.load(Ordering::Relaxed)).unwrap());
    let mismatched_worker = {
        let mismatched = mismatched.clone();
        tokio::spawn(async move { mismatched.park(mismatched_task).await })
    };
    mismatched.wait_until_reached().await;
    assert_eq!(
        control.record_parked(running.task_id, &mismatched).await.unwrap_err(),
        ScenarioError::BarrierTaskMismatch {
            task_id: running.task_id,
            barrier: "different task barrier".into(),
            reached_by: mismatched_task,
        }
    );
    mismatched.release();
    mismatched_worker.await.unwrap();
    let unreached = NamedBarrier::new("not reached");
    unreached.bind_task(running.task_id);
    assert_eq!(
        control.record_parked(running.task_id, &unreached).await.unwrap_err(),
        ScenarioError::BarrierNotReached {
            task_id: running.task_id,
            barrier: "not reached".into(),
        }
    );
    running.release();
    assert_eq!(control.settle(&[running.task_id]).await.unwrap()[0].task_id, running.task_id);
}

#[test]
#[should_panic(expected = "a pre-submit failure cannot reach an after-submission barrier")]
fn after_submission_barrier_rejects_pre_submit_failure() {
    let world = ScenarioWorld::new();
    world.block_action(
        ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX },
        1,
        ActionBarrierPoint::AfterSubmission,
        ActionOutcome::PreSubmitFailure,
        "unreachable barrier",
    );
}

#[tokio::test(start_paused = true)]
async fn barrier_wait_fails_when_the_scripted_attempt_never_reaches_it() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.block_action(
        target.clone(),
        2,
        ActionBarrierPoint::AfterSubmission,
        ActionOutcome::Success,
        "unreached second attempt",
    );
    let mut scenario = ScenarioHarness::new(world, scenario_config()).await.unwrap();
    let result = scenario.tick().await.unwrap();
    let create_id =
        result.task_id_for(|operation| matches!(operation, OperationSummary::ProposeGame { .. }));

    let error = scenario
        .wait_for_action_barrier(create_id, &target, 2, ActionBarrierPoint::AfterSubmission)
        .await
        .unwrap_err();

    assert!(matches!(
        error,
        ScenarioError::BarrierWatchdog { barrier } if barrier.contains("attempt: 2")
    ));
    scenario.settle_scheduled(&result).await.unwrap();
}

#[tokio::test]
async fn scenario_world_fallback_action_outcome_applies_to_each_game() {
    let world = ScenarioWorld::new();
    let first = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    let second = ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate());
    let first_address = first.address;
    let second_address = second.address;
    let first_target = ActionTarget::Resolve(first.target());
    let second_target = ActionTarget::Resolve(second.target());
    world.add_game(first);
    world.add_game(second);
    world.script_action_fallback(ActionOutcome::PreSubmitFailure);
    let actions = world.action_executor();

    assert!(actions.resolve_game(first_address).await.is_err());
    assert!(actions.resolve_game(second_address).await.is_err());
    assert_eq!(
        world.action_record(&first_target, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
    assert_eq!(
        world.action_record(&second_target, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
}

#[tokio::test]
async fn proof_engine_rejects_inputs_for_a_different_game_address() {
    let world = ScenarioWorld::new();
    let first = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    let second = ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate());
    let first_address = first.address;
    let second_address = second.address;
    world.add_game(first);
    world.add_game(second);
    let first =
        world.observation().games.into_iter().find(|game| game.address == first_address).unwrap();
    let inputs = GameProofInputs {
        l1_head: first.proof_inputs.l1_head,
        l1_head_number: first.proof_inputs.l1_head_number,
        starting_root: first.proof_inputs.starting_root,
        starting_ts: first.proof_inputs.starting_sequence_number,
        root_claim: first.proof_inputs.root_claim,
        claim_ts: first.proof_inputs.sequence_number,
        prestate: first.absolute_prestate,
        prover: ScenarioWorld::proposer_address(),
    };

    let error =
        world.proof_engine().prove(second_address, None, inputs, Vec::new()).await.unwrap_err();

    assert_eq!(
        error.to_string(),
        format!("proof inputs do not match scenario game {second_address}")
    );
    assert!(world.proof_record(&first.target(), 1).is_none());
}

#[tokio::test]
async fn scenario_world_confirmed_action_updates_l1_without_changing_old_blocks_or_clocks() {
    let world = ScenarioWorld::new();
    world.configure_sync_confirmations(2).unwrap();
    world.set_host_time(7_000);
    world.set_horizons(6_000, 5_000);
    let before = world.observation();
    let mut extra_data = u32::MAX.to_be_bytes().to_vec();
    extra_data.push(0x01);

    world
        .action_executor()
        .create_game(canonical_super_root(1), extra_data, U256::ONE)
        .await
        .unwrap();

    let after = world.observation();
    assert_eq!(after.latest_l1.number, before.latest_l1.number + NUM_CONFIRMATIONS + 2);
    assert_eq!(after.latest_l1.timestamp, before.latest_l1.timestamp);
    assert_eq!(after.host_time, before.host_time);
    assert_eq!(after.safe_time, before.safe_time);
    assert_eq!(after.finalized_time, before.finalized_time);
    assert_eq!(before.games, Vec::<ScenarioGame>::new());
    assert_eq!(after.games.len(), 1);

    let view = world.l1_view();
    assert_eq!(
        view.latest_game_index(BlockId::number(before.latest_l1.number)).await.unwrap(),
        None
    );
    assert_eq!(
        view.latest_game_index(BlockId::number(after.latest_l1.number - 2)).await.unwrap(),
        Some(U256::ZERO)
    );
}

#[tokio::test]
async fn scenario_world_clocks_move_independently() {
    let world = ScenarioWorld::new();
    let initial = world.observation();
    world.mine_block();
    world.mine_block();
    world.set_host_time(9_000);
    world.set_safe_time(8_000);
    world.set_finalized_time(7_000);
    world.set_latest_l1_time(6_000);

    let current = world.observation();
    let pinned = world.l1_view().block_ref(current.latest_l1.number - 2).await.unwrap().unwrap();
    assert_eq!(pinned.timestamp, initial.latest_l1.timestamp);
    assert_eq!(current.latest_l1.timestamp, 6_000);
    assert_eq!(current.host_time, 9_000);
    assert_eq!(current.safe_time, 8_000);
    assert_eq!(current.finalized_time, 7_000);
}

#[tokio::test]
async fn confirming_a_later_nonce_first_includes_timed_out_lower_nonces() {
    let world = ScenarioWorld::new();
    let existing =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).challenged();
    let existing_address = existing.address;
    let resolve_target = ActionTarget::Resolve(existing.target());
    world.add_game(existing);
    let create_target = ActionTarget::Create { sequence_number: 2, parent_game_index: u32::MAX };
    world.script_action(create_target.clone(), 1, ActionOutcome::Timeout);
    let actions = world.action_executor();
    let mut extra_data = u32::MAX.to_be_bytes().to_vec();
    extra_data.push(0x01);

    assert!(actions.create_game(canonical_super_root(2), extra_data, U256::ONE).await.is_err());
    assert_eq!(world.observation().nonce, NonceState { pending: 1, latest: 0 });

    actions.resolve_game(existing_address).await.unwrap();

    let observation = world.observation();
    assert_eq!(observation.nonce, NonceState { pending: 2, latest: 2 });
    assert!(observation.pending_transactions.is_empty());
    assert_eq!(observation.games.len(), 2);
    assert_eq!(
        world.action_record(&create_target, 1).unwrap().lifecycle,
        ActionLifecycle::IncludedLate
    );
    assert_eq!(
        world.action_record(&resolve_target, 1).unwrap().lifecycle,
        ActionLifecycle::Confirmed
    );
}

#[tokio::test]
async fn replacing_a_dropped_nonce_assigns_create_indices_in_inclusion_order() {
    let world = ScenarioWorld::new();
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();
    let first_target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    let queued_target = ActionTarget::Create { sequence_number: 2, parent_game_index: u32::MAX };
    world.script_action(first_target.clone(), 1, ActionOutcome::Timeout);
    world.script_action(queued_target.clone(), 1, ActionOutcome::Timeout);
    let actions = world.action_executor();
    let mut extra_data = u32::MAX.to_be_bytes().to_vec();
    extra_data.push(0x01);

    assert!(
        actions.create_game(canonical_super_root(1), extra_data.clone(), U256::ONE).await.is_err()
    );
    assert!(
        actions.create_game(canonical_super_root(2), extra_data.clone(), U256::ONE).await.is_err()
    );
    scenario.drop_transaction(&first_target, 1).unwrap();

    let replacement =
        actions.create_game(canonical_super_root(3), extra_data, U256::ONE).await.unwrap();
    scenario.include_transaction(&queued_target, 1, InclusionDepth::LatestOnly).unwrap();

    let observation = world.observation();
    assert_eq!(observation.nonce, NonceState { pending: 2, latest: 2 });
    assert!(observation.pending_transactions.is_empty());
    assert_eq!(observation.games.len(), 2);
    assert_eq!(observation.games[0].factory_index, U256::ZERO);
    assert_eq!(observation.games[0].root_claim, canonical_super_root(3));
    assert_eq!(observation.games[0].address, replacement.game_address);
    assert_eq!(observation.games[1].factory_index, U256::ONE);
    assert_eq!(observation.games[1].root_claim, canonical_super_root(2));
    assert!(matches!(
        world.action_record(&queued_target, 1).unwrap().effect,
        CommittedEffect::Created { factory_index, .. } if factory_index == U256::ONE
    ));
}

#[tokio::test]
async fn blocked_creation_stays_single_until_its_task_is_released() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.block_action(
        target.clone(),
        1,
        ActionBarrierPoint::AfterSubmission,
        ActionOutcome::Success,
        "create submitted",
    );
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let first = scenario.tick().await.unwrap();
    let create_id =
        first.task_id_for(|operation| matches!(operation, OperationSummary::ProposeGame { .. }));
    assert_eq!(create_id.get(), 1);
    scenario
        .wait_for_action_barrier(create_id, &target, 1, ActionBarrierPoint::AfterSubmission)
        .await
        .unwrap();
    let submitted = world.observation();
    assert_eq!(submitted.nonce, NonceState { pending: 1, latest: 0 });
    assert!(submitted.games.is_empty());
    assert_eq!(
        submitted.pending_transactions,
        vec![PendingTransactionObservation { target: target.clone(), attempt: 1, nonce: 0 }]
    );
    let submitted_record = world.action_record(&target, 1).unwrap();
    assert_eq!(
        submitted_record.inputs,
        ActionInputs::Create {
            root_claim: canonical_super_root(1),
            parent_game_index: u32::MAX,
            sequence_number: 1,
        }
    );
    assert_eq!(submitted_record.lifecycle, ActionLifecycle::Submitted);
    assert!(submitted_record.transaction_hash.is_some());
    assert_eq!(submitted_record.effect, CommittedEffect::None);
    scenario.settle(&first.task_ids_except(create_id)).await.unwrap();

    let blocked = scenario.tick().await.unwrap();
    assert!(blocked.snapshot.active_tasks.iter().any(|task| task.task_id == create_id));
    assert!(!blocked.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProposeGame { .. } | OperationSummary::ReconcileCreation { .. }
    )));
    scenario.settle_scheduled(&blocked).await.unwrap();

    scenario.release_action_barrier(&target, 1, ActionBarrierPoint::AfterSubmission).unwrap();
    scenario.settle(&[create_id]).await.unwrap();
    let confirmed = world.observation();
    assert_eq!(confirmed.nonce, NonceState { pending: 1, latest: 1 });
    assert!(confirmed.pending_transactions.is_empty());
    assert_eq!(confirmed.games.len(), 1);
    let confirmed_record = world.action_record(&target, 1).unwrap();
    assert_eq!(confirmed_record.lifecycle, ActionLifecycle::Confirmed);
    assert!(matches!(
        confirmed_record.effect,
        CommittedEffect::Created { factory_index, .. } if factory_index == U256::ZERO
    ));
}

#[tokio::test]
async fn blocked_proof_uses_one_slot_without_blocking_another_game() {
    let world = ScenarioWorld::new();
    let mut first_game =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).challenged();
    first_game.deadline = 5_000;
    let first_target = first_game.target();
    world.add_game(first_game);
    world.set_horizons(1, 1);
    world.block_proof(first_target.clone(), 1, ProofOutcome::Success, "first proof");
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let first = scenario.tick().await.unwrap();
    let first_id = first.task_id_for(|operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address, .. } if *address == first_target.address
        )
    });
    assert_eq!(first_id.get(), 1);
    scenario.wait_for_proof_barrier(first_id, &first_target, 1).await.unwrap();
    scenario.settle(&first.task_ids_except(first_id)).await.unwrap();

    let mut second_game =
        ScenarioGame::new(1, 0, 2, ScenarioWorld::default_prestate()).challenged();
    second_game.deadline = 6_000;
    let second_target = second_game.target();
    world.add_game(second_game);
    world.set_horizons(2, 2);
    let second = scenario.tick().await.unwrap();
    assert!(second.snapshot.active_tasks.iter().any(|task| task.task_id == first_id));
    assert!(!second.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == first_target.address
    )));
    let second_id = second.task_id_for(|operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address, .. } if *address == second_target.address
        )
    });
    scenario.settle_scheduled(&second).await.unwrap();
    assert_eq!(world.proof_record(&second_target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);

    scenario.release_proof_barrier(&first_target, 1).unwrap();
    scenario.settle(&[first_id]).await.unwrap();
    assert_eq!(world.proof_record(&first_target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert!(world.action_record(&ActionTarget::Prove(first_target), 1).is_some());
    assert!(world.action_record(&ActionTarget::Prove(second_target), 1).is_some());
    assert!(second_id > first_id);
}

#[tokio::test]
async fn one_submission_finishes_before_the_next_one_starts() {
    let world = ScenarioWorld::new();
    let game = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate())
        .provable_for_resolution();
    let game_target = game.target();
    world.add_game(game);
    world.set_horizons(2, 2);
    let create = ActionTarget::Create { sequence_number: 2, parent_game_index: 0 };
    let resolve = ActionTarget::Resolve(game_target);
    world.block_action(
        create.clone(),
        1,
        ActionBarrierPoint::AfterSubmission,
        ActionOutcome::Success,
        "create owns signer",
    );
    world.block_action(
        resolve.clone(),
        1,
        ActionBarrierPoint::BeforeSigner,
        ActionOutcome::Success,
        "resolve before signer",
    );
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    let create_id =
        result.task_id_for(|operation| matches!(operation, OperationSummary::ProposeGame { .. }));
    let resolve_id =
        result.task_id_for(|operation| matches!(operation, OperationSummary::ResolutionSweep));
    scenario
        .wait_for_action_barrier(resolve_id, &resolve, 1, ActionBarrierPoint::BeforeSigner)
        .await
        .unwrap();
    scenario
        .wait_for_action_barrier(create_id, &create, 1, ActionBarrierPoint::AfterSubmission)
        .await
        .unwrap();
    scenario.release_action_barrier(&resolve, 1, ActionBarrierPoint::BeforeSigner).unwrap();
    assert!(world.action_record(&resolve, 1).is_none());

    scenario.release_action_barrier(&create, 1, ActionBarrierPoint::AfterSubmission).unwrap();
    scenario.settle(&[create_id]).await.unwrap();
    scenario.settle(&[resolve_id]).await.unwrap();
    let remaining = result
        .scheduled
        .iter()
        .map(|scheduled| scheduled.task_id)
        .filter(|task_id| *task_id != create_id && *task_id != resolve_id)
        .collect::<Vec<_>>();
    scenario.settle(&remaining).await.unwrap();
    assert_eq!(world.action_record(&resolve, 1).unwrap().lifecycle, ActionLifecycle::Confirmed);
}

#[tokio::test]
async fn failed_create_before_submission_allows_a_later_create_without_consuming_a_nonce() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::PreSubmitFailure);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let failed = scenario.tick().await.unwrap();
    let create_id =
        failed.task_id_for(|operation| matches!(operation, OperationSummary::ProposeGame { .. }));
    scenario.settle_scheduled(&failed).await.unwrap();
    assert_eq!(create_id.get(), 1);
    assert_eq!(world.observation().nonce, NonceState { pending: 0, latest: 0 });
    assert!(world.observation().games.is_empty());
    assert_eq!(
        world.action_record(&target, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );

    let follow_up = scenario.tick().await.unwrap();
    assert!(follow_up.snapshot.in_flight_creation.is_some());
    assert!(follow_up.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ReconcileCreation { .. }
    )));
    assert!(
        !follow_up
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario.settle_scheduled(&follow_up).await.unwrap();

    world.set_horizons(0, 0);
    let ready = scenario.tick().await.unwrap();
    assert!(ready.snapshot.in_flight_creation.is_none());
    scenario.settle_scheduled(&ready).await.unwrap();

    world.set_horizons(1, 1);
    let later = scenario.tick().await.unwrap();
    assert!(later.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProposeGame { sequence_number: 1, parent_game_index: u32::MAX }
    )));
    scenario.settle_scheduled(&later).await.unwrap();
    assert_eq!(world.action_record(&target, 2).unwrap().lifecycle, ActionLifecycle::Confirmed);
    assert_eq!(world.observation().games.len(), 1);
}

#[tokio::test]
async fn reverted_create_consumes_a_nonce_without_creating_a_game() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Revert);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&result).await.unwrap();
    let observation = world.observation();
    assert_eq!(observation.nonce, NonceState { pending: 1, latest: 1 });
    assert!(observation.games.is_empty());
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::Reverted);
    world.set_horizons(0, 0);
    let next = scenario.tick().await.unwrap();
    assert!(next.snapshot.in_flight_creation.is_none());
    scenario.settle_scheduled(&next).await.unwrap();
}

#[tokio::test]
async fn timed_out_create_that_lands_late_is_not_duplicated() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Timeout);
    let mut config = scenario_config();
    config.sync_l1_confirmations = 2;
    let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

    let timed_out = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&timed_out).await.unwrap();
    assert_eq!(world.observation().nonce, NonceState { pending: 1, latest: 0 });
    assert!(world.observation().games.is_empty());
    assert_eq!(
        world.observation().pending_transactions,
        vec![PendingTransactionObservation { target: target.clone(), attempt: 1, nonce: 0 }]
    );
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::TimedOut);

    let held = scenario.tick().await.unwrap();
    assert!(held.snapshot.in_flight_creation.is_some());
    scenario.settle_scheduled(&held).await.unwrap();
    let pinned_block = world.observation().latest_l1.number;
    scenario.include_transaction(&target, 1, InclusionDepth::LatestOnly).unwrap();
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::IncludedLate);
    let included = world.observation();
    assert_eq!(included.nonce, NonceState { pending: 1, latest: 1 });
    assert!(included.pending_transactions.is_empty());
    assert_eq!(included.games.len(), 1);
    let view = world.l1_view();
    assert_eq!(view.latest_game_index(BlockId::number(pinned_block)).await.unwrap(), None);
    assert_eq!(
        view.latest_game_index(BlockId::number(included.latest_l1.number)).await.unwrap(),
        Some(U256::ZERO)
    );

    let adopted = scenario.tick().await.unwrap();
    assert!(adopted.snapshot.in_flight_creation.is_some());
    assert!(
        !adopted
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario.settle_scheduled(&adopted).await.unwrap();
    let learned = scenario.tick().await.unwrap();
    assert!(learned.snapshot.in_flight_creation.is_none());
    assert!(
        !learned
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario.settle_scheduled(&learned).await.unwrap();
}

#[tokio::test]
async fn dropped_create_does_not_block_a_later_create() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Timeout);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let timed_out = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&timed_out).await.unwrap();
    assert_eq!(
        world.observation().pending_transactions,
        vec![PendingTransactionObservation { target: target.clone(), attempt: 1, nonce: 0 }]
    );
    scenario.drop_transaction(&target, 1).unwrap();
    let dropped = world.observation();
    assert_eq!(dropped.nonce, NonceState { pending: 0, latest: 0 });
    assert!(dropped.pending_transactions.is_empty());
    let dropped_record = world.action_record(&target, 1).unwrap();
    assert_eq!(dropped_record.lifecycle, ActionLifecycle::Dropped);
    assert_eq!(dropped_record.effect, CommittedEffect::None);

    let follow_up = scenario.tick().await.unwrap();
    assert!(follow_up.snapshot.in_flight_creation.is_some());
    scenario.settle_scheduled(&follow_up).await.unwrap();
    world.set_horizons(0, 0);
    let ready = scenario.tick().await.unwrap();
    assert!(ready.snapshot.in_flight_creation.is_none());
    assert!(world.observation().games.is_empty());
    scenario.settle_scheduled(&ready).await.unwrap();

    world.set_horizons(1, 1);
    let later = scenario.tick().await.unwrap();
    assert!(later.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProposeGame { sequence_number: 1, parent_game_index: u32::MAX }
    )));
    scenario.settle_scheduled(&later).await.unwrap();
    let replacement = world.action_record(&target, 2).unwrap();
    assert_eq!(replacement.lifecycle, ActionLifecycle::Confirmed);
    assert_eq!(
        replacement.inputs,
        ActionInputs::Create {
            root_claim: canonical_super_root(1),
            parent_game_index: u32::MAX,
            sequence_number: 1,
        }
    );
    assert!(matches!(
        replacement.effect,
        CommittedEffect::Created { factory_index, .. } if factory_index == U256::ZERO
    ));
    let observation = world.observation();
    assert!(observation.pending_transactions.is_empty());
    assert_eq!(observation.games.len(), 1);
    assert_eq!(observation.games[0].factory_index, U256::ZERO);
}

#[tokio::test]
async fn failed_proofs_are_retried_without_blocking_other_games() {
    let world = ScenarioWorld::new();
    let games = (0..3)
        .map(|index| {
            let mut game =
                ScenarioGame::new(index, u32::MAX, index + 1, ScenarioWorld::default_prestate())
                    .challenged();
            game.deadline = 5_000 + index;
            game
        })
        .collect::<Vec<_>>();
    let failed = games[0].target();
    let panicked = games[1].target();
    let succeeded = games[2].target();
    for game in games {
        world.add_game(game);
    }
    world.set_horizons(3, 3);
    world.script_next_proof(failed.clone(), ProofOutcome::Failure);
    world.script_proof(panicked.clone(), 1, ProofOutcome::Panic);
    world.script_proof_fallback(ProofOutcome::Success);
    let mut config = scenario_config();
    config.max_concurrent_defense_tasks = NonZeroU64::new(3).unwrap();
    let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

    let first = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&first).await.unwrap();
    assert_eq!(world.proof_record(&failed, 1).unwrap().lifecycle, ProofLifecycle::Failed);
    assert_eq!(world.proof_record(&panicked, 1).unwrap().lifecycle, ProofLifecycle::Panicked);
    assert_eq!(world.proof_record(&succeeded, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert!(world.action_record(&ActionTarget::Prove(failed.clone()), 1).is_none());
    assert!(world.action_record(&ActionTarget::Prove(panicked.clone()), 1).is_none());
    assert_eq!(
        world.action_record(&ActionTarget::Prove(succeeded.clone()), 1).unwrap().effect,
        CommittedEffect::Proven { game: succeeded.address }
    );

    let retry = scenario.tick().await.unwrap();
    assert!(retry.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == failed.address
    )));
    assert!(retry.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == panicked.address
    )));
    scenario.settle_scheduled(&retry).await.unwrap();
    assert_eq!(world.proof_record(&failed, 2).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert_eq!(world.proof_record(&panicked, 2).unwrap().lifecycle, ProofLifecycle::Succeeded);
}

#[tokio::test]
async fn failed_resolution_does_not_stop_other_games() {
    let world = ScenarioWorld::new();
    let failed_game = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate())
        .provable_for_resolution();
    let succeeded_game = ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate())
        .provable_for_resolution();
    let failed = ActionTarget::Resolve(failed_game.target());
    let succeeded = ActionTarget::Resolve(succeeded_game.target());
    let succeeded_address = succeeded_game.address;
    world.add_game(failed_game);
    world.add_game(succeeded_game);
    world.set_horizons(2, 2);
    world.script_action(failed.clone(), 1, ActionOutcome::PreSubmitFailure);
    world.script_action_fallback(ActionOutcome::Success);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&result).await.unwrap();
    assert_eq!(
        world.action_record(&failed, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
    assert_eq!(
        world.action_record(&succeeded, 1).unwrap().effect,
        CommittedEffect::Resolved { game: succeeded_address }
    );
}

#[tokio::test]
async fn failed_claim_does_not_stop_other_games() {
    let world = ScenarioWorld::new();
    let failed_game =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).claimable(10);
    let succeeded_game =
        ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate()).claimable(20);
    let failed = ActionTarget::ClaimCredit(failed_game.target());
    let succeeded = ActionTarget::ClaimCredit(succeeded_game.target());
    world.add_game(failed_game);
    world.add_game(succeeded_game);
    world.set_horizons(2, 2);
    world.script_action(failed.clone(), 1, ActionOutcome::PreSubmitFailure);
    world.script_action_fallback(ActionOutcome::Success);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&result).await.unwrap();
    assert_eq!(
        world.action_record(&failed, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
    assert_eq!(world.action_record(&succeeded, 1).unwrap().lifecycle, ActionLifecycle::Confirmed);
}

#[tokio::test]
async fn claim_success_unlocks_then_pays_out() {
    let world = ScenarioWorld::new();
    let game = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).claimable(20);
    let target = ActionTarget::ClaimCredit(game.target());
    let address = game.address;
    world.add_game(game);
    world.set_horizons(1, 1);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let unlock = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&unlock).await.unwrap();
    assert_eq!(
        world.action_record(&target, 1).unwrap().effect,
        CommittedEffect::ClaimUnlocked { game: address, amount: U256::from(20) }
    );

    world.set_latest_l1_time(world.observation().latest_l1.timestamp + 20);
    let payout = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&payout).await.unwrap();
    assert_eq!(
        world.action_record(&target, 2).unwrap().effect,
        CommittedEffect::ClaimPaid { game: address, amount: U256::from(20) }
    );
}

#[tokio::test(start_paused = true)]
async fn initialization_sets_proving_duration_and_returns_failures_immediately() {
    let world = ScenarioWorld::new();
    let before = tokio::time::Instant::now();
    let scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();
    assert_eq!(scenario.max_prove_duration(), Some(DEFAULT_MAX_DURATION));
    assert_eq!(tokio::time::Instant::now(), before);

    let conflicting_world = ScenarioWorld::new();
    conflicting_world.configure_sync_confirmations(2).unwrap();
    let error = match ScenarioHarness::new(conflicting_world, scenario_config()).await {
        Ok(_) => panic!("conflicting sync-confirmation configuration should fail"),
        Err(error) => error,
    };
    assert_eq!(
        error,
        ScenarioError::Initialization(
            "scenario sync confirmations already configured as 2, cannot reconfigure as 0".into()
        )
    );

    let failing_world = ScenarioWorld::new();
    failing_world.clear_anchor_root();
    let before_failure = tokio::time::Instant::now();
    let error = match ScenarioHarness::new(failing_world.clone(), scenario_config()).await {
        Ok(_) => panic!("initialization should fail"),
        Err(error) => error,
    };
    assert_eq!(
        error,
        ScenarioError::Initialization(
            "anchor state registry has no anchor root (game creation would revert)".into()
        )
    );
    assert_eq!(tokio::time::Instant::now(), before_failure);
}

#[tokio::test]
async fn publishing_a_rotated_prestate_enables_defense_on_the_next_tick() {
    let world = ScenarioWorld::new();
    let rotated = B256::repeat_byte(0x33);
    world.rotate_registered_prestate(rotated, 7_200);
    let mut game = ScenarioGame::new(0, u32::MAX, 1, rotated).challenged();
    game.deadline = 5_000;
    let target = game.target();
    world.add_game(game);
    world.set_horizons(1, 1);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();
    assert_eq!(scenario.max_prove_duration(), Some(7_200));

    let missing = scenario.tick().await.unwrap();
    assert!(!missing.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario.settle_scheduled(&missing).await.unwrap();

    world.publish_prestate(rotated);
    let recovered = scenario.tick().await.unwrap();
    assert!(recovered.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario.settle_scheduled(&recovered).await.unwrap();
    assert_eq!(world.proof_record(&target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
}
