//! The verification round loop.
//!
//! One round is four steps, in this order and no other:
//!
//! 1. **observe** — read every chain and the L1 once, into a [`RoundObservation`];
//! 2. **decide** — reach a [`Decision`] from that value alone, with no further reads;
//! 3. **write ahead** — record an effectful decision in the verified store's WAL slot, durably,
//!    *before* any of its side effects begins; and
//! 4. **apply** — perform the side effects, then clear the slot.
//!
//! Steps 3 and 4 are what makes a crash mid-apply recoverable: a restart either finds no slot, in
//! which case nothing was started, or finds the decision and re-applies it. Every side effect an
//! applied decision has is therefore idempotent.
//!
//! ## What is applied
//!
//! [`Decision::Wait`], [`Decision::Advance`] — and, when the verifier is built with the
//! replacement routes ([`Verifier::with_replacements`]), [`Decision::Invalidate`]. Applying an
//! invalidation archives each invalid block's output preimage — the archive doubles as the deny
//! list the chain's engine consults — and rewinds the chain onto the block's parent; derivation
//! then rebuilds the height from the same L1 batch data, the rebuild hits the deny list, and a
//! deposits-only replacement is built at the same height. The frontier itself does not move: the
//! next round re-verifies the same timestamp against the replaced chain, exactly as op-supernode's
//! `applyPendingTransition` does for `DecisionInvalidate`
//! (`op-supernode/supernode/activity/interop/interop.go`).
//!
//! [`Decision::Rewind`] — the response to a committed frontier whose L1 basis reorged away — is
//! applied by the same routes: the verified results at or after that frontier's timestamp are
//! dropped, archived invalidations decided on the reorged basis are pruned (revoking their deny
//! entries), the log stores rewind onto the previous verified frontier (or clear entirely when
//! none is retained), and — only when an invalidation was revoked — every chain's engine is reset
//! onto that frontier, because the deposits-only replacement a revoked invalidation forced is
//! canonical and derivation alone will not remove it. This is op-supernode's `applyRewindPlan`
//! (`op-supernode/supernode/activity/interop/interop.go:1032-1122`), plan-building included
//! (`buildRewindPlan`, `:930-981`).
//!
//! A verifier built *without* replacement routes ([`Verifier::with_replacements`]) applies
//! neither [`Decision::Invalidate`] nor [`Decision::Rewind`]: the round logs the decision and
//! makes no progress, which holds cross-safety where it is. Nothing is written to the WAL for a
//! decision that cannot be applied: a slot holding an unappliable transition would wedge the node
//! on every restart.

use crate::{
    archive::{ArchivedOutput, OutputArchive},
    backfill::{
        backfill_chain, backfill_window, chain_backfill_range, fetch_and_seal, verification_start,
    },
    chain::{
        ChainAt, ChainError, InteropChain, L1Canonical, L1CanonicalExt, RewindableChain, RoundError,
    },
    decide::{
        ChainFrontier, Decision, RoundObservation, StepOutput, check_preconditions,
        decide_verified_result,
    },
    error::StoreError,
    kv::Kv,
    logs::LogsDb,
    verified::{
        InvalidHead, PendingTransition, RewindPlan, RoundResult, VerifiedResult, VerifiedStore,
    },
    verify::{FrontierBlock, FrontierView, LogStores, RoundVerdict, verify_round},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{ChainId, map::HashMap};
use kona_genesis::RollupConfig;
use kona_interop::{MESSAGE_EXPIRY_WINDOW, MessageRules};
use std::{collections::BTreeMap, sync::Arc};
use tracing::{debug, error, info, warn};

/// How the verifier is configured.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct VerifierConfig {
    /// The timestamp interop activates at, cluster-wide.
    ///
    /// For the initial release this is the Lagoon activation time, which every chain in the set
    /// shares — the per-chain activation check the rules apply reads each chain's own config, so
    /// the two agree by construction rather than by assumption.
    pub activation_timestamp: u64,
    /// How far back, in seconds, an initiating message may be referenced from.
    pub message_expiry_window: u64,
    /// How far behind the first verified timestamp to seal logs at cold start, in seconds.
    ///
    /// Defaults to the expiry window, so backfill covers every initiating message that could
    /// still be referenced. Zero disables backfill.
    pub log_backfill_depth: u64,
}

impl VerifierConfig {
    /// Returns the default configuration for an activation timestamp.
    pub const fn new(activation_timestamp: u64) -> Self {
        Self {
            activation_timestamp,
            message_expiry_window: MESSAGE_EXPIRY_WINDOW,
            log_backfill_depth: MESSAGE_EXPIRY_WINDOW,
        }
    }
}

/// What the loop should do before its next round.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Pace {
    /// The round made progress; run the next one immediately.
    Immediate,
    /// The round was a no-op; wait for the world to change.
    Idle,
    /// The round failed transiently; wait longer, then retry.
    Retry,
}

/// Where the verifier is in its lifecycle.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum VerifierState {
    /// Still choosing a starting timestamp and filling the log stores behind it.
    ColdStart,
    /// Verifying rounds.
    Running,
    /// Stopped for good.
    Halted,
}

/// The verifier has stopped for good.
///
/// Distinct from a transient round failure on purpose: a halted verifier stops producing verified
/// frontiers, so cross-safety stops advancing, and an operator has to act. Retrying cannot help.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[error("interop verification halted: {reason}")]
pub struct Halted {
    /// Why the verifier halted.
    pub reason: String,
}

