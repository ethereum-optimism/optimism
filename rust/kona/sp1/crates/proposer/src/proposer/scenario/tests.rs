use std::{
    collections::HashSet,
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
    SyncDisposition, TaskClass, TaskCompletionOutcome, TaskFailureClass, TaskInfo, TaskSuccess,
};

#[derive(Default)]
struct ScenarioL1View {
    head_number: AtomicU64,
    head_timestamp: AtomicU64,
    latest_head_calls: AtomicU64,
    latest_head_notify: Notify,
    fail_latest_head: AtomicBool,
    fail_respected_game_type: AtomicBool,
    fail_latest_l1_timestamp_on: AtomicU64,
    latest_l1_timestamp_calls: AtomicU64,
    confirmed_block_unavailable: AtomicBool,
    release_barrier: StdMutex<Option<NamedBarrier>>,
    task_finished: StdMutex<Option<oneshot::Receiver<()>>>,
}

impl ScenarioL1View {
    fn new() -> Self {
        Self {
            head_number: AtomicU64::new(1),
            head_timestamp: AtomicU64::new(1_000),
            ..Default::default()
        }
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
        Ok(Some(L1BlockRef {
            number: self.head_number.load(Ordering::Relaxed),
            timestamp: self.head_timestamp.load(Ordering::Relaxed),
        }))
    }

    async fn block_ref(&self, number: u64) -> anyhow::Result<Option<L1BlockRef>> {
        if self.confirmed_block_unavailable.load(Ordering::Relaxed) {
            return Ok(None);
        }
        Ok(Some(L1BlockRef { number, timestamp: self.head_timestamp.load(Ordering::Relaxed) }))
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
        Ok(self.head_timestamp.load(Ordering::Relaxed))
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
        _keys: Option<Arc<ProofKeys>>,
        _game: GameProofInputs,
        _responses: Vec<SuperRootAtTimestampResponse>,
    ) -> anyhow::Result<Vec<u8>> {
        Ok(vec![1])
    }
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
            max_price_per_pgu: 1,
            min_auction_period: 1,
        },
    }
}

async fn proposer_with(mut config: ProposerConfig, l1_view: Arc<ScenarioL1View>) -> Arc<Proposer> {
    config.prestates_url = "file:///nonexistent".parse().unwrap();
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
    info: TaskInfo,
    handle: tokio::task::JoinHandle<anyhow::Result<TaskSuccess>>,
) -> u64 {
    let task_id = proposer.next_task_id.fetch_add(1, Ordering::Relaxed);
    proposer.tasks.lock().await.insert(task_id, (handle, info));
    task_id
}

fn pending_handle() -> tokio::task::JoinHandle<anyhow::Result<TaskSuccess>> {
    tokio::spawn(async {
        std::future::pending::<()>().await;
        Ok(TaskSuccess::Completed)
    })
}

#[tokio::test(start_paused = true)]
async fn direct_tick_has_no_cadence_delay() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(86_400), view).await;
    let mut control = ScenarioControl::new(proposer, Duration::from_secs(1));
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
    assert!(result.scheduled.windows(2).all(|pair| pair[0].task_id < pair[1].task_id));
}

#[tokio::test(start_paused = true)]
async fn production_runner_keeps_immediate_and_interval_ticks() {
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
async fn unchanged_head_reaps_before_replacement_and_sync_failure_is_a_hard_gate() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    proposer.sync_state().await.unwrap();
    let (release_tx, release_rx) = oneshot::channel();
    let (done_tx, done_rx) = oneshot::channel();
    let completed = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ResolutionSweep),
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
        Some(L1BlockRef { number: 1, timestamp: 1_000 })
    );
    assert_eq!(
        second.completions.iter().map(|completion| completion.task_id).collect::<Vec<_>>(),
        vec![completed]
    );
    assert!(
        second
            .scheduled
            .iter()
            .all(|scheduled| { second.completions.last().unwrap().task_id < scheduled.task_id })
    );

    let tasks_before = proposer.tasks.lock().await.len();
    let next_id_before = proposer.next_task_id.load(Ordering::Relaxed);
    view.fail_latest_head.store(true, Ordering::Relaxed);
    assert!(proposer.cycle().await.is_err());
    assert_eq!(proposer.tasks.lock().await.len(), tasks_before);
    assert_eq!(proposer.next_task_id.load(Ordering::Relaxed), next_id_before);

    let view = Arc::new(ScenarioL1View::new());
    view.head_number.store(2, Ordering::Relaxed);
    view.confirmed_block_unavailable.store(true, Ordering::Relaxed);
    let mut config = test_config(30);
    config.sync_l1_confirmations = 1;
    let proposer = proposer_with(config, view).await;

    assert_eq!(proposer.sync_state().await.unwrap(), SyncDisposition::ConfirmedBlockUnavailable);
    assert_eq!(*proposer.last_successful_pinned_l1.read().await, None);
}

