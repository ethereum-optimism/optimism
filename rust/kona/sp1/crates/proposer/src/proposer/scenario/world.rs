use std::{
    collections::{BTreeMap, HashMap, VecDeque},
    fs,
    hash::Hash,
    io::Write,
    num::{NonZeroU64, NonZeroUsize},
    path::{Path, PathBuf},
    sync::{
        Arc, Mutex as StdMutex,
        atomic::{AtomicU64, Ordering},
    },
    time::Duration,
};

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, U256};
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, anyhow, bail};
use async_trait::async_trait;
use flate2::{Compression, write::GzEncoder};
use kona_sp1_host_utils::metrics::MetricsListen;
use kona_sp1_super_range_executor::{
    BlockId as SuperBlockId, SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1,
};
use tokio::sync::Mutex as AsyncMutex;

use super::{NamedBarrier, ScenarioControl, ScenarioError};
use crate::{
    TX_REVERTED_PREFIX, ZK_GAME_TYPE,
    config::{
        AGGREGATION_ARTIFACT_SUFFIX, ProofProviderConfig, ProofProviderKind, ProposalSafety,
        ProposerConfig, RANGE_ARTIFACT_SUFFIX, RangeSplitCount,
    },
    contract::{GameStatus, ProposalStatus, ZKGameArgs},
    ports::{
        ActionExecutor, AnchorRoot, BondState, ClaimPreflight, FactoryGame, GameClaim,
        GameCreationReceipt, GameIdentity, GameLifecycle, GameStanding, GameValidity, L1BlockRef,
        L1View, NonceState, ProofEngine, ProofInputs, ProposalHorizon, QueryTime,
        SuperRootAtTimestamp, SuperRootSource, WithdrawalState,
    },
    proposer::{CycleResult, OperationSummary, PrestateCache, Proposer, TaskCompletion, TaskId},
    prover::ProofKeys,
    proving::GameProofInputs,
    signer::NUM_CONFIRMATIONS,
    superroot::SuperRootAt,
};

const INITIAL_BLOCK: u64 = 10;
const INITIAL_L1_TIME: u64 = 1_000;
const DEFAULT_PRESTATE_BYTE: u8 = 0x11;
pub(super) const DEFAULT_MAX_DURATION: u64 = 3_600;