/// One chain's invalidation route: the archive its invalidated outputs are recorded in, and the
/// seam that rewinds it.
///
/// The archive doubles as the deny list the chain's engine consults, which is what turns the
/// post-rewind rebuild into a deposits-only replacement — the same one record op-supernode keeps
/// (its deny list *is* its invalidated-output store, one entry of payload hash, decision
/// timestamp, state root and message-passer storage root).
#[derive(Debug)]
pub struct ChainReplacement<K> {
    /// The chain's invalidated-output archive.
    pub archive: Arc<OutputArchive<K>>,
    /// The rewind seam over the chain.
    pub chain: Arc<dyn RewindableChain>,
}

/// The timestamp-lockstep interop verifier.
///
/// Owns the stores it writes and the read-only seams it observes through. One instance per
/// process: the chain set is fixed at construction, as it is everywhere else in lokahi, because a
/// chain joining midway would change the answer to questions already answered.
#[derive(Debug)]
pub struct Verifier<K> {
    /// The chains being verified, by id.
    chains: BTreeMap<ChainId, Arc<dyn InteropChain>>,
    /// Each chain's rollup config, as the shared rules' fallback lookup wants them.
    rollup_configs: HashMap<u64, RollupConfig>,
    /// The L1 every chain derives from.
    l1: Arc<dyn L1Canonical>,
    /// The verified frontier and the WAL slot.
    verified: VerifiedStore<K>,
    /// Each chain's invalidation route, when this verifier applies invalidations.
    ///
    /// Empty on a verifier built without [`Verifier::with_replacements`], which then holds the
    /// frontier on a [`Decision::Invalidate`] instead of applying it.
    replacements: BTreeMap<ChainId, ChainReplacement<K>>,
    /// Each chain's log store.
    stores: LogStores,
    /// The configuration.
    config: VerifierConfig,
    /// The first timestamp this verifier will attempt, once chosen. [`None`] while cold-starting.
    start: Option<u64>,
    /// The L1 block the verifier has considered up to.
    current_l1: Option<BlockNumHash>,
    /// Why the verifier halted, once it has.
    halted: Option<String>,
    /// The timestamp verification stops at, when a test has asked it to.
    ///
    /// Test-control state, and the only mutable knob on this type that production never turns.
    /// It lives on the verifier rather than beside it because the check it drives has to be the
    /// one the round loop makes: a pause enforced anywhere else would still let a round already
    /// in flight commit the timestamp the test wanted held back. See [`Self::set_pause`].
    pause_at: Option<u64>,
    /// How many cold-start attempts have been made in this process.
    ///
    /// Counted rather than derived because the thing tests ask about is the *retry loop*: a
    /// verifier waiting for a chain's first safe head is indistinguishable, from the outside,
    /// from one that has not been stepped at all. See [`Self::backfill_attempts`].
    backfill_attempts: u32,
}

impl<K: Kv> Verifier<K> {
    /// Builds a verifier over the given chains and stores.
    ///
    /// Resumes rather than cold-starts when the verified store already holds a frontier: the next
    /// timestamp is one past the last committed one, and backfill is skipped, because the log
    /// stores were filled when that frontier was committed. That last part is checked rather than
    /// assumed, because the log stores are separate databases from the verified store: a resumed
    /// verifier whose stores lost the window would read a valid executing message as a protocol
    /// violation. A store that cannot cover the window halts the verifier here, before its first
    /// round.
    pub fn new(
        chains: Vec<Arc<dyn InteropChain>>,
        l1: Arc<dyn L1Canonical>,
        verified: VerifiedStore<K>,
        stores: LogStores,
        config: VerifierConfig,
    ) -> Result<Self, RoundError> {
        let rollup_configs =
            chains.iter().map(|c| (c.chain_id(), c.rollup_config().clone())).collect();
        let chains: BTreeMap<_, _> = chains.into_iter().map(|c| (c.chain_id(), c)).collect();

        if let Some(missing) = chains.keys().find(|id| !stores.contains_key(id)) {
            return Err(RoundError::Invariant(format!(
                "chain {missing} has no log store: every chain the verifier follows must have one, \
                 or a message referencing it could not be answered"
            )));
        }

        // A slot in the WAL is also a resume: it is written only after cold start has finished, so
        // its timestamp is a start the verifier already chose. Re-running cold start instead would
        // pick a start from today's safe heads while a transition for the old one sits unapplied.
        // A rewind slot carries no such timestamp — over an empty store it *means* everything was
        // dropped — so cold start runs again once the slot has been replayed (see `step`).
        let start = match verified.last_timestamp() {
            Some(last) => Some(last + 1),
            None => verified
                .pending()?
                .and_then(|pending| pending.result().map(|result| result.verified.timestamp)),
        };
        if let Some(start) = start {
            info!(
                target: "lokahi_interop",
                start,
                activation = config.activation_timestamp,
                "Resuming interop verification"
            );
        }

        // Resuming skips backfill, so this is the only place the stores' coverage is established.
        let halted = match start {
            Some(start) => Self::log_history_gap(&chains, &stores, &config, start)?,
            None => None,
        };
        if let Some(reason) = &halted {
            error!(
                target: "lokahi_interop",
                reason,
                remediation = "reseed the interop data directory, or advance the activation \
                               timestamp past the gap and rederive",
                "Interop verification halted at startup"
            );
        }

        Ok(Self {
            chains,
            rollup_configs,
            l1,
            verified,
            replacements: BTreeMap::new(),
            stores,
            config,
            start,
            current_l1: None,
            halted,
            pause_at: None,
            backfill_attempts: 0,
        })
    }