#[tokio::test]
async fn planning_failures_keep_later_task_classes_isolated() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    proposer.state.write().await.games.insert(U256::ONE, challenged_game(1, 5_000));
    view.fail_respected_game_type.store(true, Ordering::Relaxed);

    let result = proposer.cycle().await.unwrap();
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

    let result = proposer.cycle().await.unwrap();
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
async fn snapshots_are_ordered_immutable_and_creation_summaries_are_distinct() {
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
    let claim_id = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ClaimSweep),
        pending_handle(),
    )
    .await;
    let resolution_id = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ResolutionSweep),
        pending_handle(),
    )
    .await;

    let result = proposer.cycle().await.unwrap();
    assert_eq!(
        result.snapshot.last_successful_pinned_l1,
        Some(L1BlockRef { number: 1, timestamp: 1_000 })
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
        vec![resolution_id, claim_id]
    );
    let snapshot = result.snapshot.clone();
    proposer.pending_games.write().await.clear();
    proposer.state.write().await.anchor_game = None;
    assert_eq!(result.snapshot, snapshot);

    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let proposed = proposer.cycle().await.unwrap();
    let proposal = proposed
        .scheduled
        .iter()
        .find(|scheduled| {
            matches!(
                scheduled.operation,
                OperationSummary::ProposeGame {
                    sequence_number: 3_600,
                    parent_game_index: u32::MAX
                }
            )
        })
        .unwrap();
    let proposed_tasks = proposer.tasks.lock().await;
    assert_eq!(proposed_tasks.get(&proposal.task_id).unwrap().1.class, TaskClass::Creation);
    assert_eq!(
        proposed_tasks.get(&proposal.task_id).unwrap().1.deduplication,
        crate::proposer::TaskDeduplication::Creation
    );
    drop(proposed_tasks);

    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    proposer.sync_state().await.unwrap();
    *proposer.in_flight_creation.lock().await = Some(InFlightCreation {
        root_claim: B256::left_padding_from(&[9]),
        extra_data: vec![1, 2, 3],
        sequence_number: 7_200,
        parent_game_index: 4,
    });
    let reconciled = proposer.cycle().await.unwrap();
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
    let tasks = proposer.tasks.lock().await;
    let info = &tasks.get(&creation.task_id).unwrap().1;
    assert_eq!(info.class, TaskClass::Creation);
    assert_eq!(info.deduplication, crate::proposer::TaskDeduplication::Creation);
}

