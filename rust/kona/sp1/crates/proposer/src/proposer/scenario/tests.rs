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

#[tokio::test]
async fn action_fallback_applies_independently_to_each_target() {
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
async fn confirmed_actions_preserve_pinned_snapshots_and_logical_time() {
    let world = ScenarioWorld::new();
    world.set_sync_confirmations(2);
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
async fn scenario_clocks_move_independently() {
    let world = ScenarioWorld::new();
    world.set_sync_confirmations(2);
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
async fn blocked_creation_stays_single_until_its_task_is_released() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.block_action(
        target.clone(),
        1,
        BarrierPoint::AfterSubmission,
        ActionOutcome::Success,
        "create submitted",
    );
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let first = scenario.tick().await.unwrap();
    let create_id = scheduled_task(&first, |operation| {
        matches!(operation, OperationSummary::ProposeGame { .. })
    });
    assert_eq!(create_id, 1);
    scenario
        .wait_for_action_barrier(create_id, &target, 1, BarrierPoint::AfterSubmission)
        .await
        .unwrap();
    scenario.settle(&other_task_ids(&first, create_id)).await.unwrap();

    let blocked = scenario.tick().await.unwrap();
    assert!(blocked.snapshot.active_tasks.iter().any(|task| task.task_id == create_id));
    assert!(!blocked.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProposeGame { .. } | OperationSummary::ReconcileCreation { .. }
    )));
    scenario
        .settle(&blocked.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    scenario.release_action_barrier(&target, 1, BarrierPoint::AfterSubmission).unwrap();
    scenario.settle(&[create_id]).await.unwrap();
    assert_eq!(world.observation().games.len(), 1);
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::Confirmed);
}