    /// Wires the invalidation routes, one per chain, making this verifier apply
    /// [`Decision::Invalidate`] rather than hold on it.
    ///
    /// Every chain of the verifier must have a route: an invalidation names the chains it found
    /// invalid only at decision time, and a chain that could be verified but not rewound would
    /// turn an applied decision into a permanent error mid-apply.
    pub fn with_replacements(
        mut self,
        replacements: BTreeMap<ChainId, ChainReplacement<K>>,
    ) -> Result<Self, RoundError> {
        if let Some(missing) = self.chains.keys().find(|id| !replacements.contains_key(id)) {
            return Err(RoundError::Invariant(format!(
                "chain {missing} has no invalidation route: every chain the verifier follows must \
                 have one, or a block it finds invalid could not be replaced"
            )));
        }
        self.replacements = replacements;
        Ok(self)
    }

    /// Returns where the verifier is in its lifecycle.
    pub const fn state(&self) -> VerifierState {
        if self.halted.is_some() {
            VerifierState::Halted
        } else if self.start.is_none() {
            VerifierState::ColdStart
        } else {
            VerifierState::Running
        }
    }

    /// Returns the verified store.
    pub const fn verified(&self) -> &VerifiedStore<K> {
        &self.verified
    }

    /// Returns the earliest timestamp this verifier covers.
    ///
    /// Once anything is committed, the store's first timestamp is authoritative and cannot move.
    pub fn first_verifiable_timestamp(&self) -> Option<u64> {
        self.verified.first_timestamp().or(self.start)
    }

    /// Returns the timestamp the cluster activates interop at.
    pub const fn activation_timestamp(&self) -> u64 {
        self.config.activation_timestamp
    }

    /// Returns the timestamp this verifier's round loop began at, or [`None`] while cold-starting.
    ///
    /// Distinct from [`Self::first_verifiable_timestamp`], which is the lowest timestamp the
    /// verifier *covers*: once a frontier has been committed, that is the store's first timestamp
    /// and cannot move, while this stays the start the current run chose.
    pub const fn verification_start(&self) -> Option<u64> {
        self.start
    }

    /// Returns how many cold-start attempts this verifier has made.
    ///
    /// Zero once the count is no longer moving *and* [`Self::verification_start`] is set on a
    /// resumed verifier: resuming chooses its start in [`Self::new`], so no attempt is ever made.
    pub const fn backfill_attempts(&self) -> u32 {
        self.backfill_attempts
    }

    /// Returns one chain's log store, or [`None`] when the chain is not followed.
    ///
    /// The store is the verifier's, handed out behind a shared pointer rather than copied: a
    /// reader gets the same database the round loop seals into, so what it reports cannot lag
    /// what the loop has written.
    pub fn logs(&self, chain_id: ChainId) -> Option<&Arc<dyn LogsDb>> {
        self.stores.get(&chain_id)
    }

    /// Stops the round loop at `timestamp`, or clears an existing pause with [`None`].
    ///
    /// Test control, and inclusive and forward-looking on purpose: a verifier already past
    /// `timestamp` still stops, so a test that asks late is not silently given a running verifier.
    /// It matches op-supernode's `PauseAt`, whose zero means "clear" — expressed here as [`None`]
    /// so a caller cannot mean timestamp zero and get a clear.
    ///
    /// Production never calls this. The pause is checked where the round loop decides which
    /// timestamp to attempt, so a round already in flight finishes and no later one starts.
    pub const fn set_pause(&mut self, timestamp: Option<u64>) {
        self.pause_at = timestamp;
    }

    /// Returns whether a pause holds the round loop back from `next_timestamp`.
    fn paused_at(&self, next_timestamp: u64) -> bool {
        self.pause_at.is_some_and(|pause| next_timestamp >= pause)
    }

    /// Returns the L1 block the verifier has considered up to, or [`None`] before the first
    /// advance.
    ///
    /// This is the highest L1 block any chain's verified block was derived from. op-supernode
    /// additionally caps it at the lowest L1 block *any* chain has derived from, so the value is
    /// safe to publish on its own rather than only once aggregated. That cap needs the L1 half of
    /// each chain's sync status, which the L1 watcher owns and this seam does not carry; until it
    /// does, a consumer must combine this with its own view by taking the minimum.
    pub const fn current_l1(&self) -> Option<BlockNumHash> {
        self.current_l1
    }

