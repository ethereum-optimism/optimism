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
//! ## What this phase applies
//!
//! Only [`Decision::Wait`] and [`Decision::Advance`]. A round that reaches
//! [`Decision::Invalidate`] or [`Decision::Rewind`] logs it and makes no progress — the verified
//! frontier simply stops, which holds cross-safety where it is instead of promoting a block the
//! verifier believes is wrong. Nothing is written to the WAL for a decision that cannot be
//! applied: a slot holding an unappliable transition would wedge the node on every restart.

use crate::{
    backfill::{
        backfill_chain, backfill_window, chain_backfill_range, fetch_and_seal, verification_start,
    },
    chain::{ChainAt, ChainError, InteropChain, L1Canonical, L1CanonicalExt, RoundError},
    decide::{
        ChainFrontier, Decision, RoundObservation, StepOutput, check_preconditions,
        decide_verified_result,
    },
    kv::Kv,
    logs::LogsDb,
    verified::{InvalidHead, PendingTransition, RoundResult, VerifiedResult, VerifiedStore},
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
}

impl<K: Kv> Verifier<K> {
    /// Builds a verifier over the given chains and stores.
    ///
    /// Resumes rather than cold-starts when the verified store already holds a frontier: the next
    /// timestamp is one past the last committed one, and backfill is skipped, because the log
    /// stores were filled when that frontier was committed.
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
        let start = match verified.last_timestamp() {
            Some(last) => Some(last + 1),
            None => verified.pending()?.map(|pending| pending.result().verified.timestamp),
        };
        if let Some(start) = start {
            info!(
                target: "lokahi_interop",
                start,
                activation = config.activation_timestamp,
                "Resuming interop verification"
            );
        }

        Ok(Self {
            chains,
            rollup_configs,
            l1,
            verified,
            stores,
            config,
            start,
            current_l1: None,
            halted: None,
        })
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

        let outcome = if self.start.is_none() {
            self.advance_cold_start().await
        } else {
            self.progress().await
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
    async fn progress(&mut self) -> Result<bool, RoundError> {
        // A slot left over from a previous run is applied before anything is observed: the world
        // it was decided against is gone, but the decision was already committed to.
        if let Some(pending) = self.verified.pending()? {
            return self.apply(pending).await;
        }

        let observation = self.observe().await?;
        let output = match check_preconditions(&observation) {
            Some(early) => early,
            None => {
                let verdict = self.verify(&observation).await?;
                decide_verified_result(self.into_round_result(verdict).await?)
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
            decision @ (Decision::Invalidate | Decision::Rewind) => {
                self.hold(decision, &output);
                Ok(false)
            }
        }
    }

    /// Records a decision this phase does not apply.
    fn hold(&self, decision: Decision, output: &StepOutput) {
        warn!(
            target: "lokahi_interop",
            %decision,
            timestamp = output.result.verified.timestamp,
            chains = ?output.result.invalid_heads.keys().collect::<Vec<_>>(),
            "Interop verification reached a decision this phase does not apply; the verified \
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
    async fn into_round_result(&self, verdict: RoundVerdict) -> Result<RoundResult, RoundError> {
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
            PendingTransition::Invalidate(result) => Err(RoundError::Permanent(format!(
                "the write-ahead log holds an invalidation at timestamp {} that this build cannot \
                 apply; it was written by a later version",
                result.verified.timestamp
            ))),
        }
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
}