static ARTIFACT_DIR_ID: AtomicU64 = AtomicU64::new(1);

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(super) enum ReadTarget {
    Global,
    Block(u64),
    FactoryIndex(U256),
    Game(Address),
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(super) enum ReadBoundary {
    LatestHead,
    BlockRef,
    RegisteredGameArgs,
    AnchorRoot,
    LatestGameIndex,
    RegisteredAnchorGame,
    FactoryGame,
    GameClaim,
    GameIdentity,
    GameValidity,
    GameLifecycle,
    ParentGameStatus,
    BondState,
    InitBond,
    GameStatus,
    ClaimCredit,
    ClaimWithdrawal,
    WethDelay,
    GameByUuid,
    GameCreator,
    NonceState,
    RespectedGameType,
    ParentStanding,
    GameStanding,
    ProofStatus,
    ProofInputs,
    AnchorStateRegistry,
    LatestL1Timestamp,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(super) struct ReadKey {
    pub(super) boundary: ReadBoundary,
    pub(super) target: ReadTarget,
}

impl ReadKey {
    pub(super) const fn global(boundary: ReadBoundary) -> Self {
        Self { boundary, target: ReadTarget::Global }
    }

    pub(super) const fn game(boundary: ReadBoundary, game: Address) -> Self {
        Self { boundary, target: ReadTarget::Game(game) }
    }

    pub(super) const fn factory(boundary: ReadBoundary, index: U256) -> Self {
        Self { boundary, target: ReadTarget::FactoryIndex(index) }
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(super) struct GameTarget {
    pub(super) factory_index: U256,
    pub(super) address: Address,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(super) enum ActionTarget {
    Create { sequence_number: u64, parent_game_index: u32 },
    Prove(GameTarget),
    Resolve(GameTarget),
    ClaimCredit(GameTarget),
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum ActionOutcome {
    Success,
    PreSubmitFailure,
    Revert,
    Timeout,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum ProofOutcome {
    Success,
    Failure,
    Panic,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(super) enum BarrierPoint {
    BeforeSigner,
    AfterSubmission,
    Proof,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct BarrierKey {
    point: BarrierPoint,
    target: ScriptTarget,
    attempt: u64,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
enum ScriptTarget {
    Action(ActionTarget),
    Proof(GameTarget),
}

#[derive(Clone)]
struct ActionScript {
    outcome: ActionOutcome,
    before_signer: Option<NamedBarrier>,
    after_submission: Option<NamedBarrier>,
}

impl ActionScript {
    const fn immediate(outcome: ActionOutcome) -> Self {
        Self { outcome, before_signer: None, after_submission: None }
    }
}

#[derive(Clone)]
struct ProofScript {
    outcome: ProofOutcome,
    barrier: Option<NamedBarrier>,
}

impl ProofScript {
    const fn immediate(outcome: ProofOutcome) -> Self {
        Self { outcome, barrier: None }
    }
}

struct Scripts<K, V> {
    exact: HashMap<(K, u64), V>,
    queued: HashMap<K, VecDeque<V>>,
    fallback: Option<V>,
    attempts: HashMap<K, u64>,
}

impl<K, V> Default for Scripts<K, V> {
    fn default() -> Self {
        Self {
            exact: HashMap::new(),
            queued: HashMap::new(),
            fallback: None,
            attempts: HashMap::new(),
        }
    }
}

impl<K, V> Scripts<K, V>
where
    K: Clone + Eq + Hash,
    V: Clone,
{
    fn script_exact(&mut self, key: K, attempt: u64, value: V) {
        assert!(attempt > 0, "script attempts are one-based");
        assert!(self.exact.insert((key, attempt), value).is_none(), "duplicate exact script");
    }

    fn script_next(&mut self, key: K, value: V) {
        self.queued.entry(key).or_default().push_back(value);
    }

    fn script_fallback(&mut self, value: V) {
        assert!(self.fallback.replace(value).is_none(), "duplicate fallback script");
    }

    fn next(&mut self, key: &K) -> (u64, Option<V>) {
        let attempt = self.attempts.entry(key.clone()).or_default();
        *attempt += 1;
        let current = *attempt;
        let value = self
            .exact
            .remove(&(key.clone(), current))
            .or_else(|| self.queued.get_mut(key).and_then(VecDeque::pop_front))
            .or_else(|| self.fallback.clone());
        (current, value)
    }

    fn attempts(&self, key: &K) -> u64 {
        self.attempts.get(key).copied().unwrap_or_default()
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct ScenarioGame {
    pub(super) factory_index: U256,
    pub(super) address: Address,
    pub(super) game_type: u32,
    pub(super) parent_index: u32,
    pub(super) sequence_number: u64,
    pub(super) root_claim: B256,
    pub(super) extra_data: Vec<u8>,
    pub(super) proposal_status: ProposalStatus,
    pub(super) status: GameStatus,
    pub(super) deadline: u64,
    pub(super) finalized: bool,
    pub(super) was_respected: bool,
    pub(super) absolute_prestate: B256,
    pub(super) creator: Address,
    pub(super) weth: Address,
    pub(super) anchor_state_registry: Address,
    pub(super) standing: GameStanding,
    pub(super) bond: BondState,
    pub(super) proof_inputs: ProofInputs,
}

impl ScenarioGame {
    pub(super) fn new(index: u64, parent_index: u32, sequence_number: u64, prestate: B256) -> Self {
        let address = deterministic_address(0x40, index);
        let mut extra_data = parent_index.to_be_bytes().to_vec();
        extra_data.push(0x01);
        Self {
            factory_index: U256::from(index),
            address,
            game_type: ZK_GAME_TYPE,
            parent_index,
            sequence_number,
            root_claim: canonical_super_root(sequence_number),
            extra_data,
            proposal_status: ProposalStatus::Unchallenged,
            status: GameStatus::InProgress,
            deadline: INITIAL_L1_TIME + DEFAULT_MAX_DURATION,
            finalized: false,
            was_respected: true,
            absolute_prestate: prestate,
            creator: deterministic_address(0x50, index),
            weth: deterministic_address(0x60, 1),
            anchor_state_registry: deterministic_address(0x70, 1),
            standing: GameStanding { blacklisted: false, retired: false },
            bond: BondState {
                credit: U256::ZERO,
                withdrawal_amount: U256::ZERO,
                withdrawal_timestamp: U256::ZERO,
                delay: U256::from(10),
            },
            proof_inputs: ProofInputs {
                l1_head: deterministic_hash(0x80, INITIAL_BLOCK),
                l1_head_number: INITIAL_BLOCK,
                starting_root: canonical_super_root(sequence_number.saturating_sub(1)),
                starting_sequence_number: sequence_number.saturating_sub(1),
                root_claim: canonical_super_root(sequence_number),
                sequence_number,
            },
        }
    }

    pub(super) fn challenged(mut self) -> Self {
        self.proposal_status = ProposalStatus::Challenged;
        self
    }

    pub(super) fn provable_for_resolution(mut self) -> Self {
        self.proposal_status = ProposalStatus::ChallengedAndValidProofProvided;
        self
    }

    pub(super) fn claimable(mut self, credit: u64) -> Self {
        self.status = GameStatus::DefenderWins;
        self.proposal_status = ProposalStatus::Resolved;
        self.finalized = true;
        self.bond.credit = U256::from(credit);
        self
    }

    pub(super) fn target(&self) -> GameTarget {
        GameTarget { factory_index: self.factory_index, address: self.address }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) enum SuperRootSetting {
    Available { root: B256, proof: Vec<u8>, current_l1: u64, required_l1: u64 },
    Absent { current_l1: u64, local_safe: u64, finalized: u64 },
    Failure(String),
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) enum SuperRootQueryRecord {
    Horizon {
        request_time: u64,
        safe: u64,
        finalized: u64,
    },
    AtTimestamp {
        timestamp: u64,
        current_l1: u64,
        required_l1: Option<u64>,
        safe: u64,
        local_safe: u64,
        finalized: u64,
        available: bool,
    },
    FailedAtTimestamp {
        timestamp: u64,
        error: String,
    },
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum ActionLifecycle {
    PreSubmitFailed,
    Submitted,
    Confirmed,
    Reverted,
    TimedOut,
    IncludedLate,
    Dropped,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) enum CommittedEffect {
    None,
    Created { factory_index: U256, address: Address },
    Proven { game: Address },
    Resolved { game: Address },
    ClaimUnlocked { game: Address, amount: U256 },
    ClaimPaid { game: Address, amount: U256 },
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) enum ActionInputs {
    Create { root_claim: B256, parent_game_index: u32, sequence_number: u64 },
    Prove { game: Address },
    Resolve { game: Address },
    ClaimCredit { game: Address, recipient: Address },
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct ActionRecord {
    pub(super) target: ActionTarget,
    pub(super) attempt: u64,
    pub(super) inputs: ActionInputs,
    pub(super) lifecycle: ActionLifecycle,
    pub(super) transaction_hash: Option<B256>,
    pub(super) effect: CommittedEffect,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum ProofLifecycle {
    Parked,
    Succeeded,
    Failed,
    Panicked,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct ProofRecord {
    pub(super) target: GameTarget,
    pub(super) attempt: u64,
    pub(super) lifecycle: ProofLifecycle,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum InclusionDepth {
    LatestOnly,
    Confirmed,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct PendingTransactionObservation {
    pub(super) target: ActionTarget,
    pub(super) attempt: u64,
    pub(super) nonce: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct WorldObservation {
    pub(super) latest_l1: L1BlockRef,
    pub(super) host_time: u64,
    pub(super) safe_time: u64,
    pub(super) finalized_time: u64,
    pub(super) nonce: NonceState,
    pub(super) games: Vec<ScenarioGame>,
    pub(super) pending_transactions: Vec<PendingTransactionObservation>,
}

#[derive(Clone)]
struct L1State {
    block: L1BlockRef,
    registered_args: ZKGameArgs,
    anchor_root: AnchorRoot,
    registered_anchor_game: Address,
    respected_game_type: u32,
    init_bond: U256,
    latest_nonce: u64,
    games: BTreeMap<U256, ScenarioGame>,
}

#[derive(Clone)]
enum PendingEffect {
    Create {
        index: U256,
        address: Address,
        root_claim: B256,
        extra_data: Vec<u8>,
        sequence_number: u64,
        parent_game_index: u32,
    },
    Prove(Address),
    Resolve(Address),
    Claim(Address),
}

#[derive(Clone)]
struct PendingTransaction {
    target: ActionTarget,
    attempt: u64,
    nonce: u64,
    effect: PendingEffect,
}

struct ArtifactDirectory(PathBuf);

impl Drop for ArtifactDirectory {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

struct WorldData {
    states: BTreeMap<u64, Arc<L1State>>,
    latest_block: u64,
    host_time: u64,
    safe_time: u64,
    finalized_time: u64,
    sync_confirmations: u64,
    pending_nonce: u64,
    superroots: HashMap<u64, SuperRootSetting>,
    read_scripts: Scripts<ReadKey, String>,
    action_scripts: Scripts<ActionTarget, ActionScript>,
    proof_scripts: Scripts<GameTarget, ProofScript>,
    barriers: HashMap<BarrierKey, NamedBarrier>,
    superroot_journal: Vec<SuperRootQueryRecord>,
    action_records: HashMap<(ActionTarget, u64), ActionRecord>,
    proof_records: HashMap<(GameTarget, u64), ProofRecord>,
    pending_transactions: HashMap<(ActionTarget, u64), PendingTransaction>,
}

#[derive(Clone)]
pub(super) struct ScenarioWorld {
    data: Arc<StdMutex<WorldData>>,
    signer_gate: Arc<AsyncMutex<()>>,
    artifacts: Arc<ArtifactDirectory>,
}

impl ScenarioWorld {
    pub(super) fn new() -> Self {
        let artifacts = create_artifact_directory();
        let prestate = Self::default_prestate();
        publish_prestate_at(&artifacts.0, prestate);
        let registry = deterministic_address(0x70, 1);
        let weth = deterministic_address(0x60, 1);
        let initial = L1State {
            block: L1BlockRef { number: INITIAL_BLOCK, timestamp: INITIAL_L1_TIME },
            registered_args: ZKGameArgs {
                absolute_prestate: prestate,
                verifier: deterministic_address(0x71, 1),
                max_challenge_duration: DEFAULT_MAX_DURATION,
                max_prove_duration: DEFAULT_MAX_DURATION,
                challenger_bond: U256::ONE,
                anchor_state_registry: registry,
                weth,
            },
            anchor_root: AnchorRoot { root: canonical_super_root(0), sequence_number: U256::ZERO },
            registered_anchor_game: Address::ZERO,
            respected_game_type: ZK_GAME_TYPE,
            init_bond: U256::ONE,
            latest_nonce: 0,
            games: BTreeMap::new(),
        };
        let mut states = BTreeMap::new();
        states.insert(INITIAL_BLOCK, Arc::new(initial));
        Self {
            data: Arc::new(StdMutex::new(WorldData {
                states,
                latest_block: INITIAL_BLOCK,
                host_time: INITIAL_L1_TIME,
                safe_time: 4,
                finalized_time: 4,
                sync_confirmations: 0,
                pending_nonce: 0,
                superroots: HashMap::new(),
                read_scripts: Scripts::default(),
                action_scripts: Scripts::default(),
                proof_scripts: Scripts::default(),
                barriers: HashMap::new(),
                superroot_journal: Vec::new(),
                action_records: HashMap::new(),
                proof_records: HashMap::new(),
                pending_transactions: HashMap::new(),
            })),
            signer_gate: Arc::new(AsyncMutex::new(())),
            artifacts: Arc::new(artifacts),
        }
    }

    pub(super) const fn default_prestate() -> B256 {
        B256::repeat_byte(DEFAULT_PRESTATE_BYTE)
    }

    pub(super) const fn proposer_address() -> Address {
        Address::repeat_byte(0x22)
    }

    pub(super) fn prestates_url(&self) -> Url {
        Url::from_directory_path(&self.artifacts.0).expect("artifact path must form a file URL")
    }

    pub(super) fn publish_prestate(&self, prestate: B256) {
        publish_prestate_at(&self.artifacts.0, prestate);
    }

    pub(super) fn add_game(&self, mut game: ScenarioGame) -> L1BlockRef {
        self.append_block(|state| {
            game.proof_inputs.l1_head_number = state.block.number;
            game.proof_inputs.l1_head = deterministic_hash(0x80, state.block.number);
            if game.parent_index == u32::MAX {
                game.proof_inputs.starting_root = state.anchor_root.root;
                game.proof_inputs.starting_sequence_number =
                    state.anchor_root.sequence_number.to::<u64>();
            } else if let Some(parent) = state.games.get(&U256::from(game.parent_index)) {
                game.proof_inputs.starting_root = parent.root_claim;
                game.proof_inputs.starting_sequence_number = parent.sequence_number;
            }
            state.games.insert(game.factory_index, game);
        })
    }

    pub(super) fn update_game(
        &self,
        address: Address,
        update: impl FnOnce(&mut ScenarioGame),
    ) -> L1BlockRef {
        self.append_block(|state| {
            let game = state
                .games
                .values_mut()
                .find(|game| game.address == address)
                .expect("scenario game must exist");
            update(game);
        })
    }

    pub(super) fn rotate_registered_prestate(
        &self,
        prestate: B256,
        max_prove_duration: u64,
    ) -> L1BlockRef {
        self.append_block(|state| {
            state.registered_args.absolute_prestate = prestate;
            state.registered_args.max_prove_duration = max_prove_duration;
        })
    }

    pub(super) fn set_anchor(&self, game: Address, sequence_number: u64) -> L1BlockRef {
        self.append_block(|state| {
            state.registered_anchor_game = game;
            state.anchor_root = AnchorRoot {
                root: canonical_super_root(sequence_number),
                sequence_number: U256::from(sequence_number),
            };
        })
    }

    pub(super) fn set_latest_l1_time(&self, timestamp: u64) -> L1BlockRef {
        self.append_block(|state| state.block.timestamp = timestamp)
    }

    pub(super) fn mine_block(&self) -> L1BlockRef {
        self.append_block(|_| {})
    }

    pub(super) fn set_host_time(&self, timestamp: u64) {
        self.lock().host_time = timestamp;
    }

    pub(super) fn set_horizons(&self, safe: u64, finalized: u64) {
        let mut data = self.lock();
        data.safe_time = safe;
        data.finalized_time = finalized;
    }

    pub(super) fn set_safe_time(&self, safe: u64) {
        self.lock().safe_time = safe;
    }

    pub(super) fn set_finalized_time(&self, finalized: u64) {
        self.lock().finalized_time = finalized;
    }

    pub(super) fn set_sync_confirmations(&self, confirmations: u64) {
        self.lock().sync_confirmations = confirmations;
    }

    pub(super) fn set_superroot(&self, timestamp: u64, setting: SuperRootSetting) {
        self.lock().superroots.insert(timestamp, setting);
    }

    pub(super) fn script_read_failure(&self, key: ReadKey, attempt: u64, error: &str) {
        self.lock().read_scripts.script_exact(key, attempt, error.to_string());
    }

    pub(super) fn script_next_read_failure(&self, key: ReadKey, error: &str) {
        self.lock().read_scripts.script_next(key, error.to_string());
    }

    pub(super) fn script_read_fallback(&self, error: &str) {
        self.lock().read_scripts.script_fallback(error.to_string());
    }

    pub(super) fn read_attempts(&self, key: &ReadKey) -> u64 {
        self.lock().read_scripts.attempts(key)
    }

    pub(super) fn script_action(&self, target: ActionTarget, attempt: u64, outcome: ActionOutcome) {
        self.lock().action_scripts.script_exact(target, attempt, ActionScript::immediate(outcome));
    }

    pub(super) fn script_action_fallback(&self, outcome: ActionOutcome) {
        self.lock().action_scripts.script_fallback(ActionScript::immediate(outcome));
    }

    pub(super) fn block_action(
        &self,
        target: ActionTarget,
        attempt: u64,
        point: BarrierPoint,
        outcome: ActionOutcome,
        name: &str,
    ) {
        assert!(point != BarrierPoint::Proof, "proof barriers use block_proof");
        let barrier = NamedBarrier::new(name);
        let script = match point {
            BarrierPoint::BeforeSigner => ActionScript {
                outcome,
                before_signer: Some(barrier.clone()),
                after_submission: None,
            },
            BarrierPoint::AfterSubmission => ActionScript {
                outcome,
                before_signer: None,
                after_submission: Some(barrier.clone()),
            },
            BarrierPoint::Proof => unreachable!(),
        };
        let mut data = self.lock();
        data.action_scripts.script_exact(target.clone(), attempt, script);
        data.barriers
            .insert(BarrierKey { point, target: ScriptTarget::Action(target), attempt }, barrier);
    }

    pub(super) fn script_proof(&self, target: GameTarget, attempt: u64, outcome: ProofOutcome) {
        self.lock().proof_scripts.script_exact(target, attempt, ProofScript::immediate(outcome));
    }

    pub(super) fn script_next_proof(&self, target: GameTarget, outcome: ProofOutcome) {
        self.lock().proof_scripts.script_next(target, ProofScript::immediate(outcome));
    }

    pub(super) fn script_proof_fallback(&self, outcome: ProofOutcome) {
        self.lock().proof_scripts.script_fallback(ProofScript::immediate(outcome));
    }

    pub(super) fn block_proof(
        &self,
        target: GameTarget,
        attempt: u64,
        outcome: ProofOutcome,
        name: &str,
    ) {
        let barrier = NamedBarrier::new(name);
        let mut data = self.lock();
        data.proof_scripts.script_exact(
            target.clone(),
            attempt,
            ProofScript { outcome, barrier: Some(barrier.clone()) },
        );
        data.barriers.insert(
            BarrierKey { point: BarrierPoint::Proof, target: ScriptTarget::Proof(target), attempt },
            barrier,
        );
    }

    pub(super) fn action_record(
        &self,
        target: &ActionTarget,
        attempt: u64,
    ) -> Option<ActionRecord> {
        self.lock().action_records.get(&(target.clone(), attempt)).cloned()
    }

    pub(super) fn proof_record(&self, target: &GameTarget, attempt: u64) -> Option<ProofRecord> {
        self.lock().proof_records.get(&(target.clone(), attempt)).cloned()
    }

    pub(super) fn superroot_journal(&self) -> Vec<SuperRootQueryRecord> {
        self.lock().superroot_journal.clone()
    }

    pub(super) fn observation(&self) -> WorldObservation {
        let data = self.lock();
        let latest = data.latest_state();
        let mut games = latest.games.values().cloned().collect::<Vec<_>>();
        games.sort_unstable_by_key(|game| game.factory_index);
        let mut pending_transactions = data
            .pending_transactions
            .values()
            .map(|transaction| PendingTransactionObservation {
                target: transaction.target.clone(),
                attempt: transaction.attempt,
                nonce: transaction.nonce,
            })
            .collect::<Vec<_>>();
        pending_transactions.sort_unstable_by(|left, right| {
            left.target.cmp(&right.target).then_with(|| left.attempt.cmp(&right.attempt))
        });
        WorldObservation {
            latest_l1: latest.block,
            host_time: data.host_time,
            safe_time: data.safe_time,
            finalized_time: data.finalized_time,
            nonce: NonceState { pending: data.pending_nonce, latest: latest.latest_nonce },
            games,
            pending_transactions,
        }
    }

    pub(super) fn include_transaction(
        &self,
        target: &ActionTarget,
        attempt: u64,
        depth: InclusionDepth,
    ) -> Result<CommittedEffect> {
        self.lock().include_transaction(target, attempt, depth, true, true)
    }

    pub(super) fn drop_transaction(&self, target: &ActionTarget, attempt: u64) -> Result<()> {
        let mut data = self.lock();
        data.pending_transactions
            .remove(&(target.clone(), attempt))
            .with_context(|| format!("no pending transaction for {target:?} attempt {attempt}"))?;
        let latest_nonce = data.latest_state().latest_nonce;
        data.pending_nonce = data
            .pending_transactions
            .values()
            .map(|transaction| transaction.nonce + 1)
            .max()
            .unwrap_or(latest_nonce)
            .max(latest_nonce);
        let record = data
            .action_records
            .get_mut(&(target.clone(), attempt))
            .context("pending transaction has no action record")?;
        record.lifecycle = ActionLifecycle::Dropped;
        record.effect = CommittedEffect::None;
        Ok(())
    }

    pub(super) fn l1_view(&self) -> Arc<dyn L1View> {
        Arc::new(FakeL1View(self.clone()))
    }

    pub(super) fn superroot_source(&self) -> Arc<dyn SuperRootSource> {
        Arc::new(FakeSuperRootSource(self.clone()))
    }

    pub(super) fn query_time(&self) -> Arc<dyn QueryTime> {
        Arc::new(FakeQueryTime(self.clone()))
    }

    pub(super) fn proof_engine(&self) -> Arc<dyn ProofEngine> {
        Arc::new(FakeProofEngine(self.clone()))
    }

    pub(super) fn action_executor(&self) -> Arc<dyn ActionExecutor> {
        Arc::new(FakeActionExecutor(self.clone()))
    }

    fn append_block(&self, update: impl FnOnce(&mut L1State)) -> L1BlockRef {
        self.lock().append_block(update)
    }

    fn lock(&self) -> std::sync::MutexGuard<'_, WorldData> {
        self.data.lock().expect("scenario world lock poisoned")
    }

    fn barrier(&self, key: &BarrierKey) -> Option<NamedBarrier> {
        self.lock().barriers.get(key).cloned()
    }

    fn fail_read(&self, key: ReadKey) -> Result<()> {
        let (_, failure) = self.lock().read_scripts.next(&key);
        if let Some(failure) = failure {
            bail!(failure)
        }
        Ok(())
    }
}

impl WorldData {
    fn latest_state(&self) -> Arc<L1State> {
        self.states.get(&self.latest_block).expect("latest world block must exist").clone()
    }

    fn state_at(&self, block: BlockId) -> Result<Arc<L1State>> {
        let number = if block.is_latest() {
            self.latest_block
        } else {
            block.as_u64().context("scenario only supports latest or numbered L1 reads")?
        };
        self.states.get(&number).cloned().with_context(|| format!("L1 block {number} unavailable"))
    }

    fn append_block(&mut self, update: impl FnOnce(&mut L1State)) -> L1BlockRef {
        let mut next = self.latest_state().as_ref().clone();
        next.block.number += 1;
        update(&mut next);
        self.latest_block = next.block.number;
        let block = next.block;
        self.states.insert(block.number, Arc::new(next));
        block
    }

    fn game_target(&self, address: Address) -> Result<GameTarget> {
        self.latest_state()
            .games
            .values()
            .find(|game| game.address == address)
            .map(|game| GameTarget { factory_index: game.factory_index, address: game.address })
            .with_context(|| format!("unknown scenario game {address}"))
    }

    fn proof_target(&self, game: &GameProofInputs) -> Result<GameTarget> {
        let matches = self
            .latest_state()
            .games
            .values()
            .find(|candidate| {
                candidate.proof_inputs.l1_head == game.l1_head &&
                    candidate.proof_inputs.l1_head_number == game.l1_head_number &&
                    candidate.proof_inputs.starting_root == game.starting_root &&
                    candidate.proof_inputs.starting_sequence_number == game.starting_ts &&
                    candidate.proof_inputs.root_claim == game.root_claim &&
                    candidate.proof_inputs.sequence_number == game.claim_ts &&
                    candidate.absolute_prestate == game.prestate
            })
            .cloned();
        matches
            .map(|candidate| candidate.target())
            .context("proof inputs do not uniquely identify a scenario game")
    }

    fn include_transaction(
        &mut self,
        target: &ActionTarget,
        attempt: u64,
        depth: InclusionDepth,
        apply_effect: bool,
        late: bool,
    ) -> Result<CommittedEffect> {
        let transaction = self
            .pending_transactions
            .remove(&(target.clone(), attempt))
            .with_context(|| format!("no pending transaction for {target:?} attempt {attempt}"))?;
        let effect_plan = transaction.effect.clone();
        let nonce = transaction.nonce;
        let effect = self.commit_transaction(effect_plan, nonce, apply_effect)?;
        let confirmations = match depth {
            InclusionDepth::LatestOnly => 1,
            InclusionDepth::Confirmed => NUM_CONFIRMATIONS + self.sync_confirmations,
        };
        for _ in 1..confirmations {
            self.append_block(|_| {});
        }
        let latest_nonce = self.latest_state().latest_nonce;
        self.pending_nonce = self
            .pending_transactions
            .values()
            .map(|pending| pending.nonce + 1)
            .max()
            .unwrap_or(latest_nonce)
            .max(latest_nonce);
        let record = self
            .action_records
            .get_mut(&(target.clone(), attempt))
            .context("pending transaction has no action record")?;
        record.lifecycle = if apply_effect {
            if late { ActionLifecycle::IncludedLate } else { ActionLifecycle::Confirmed }
        } else {
            ActionLifecycle::Reverted
        };
        record.effect = effect.clone();
        Ok(effect)
    }

    fn commit_transaction(
        &mut self,
        plan: PendingEffect,
        nonce: u64,
        apply_effect: bool,
    ) -> Result<CommittedEffect> {
        let mut committed = CommittedEffect::None;
        self.append_block(|state| {
            state.latest_nonce = state.latest_nonce.max(nonce + 1);
            if !apply_effect {
                return;
            }
            committed = match plan {
                PendingEffect::Create {
                    index,
                    address,
                    root_claim,
                    extra_data,
                    sequence_number,
                    parent_game_index,
                } => {
                    let mut game = ScenarioGame::new(
                        index.to::<u64>(),
                        parent_game_index,
                        sequence_number,
                        state.registered_args.absolute_prestate,
                    );
                    game.address = address;
                    game.root_claim = root_claim;
                    game.extra_data = extra_data;
                    game.creator = ScenarioWorld::proposer_address();
                    game.weth = state.registered_args.weth;
                    game.anchor_state_registry = state.registered_args.anchor_state_registry;
                    game.deadline =
                        state.block.timestamp + state.registered_args.max_challenge_duration;
                    game.proof_inputs.l1_head_number = state.block.number;
                    game.proof_inputs.l1_head = deterministic_hash(0x80, state.block.number);
                    if parent_game_index == u32::MAX {
                        game.proof_inputs.starting_root = state.anchor_root.root;
                        game.proof_inputs.starting_sequence_number =
                            state.anchor_root.sequence_number.to::<u64>();
                    } else if let Some(parent) = state.games.get(&U256::from(parent_game_index)) {
                        game.proof_inputs.starting_root = parent.root_claim;
                        game.proof_inputs.starting_sequence_number = parent.sequence_number;
                    }
                    state.games.insert(index, game);
                    CommittedEffect::Created { factory_index: index, address }
                }
                PendingEffect::Prove(address) => {
                    let game = state
                        .games
                        .values_mut()
                        .find(|game| game.address == address)
                        .expect("proved scenario game must exist");
                    game.proposal_status = match game.proposal_status {
                        ProposalStatus::Unchallenged => {
                            ProposalStatus::UnchallengedAndValidProofProvided
                        }
                        _ => ProposalStatus::ChallengedAndValidProofProvided,
                    };
                    CommittedEffect::Proven { game: address }
                }
                PendingEffect::Resolve(address) => {
                    let game = state
                        .games
                        .values_mut()
                        .find(|game| game.address == address)
                        .expect("resolved scenario game must exist");
                    game.status = GameStatus::DefenderWins;
                    game.proposal_status = ProposalStatus::Resolved;
                    game.finalized = true;
                    CommittedEffect::Resolved { game: address }
                }
                PendingEffect::Claim(address) => {
                    let game = state
                        .games
                        .values_mut()
                        .find(|game| game.address == address)
                        .expect("claimed scenario game must exist");
                    if game.bond.credit == U256::ZERO {
                        let amount = game.bond.withdrawal_amount;
                        game.bond.withdrawal_amount = U256::ZERO;
                        CommittedEffect::ClaimPaid { game: address, amount }
                    } else {
                        let amount = game.bond.credit;
                        game.bond.credit = U256::ZERO;
                        game.bond.withdrawal_amount = amount;
                        game.bond.withdrawal_timestamp = U256::from(state.block.timestamp);
                        CommittedEffect::ClaimUnlocked { game: address, amount }
                    }
                }
            };
        });
        Ok(committed)
    }
}

fn deterministic_address(tag: u8, value: u64) -> Address {
    let mut bytes = [0u8; 20];
    bytes[0] = tag;
    bytes[12..].copy_from_slice(&value.to_be_bytes());
    Address::from(bytes)
}

fn deterministic_hash(tag: u8, value: u64) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[0] = tag;
    bytes[24..].copy_from_slice(&value.to_be_bytes());
    B256::from(bytes)
}

pub(super) fn canonical_super_root(timestamp: u64) -> B256 {
    deterministic_hash(0x90, timestamp)
}

fn super_root_timestamp(root: B256) -> Result<u64> {
    anyhow::ensure!(root[0] == 0x90, "unknown scenario super root {root}");
    Ok(u64::from_be_bytes(root[24..].try_into().expect("eight-byte timestamp")))
}

fn create_artifact_directory() -> ArtifactDirectory {
    let id = ARTIFACT_DIR_ID.fetch_add(1, Ordering::Relaxed);
    let path =
        std::env::temp_dir().join(format!("kona-sp1-scenario-world-{}-{id}", std::process::id()));
    fs::create_dir_all(&path).expect("create scenario artifact directory");
    ArtifactDirectory(path)
}

fn publish_prestate_at(directory: &Path, prestate: B256) {
    for suffix in [AGGREGATION_ARTIFACT_SUFFIX, RANGE_ARTIFACT_SUFFIX] {
        let mut encoder = GzEncoder::new(Vec::new(), Compression::fast());
        encoder.write_all(&[0x7f, b'E', b'L', b'F']).expect("compress scenario ELF");
        let compressed = encoder.finish().expect("finish scenario ELF compression");
        fs::write(directory.join(format!("{prestate}{suffix}")), compressed)
            .expect("publish scenario prestate artifact");
    }
}

#[derive(Clone)]
struct FakeL1View(ScenarioWorld);

impl FakeL1View {
    fn state(&self, boundary: ReadBoundary, block: BlockId) -> Result<Arc<L1State>> {
        let number = {
            let data = self.0.lock();
            if block.is_latest() {
                data.latest_block
            } else {
                block.as_u64().context("scenario L1 read requires a block number")?
            }
        };
        self.0.fail_read(ReadKey { boundary, target: ReadTarget::Block(number) })?;
        self.0.lock().state_at(block)
    }

    fn latest_state(&self, boundary: ReadBoundary, target: ReadTarget) -> Result<Arc<L1State>> {
        self.0.fail_read(ReadKey { boundary, target })?;
        Ok(self.0.lock().latest_state())
    }
}

#[async_trait]
impl L1View for FakeL1View {
    async fn latest_head(&self) -> Result<Option<L1BlockRef>> {
        self.0.fail_read(ReadKey::global(ReadBoundary::LatestHead))?;
        Ok(Some(self.0.lock().latest_state().block))
    }

    async fn block_ref(&self, number: u64) -> Result<Option<L1BlockRef>> {
        self.0.fail_read(ReadKey {
            boundary: ReadBoundary::BlockRef,
            target: ReadTarget::Block(number),
        })?;
        Ok(self.0.lock().states.get(&number).map(|state| state.block))
    }

    async fn registered_game_args(&self, block: BlockId) -> Result<ZKGameArgs> {
        Ok(self.state(ReadBoundary::RegisteredGameArgs, block)?.registered_args.clone())
    }

    async fn anchor_root(&self, _registry: Address, block: BlockId) -> Result<AnchorRoot> {
        Ok(self.state(ReadBoundary::AnchorRoot, block)?.anchor_root)
    }

    async fn latest_game_index(&self, block: BlockId) -> Result<Option<U256>> {
        Ok(self.state(ReadBoundary::LatestGameIndex, block)?.games.keys().next_back().copied())
    }

    async fn registered_anchor_game(&self, block: BlockId) -> Result<Address> {
        Ok(self.state(ReadBoundary::RegisteredAnchorGame, block)?.registered_anchor_game)
    }

    async fn factory_game(&self, index: U256, block: BlockId) -> Result<FactoryGame> {
        self.0.fail_read(ReadKey::factory(ReadBoundary::FactoryGame, index))?;
        let state = self.0.lock().state_at(block)?;
        let game = state.games.get(&index).context("factory game missing")?;
        Ok(FactoryGame { address: game.address, game_type: game.game_type })
    }

    async fn game_claim(&self, game: Address, block: BlockId) -> Result<GameClaim> {
        self.0.fail_read(ReadKey::game(ReadBoundary::GameClaim, game))?;
        let state = self.0.lock().state_at(block)?;
        let game = game_by_address(&state, game)?;
        Ok(GameClaim {
            status: game.proposal_status as u8,
            deadline: game.deadline,
            parent_index: game.parent_index,
        })
    }

    async fn game_identity(&self, game: Address, block: BlockId) -> Result<GameIdentity> {
        self.0.fail_read(ReadKey::game(ReadBoundary::GameIdentity, game))?;
        let state = self.0.lock().state_at(block)?;
        let game = game_by_address(&state, game)?;
        Ok(GameIdentity {
            anchor_state_registry: game.anchor_state_registry,
            weth: game.weth,
            creator: game.creator,
            sequence_number: U256::from(game.sequence_number),
        })
    }

    async fn game_validity(&self, game: Address, block: BlockId) -> Result<GameValidity> {
        self.0.fail_read(ReadKey::game(ReadBoundary::GameValidity, game))?;
        let state = self.0.lock().state_at(block)?;
        let game = game_by_address(&state, game)?;
        Ok(GameValidity {
            root_claim: game.root_claim,
            was_respected: game.was_respected,
            status: game.status,
            absolute_prestate: game.absolute_prestate,
        })
    }

    async fn game_lifecycle(
        &self,
        game: Address,
        _registry: Address,
        block: BlockId,
    ) -> Result<GameLifecycle> {
        self.0.fail_read(ReadKey::game(ReadBoundary::GameLifecycle, game))?;
        let state = self.0.lock().state_at(block)?;
        let game = game_by_address(&state, game)?;
        Ok(GameLifecycle {
            proposal_status: game.proposal_status,
            deadline: game.deadline,
            parent_index: game.parent_index,
            status: game.status,
            is_finalized: game.finalized,
        })
    }

    async fn parent_game_status(&self, parent_index: u32, block: BlockId) -> Result<u8> {
        self.0.fail_read(ReadKey::factory(
            ReadBoundary::ParentGameStatus,
            U256::from(parent_index),
        ))?;
        let state = self.0.lock().state_at(block)?;
        Ok(state.games.get(&U256::from(parent_index)).context("parent game missing")?.status as u8)
    }

    async fn bond_state(
        &self,
        game: Address,
        _weth: Address,
        _proposer: Address,
        block: BlockId,
    ) -> Result<BondState> {
        self.0.fail_read(ReadKey::game(ReadBoundary::BondState, game))?;
        let state = self.0.lock().state_at(block)?;
        Ok(game_by_address(&state, game)?.bond)
    }

    async fn init_bond(&self) -> Result<U256> {
        Ok(self.latest_state(ReadBoundary::InitBond, ReadTarget::Global)?.init_bond)
    }

    async fn game_status(&self, game: Address) -> Result<u8> {
        let state = self.latest_state(ReadBoundary::GameStatus, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.status as u8)
    }

    async fn claim_preflight(
        &self,
        game: Address,
        _weth: Address,
        _proposer: Address,
    ) -> ClaimPreflight {
        let state = self.0.lock().latest_state();
        let credit = self
            .0
            .fail_read(ReadKey::game(ReadBoundary::ClaimCredit, game))
            .and_then(|()| game_by_address(&state, game).map(|game| game.bond.credit));
        let withdrawal =
            self.0.fail_read(ReadKey::game(ReadBoundary::ClaimWithdrawal, game)).and_then(|()| {
                game_by_address(&state, game).map(|game| WithdrawalState {
                    amount: game.bond.withdrawal_amount,
                    timestamp: game.bond.withdrawal_timestamp,
                })
            });
        ClaimPreflight { credit, withdrawal }
    }

    async fn weth_delay(&self, weth: Address) -> Result<U256> {
        let state = self.latest_state(ReadBoundary::WethDelay, ReadTarget::Global)?;
        Ok(state
            .games
            .values()
            .find(|game| game.weth == weth)
            .map_or_else(|| U256::from(10), |game| game.bond.delay))
    }

    async fn game_by_uuid(&self, root_claim: B256, extra_data: Vec<u8>) -> Result<Address> {
        let state = self.latest_state(ReadBoundary::GameByUuid, ReadTarget::Global)?;
        Ok(state
            .games
            .values()
            .find(|game| game.root_claim == root_claim && game.extra_data == extra_data)
            .map_or(Address::ZERO, |game| game.address))
    }

    async fn game_creator(&self, game: Address) -> Result<Address> {
        let state = self.latest_state(ReadBoundary::GameCreator, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.creator)
    }

    async fn nonce_state(&self, _proposer: Address) -> Result<NonceState> {
        self.0.fail_read(ReadKey::global(ReadBoundary::NonceState))?;
        let data = self.0.lock();
        Ok(NonceState { pending: data.pending_nonce, latest: data.latest_state().latest_nonce })
    }

    async fn respected_game_type(&self, block: BlockId) -> Result<u32> {
        Ok(self.state(ReadBoundary::RespectedGameType, block)?.respected_game_type)
    }

    async fn parent_standing(&self, game: Address, _registry: Address) -> Result<GameStanding> {
        let state = self.latest_state(ReadBoundary::ParentStanding, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.standing)
    }

    async fn game_standing(&self, game: Address, _registry: Address) -> Result<GameStanding> {
        let state = self.latest_state(ReadBoundary::GameStanding, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.standing)
    }

    async fn proof_status(&self, game: Address) -> Result<u8> {
        let state = self.latest_state(ReadBoundary::ProofStatus, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.proposal_status as u8)
    }

    async fn proof_inputs(&self, game: Address) -> Result<ProofInputs> {
        let state = self.latest_state(ReadBoundary::ProofInputs, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.proof_inputs)
    }

    async fn anchor_state_registry(&self, game: Address) -> Result<Address> {
        let state = self.latest_state(ReadBoundary::AnchorStateRegistry, ReadTarget::Game(game))?;
        Ok(game_by_address(&state, game)?.anchor_state_registry)
    }

    async fn latest_l1_timestamp(&self) -> Result<u64> {
        Ok(self.latest_state(ReadBoundary::LatestL1Timestamp, ReadTarget::Global)?.block.timestamp)
    }
}

fn game_by_address(state: &L1State, address: Address) -> Result<&ScenarioGame> {
    state
        .games
        .values()
        .find(|game| game.address == address)
        .with_context(|| format!("scenario game {address} missing"))
}

#[derive(Clone)]
struct FakeQueryTime(ScenarioWorld);

impl QueryTime for FakeQueryTime {
    fn unix_timestamp(&self) -> Result<u64> {
        Ok(self.0.lock().host_time)
    }
}

#[derive(Clone)]
struct FakeSuperRootSource(ScenarioWorld);

#[async_trait]
impl SuperRootSource for FakeSuperRootSource {
    async fn proposal_horizon(&self, timestamp: u64) -> Result<ProposalHorizon> {
        let mut data = self.0.lock();
        let horizon = ProposalHorizon {
            safe_timestamp: data.safe_time,
            finalized_timestamp: data.finalized_time,
        };
        data.superroot_journal.push(SuperRootQueryRecord::Horizon {
            request_time: timestamp,
            safe: horizon.safe_timestamp,
            finalized: horizon.finalized_timestamp,
        });
        Ok(horizon)
    }

    async fn super_root_at_timestamp(&self, timestamp: u64) -> Result<SuperRootAtTimestamp> {
        let mut data = self.0.lock();
        let safe = data.safe_time;
        let finalized = data.finalized_time;
        let setting = data.superroots.get(&timestamp).cloned().unwrap_or_else(|| {
            if timestamp <= safe {
                SuperRootSetting::Available {
                    root: canonical_super_root(timestamp),
                    proof: vec![0x01],
                    current_l1: 2,
                    required_l1: 1,
                }
            } else {
                SuperRootSetting::Absent { current_l1: 2, local_safe: safe, finalized }
            }
        });
        let response = match setting {
            SuperRootSetting::Available { root, proof, current_l1, required_l1 } => {
                let response = superroot_response(
                    timestamp,
                    safe,
                    safe,
                    finalized,
                    current_l1,
                    Some((required_l1, root)),
                );
                data.superroot_journal.push(SuperRootQueryRecord::AtTimestamp {
                    timestamp,
                    current_l1,
                    required_l1: Some(required_l1),
                    safe,
                    local_safe: safe,
                    finalized,
                    available: true,
                });
                SuperRootAtTimestamp {
                    response,
                    root: Some(SuperRootAt { proof_bytes: proof, super_root: root }),
                }
            }
            SuperRootSetting::Absent { current_l1, local_safe, finalized } => {
                let response =
                    superroot_response(timestamp, safe, local_safe, finalized, current_l1, None);
                data.superroot_journal.push(SuperRootQueryRecord::AtTimestamp {
                    timestamp,
                    current_l1,
                    required_l1: None,
                    safe,
                    local_safe,
                    finalized,
                    available: false,
                });
                SuperRootAtTimestamp { response, root: None }
            }
            SuperRootSetting::Failure(error) => {
                data.superroot_journal.push(SuperRootQueryRecord::FailedAtTimestamp {
                    timestamp,
                    error: error.clone(),
                });
                return Err(anyhow!(error));
            }
        };
        Ok(response)
    }
}

fn superroot_response(
    timestamp: u64,
    safe: u64,
    local_safe: u64,
    finalized: u64,
    current_l1: u64,
    data: Option<(u64, B256)>,
) -> SuperRootAtTimestampResponse {
    SuperRootAtTimestampResponse {
        current_l1: SuperBlockId { number: current_l1, ..Default::default() },
        current_safe_timestamp: safe,
        current_local_safe_timestamp: local_safe,
        current_finalized_timestamp: finalized,
        optimistic_at_timestamp: Default::default(),
        chain_ids: Vec::new(),
        data: data.map(|(required_l1, root)| SuperRootResponseData {
            verified_required_l1: SuperBlockId { number: required_l1, ..Default::default() },
            super_v1: SuperV1 { timestamp, chains: Vec::new() },
            super_root: root,
        }),
    }
}

#[derive(Clone)]
struct FakeProofEngine(ScenarioWorld);

#[async_trait]
impl ProofEngine for FakeProofEngine {
    async fn prove(
        &self,
        _keys: Option<Arc<ProofKeys>>,
        game: GameProofInputs,
        _responses: Vec<SuperRootAtTimestampResponse>,
    ) -> Result<Vec<u8>> {
        let (target, attempt, script) = {
            let mut data = self.0.lock();
            let target = data.proof_target(&game)?;
            let (attempt, script) = data.proof_scripts.next(&target);
            let script = script.unwrap_or_else(|| ProofScript::immediate(ProofOutcome::Success));
            data.proof_records.insert(
                (target.clone(), attempt),
                ProofRecord {
                    target: target.clone(),
                    attempt,
                    lifecycle: if script.barrier.is_some() {
                        ProofLifecycle::Parked
                    } else {
                        ProofLifecycle::Succeeded
                    },
                },
            );
            (target, attempt, script)
        };
        if let Some(barrier) = script.barrier {
            barrier.park_unassigned().await;
        }
        let lifecycle = match script.outcome {
            ProofOutcome::Success => ProofLifecycle::Succeeded,
            ProofOutcome::Failure => ProofLifecycle::Failed,
            ProofOutcome::Panic => ProofLifecycle::Panicked,
        };
        self.0
            .lock()
            .proof_records
            .get_mut(&(target.clone(), attempt))
            .expect("proof record must exist")
            .lifecycle = lifecycle;
        match script.outcome {
            ProofOutcome::Success => Ok(vec![0xa0, attempt as u8]),
            ProofOutcome::Failure => {
                bail!("scripted proof failure for {target:?} attempt {attempt}")
            }
            ProofOutcome::Panic => panic!("scripted proof panic for {target:?} attempt {attempt}"),
        }
    }
}

#[derive(Clone)]
struct FakeActionExecutor(ScenarioWorld);

struct ActionResult {
    transaction_hash: B256,
    created_address: Option<Address>,
}

impl FakeActionExecutor {
    async fn execute(
        &self,
        target: ActionTarget,
        inputs: ActionInputs,
        effect: PendingEffect,
    ) -> Result<ActionResult> {
        let (attempt, script) = {
            let mut data = self.0.lock();
            let (attempt, script) = data.action_scripts.next(&target);
            (attempt, script.unwrap_or_else(|| ActionScript::immediate(ActionOutcome::Success)))
        };
        if let Some(barrier) = script.before_signer.clone() {
            barrier.park_unassigned().await;
        }

        let _signer = self.0.signer_gate.lock().await;
        if script.outcome == ActionOutcome::PreSubmitFailure {
            self.0.lock().action_records.insert(
                (target.clone(), attempt),
                ActionRecord {
                    target: target.clone(),
                    attempt,
                    inputs,
                    lifecycle: ActionLifecycle::PreSubmitFailed,
                    transaction_hash: None,
                    effect: CommittedEffect::None,
                },
            );
            bail!("scripted pre-submit failure for {target:?} attempt {attempt}")
        }

        let (transaction_hash, created_address) = {
            let mut data = self.0.lock();
            let nonce = data.pending_nonce;
            data.pending_nonce += 1;
            let transaction_hash = deterministic_hash(0xa0, nonce);
            let created_address = match &effect {
                PendingEffect::Create { address, .. } => Some(*address),
                _ => None,
            };
            data.pending_transactions.insert(
                (target.clone(), attempt),
                PendingTransaction { target: target.clone(), attempt, nonce, effect },
            );
            data.action_records.insert(
                (target.clone(), attempt),
                ActionRecord {
                    target: target.clone(),
                    attempt,
                    inputs,
                    lifecycle: ActionLifecycle::Submitted,
                    transaction_hash: Some(transaction_hash),
                    effect: CommittedEffect::None,
                },
            );
            (transaction_hash, created_address)
        };

        if let Some(barrier) = script.after_submission {
            barrier.park_unassigned().await;
        }
        match script.outcome {
            ActionOutcome::Success => {
                self.0.lock().include_transaction(
                    &target,
                    attempt,
                    InclusionDepth::Confirmed,
                    true,
                    false,
                )?;
                Ok(ActionResult { transaction_hash, created_address })
            }
            ActionOutcome::Revert => {
                self.0.lock().include_transaction(
                    &target,
                    attempt,
                    InclusionDepth::Confirmed,
                    false,
                    false,
                )?;
                bail!("{TX_REVERTED_PREFIX} scripted receipt")
            }
            ActionOutcome::Timeout => {
                self.0
                    .lock()
                    .action_records
                    .get_mut(&(target.clone(), attempt))
                    .expect("action record must exist")
                    .lifecycle = ActionLifecycle::TimedOut;
                bail!("scripted confirmation timeout for {target:?} attempt {attempt}")
            }
            ActionOutcome::PreSubmitFailure => unreachable!(),
        }
    }
}

#[async_trait]
impl ActionExecutor for FakeActionExecutor {
    async fn create_game(
        &self,
        root_claim: B256,
        extra_data: Vec<u8>,
        _init_bond: U256,
    ) -> Result<GameCreationReceipt> {
        let sequence_number = super_root_timestamp(root_claim)?;
        let parent_game_index = u32::from_be_bytes(
            extra_data.get(..4).context("create extra data lacks parent index")?.try_into()?,
        );
        let (index, address) = {
            let data = self.0.lock();
            let next_index = data
                .latest_state()
                .games
                .keys()
                .next_back()
                .map_or(0, |index| index.to::<u64>() + 1);
            let reserved = data
                .pending_transactions
                .values()
                .filter(|transaction| matches!(transaction.effect, PendingEffect::Create { .. }))
                .count() as u64;
            let index = U256::from(next_index + reserved);
            (index, deterministic_address(0x40, index.to::<u64>()))
        };
        let target = ActionTarget::Create { sequence_number, parent_game_index };
        let result = self
            .execute(
                target,
                ActionInputs::Create { root_claim, parent_game_index, sequence_number },
                PendingEffect::Create {
                    index,
                    address,
                    root_claim,
                    extra_data,
                    sequence_number,
                    parent_game_index,
                },
            )
            .await?;
        Ok(GameCreationReceipt {
            game_address: result.created_address.expect("create action must reserve an address"),
            transaction_hash: result.transaction_hash,
        })
    }

    async fn prove_game(&self, game: Address, _proof: Vec<u8>) -> Result<B256> {
        let target = self.0.lock().game_target(game)?;
        Ok(self
            .execute(
                ActionTarget::Prove(target),
                ActionInputs::Prove { game },
                PendingEffect::Prove(game),
            )
            .await?
            .transaction_hash)
    }

    async fn resolve_game(&self, game: Address) -> Result<B256> {
        let target = self.0.lock().game_target(game)?;
        Ok(self
            .execute(
                ActionTarget::Resolve(target),
                ActionInputs::Resolve { game },
                PendingEffect::Resolve(game),
            )
            .await?
            .transaction_hash)
    }

    async fn claim_credit(&self, game: Address, recipient: Address) -> Result<B256> {
        let target = self.0.lock().game_target(game)?;
        Ok(self
            .execute(
                ActionTarget::ClaimCredit(target),
                ActionInputs::ClaimCredit { game, recipient },
                PendingEffect::Claim(game),
            )
            .await?
            .transaction_hash)
    }
}

pub(super) struct ScenarioHarness {
    pub(super) world: ScenarioWorld,
    pub(super) proposer: Arc<Proposer>,
    control: ScenarioControl,
    config: ProposerConfig,
}

impl ScenarioHarness {
    pub(super) async fn new(
        world: ScenarioWorld,
        config: ProposerConfig,
    ) -> Result<Self, ScenarioError> {
        let proposer = build_proposer(&world, &config)
            .await
            .map_err(|error| ScenarioError::Initialization(error.to_string()))?;
        proposer
            .validate_and_init()
            .await
            .map_err(|error| ScenarioError::Initialization(error.to_string()))?;
        let control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
        Ok(Self { world, proposer, control, config })
    }

    pub(super) async fn uninitialized(
        world: &ScenarioWorld,
        config: &ProposerConfig,
    ) -> Result<Arc<Proposer>> {
        build_proposer(world, config).await
    }

    pub(super) async fn restart(&mut self) -> Result<(), ScenarioError> {
        let tasks = self.proposer.tasks.lock().await;
        if let Some(task_id) = tasks.keys().min().copied() {
            return Err(ScenarioError::RunningTask { task_id });
        }
        drop(tasks);
        let proposer = build_proposer(&self.world, &self.config)
            .await
            .map_err(|error| ScenarioError::Initialization(error.to_string()))?;
        proposer
            .validate_and_init()
            .await
            .map_err(|error| ScenarioError::Initialization(error.to_string()))?;
        self.control = ScenarioControl::new(proposer.clone(), Duration::from_secs(1));
        self.proposer = proposer;
        Ok(())
    }

    pub(super) async fn tick(&mut self) -> Result<CycleResult, ScenarioError> {
        self.control.tick().await
    }

    pub(super) async fn settle(
        &mut self,
        task_ids: &[TaskId],
    ) -> Result<Vec<TaskCompletion>, ScenarioError> {
        self.control.settle(task_ids).await
    }

    pub(super) async fn wait_for_action_barrier(
        &mut self,
        task_id: TaskId,
        target: &ActionTarget,
        attempt: u64,
        point: BarrierPoint,
    ) -> Result<(), ScenarioError> {
        let key = BarrierKey { point, target: ScriptTarget::Action(target.clone()), attempt };
        self.bind_reached_barrier(task_id, key).await
    }

    pub(super) async fn wait_for_proof_barrier(
        &mut self,
        task_id: TaskId,
        target: &GameTarget,
        attempt: u64,
    ) -> Result<(), ScenarioError> {
        let key = BarrierKey {
            point: BarrierPoint::Proof,
            target: ScriptTarget::Proof(target.clone()),
            attempt,
        };
        self.bind_reached_barrier(task_id, key).await
    }

    pub(super) fn release_action_barrier(
        &self,
        target: &ActionTarget,
        attempt: u64,
        point: BarrierPoint,
    ) -> Result<(), ScenarioError> {
        self.release_barrier(BarrierKey {
            point,
            target: ScriptTarget::Action(target.clone()),
            attempt,
        })
    }

    pub(super) fn release_proof_barrier(
        &self,
        target: &GameTarget,
        attempt: u64,
    ) -> Result<(), ScenarioError> {
        self.release_barrier(BarrierKey {
            point: BarrierPoint::Proof,
            target: ScriptTarget::Proof(target.clone()),
            attempt,
        })
    }

    async fn bind_reached_barrier(
        &mut self,
        task_id: TaskId,
        key: BarrierKey,
    ) -> Result<(), ScenarioError> {
        let barrier_name = format!("{key:?}");
        let barrier = self
            .world
            .barrier(&key)
            .ok_or_else(|| ScenarioError::UnknownBarrier { barrier: barrier_name.clone() })?;
        let operation = self
            .proposer
            .tasks
            .lock()
            .await
            .get(&task_id)
            .map(|(_, operation)| operation.clone())
            .ok_or(ScenarioError::UnknownTask { task_id })?;
        if !barrier_matches_operation(&key, &operation) {
            return Err(ScenarioError::BarrierOperationMismatch { task_id, barrier: barrier_name });
        }
        barrier.wait_until_reached().await;
        barrier.bind_task(task_id);
        self.control.record_parked(task_id, &barrier).await
    }

    fn release_barrier(&self, key: BarrierKey) -> Result<(), ScenarioError> {
        let barrier = self
            .world
            .barrier(&key)
            .ok_or_else(|| ScenarioError::UnknownBarrier { barrier: format!("{key:?}") })?;
        barrier.release();
        Ok(())
    }
}

fn barrier_matches_operation(key: &BarrierKey, operation: &OperationSummary) -> bool {
    match (&key.target, operation) {
        (
            ScriptTarget::Action(ActionTarget::Create { sequence_number, parent_game_index }),
            OperationSummary::ProposeGame {
                sequence_number: scheduled_sequence,
                parent_game_index: scheduled_parent,
            },
        ) => sequence_number == scheduled_sequence && parent_game_index == scheduled_parent,
        (
            ScriptTarget::Action(ActionTarget::Prove(target)),
            OperationSummary::ProveGame { factory_index, address, .. },
        ) |
        (
            ScriptTarget::Proof(target),
            OperationSummary::ProveGame { factory_index, address, .. },
        ) => target.factory_index == *factory_index && target.address == *address,
        (ScriptTarget::Action(ActionTarget::Resolve(_)), OperationSummary::ResolutionSweep) |
        (ScriptTarget::Action(ActionTarget::ClaimCredit(_)), OperationSummary::ClaimSweep) => true,
        _ => false,
    }
}

async fn build_proposer(world: &ScenarioWorld, config: &ProposerConfig) -> Result<Arc<Proposer>> {
    let mut config = config.clone();
    config.prestates_url = world.prestates_url();
    world.set_sync_confirmations(config.sync_l1_confirmations);
    let prestates =
        Arc::new(PrestateCache::with_retry_window(config.prestates_url.clone(), Duration::ZERO));
    Ok(Arc::new(
        Proposer::new_with_dependencies(
            config,
            ScenarioWorld::proposer_address(),
            world.l1_view(),
            world.query_time(),
            world.superroot_source(),
            world.proof_engine(),
            world.action_executor(),
            prestates,
        )
        .await?,
    ))
}

pub(super) fn scenario_config() -> ProposerConfig {
    ProposerConfig {
        l1_rpc: "http://127.0.0.1:1".parse().unwrap(),
        superroot_rpcs: vec!["http://127.0.0.1:1".parse().unwrap()],
        factory_address: Address::ZERO,
        prestates_url: "file:///replaced-by-scenario-world".parse().unwrap(),
        proposal_interval_seconds: 1,
        proposal_safety: ProposalSafety::Finalized,
        fetch_interval: 86_400,
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
        max_concurrent_defense_tasks: NonZeroU64::new(2).unwrap(),
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

pub(super) fn scheduled_task(
    result: &CycleResult,
    predicate: impl Fn(&OperationSummary) -> bool,
) -> TaskId {
    result
        .scheduled
        .iter()
        .find(|scheduled| predicate(&scheduled.operation))
        .map(|scheduled| scheduled.task_id)
        .expect("expected scheduled operation")
}

pub(super) fn other_task_ids(result: &CycleResult, excluded: TaskId) -> Vec<TaskId> {
    result
        .scheduled
        .iter()
        .filter(|scheduled| scheduled.task_id != excluded)
        .map(|scheduled| scheduled.task_id)
        .collect()
}