    /// Runs one iteration of the loop, returning how long to wait before the next.
    ///
    /// Never returns a transient failure: a chain that is down, an L1 RPC that timed out and a
    /// chain that has not derived far enough are all ordinary conditions during startup and
    /// operation, and each costs a backoff rather than the activity. Only a condition no retry can
    /// fix halts the verifier, and it stays halted.
    pub async fn step(&mut self) -> Result<Pace, Halted> {
        if let Some(reason) = &self.halted {
            return Err(Halted { reason: reason.clone() });
        }

        // A slot left over from a previous run is applied before anything else — even before cold
        // start: the world the decision was made against is gone, but the decision was already
        // committed to. In particular a full rewind that crashed mid-apply leaves a rewind slot
        // over an empty verified store, and its replay must run before cold start backfills, or
        // the replay's store clear would wipe what backfill just wrote.
        let outcome = match self.verified.pending() {
            Err(err) => Err(err.into()),
            Ok(Some(pending)) => self.apply(pending).await,
            Ok(None) if self.start.is_none() => self.advance_cold_start().await,
            Ok(None) => self.progress().await,
        };

        match outcome {
            Ok(true) => Ok(Pace::Immediate),
            Ok(false) => Ok(Pace::Idle),
            Err(err) if err.is_transient() => {
                warn!(target: "lokahi_interop", %err, "Interop verification round failed; retrying");
                Ok(Pace::Retry)
            }
            Err(err) => {
                let reason = err.to_string();
                error!(
                    target: "lokahi_interop",
                    %err,
                    remediation = "reseed the interop data directory, or advance the activation \
                                   timestamp past the gap and rederive",
                    "Interop verification halted"
                );
                self.halted = Some(reason.clone());
                Err(Halted { reason })
            }
        }
    }

    /// Runs one cold-start step, returning whether it completed initialization.
    ///
    /// Cold start runs alongthe chains' own startup, so every failure it can hit — a chain not
    /// attached yet, an execution layer not up, a safe head not recorded — is expected and retried
    /// rather than fatal.
    async fn advance_cold_start(&mut self) -> Result<bool, RoundError> {
        // Counted at the top, so an attempt that gives up below still counts as one: a test
        // waiting for the retry loop to engage is waiting for exactly those.
        self.backfill_attempts = self.backfill_attempts.saturating_add(1);

        let mut first_safe_heads = Vec::with_capacity(self.chains.len());
        for (&chain_id, chain) in &self.chains {
            match chain.first_safe_head_timestamp().await {
                Ok(timestamp) => first_safe_heads.push((chain_id, timestamp)),
                Err(ChainError::NotReady) => {
                    debug!(
                        target: "lokahi_interop",
                        chain_id,
                        "Interop cold start: waiting for a first safe head"
                    );
                    return Ok(false);
                }
                Err(source) => return Err(RoundError::chain(chain_id, source)),
            }
        }

        let start = verification_start(self.config.activation_timestamp, &first_safe_heads);
        self.backfill(start).await?;
        self.start = Some(start);
        info!(
            target: "lokahi_interop",
            start,
            activation = self.config.activation_timestamp,
            "Interop cold start complete"
        );
        Ok(true)
    }

    /// Fills every chain's log store over the window behind `start`.
    async fn backfill(&self, start: u64) -> Result<(), RoundError> {
        let Some(window) = backfill_window(
            self.config.activation_timestamp,
            start,
            self.config.log_backfill_depth,
        ) else {
            return Ok(());
        };

        for (chain_id, chain) in &self.chains {
            let Some(range) = chain_backfill_range(chain.as_ref(), window).await? else { continue };
            backfill_chain(chain.as_ref(), self.store(*chain_id)?.as_ref(), range).await?;
        }
        Ok(())
    }

    /// Runs one verification round, returning whether the verified frontier advanced.
    ///
    /// Runs with no transition in flight: a leftover WAL slot is applied by [`Self::step`] before
    /// this is reached.
    async fn progress(&mut self) -> Result<bool, RoundError> {
        let observation = self.observe().await?;
        let output = match check_preconditions(&observation) {
            Some(early) => early,
            None => {
                let verdict = self.verify(&observation).await?;
                decide_verified_result(self.round_result(verdict).await?)
            }
        };

        match output.decision {
            Decision::Wait => Ok(false),
            Decision::Advance => {
                let pending = PendingTransition::Advance(output.result);
                // Durable before the first side effect, cleared after the last.
                self.verified.set_pending(&pending)?;
                self.apply(pending).await
            }
            Decision::Invalidate if !self.replacements.is_empty() => {
                let pending = PendingTransition::Invalidate(output.result);
                // Durable before the first side effect, cleared after the last — and the archive
                // write inside the apply is driven from this slot on every replay, so a crash
                // between here and the rewind cannot lose the invalidated output.
                self.verified.set_pending(&pending)?;
                self.apply(pending).await
            }
            Decision::Rewind if !self.replacements.is_empty() => {
                // The observation sets `l1_needs_rewind` only after reading the committed
                // frontier, so its absence here is a broken invariant rather than a case.
                let last = observation.last_verified.as_ref().ok_or_else(|| {
                    RoundError::Invariant(
                        "a rewind was decided without a committed frontier".into(),
                    )
                })?;
                let plan = self.build_rewind_plan(last)?;
                let pending = PendingTransition::Rewind(plan);
                // Durable before the first side effect, cleared after the last. The plan itself —
                // the previous frontier and whether the engines move — is decided *before* the
                // stores change, so a crash replay applies what was decided rather than
                // re-deriving a plan from stores the first attempt already rewound
                // (op-supernode WALs its `RewindPlan` the same way, `interop.go:732-740`).
                self.verified.set_pending(&pending)?;
                self.apply(pending).await
            }
            decision @ (Decision::Invalidate | Decision::Rewind) => {
                Self::hold(decision, observation.next_timestamp, &output);
                Ok(false)
            }
        }
    }

