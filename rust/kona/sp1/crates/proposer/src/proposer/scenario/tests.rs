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

use super::{NamedBarrier, ScenarioControl, ScenarioError};
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
async fn tick_does_not_wait_for_fetch_interval() {
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
async fn run_ticks_immediately_then_waits_for_fetch_interval() {
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
async fn tick_reaps_finished_tasks_before_scheduling_replacements() {
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
async fn sync_error_skips_reaping_and_scheduling() {
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
async fn tick_schedules_other_work_when_creation_planning_fails() {
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
async fn tick_schedules_resolution_and_claims_when_defense_planning_fails() {
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
async fn snapshot_is_sorted_and_detached_from_proposer_state() {
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
async fn tick_distinguishes_proposal_and_reconciliation_summaries() {
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
async fn settle_timeout_preserves_unfinished_tasks() {
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
async fn task_finishing_after_reap_remains_settleable() {
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
async fn settle_rejects_task_reaped_by_later_tick() {
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
async fn tick_caps_defense_tasks_at_concurrency_limit() {
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
    assert_eq!(proposer.peak_concurrent_defense_tasks.load(Ordering::Relaxed), 2);
    active.release();
    control.settle(&[active.task_id]).await.unwrap();
}

#[tokio::test]
async fn tick_does_not_schedule_singletons_when_they_are_active() {
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
async fn tick_rejects_an_unparked_running_task() {
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
async fn tick_accepts_a_recorded_parked_task() {
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
async fn record_parked_requires_task_to_reach_matching_barrier() {
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