#[tokio::test]
async fn blocked_proof_keeps_its_slot_and_allows_another_defense() {
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
    let first_id = scheduled_task(&first, |operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address, .. } if *address == first_target.address
        )
    });
    assert_eq!(first_id, 1);
    scenario.wait_for_proof_barrier(first_id, &first_target, 1).await.unwrap();
    scenario.settle(&other_task_ids(&first, first_id)).await.unwrap();

    let mut second_game =
        ScenarioGame::new(1, 0, 2, ScenarioWorld::default_prestate()).challenged();
    second_game.deadline = 6_000;
    let second_target = second_game.target();
    world.add_game(second_game);
    world.set_horizons(2, 2);
    let second = scenario.tick().await.unwrap();
    assert!(!second.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == first_target.address
    )));
    let second_id = scheduled_task(&second, |operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address, .. } if *address == second_target.address
        )
    });
    assert_eq!(
        scenario
            .proposer
            .tasks
            .lock()
            .await
            .values()
            .filter(|(_, operation)| matches!(operation, OperationSummary::ProveGame { .. }))
            .count(),
        2
    );
    scenario
        .settle(&second.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(world.proof_record(&second_target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);

    scenario.release_proof_barrier(&first_target, 1).unwrap();
    scenario.settle(&[first_id]).await.unwrap();
    assert_eq!(world.proof_record(&first_target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert!(world.action_record(&ActionTarget::Prove(first_target), 1).is_some());
    assert!(world.action_record(&ActionTarget::Prove(second_target), 1).is_some());
    assert!(second_id > first_id);
}

#[tokio::test]
async fn signer_gate_hides_queued_submissions_until_the_holder_finishes() {
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
        BarrierPoint::AfterSubmission,
        ActionOutcome::Success,
        "create owns signer",
    );
    world.block_action(
        resolve.clone(),
        1,
        BarrierPoint::BeforeSigner,
        ActionOutcome::Success,
        "resolve before signer",
    );
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    let create_id = scheduled_task(&result, |operation| {
        matches!(operation, OperationSummary::ProposeGame { .. })
    });
    let resolve_id =
        scheduled_task(&result, |operation| matches!(operation, OperationSummary::ResolutionSweep));
    scenario
        .wait_for_action_barrier(resolve_id, &resolve, 1, BarrierPoint::BeforeSigner)
        .await
        .unwrap();
    scenario
        .wait_for_action_barrier(create_id, &create, 1, BarrierPoint::AfterSubmission)
        .await
        .unwrap();
    scenario.release_action_barrier(&resolve, 1, BarrierPoint::BeforeSigner).unwrap();
    assert!(world.action_record(&resolve, 1).is_none());

    scenario.release_action_barrier(&create, 1, BarrierPoint::AfterSubmission).unwrap();
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
async fn pre_submit_failure_reconciles_without_consuming_a_nonce() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::PreSubmitFailure);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let failed = scenario.tick().await.unwrap();
    let create_id = scheduled_task(&failed, |operation| {
        matches!(operation, OperationSummary::ProposeGame { .. })
    });
    scenario
        .settle(&failed.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(create_id, 1);
    assert_eq!(world.observation().nonce, NonceState { pending: 0, latest: 0 });
    assert!(world.observation().games.is_empty());
    assert_eq!(
        world.action_record(&target, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );

    let reconcile = scenario.tick().await.unwrap();
    assert!(reconcile.snapshot.in_flight_creation.is_some());
    assert!(reconcile.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ReconcileCreation { .. }
    )));
    assert!(
        !reconcile
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario
        .settle(&reconcile.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    world.set_horizons(0, 0);
    let cleared = scenario.tick().await.unwrap();
    assert!(cleared.snapshot.in_flight_creation.is_none());
    scenario
        .settle(&cleared.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}

#[tokio::test]
async fn create_revert_is_terminal_without_a_world_effect() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Revert);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    scenario
        .settle(&result.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    let observation = world.observation();
    assert_eq!(observation.nonce, NonceState { pending: 1, latest: 1 });
    assert!(observation.games.is_empty());
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::Reverted);
    world.set_horizons(0, 0);
    let next = scenario.tick().await.unwrap();
    assert!(next.snapshot.in_flight_creation.is_none());
    scenario
        .settle(&next.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}

#[tokio::test]
async fn timed_out_create_can_land_late_and_be_adopted() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Timeout);
    let mut config = scenario_config();
    config.sync_l1_confirmations = 2;
    let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

    let timed_out = scenario.tick().await.unwrap();
    scenario
        .settle(&timed_out.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(world.observation().nonce, NonceState { pending: 1, latest: 0 });
    assert!(world.observation().games.is_empty());
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::TimedOut);

    let held = scenario.tick().await.unwrap();
    assert!(held.snapshot.in_flight_creation.is_some());
    scenario
        .settle(&held.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    world.include_transaction(&target, 1, InclusionDepth::LatestOnly).unwrap();
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::IncludedLate);
    assert_eq!(world.observation().games.len(), 1);

    let adopted = scenario.tick().await.unwrap();
    assert!(adopted.snapshot.in_flight_creation.is_some());
    assert!(
        !adopted
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario
        .settle(&adopted.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    let learned = scenario.tick().await.unwrap();
    assert!(learned.snapshot.in_flight_creation.is_none());
    assert!(
        !learned
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario
        .settle(&learned.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}

#[tokio::test]
async fn dropped_create_drains_the_nonce_and_clears_on_reconciliation() {
    let world = ScenarioWorld::new();
    world.set_horizons(1, 1);
    let target = ActionTarget::Create { sequence_number: 1, parent_game_index: u32::MAX };
    world.script_action(target.clone(), 1, ActionOutcome::Timeout);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let timed_out = scenario.tick().await.unwrap();
    scenario
        .settle(&timed_out.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    world.drop_transaction(&target, 1).unwrap();
    assert_eq!(world.observation().nonce, NonceState { pending: 0, latest: 0 });
    assert_eq!(world.action_record(&target, 1).unwrap().lifecycle, ActionLifecycle::Dropped);

    let reconcile = scenario.tick().await.unwrap();
    assert!(reconcile.snapshot.in_flight_creation.is_some());
    scenario
        .settle(&reconcile.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    world.set_horizons(0, 0);
    let cleared = scenario.tick().await.unwrap();
    assert!(cleared.snapshot.in_flight_creation.is_none());
    assert!(world.observation().games.is_empty());
    scenario
        .settle(&cleared.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}

#[tokio::test]
async fn proof_faults_and_fallbacks_are_scoped_to_target_and_attempt() {
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
    scenario
        .settle(&first.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
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
    scenario
        .settle(&retry.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(world.proof_record(&failed, 2).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert_eq!(world.proof_record(&panicked, 2).unwrap().lifecycle, ProofLifecycle::Succeeded);
}

#[tokio::test]
async fn action_failures_do_not_hide_other_targets_or_claim_effects() {
    let world = ScenarioWorld::new();
    let resolve_failed_game = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate())
        .provable_for_resolution();
    let resolve_succeeded_game =
        ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate())
            .provable_for_resolution();
    let claim_failed_game =
        ScenarioGame::new(2, u32::MAX, 3, ScenarioWorld::default_prestate()).claimable(10);
    let claim_succeeded_game =
        ScenarioGame::new(3, u32::MAX, 4, ScenarioWorld::default_prestate()).claimable(20);
    let resolve_failed = ActionTarget::Resolve(resolve_failed_game.target());
    let resolve_succeeded = ActionTarget::Resolve(resolve_succeeded_game.target());
    let claim_failed = ActionTarget::ClaimCredit(claim_failed_game.target());
    let claim_succeeded = ActionTarget::ClaimCredit(claim_succeeded_game.target());
    for game in
        [resolve_failed_game, resolve_succeeded_game, claim_failed_game, claim_succeeded_game]
    {
        world.add_game(game);
    }
    world.set_horizons(4, 4);
    world.script_next_action(resolve_failed.clone(), ActionOutcome::PreSubmitFailure);
    world.script_next_action(claim_failed.clone(), ActionOutcome::PreSubmitFailure);
    world.script_action_fallback(ActionOutcome::Success);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let first = scenario.tick().await.unwrap();
    scenario
        .settle(&first.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(
        world.action_record(&resolve_failed, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
    assert_eq!(
        world.action_record(&resolve_succeeded, 1).unwrap().effect,
        CommittedEffect::Resolved {
            game: match &resolve_succeeded {
                ActionTarget::Resolve(target) => target.address,
                _ => unreachable!(),
            }
        }
    );
    assert_eq!(
        world.action_record(&claim_failed, 1).unwrap().lifecycle,
        ActionLifecycle::PreSubmitFailed
    );
    let claim_address = match &claim_succeeded {
        ActionTarget::ClaimCredit(target) => target.address,
        _ => unreachable!(),
    };
    assert_eq!(
        world.action_record(&claim_succeeded, 1).unwrap().effect,
        CommittedEffect::ClaimUnlocked { game: claim_address, amount: U256::from(20) }
    );

    world.set_latest_l1_time(world.observation().latest_l1.timestamp + 20);
    let payout = scenario.tick().await.unwrap();
    scenario
        .settle(&payout.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(
        world.action_record(&claim_succeeded, 2).unwrap().effect,
        CommittedEffect::ClaimPaid { game: claim_address, amount: U256::from(20) }
    );
}

#[tokio::test]
async fn read_faults_stay_at_their_scripted_failure_boundaries() {
    let world = ScenarioWorld::new();
    let first_game = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    let first_address = first_game.address;
    world.add_game(first_game);
    world.set_horizons(1, 1);
    let factory_key = ReadKey::factory(ReadBoundary::FactoryGame, U256::ZERO);
    world.script_read_failure(factory_key.clone(), 1, "factory walk failed");
    let unused_bond = ReadKey::game(ReadBoundary::BondState, first_address);
    world.script_read_failure(unused_bond.clone(), 1, "unused bond fault");
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    assert!(matches!(scenario.tick().await, Err(ScenarioError::Cycle(_))));
    assert_eq!(world.read_attempts(&factory_key), 1);
    let pending_game = ScenarioGame::new(1, u32::MAX, 5, ScenarioWorld::default_prestate());
    world.add_game(pending_game);
    world.set_superroot(5, SuperRootSetting::Absent { current_l1: 4, local_safe: 1, finalized: 1 });
    let recovered = scenario.tick().await.unwrap();
    assert_eq!(
        recovered.snapshot.pending_games.iter().map(|game| game.factory_index).collect::<Vec<_>>(),
        vec![U256::ONE]
    );
    assert_eq!(world.read_attempts(&unused_bond), 0);
    scenario
        .settle(&recovered.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    let mut challenged =
        ScenarioGame::new(2, u32::MAX, 2, ScenarioWorld::default_prestate()).challenged();
    challenged.deadline = 6_000;
    let challenged_address = challenged.address;
    world.add_game(challenged);
    world.set_horizons(2, 2);
    let cached_refresh = ReadKey::game(ReadBoundary::GameLifecycle, first_address);
    let refresh_attempts = world.read_attempts(&cached_refresh);
    world.script_next_read_failure(cached_refresh.clone(), "cached refresh failed");
    let isolated = scenario.tick().await.unwrap();
    assert_eq!(world.read_attempts(&cached_refresh), refresh_attempts + 1);
    assert!(isolated.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == challenged_address
    )));
    scenario
        .settle(&isolated.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    world.script_read_fallback("ordered fallback");
    assert!(world.l1_view().latest_l1_timestamp().await.is_err());
}

#[tokio::test]
async fn superroot_journal_separates_horizons_from_proof_span_queries() {
    let world = ScenarioWorld::new();
    let mut game =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).challenged();
    game.deadline = 5_000;
    world.add_game(game);
    world.set_host_time(9_000);
    world.set_horizons(1, 1);
    world.set_superroot(
        0,
        SuperRootSetting::Available {
            root: canonical_super_root(0),
            proof: vec![0x01],
            current_l1: 10,
            required_l1: 9,
        },
    );
    world.set_superroot(
        1,
        SuperRootSetting::Available {
            root: canonical_super_root(1),
            proof: vec![0x01],
            current_l1: 11,
            required_l1: 10,
        },
    );
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let result = scenario.tick().await.unwrap();
    scenario
        .settle(&result.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    let journal = world.superroot_journal();
    assert!(journal.iter().any(|record| matches!(
        record,
        SuperRootQueryRecord::Horizon { request_time: 9_000, safe: 1, finalized: 1 }
    )));
    assert!(journal.iter().any(|record| matches!(
        record,
        SuperRootQueryRecord::AtTimestamp {
            timestamp: 0,
            current_l1: 10,
            required_l1: Some(9),
            available: true,
            ..
        }
    )));
    assert!(journal.iter().any(|record| matches!(
        record,
        SuperRootQueryRecord::AtTimestamp {
            timestamp: 1,
            current_l1: 11,
            required_l1: Some(10),
            safe: 1,
            local_safe: 1,
            finalized: 1,
            available: true,
        }
    )));

    world.set_superroot(2, SuperRootSetting::Failure("superroot unavailable".into()));
    assert!(world.superroot_source().super_root_at_timestamp(2).await.is_err());
    assert!(world.superroot_journal().iter().any(|record| matches!(
        record,
        SuperRootQueryRecord::FailedAtTimestamp { timestamp: 2, error }
            if error == "superroot unavailable"
    )));
}

#[tokio::test(start_paused = true)]
async fn scenario_initialization_runs_once_and_returns_errors_without_retrying() {
    let world = ScenarioWorld::new();
    let initial_block = world.observation().latest_l1.number;
    let registered_args = ReadKey {
        boundary: ReadBoundary::RegisteredGameArgs,
        target: ReadTarget::Block(initial_block),
    };
    let before = tokio::time::Instant::now();
    let scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();
    assert_eq!(world.read_attempts(&registered_args), 1);
    assert_eq!(scenario.proposer.max_prove_duration.get(), Some(&DEFAULT_MAX_DURATION));
    assert_eq!(tokio::time::Instant::now(), before);

    let failing_world = ScenarioWorld::new();
    let failing_block = failing_world.observation().latest_l1.number;
    let failing_read = ReadKey {
        boundary: ReadBoundary::RegisteredGameArgs,
        target: ReadTarget::Block(failing_block),
    };
    failing_world.script_read_failure(failing_read.clone(), 1, "initialization unavailable");
    let before_failure = tokio::time::Instant::now();
    let error = match ScenarioHarness::new(failing_world.clone(), scenario_config()).await {
        Ok(_) => panic!("initialization should fail"),
        Err(error) => error,
    };
    assert_eq!(error, ScenarioError::Initialization("initialization unavailable".into()));
    assert_eq!(failing_world.read_attempts(&failing_read), 1);
    assert_eq!(tokio::time::Instant::now(), before_failure);
}

#[tokio::test]
async fn uninitialized_defense_error_does_not_suppress_sweeps() {
    let world = ScenarioWorld::new();
    let mut first =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).challenged();
    first.deadline = 5_000;
    let first_address = first.address;
    let mut second =
        ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate()).challenged();
    second.deadline = 6_000;
    world.add_game(first);
    world.add_game(second);
    world.set_horizons(2, 2);
    let config = scenario_config();
    let proposer = ScenarioHarness::uninitialized(&world, &config).await.unwrap();
    let error = proposer.should_skip_proving(first_address, 5_000, true).await.unwrap_err();
    assert!(error.to_string().contains("max_prove_duration must be set via try_init"));
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));

    let result = control.tick().await.unwrap();
    assert!(
        !result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ProveGame { .. }))
    );
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ResolutionSweep))
    );
    assert!(
        result
            .scheduled
            .iter()
            .any(|scheduled| matches!(scheduled.operation, OperationSummary::ClaimSweep))
    );
    control
        .settle(&result.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}

#[tokio::test]
async fn published_rotated_prestate_becomes_owned_on_the_next_tick() {
    let world = ScenarioWorld::new();
    let rotated = B256::repeat_byte(0x33);
    world.rotate_registered_prestate(rotated, 7_200);
    let mut game = ScenarioGame::new(0, u32::MAX, 1, rotated).challenged();
    game.deadline = 5_000;
    let target = game.target();
    world.add_game(game);
    world.set_horizons(1, 1);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();
    assert_eq!(scenario.proposer.max_prove_duration.get(), Some(&7_200));

    let missing = scenario.tick().await.unwrap();
    assert!(!missing.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario
        .settle(&missing.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    world.publish_prestate(rotated);
    let recovered = scenario.tick().await.unwrap();
    assert!(recovered.scheduled.iter().any(|scheduled| matches!(
        scheduled.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario
        .settle(&recovered.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(world.proof_record(&target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
}

#[tokio::test]
async fn restart_rebuilds_external_games_and_resets_local_guards_and_ids() {
    let world = ScenarioWorld::new();
    let mut game =
        ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate()).challenged();
    game.deadline = 5_000;
    let address = game.address;
    world.add_game(game);
    world.update_game(address, |game| {
        game.proof_inputs.starting_root = B256::repeat_byte(0xee);
    });
    world.set_anchor(address, 0);
    world.set_horizons(1, 1);
    let mut scenario = ScenarioHarness::new(world.clone(), scenario_config()).await.unwrap();

    let terminal = scenario.tick().await.unwrap();
    let terminal_id = scheduled_task(&terminal, |operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address: scheduled, .. } if *scheduled == address
        )
    });
    scenario
        .settle(&terminal.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(terminal_id, 1);
    assert!(scenario.proposer.undefendable.lock().await.contains(&address));

    scenario.restart().await.unwrap();
    assert!(scenario.proposer.undefendable.lock().await.is_empty());
    let retried = scenario.tick().await.unwrap();
    let retried_id = scheduled_task(&retried, |operation| {
        matches!(
            operation,
            OperationSummary::ProveGame { address: scheduled, .. } if *scheduled == address
        )
    });
    assert_eq!(retried_id, 1);
    assert_eq!(retried.snapshot.canonical_head_index, Some(U256::ZERO));
    scenario
        .settle(&retried.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();

    world.update_game(address, |game| {
        game.proposal_status = ProposalStatus::ChallengedAndValidProofProvided;
        game.proof_inputs.starting_root = canonical_super_root(0);
    });
    world.set_horizons(2, 2);
    scenario.restart().await.unwrap();
    let create_target = ActionTarget::Create { sequence_number: 2, parent_game_index: u32::MAX };
    world.script_action(create_target.clone(), 1, ActionOutcome::Timeout);
    let uncertain = scenario.tick().await.unwrap();
    scenario
        .settle(&uncertain.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
    assert_eq!(
        world.action_record(&create_target, 1).unwrap().lifecycle,
        ActionLifecycle::TimedOut
    );
    world.drop_transaction(&create_target, 1).unwrap();

    scenario.restart().await.unwrap();
    world.set_horizons(1, 1);
    let fresh = scenario.tick().await.unwrap();
    assert!(fresh.snapshot.in_flight_creation.is_none());
    assert!(fresh.snapshot.active_tasks.is_empty());
    assert_eq!(fresh.snapshot.canonical_head_index, Some(U256::ZERO));
    assert_eq!(fresh.scheduled.iter().map(|scheduled| scheduled.task_id).min(), Some(1));
    scenario
        .settle(&fresh.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>())
        .await
        .unwrap();
}