    /// Records a decision this verifier does not apply: a rewind or an invalidation, on a
    /// verifier built without replacement routes.
    ///
    /// The timestamp comes from the observation rather than from the result: a rewind carries no
    /// result, so reading it there would report zero.
    fn hold(decision: Decision, timestamp: u64, output: &StepOutput) {
        warn!(
            target: "lokahi_interop",
            %decision,
            timestamp,
            chains = ?output.result.invalid_heads.keys().collect::<Vec<_>>(),
            "Interop verification reached a decision this verifier does not apply; the verified \
             frontier holds where it is"
        );
    }

    /// Observes every chain and the L1 once.
    async fn observe(&self) -> Result<RoundObservation, RoundError> {
        let last_timestamp = self.verified.last_timestamp();
        let last_verified = match last_timestamp {
            Some(timestamp) => Some(self.verified.get(timestamp)?),
            None => None,
        };
        let next_timestamp = match last_timestamp {
            Some(last) => last + 1,
            // `progress` runs only once cold start has chosen a start.
            None => self.start.ok_or_else(|| {
                RoundError::Invariant("a round was run before cold start finished".into())
            })?,
        };

        let mut observation = RoundObservation {
            last_verified,
            next_timestamp,
            frontier: None,
            l1_consistent: false,
            l1_needs_rewind: false,
        };

        // A pause is honoured here, where the timestamp to attempt has just been chosen and
        // nothing has been observed yet, which is where op-supernode checks it too. A frontier-less
        // observation is a `Decision::Wait`, so the round loop idles without deciding anything —
        // the paused verifier answers reads from the frontier it already committed, and commits
        // nothing further.
        if self.paused_at(next_timestamp) {
            debug!(
                target: "lokahi_interop",
                next_timestamp,
                pause_at = self.pause_at,
                "Interop verification is paused"
            );
            return Ok(observation);
        }

        let Some(frontier) = self.observe_frontier(next_timestamp).await? else {
            return Ok(observation);
        };
        observation.frontier = Some(frontier);

        // The committed frontier is checked first and on its own. If the L1 block it rests on has
        // left the canonical chain, everything built on it has to go, and no amount of waiting
        // changes that.
        if let Some(last) = &observation.last_verified &&
            !self.l1.all_canonical(&[last.l1_inclusion]).await.map_err(RoundError::L1)?
        {
            observation.l1_needs_rewind = true;
            return Ok(observation);
        }

        // The new frontier is checked separately: a chain whose own L1 head is stale is behind an
        // L1 reorg it has not caught up with, which waiting fixes and rewinding would not.
        observation.l1_consistent = self
            .l1
            .all_canonical(&observation.frontier_l1_blocks())
            .await
            .map_err(RoundError::L1)?;

        Ok(observation)
    }

    /// Asks every chain for its block at `timestamp`, or returns [`None`] if any cannot answer.
    async fn observe_frontier(
        &self,
        timestamp: u64,
    ) -> Result<Option<BTreeMap<ChainId, ChainFrontier>>, RoundError> {
        let mut frontier = BTreeMap::new();
        for (&chain_id, chain) in &self.chains {
            let at = chain
                .local_safe_at(timestamp)
                .await
                .map_err(|source| RoundError::chain(chain_id, source))?;
            match at {
                ChainAt::Derived { block, l1 } => {
                    frontier.insert(chain_id, ChainFrontier { block, l1_inclusion: l1 });
                }
                ChainAt::NotYet => {
                    debug!(
                        target: "lokahi_interop",
                        chain_id,
                        timestamp,
                        "Chain has not reached the round's timestamp yet"
                    );
                    return Ok(None);
                }
                ChainAt::HistoryUnavailable => {
                    return Err(RoundError::Permanent(format!(
                        "chain {chain_id}: which L1 block made timestamp {timestamp} safe is no \
                         longer recorded on this node"
                    )));
                }
                ChainAt::BeforeGenesis => {
                    return Err(RoundError::Permanent(format!(
                        "chain {chain_id}: timestamp {timestamp} predates the chain's genesis, so \
                         no block of it can ever carry that timestamp"
                    )));
                }
            }
        }
        Ok(Some(frontier))
    }

    /// Fetches the round's frontier blocks and verifies their messages.
    async fn verify(&self, observation: &RoundObservation) -> Result<RoundVerdict, RoundError> {
        let frontier = observation
            .frontier
            .as_ref()
            .ok_or_else(|| RoundError::Invariant("verification ran without a frontier".into()))?;
        let l1_inclusion = observation
            .l1_inclusion()
            .ok_or_else(|| RoundError::Invariant("a frontier with no L1 inclusion".into()))?;

        let mut blocks = BTreeMap::new();
        for (&chain_id, chain_frontier) in frontier {
            let chain = self.chain(chain_id)?;
            let logs = chain
                .block_logs(chain_frontier.block)
                .await
                .map_err(|source| RoundError::chain(chain_id, source))?;
            blocks.insert(chain_id, FrontierBlock::index(chain_id, &logs));
        }
        let view = FrontierView::new(blocks);

        let rules = MessageRules::new(&self.rollup_configs, self.config.message_expiry_window);
        verify_round(
            observation.next_timestamp,
            l1_inclusion,
            frontier,
            &view,
            &self.stores,
            &rules,
        )
    }