#[tokio::test]
async fn shared_finalizer_reports_outcomes_and_task_lifecycle() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
    let info = || TaskInfo::from_operation(OperationSummary::ResolutionSweep);
    let success =
        insert_task(&proposer, info(), tokio::spawn(async { Ok(TaskSuccess::Completed) })).await;
    let failed =
        insert_task(&proposer, info(), tokio::spawn(async { anyhow::bail!("worker failed") }))
            .await;
    let terminal = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ProveGame {
            factory_index: U256::ONE,
            address: Address::left_padding_from(&[1]),
            purpose: ProvingPurpose::Defense,
        }),
        tokio::spawn(async { Ok(TaskSuccess::TerminallyUnprovable) }),
    )
    .await;
    let panicked = insert_task(
        &proposer,
        info(),
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
    assert_eq!(
        control.settle(&[success]).await.unwrap_err(),
        ScenarioError::AlreadyFinalized { task_id: success }
    );
    let unknown = proposer.next_task_id.load(Ordering::Relaxed) + 10;
    assert_eq!(
        control.settle(&[unknown]).await.unwrap_err(),
        ScenarioError::UnknownTask { task_id: unknown }
    );

    let completed_before_timeout =
        insert_task(&proposer, info(), tokio::spawn(async { Ok(TaskSuccess::Completed) })).await;
    let barrier = NamedBarrier::new("partial settlement barrier");
    let task_barrier = barrier.clone();
    let blocked = insert_task(
        &proposer,
        info(),
        tokio::spawn(async move {
            task_barrier.park().await;
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    barrier.wait_until_reached().await;
    let mut short_control = ScenarioControl::new(proposer.clone(), Duration::from_millis(10));
    assert_eq!(
        short_control.settle(&[completed_before_timeout, blocked]).await.unwrap_err(),
        ScenarioError::SettlementWatchdog {
            task_ids: vec![completed_before_timeout, blocked],
            completions: vec![crate::proposer::TaskCompletion {
                task_id: completed_before_timeout,
                class: TaskClass::Resolution,
                target: crate::proposer::OperationTarget::AllGames,
                outcome: TaskCompletionOutcome::Success,
            }],
        }
    );
    assert_eq!(
        short_control.settle(&[completed_before_timeout]).await.unwrap_err(),
        ScenarioError::AlreadyFinalized { task_id: completed_before_timeout }
    );
    barrier.release();
    assert_eq!(short_control.settle(&[blocked]).await.unwrap()[0].task_id, blocked);
}

#[tokio::test]
async fn task_finishing_during_tick_remains_explicitly_settleable() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view.clone()).await;
    proposer.sync_state().await.unwrap();
    let barrier = NamedBarrier::new("completion barrier");
    let (done_tx, done_rx) = oneshot::channel();
    *view.release_barrier.lock().unwrap() = Some(barrier.clone());
    *view.task_finished.lock().unwrap() = Some(done_rx);
    let task_barrier = barrier.clone();
    let task_id = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ResolutionSweep),
        tokio::spawn(async move {
            task_barrier.park().await;
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
}

#[tokio::test]
async fn proving_dedup_and_singleton_caps_are_preserved() {
    let view = Arc::new(ScenarioL1View::new());
    let mut config = test_config(30);
    config.max_concurrent_defense_tasks = NonZeroU64::new(2).unwrap();
    let proposer = proposer_with(config, view.clone()).await;
    proposer.sync_state().await.unwrap();
    {
        let mut state = proposer.state.write().await;
        for (index, deadline) in [(1, 5_000), (2, 6_000), (3, 7_000)] {
            state.games.insert(U256::from(index), challenged_game(index, deadline));
        }
    }
    let game_one = challenged_game(1, 5_000);
    insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ProveGame {
            factory_index: game_one.index,
            address: game_one.address,
            purpose: ProvingPurpose::Defense,
        }),
        pending_handle(),
    )
    .await;
    insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ProposeGame {
            sequence_number: 9,
            parent_game_index: 8,
        }),
        pending_handle(),
    )
    .await;
    insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ResolutionSweep),
        pending_handle(),
    )
    .await;
    insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ClaimSweep),
        pending_handle(),
    )
    .await;

    let result = proposer.cycle().await.unwrap();
    let proving = result
        .scheduled
        .iter()
        .filter_map(|scheduled| match scheduled.operation {
            OperationSummary::ProveGame { address, .. } => Some(address),
            _ => None,
        })
        .collect::<Vec<_>>();
    assert_eq!(proving, vec![challenged_game(2, 6_000).address]);
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
    let active_proving = proposer
        .tasks
        .lock()
        .await
        .values()
        .filter(|(_, info)| info.class == TaskClass::Proving)
        .count();
    assert_eq!(active_proving, 2);
    assert_eq!(
        proposer
            .tasks
            .lock()
            .await
            .values()
            .filter_map(|(_, info)| match info.deduplication {
                crate::proposer::TaskDeduplication::Proving(address) => Some(address),
                _ => None,
            })
            .collect::<HashSet<_>>()
            .len(),
        2
    );
}

#[tokio::test]
async fn tick_enforces_running_task_discipline_and_settlement_watchdog() {
    let view = Arc::new(ScenarioL1View::new());
    let proposer = proposer_with(test_config(30), view).await;
    let barrier = NamedBarrier::new("resolution barrier");
    let task_barrier = barrier.clone();
    let (barrier_released_tx, barrier_released_rx) = oneshot::channel();
    let (finish_tx, finish_rx) = oneshot::channel();
    let running = insert_task(
        &proposer,
        TaskInfo::from_operation(OperationSummary::ResolutionSweep),
        tokio::spawn(async move {
            task_barrier.park().await;
            let _ = barrier_released_tx.send(());
            let _ = finish_rx.await;
            Ok(TaskSuccess::Completed)
        }),
    )
    .await;
    barrier.wait_until_reached().await;
    let mut control = ScenarioControl::new(proposer.clone(), Duration::from_millis(10));

    assert_eq!(control.tick().await.unwrap_err(), ScenarioError::RunningTask { task_id: running });
    let unreached = NamedBarrier::new("not reached");
    assert_eq!(
        control.record_parked(running, &unreached).await.unwrap_err(),
        ScenarioError::BarrierNotReached { task_id: running, barrier: "not reached".into() }
    );
    control.record_parked(running, &barrier).await.unwrap();
    let parked_tick = control.tick().await.unwrap();
    let scheduled_ids =
        parked_tick.scheduled.iter().map(|scheduled| scheduled.task_id).collect::<Vec<_>>();
    control.settle(&scheduled_ids).await.unwrap();
    assert_eq!(
        control.settle(&[running]).await.unwrap_err(),
        ScenarioError::SettlementWatchdog { task_ids: vec![running], completions: vec![] }
    );
    barrier.release();
    let _ = barrier_released_rx.await;
    assert_eq!(control.tick().await.unwrap_err(), ScenarioError::RunningTask { task_id: running });
    let _ = finish_tx.send(());
    assert_eq!(control.settle(&[running]).await.unwrap()[0].task_id, running);
}