    /// Turns a verdict's named invalid blocks into the round result, reading each one's output
    /// preimage.
    ///
    /// Kept out of verification so the verdict stays a function of the round's inputs: the
    /// preimage is only needed to *describe* an already-decided invalidity to the optimistic
    /// superroot branch, never to reach it.
    async fn round_result(&self, verdict: RoundVerdict) -> Result<RoundResult, RoundError> {
        let RoundVerdict { verified, invalid } = verdict;
        let mut invalid_heads = BTreeMap::new();
        for (chain_id, reason) in invalid {
            let block = *verified.l2_heads.get(&chain_id).ok_or_else(|| {
                RoundError::Invariant(format!(
                    "chain {chain_id} was found invalid but is not in the round's frontier"
                ))
            })?;
            warn!(target: "lokahi_interop", chain_id, ?block, %reason, "Block found invalid");
            let output = self
                .chain(chain_id)?
                .output_at(block.number)
                .await
                .map_err(|source| RoundError::chain(chain_id, source))?;
            invalid_heads.insert(
                chain_id,
                InvalidHead {
                    block,
                    state_root: output.state_root,
                    message_passer_storage_root: output.bridge_storage_root,
                },
            );
        }
        Ok(RoundResult { verified, invalid_heads })
    }

    /// Applies a write-ahead-logged transition, returning whether the frontier advanced.
    async fn apply(&mut self, pending: PendingTransition) -> Result<bool, RoundError> {
        match pending {
            PendingTransition::Advance(result) => self.apply_advance(result).await,
            PendingTransition::Invalidate(result) => self.apply_invalidate(result).await,
            PendingTransition::Rewind(plan) => self.apply_rewind(plan).await,
        }
    }

    /// Builds the plan for rewinding off a committed frontier whose L1 basis reorged away.
    ///
    /// op-supernode's `buildRewindPlan` (`op-supernode/supernode/activity/interop/interop.go:
    /// 930-981`): everything at or after the last verified timestamp is dropped; the engines are
    /// reset — onto the timestamp one before it — only when an archived invalidation decided at
    /// or after that timestamp exists (`shouldResetEnginesOnRewind`, `:1019-1030`); and the log
    /// stores' landing frontier is the previous verified result, or nothing when the last
    /// timestamp is also the first retained one, which makes the rewind a full one.
    ///
    /// Any failure here aborts the round before the WAL is written, and the decision is
    /// re-evaluated next round — the same posture as op-supernode, whose build failures return
    /// without persisting a transition.
    fn build_rewind_plan(&self, last: &VerifiedResult) -> Result<RewindPlan, RoundError> {
        let rewind_at_or_after = last.timestamp;

        let mut reset_engines = false;
        for replacement in self.replacements.values() {
            if replacement.archive.has_at_or_after(rewind_at_or_after)? {
                reset_engines = true;
                break;
            }
        }
        let reset_chains_to = if reset_engines {
            // Unreachable through the round loop: an archived decision timestamp is a verified
            // round's, and verification starts at or after activation, which is past zero.
            Some(rewind_at_or_after.checked_sub(1).ok_or_else(|| {
                RoundError::Invariant("cannot reset the chains behind timestamp zero".into())
            })?)
        } else {
            None
        };

        let first = self.verified.first_timestamp().ok_or_else(|| {
            RoundError::Invariant("a rewind was decided with nothing committed".into())
        })?;
        let target_heads = if rewind_at_or_after <= first {
            // No earlier frontier is retained, so everything goes: op-supernode's full-rewind
            // branch, which clears the log stores rather than rewinding them onto a frontier.
            None
        } else {
            Some(self.verified.get(rewind_at_or_after - 1)?.l2_heads)
        };

        Ok(RewindPlan { rewind_at_or_after, reset_chains_to, target_heads })
    }

    /// Drops what was verified on the reorged basis, then clears the WAL slot.
    ///
    /// The order is op-supernode's `applyRewindPlan` (`op-supernode/supernode/activity/interop/
    /// interop.go:1032-1096`): the verified results at or after the rewound timestamp go first;
    /// then each chain's archive is pruned of outputs whose invalidation was decided at or after
    /// it — the archive is the deny list, so this revokes denials whose basis no longer stands —
    /// then the log stores rewind onto the previous verified frontier (or clear entirely on a
    /// full rewind); and only after everything above succeeded are the engines reset, when the
    /// plan says an invalidation was revoked. The pruning must precede the engine move, so no
    /// window exists in which derivation rebuilds a block that is still denied by a revoked
    /// decision.
    ///
    /// Nothing is committed and no progress is reported: the next round re-observes from the new
    /// frontier, on the new L1. A failure anywhere leaves the WAL slot in place and costs a
    /// backoff — every side effect here is idempotent, so the retry re-applies the plan whole.
    /// (op-supernode records per-chain failures and joins them at the end; the slot-preserving
    /// retry of the whole idempotent plan is the same recovery, reached on the first failure.)
    async fn apply_rewind(&mut self, plan: RewindPlan) -> Result<bool, RoundError> {
        // The cursor described progress on frontiers that are being dropped. op-supernode zeroes
        // it before applying (`interop.go:783-785`); the next advance re-establishes it.
        self.current_l1 = None;

        if plan.reset_chains_to.is_some() && self.replacements.is_empty() {
            // A replay on a verifier built without routes cannot move the engines the plan says
            // must move, and silently skipping them would leave denied-and-revoked replacements
            // canonical forever.
            return Err(RoundError::Invariant(
                "a write-ahead rewind requires the chains reset, but this verifier has no \
                 replacement routes"
                    .into(),
            ));
        }

        self.verified.rewind(plan.rewind_at_or_after)?;

        for (&chain_id, replacement) in &self.replacements {
            let removed = replacement.archive.prune_at_or_after(plan.rewind_at_or_after)?;
            if !removed.is_empty() {
                warn!(
                    target: "lokahi_interop",
                    chain_id,
                    ?removed,
                    "Pruned archived outputs whose invalidation basis was reorged out"
                );
            }
        }

        match &plan.target_heads {
            // A full rewind: nothing verified remains, so nothing sealed can be relied on either.
            None => {
                for store in self.stores.values() {
                    store.clear()?;
                }
            }
            Some(heads) => {
                for (chain_id, head) in heads {
                    let Some(store) = self.stores.get(chain_id) else { continue };
                    // op-supernode's guards (`interop.go:1080-1086`): nothing sealed, already on
                    // the landing head, or behind it — nothing to rewind. A store *at* the head's
                    // height on a different block falls through, and the store's own conflict
                    // check names it.
                    if let Some(latest) = store.latest_sealed_block() &&
                        latest != *head &&
                        latest.number >= head.number
                    {
                        info!(
                            target: "lokahi_interop",
                            chain_id,
                            from = ?latest,
                            to = ?head,
                            "Rewinding the log store onto the previous verified frontier"
                        );
                        store.rewind(*head)?;
                    }
                }
            }
        }

        if plan.reset_chains_to.is_some() {
            self.reset_chains(&plan).await?;
        }

        self.verified.clear_pending()?;
        warn!(
            target: "lokahi_interop",
            timestamp = plan.rewind_at_or_after,
            engines_reset = plan.reset_chains_to.is_some(),
            "Rewound the verified frontier: its L1 basis was reorged out"
        );
        Ok(false)
    }

    /// Resets every chain's engine onto the rewind's landing frontier.
    ///
    /// All chains, not only those with revoked invalidations — op-supernode's
    /// `resetChainEnginesIfNeeded` walks every chain (`interop.go:1099-1122`): the frontier is
    /// verified as a set, so the set lands together. The target is the plan's recorded head for
    /// the chain; on a full rewind there is no recorded frontier, so the target is resolved from
    /// the chain as the block covering the reset timestamp — what op-supernode's
    /// `captureRewindPayloadsAtTimestamp` does (`interop.go:1002-1016`). Resolution reads below
    /// everything the reorg touched, so it answers the same block on every replay.
    async fn reset_chains(&self, plan: &RewindPlan) -> Result<(), RoundError> {
        let Some(reset_to) = plan.reset_chains_to else { return Ok(()) };
        for (&chain_id, replacement) in &self.replacements {
            let target = match plan.target_heads.as_ref().and_then(|heads| heads.get(&chain_id)) {
                Some(head) => *head,
                None => {
                    let chain = self.chain(chain_id)?;
                    let number = chain
                        .block_number_at_timestamp(reset_to)
                        .await
                        .map_err(|source| RoundError::chain(chain_id, source))?;
                    let output = chain
                        .output_at(number)
                        .await
                        .map_err(|source| RoundError::chain(chain_id, source))?;
                    BlockNumHash { number, hash: output.block_hash }
                }
            };
            replacement
                .chain
                .reset_to(target)
                .await
                .map_err(|source| RoundError::chain(chain_id, source))?;
            warn!(
                target: "lokahi_interop",
                chain_id,
                number = target.number,
                hash = %target.hash,
                "Reset the chain onto the last verified frontier after revoking invalidations"
            );
        }
        Ok(())
    }

    /// Archives each invalid head's output and rewinds its chain, then clears the WAL slot.
    ///
    /// The frontier is deliberately not committed: the round found these blocks invalid, so
    /// nothing about this timestamp is verified. The next round re-verifies the same timestamp
    /// against the replaced chain, which is what op-supernode's `applyPendingTransition` does for
    /// `DecisionInvalidate` (`op-supernode/supernode/activity/interop/interop.go`) — it too
    /// invalidates, resumes, and returns without committing.
    ///
    /// Order within a chain matters and is op-supernode's (`InvalidateBlock`,
    /// `op-supernode/supernode/chain_container/invalidation.go:392-465`): the archive record —
    /// its deny list — is durable *before* the rewind, so no window exists in which the chain is
    /// off the block but nothing stops derivation from rebuilding it verbatim. Both side effects
    /// are idempotent, and both run on every crash replay: in particular the archive write is
    /// unconditional, because a replay may find the block already replaced and skip the rewind —
    /// skipping the write there too would lose the output roots permanently, and a missing
    /// archive entry fails silently at the optimistic superroot branch.
    ///
    /// A failed rewind leaves the slot in place — "transition preserved for retry" — and costs a
    /// backoff; every [`ChainError`] is transient by contract.
    async fn apply_invalidate(&self, result: RoundResult) -> Result<bool, RoundError> {
        let decision_timestamp = result.verified.timestamp;

        for (&chain_id, invalid) in &result.invalid_heads {
            // No parent exists to rewind onto, and no round can decide this: block one of any
            // chain is the earliest verifiable block, so a slot naming height zero is damage, not
            // a decision to retry.
            if invalid.block.number == 0 {
                return Err(RoundError::Permanent(format!(
                    "chain {chain_id}: the genesis block cannot be invalidated: it has no parent \
                     to rewind onto"
                )));
            }
            let replacement = self.replacements.get(&chain_id).ok_or_else(|| {
                RoundError::Invariant(format!("chain {chain_id} has no invalidation route"))
            })?;

            // The archive doubles as the deny list, so this write is what turns derivation's
            // rebuild of the height into the deposits-only replacement — and what the optimistic
            // superroot branch serves for this height ever after.
            replacement.archive.record(
                invalid.block.number,
                ArchivedOutput { output_root: invalid.output_root(), decision_timestamp },
            )?;

            let rewound = replacement
                .chain
                .rewind_off(invalid.block)
                .await
                .map_err(|source| RoundError::chain(chain_id, source))?;
            warn!(
                target: "lokahi_interop",
                chain_id,
                number = invalid.block.number,
                hash = %invalid.block.hash,
                decision_timestamp,
                rewound,
                "Applied an invalidation: archived the block's output and took the chain off it"
            );
        }

        self.verified.clear_pending()?;
        Ok(false)
    }

    /// Seals the round's blocks, commits its frontier, and clears the WAL slot.
    ///
    /// Order matters and is the reverse of what reads want: the log stores are written first, so
    /// that a committed frontier is never the frontier of a timestamp whose logs are missing. Both
    /// writes are idempotent, so a crash anywhere in here replays cleanly.
    async fn apply_advance(&mut self, result: RoundResult) -> Result<bool, RoundError> {
        let VerifiedResult { timestamp, l1_inclusion, ref l2_heads } = result.verified;

        for (&chain_id, &block) in l2_heads {
            fetch_and_seal(self.chain(chain_id)?.as_ref(), self.store(chain_id)?.as_ref(), block)
                .await?;
        }

        self.verified.commit(&result.verified)?;
        self.verified.clear_pending()?;
        self.current_l1 = Some(l1_inclusion);

        info!(
            target: "lokahi_interop",
            timestamp,
            l1 = l1_inclusion.number,
            "Committed a verified frontier"
        );
        Ok(true)
    }

    /// Returns one chain, or the invariant failure that it is not in the set.
    fn chain(&self, chain_id: ChainId) -> Result<&Arc<dyn InteropChain>, RoundError> {
        self.chains.get(&chain_id).ok_or_else(|| {
            RoundError::Invariant(format!("chain {chain_id} is not in the verifier's chain set"))
        })
    }

    /// Returns one chain's log store, or the invariant failure that it has none.
    fn store(&self, chain_id: ChainId) -> Result<&Arc<dyn LogsDb>, RoundError> {
        self.stores
            .get(&chain_id)
            .ok_or_else(|| RoundError::Invariant(format!("chain {chain_id} has no log store")))
    }

    /// Returns why the log stores cannot answer what rounds from `start` will ask them, if they
    /// cannot.
    ///
    /// A round at timestamp `t` answers an existence question about `t` itself from its own view,
    /// and every earlier timestamp from the chain's log store. The log stores are separate
    /// databases from the verified store, and resuming does not backfill, so a store that was
    /// truncated, reseeded or lost leaves the round loop unable to answer a question the
    /// protocol says has an answer. Naming that here halts the verifier once, at startup, with
    /// the cause attached — the alternative is discovering it mid-round, where a missing block
    /// is indistinguishable from an invalid one.
    ///
    /// The bound is the earliest timestamp a round from `start` can be asked about, which is the
    /// latest of three: the expiry window behind `start`, because no older initiating message
    /// can be executed; the interop activation time, because no earlier message can be
    /// initiated; and the chain's own genesis, because no earlier block of it exists. The last
    /// two are the bounds [`backfill_window`] and [`chain_backfill_range`] already apply when
    /// filling the stores.
    fn log_history_gap(
        chains: &BTreeMap<ChainId, Arc<dyn InteropChain>>,
        stores: &LogStores,
        config: &VerifierConfig,
        start: u64,
    ) -> Result<Option<String>, RoundError> {
        for (&chain_id, chain) in chains {
            // Checked by the caller before this runs; skipping is the caller's error to report.
            let Some(store) = stores.get(&chain_id) else { continue };

            let earliest = start
                .saturating_sub(config.message_expiry_window)
                .max(config.activation_timestamp)
                .max(chain.rollup_config().genesis.l2_time);
            // Nothing below `start` can be referenced, so holding no history at all is correct.
            if earliest >= start {
                continue;
            }

            let held = match store.first_sealed_block() {
                Ok(first) if first.timestamp <= earliest => continue,
                Ok(first) => format!("starts at timestamp {}", first.timestamp),
                // A store with nothing sealed reports its first block as still to come.
                Err(StoreError::Future) => "is empty".to_string(),
                Err(err) => return Err(RoundError::Store(err)),
            };
            return Ok(Some(format!(
                "chain {chain_id}: the log store {held}, but verification resumes at timestamp \
                 {start}, where an executing message may reference back to {earliest}; the store \
                 is missing history the round loop needs and must be refilled to continue"
            )));
        }
        Ok(None)
    }
}
