//! Serial discovery with nested callbacks through the real atomic router and inbox.
//!
//! The envelope callback can wrap router calldata in a signed ERC-4337 operation.
//! Every chain retains one live transaction and receives one final canonical run.
//! No database is committed here; whole-block verification remains the caller's job.
pub mod abi;
mod messages;
mod tape;
use crate::{Application, Candidate, Endpoint, Limits, Progress, Verified, discover_with};
use abi::*;
use alloy_eips::eip2930::{AccessList, AccessListItem};
use alloy_sol_types::{SolCall, SolEvent, SolValue};
use op_revm::{OpContext, OpTransaction};
use revm::{
    Database,
    context::TxEnv,
    primitives::{Address, B256, Bytes, Log, U256, address, keccak256},
};
use std::collections::BTreeMap;

/// Existing `CrossL2Inbox` predeploy; no protocol changes are made by this adapter.
pub const INBOX: Address = address!("4200000000000000000000000000000000000022");
/// Pinned chain environment, state and application budget.
#[derive(Debug)]
pub struct Chain<DB: Database> {
    /// Isolated prefix snapshot. The block environment must be the candidate's exact environment.
    pub context: OpContext<DB>,
    /// Same deployed router implementation/address on every chain.
    pub router: Address,
    /// EOA or ERC-4337 account that calls the root router.
    pub sender: Address,
    /// Logs in earlier transactions, excluding logs of this transaction's envelope.
    pub prefix_logs: u32,
    /// Fixed gas forwarded to top-level application calls; nested callbacks share local gas.
    pub application_gas: u64,
}
/// Discovery/canonical validation failure; candidates must be discarded, never retried in a loop.
#[derive(Debug, thiserror::Error)]
pub enum Error<DB: Database> {
    /// EVM continuation failure.
    #[error(transparent)]
    Core(#[from] crate::Error<DB>),
    /// Invalid configuration, transcript or protocol messages.
    #[error("{0}")]
    Invalid(&'static str),
    /// Invalid contract ABI response.
    #[error(transparent)]
    Abi(#[from] alloy_sol_types::Error),
    /// Envelope preparation/signing failure.
    #[error("envelope: {0}")]
    Envelope(String),
}
/// Exact canonical transaction and its isolated effects.
#[derive(Debug)]
pub struct Included {
    /// Envelope used by the single final replay.
    pub transaction: OpTransaction<TxEnv>,
    /// Canonical receipt and state; not automatically committed.
    pub execution: Verified,
}
/// A locally verified bundle. Run the existing whole-block/protocol gate before publishing.
#[derive(Debug)]
pub struct Bundle {
    /// Included transactions, one per participating chain.
    pub chains: BTreeMap<u64, Included>,
    /// Whether the root application rolled back.
    pub reverted: bool,
}
#[derive(Clone, Debug, Default)]
struct Tape {
    witnesses: Vec<AtomicResultWitness>,
    calls: Vec<AtomicRemoteCall>,
    callbacks: Vec<AtomicCallback>,
    completion: Identifier,
}
struct Live<DB: Database> {
    progress: Option<Progress<DB>>,
    tape: Tape,
    accesses: Vec<B256>,
    discovery_data: Bytes,
}
/// Prepare the exact outer transaction, including the supplied inbox access list.
/// Called once for discovery and once for final replay per included chain.
/// Local software signing is expected; this is not a one-signature wallet flow.
pub trait Envelope {
    /// Bind router data and accesses into an ordinary signed user or bundler envelope.
    fn prepare(
        &mut self,
        chain: u64,
        router_data: Bytes,
        accesses: AccessList,
    ) -> Result<OpTransaction<TxEnv>, String>;
}
impl<F> Envelope for F
where
    F: FnMut(u64, Bytes, AccessList) -> Result<OpTransaction<TxEnv>, String>,
{
    fn prepare(
        &mut self,
        chain: u64,
        data: Bytes,
        accesses: AccessList,
    ) -> Result<OpTransaction<TxEnv>, String> {
        self(chain, data, accesses)
    }
}
/// Builds repeated, multi-destination and nested callback bundles with suspended frames.
/// Each callback executes inside the destination chain's original transaction.
#[derive(Debug)]
pub struct Builder<DB: Database, E> {
    /// Pinned chain snapshots, keyed by chain ID.
    pub chains: BTreeMap<u64, Chain<DB>>,
    /// Ordinary transaction / signed ERC-4337 preparation.
    pub envelope: E,
    /// Global remote-call bound (and each destination's batch capacity).
    pub max_calls: u16,
    /// Maximum endpoint calldata or response length.
    pub max_bytes: usize,
}
impl<DB: Database + Clone, E: Envelope> Builder<DB, E>
where
    DB::Error: Clone,
{
    /// Discover once, materialize the tape, replay once, then match all surviving messages.
    pub fn build(
        &mut self,
        root: u64,
        nonce: U256,
        target: Address,
        data: Bytes,
    ) -> Result<Bundle, Error<DB>> {
        let chain = self.chains.get(&root).ok_or(Error::Invalid("missing root chain"))?;
        if self.max_calls == 0 || self.max_bytes == 0 || chain.application_gas == 0 {
            return Err(Error::Invalid("empty discovery budget"));
        }
        for (id, peer) in &self.chains {
            if peer.context.cfg.chain_id != *id ||
                peer.router != chain.router ||
                peer.context.block.timestamp != chain.context.block.timestamp ||
                peer.application_gas == 0
            {
                return Err(Error::Invalid("inconsistent pinned chain configuration"));
            }
        }
        let bundle = keccak256((U256::from(root), chain.router, chain.sender, nonce).abi_encode());
        let mut live = BTreeMap::new();
        let root_tape = Tape::default();
        let root_data = |t: &Tape, gas| {
            executeRootNestedCall {
                nonce,
                target,
                data: data.clone(),
                witnesses: t.witnesses.clone(),
                callbacks: t.callbacks.clone(),
                applicationGas: gas,
            }
            .abi_encode()
            .into()
        };
        let remote_data = |t: &Tape, gas, max| {
            executeRemoteNestedCall {
                bundleId: bundle,
                calls: t.calls.clone(),
                witnesses: t.witnesses.clone(),
                callbacks: t.callbacks.clone(),
                rootCompletion: t.completion.clone(),
                applicationGas: gas,
                maxCalls: max,
            }
            .abi_encode()
            .into()
        };
        let discovery_data: Bytes = root_data(&root_tape, chain.application_gas);
        let root_progress = self.start(root, discovery_data.clone())?;
        live.insert(
            root,
            Live {
                progress: Some(root_progress),
                tape: root_tape,
                accesses: Vec::new(),
                discovery_data,
            },
        );
        let mut calls = 0usize;
        self.run_until_result(root, None, bundle, &mut live, &mut calls, 0)?;
        let root_candidate = candidate(live[&root].progress.as_ref().unwrap())?;
        validate_route(root_candidate, &live[&root].discovery_data)?;
        let completion = self
            .last_identifier(root, root_candidate.result().logs(), BundleCompleted::SIGNATURE_HASH)
            .ok();
        let reverted = !root_candidate.routes()[0].result.is_ok();
        if reverted == completion.is_some() {
            return Err(Error::Invalid("root outcome/completion mismatch"));
        }
        let destinations = live.keys().copied().filter(|id| *id != root).collect::<Vec<_>>();
        for id in destinations {
            if reverted {
                let aborted = match live[&id].progress.as_ref().unwrap() {
                    Progress::Complete(candidate) => {
                        validate_route(candidate, &live[&id].discovery_data)?;
                        !candidate.routes()[0].result.is_ok()
                    }
                    Progress::Paused(_) => false,
                };
                if !aborted {
                    live.remove(&id);
                }
            } else {
                let dest = live.get_mut(&id).unwrap();
                dest.tape.completion = completion.clone().unwrap();
                let Progress::Paused(waiting) = dest.progress.take().unwrap() else {
                    return Err(Error::Invalid("completed destination before root"));
                };
                let reply = tape::quote(&waiting, &self.chains[&id], &dest.tape)?;
                dest.progress = Some(waiting.resolve(reply)?);
                self.advance(id, &mut live)?;
                candidate(live[&id].progress.as_ref().unwrap())?;
            }
        }
        let mut included = BTreeMap::new();
        for (id, transaction) in live {
            let gas = self.chains[&id].application_gas;
            let calldata = if id == root {
                root_data(&transaction.tape, gas)
            } else {
                remote_data(&transaction.tape, gas, self.max_calls)
            };
            let tx = self
                .envelope
                .prepare(id, calldata.clone(), access_list(transaction.accesses))
                .map_err(Error::Envelope)?;
            let context = self.chains[&id].context.clone().with_tx(tx.clone());
            let canonical = discover_with(
                context,
                Vec::new(),
                Some(application(&self.chains[&id])),
                self.limits(),
            )?;
            let canonical = match canonical {
                Progress::Complete(c) => c,
                _ => unreachable!("no canonical hooks"),
            };
            let speculative = candidate(transaction.progress.as_ref().unwrap())?;
            validate_route(speculative, &transaction.discovery_data)?;
            validate_route(&canonical, &calldata)?;
            if canonical.routes()[0].result.is_ok() == reverted ||
                speculative.routes()[0].result.result != canonical.routes()[0].result.result ||
                speculative.routes()[0].result.output != canonical.routes()[0].result.output
            {
                return Err(Error::Core(crate::Error::Diverged));
            }
            if speculative.observations() != canonical.observations() ||
                protocol_logs(speculative.result().logs(), self.chains[&id].router) !=
                    protocol_logs(canonical.result().logs(), self.chains[&id].router) ||
                speculative.result().is_success() != canonical.result().is_success()
            {
                return Err(Error::Core(crate::Error::Diverged));
            }
            included.insert(id, Included { transaction: tx, execution: canonical.into_result() });
        }
        self.verify_messages(&included, reverted)?;
        Ok(Bundle { chains: included, reverted })
    }
    // Drive a chain until its transaction finishes or a particular incoming call reports
    // its result. Other outbound calls encountered on the way recurse through this scheduler.
    // Every local callback remains an ordinary CALL in the original machine's journal.
    fn run_until_result(
        &mut self,
        id: u64,
        expected: Option<(u64, U256)>,
        bundle: B256,
        live: &mut BTreeMap<u64, Live<DB>>,
        calls: &mut usize,
        depth: usize,
    ) -> Result<Option<AtomicResultWitness>, Error<DB>> {
        loop {
            self.advance(id, live)?;
            let Progress::Paused(paused) = live[&id].progress.as_ref().unwrap() else {
                if expected.is_some() {
                    return Err(Error::Invalid("incoming call ended before acknowledgement"));
                }
                return Ok(None);
            };
            if paused.request().input.starts_with(&callFinishedCall::SELECTOR) {
                let result = callFinishedCall::abi_decode(&paused.request().input)?;
                let source: u64 =
                    result.sourceChain.try_into().map_err(|_| Error::Invalid("source overflow"))?;
                if expected != Some((source, result.sequence)) {
                    return Err(Error::Invalid("unexpected incoming call result"));
                }
                return Ok(Some(AtomicResultWitness {
                    identifier: if result.success {
                        self.last_identifier(id, paused.logs(), CallResult::SIGNATURE_HASH)?
                    } else {
                        Identifier::default()
                    },
                    success: result.success,
                    returnData: result.result,
                }));
            }
            if !paused.request().input.starts_with(&callbackAtCall::SELECTOR) {
                return Err(Error::Invalid("unexpected application yield"));
            }
            let waiting = callbackAtCall::abi_decode(&paused.request().input)?;
            let sequence: usize = waiting
                .request
                .sequence
                .try_into()
                .map_err(|_| Error::Invalid("sequence overflow"))?;
            if waiting.index != U256::ZERO ||
                sequence != live[&id].tape.witnesses.len() ||
                *calls >= usize::from(self.max_calls) ||
                depth >= 32
            {
                return Err(Error::Invalid("call depth, budget or sequence mismatch"));
            }
            let destination: u64 = waiting
                .request
                .chainId
                .try_into()
                .map_err(|_| Error::Invalid("chain ID overflow"))?;
            if destination == id || !self.chains.contains_key(&destination) {
                return Err(Error::Invalid("unsupported remote destination"));
            }
            let item = AtomicRemoteCall {
                identifier: self.last_identifier(
                    id,
                    paused.logs(),
                    CallRequested::SIGNATURE_HASH,
                )?,
                sequence: waiting.request.sequence,
                sender: waiting.request.sender,
                target: waiting.request.target,
                data: waiting.request.data,
            };
            *calls += 1;
            // Reserve the slot: a callback may issue sequence N+1 before N returns.
            live.get_mut(&id).unwrap().tape.witnesses.push(AtomicResultWitness::default());
            let witness =
                self.execute_incoming(destination, item, bundle, live, calls, depth + 1)?;
            live.get_mut(&id).unwrap().tape.witnesses[sequence] = witness;
            let source = live.get_mut(&id).unwrap();
            let progress = source.progress.take().unwrap();
            if let Progress::Paused(paused) = &progress &&
                paused.request().input.starts_with(&callbackAtCall::SELECTOR)
            {
                let cursor = callbackAtCall::abi_decode(&paused.request().input)?;
                if cursor.request.sequence != U256::from(sequence) {
                    return Err(Error::Invalid("outbound continuation changed"));
                }
                let reply = tape::quote(paused, &self.chains[&id], &source.tape)?;
                let Progress::Paused(paused) = progress else { unreachable!() };
                source.progress = Some(paused.resolve(reply)?);
            } else {
                // A failing callback can unwind this chain before its remote ancestor returns.
                // Preserve its ancestor acknowledgement or completed transaction while the
                // corresponding scheduler frame finishes resolving reserved witnesses.
                source.progress = Some(progress);
            }
        }
    }
    fn execute_incoming(
        &mut self,
        id: u64,
        item: AtomicRemoteCall,
        bundle: B256,
        live: &mut BTreeMap<u64, Live<DB>>,
        calls: &mut usize,
        depth: usize,
    ) -> Result<AtomicResultWitness, Error<DB>> {
        if let std::collections::btree_map::Entry::Vacant(entry) = live.entry(id) {
            let data: Bytes = executeRemoteNestedCall {
                bundleId: bundle,
                calls: Vec::new(),
                witnesses: Vec::new(),
                callbacks: Vec::new(),
                rootCompletion: Identifier::default(),
                applicationGas: self.chains[&id].application_gas,
                maxCalls: self.max_calls,
            }
            .abi_encode()
            .into();
            let progress = self.start(id, data.clone())?;
            entry.insert(Live {
                progress: Some(progress),
                tape: Tape::default(),
                accesses: Vec::new(),
                discovery_data: data,
            });
            self.advance(id, live)?;
        }
        let transaction = live.get_mut(&id).unwrap();
        let Progress::Paused(waiting) = transaction.progress.take().unwrap() else {
            return Err(Error::Invalid("destination already completed"));
        };
        let expected = (
            item.identifier.chainId.try_into().map_err(|_| Error::Invalid("source overflow"))?,
            item.sequence,
        );
        let callback_index = if waiting.request().input.starts_with(&remoteCallAtCall::SELECTOR) {
            let cursor = remoteCallAtCall::abi_decode(&waiting.request().input)?.cursor;
            if cursor.index != U256::from(transaction.tape.calls.len()) {
                return Err(Error::Invalid("destination cursor mismatch"));
            }
            transaction.tape.calls.push(item);
            None
        } else if waiting.request().input.starts_with(&callbackAtCall::SELECTOR) {
            let cursor = callbackAtCall::abi_decode(&waiting.request().input)?;
            let count = transaction
                .tape
                .callbacks
                .iter()
                .filter(|c| c.waitingSequence == cursor.request.sequence)
                .count();
            if cursor.index != U256::from(count) {
                return Err(Error::Invalid("callback cursor mismatch"));
            }
            let index = transaction.tape.callbacks.len();
            transaction.tape.callbacks.push(AtomicCallback {
                waitingSequence: cursor.request.sequence,
                call: item,
                success: false,
            });
            Some(index)
        } else {
            return Err(Error::Invalid("destination is not awaiting work"));
        };
        let reply = tape::quote(&waiting, &self.chains[&id], &transaction.tape)?;
        transaction.progress = Some(waiting.resolve(reply)?);
        let result = self
            .run_until_result(id, Some(expected), bundle, live, calls, depth)?
            .ok_or(Error::Invalid("missing incoming result"))?;
        let transaction = live.get_mut(&id).unwrap();
        if let Some(index) = callback_index {
            transaction.tape.callbacks[index].success = result.success;
        }
        let Progress::Paused(ack) = transaction.progress.take().unwrap() else {
            unreachable!("result acknowledgement")
        };
        transaction.progress = Some(ack.resume()?);
        self.advance(id, live)?;
        Ok(result)
    }
    fn limits(&self) -> Limits {
        Limits { calls: usize::from(self.max_calls) * 16 + 16, bytes_per_call: self.max_bytes }
    }
    fn start(&mut self, id: u64, data: Bytes) -> Result<Progress<DB>, Error<DB>> {
        let tx =
            self.envelope.prepare(id, data, access_list(Vec::new())).map_err(Error::Envelope)?;
        let chain = &self.chains[&id];
        let endpoints = [
            witnessAtCall::SELECTOR,
            remoteCallAtCall::SELECTOR,
            callbackAtCall::SELECTOR,
            callFinishedCall::SELECTOR,
            callbackStatusCall::SELECTOR,
            witnessCountCall::SELECTOR,
            completionIdentifierCall::SELECTOR,
        ]
        .into_iter()
        .map(|selector| Endpoint { address: chain.router, selector: Some(selector) })
        .chain([Endpoint { address: INBOX, selector: Some(validateMessageCall::SELECTOR) }])
        .collect();
        Ok(discover_with(
            chain.context.clone().with_tx(tx),
            endpoints,
            Some(application(chain)),
            self.limits(),
        )?)
    }
    fn advance(&self, id: u64, live: &mut BTreeMap<u64, Live<DB>>) -> Result<(), Error<DB>> {
        let chain = &self.chains[&id];
        let transaction = live.get_mut(&id).unwrap();
        loop {
            let progress = transaction.progress.take().unwrap();
            let Progress::Paused(mut paused) = progress else {
                transaction.progress = Some(progress);
                return Ok(());
            };
            let input = paused.request().input.clone();
            if paused.request().endpoint == INBOX {
                if paused.request().caller != chain.router {
                    return Err(Error::Invalid("application inbox probes are unsupported"));
                }
                let call = validateMessageCall::abi_decode(&input)?;
                let accesses = messages::access(&call.id, call.msgHash)?;
                paused.warm_slot(INBOX, messages::word(accesses[1]));
                transaction.accesses.extend(accesses);
                transaction.progress = Some(paused.resume()?);
            } else if input.starts_with(&callbackStatusCall::SELECTOR) ||
                input.starts_with(&witnessCountCall::SELECTOR) ||
                input.starts_with(&completionIdentifierCall::SELECTOR) ||
                input.starts_with(&witnessAtCall::SELECTOR)
            {
                let reply = tape::quote(&paused, chain, &transaction.tape)?;
                transaction.progress = Some(paused.resolve(reply)?);
            } else {
                transaction.progress = Some(Progress::Paused(paused));
                return Ok(());
            }
        }
    }
    fn last_identifier(
        &self,
        chain: u64,
        logs: &[Log],
        topic: B256,
    ) -> Result<Identifier, Error<DB>> {
        let config = &self.chains[&chain];
        let index = logs
            .iter()
            .rposition(|log| {
                log.address == config.router && log.data.topics().first() == Some(&topic)
            })
            .ok_or(Error::Invalid("missing initiating log"))?;
        Ok(Identifier {
            origin: config.router,
            blockNumber: config.context.block.number,
            logIndex: U256::from(config.prefix_logs) + U256::from(index),
            timestamp: config.context.block.timestamp,
            chainId: U256::from(chain),
        })
    }
    fn verify_messages(
        &self,
        included: &BTreeMap<u64, Included>,
        reverted: bool,
    ) -> Result<(), Error<DB>> {
        for (chain, included_tx) in included {
            let logs = included_tx.execution.result.logs();
            if reverted && !protocol_logs(logs, self.chains[chain].router).is_empty() {
                return Err(Error::Invalid("aborted bundle emitted protocol logs"));
            }
            for log in logs.iter().filter(|l| l.address == INBOX) {
                let message = ExecutingMessage::decode_log(log)?;
                let id = &message.data.id;
                let source: u64 =
                    id.chainId.try_into().map_err(|_| Error::Invalid("source overflow"))?;
                let source_tx =
                    included.get(&source).ok_or(Error::Invalid("missing message source"))?;
                let source_config = &self.chains[&source];
                let index: usize =
                    id.logIndex.try_into().map_err(|_| Error::Invalid("index overflow"))?;
                let source_log = index
                    .checked_sub(source_config.prefix_logs as usize)
                    .and_then(|index| source_tx.execution.result.logs().get(index))
                    .ok_or(Error::Invalid("missing source log"))?;
                if source == *chain ||
                    id.blockNumber != source_config.context.block.number ||
                    id.timestamp != source_config.context.block.timestamp ||
                    id.origin != source_log.address ||
                    messages::payload(source_log) != message.data.msgHash
                {
                    return Err(Error::Invalid("message mismatch"));
                }
            }
        }
        Ok(())
    }
}
fn access_list(storage_keys: Vec<B256>) -> AccessList {
    AccessList(vec![AccessListItem { address: INBOX, storage_keys }])
}
fn application<DB: Database>(chain: &Chain<DB>) -> Application {
    Application { router: chain.router, entry_caller: chain.sender, infrastructure: vec![INBOX] }
}
fn validate_route<DB: Database>(candidate: &Candidate, data: &Bytes) -> Result<(), Error<DB>> {
    if candidate.routes().len() != 1 || candidate.routes()[0].request.input != *data {
        return Err(Error::Invalid("missing or substituted router invocation"));
    }
    Ok(())
}
const fn candidate<DB: Database>(progress: &Progress<DB>) -> Result<&Candidate, Error<DB>> {
    match progress {
        Progress::Complete(candidate) => Ok(candidate),
        _ => Err(Error::Invalid("unfinished chain")),
    }
}
fn protocol_logs(logs: &[Log], router: Address) -> Vec<&Log> {
    logs.iter().filter(|l| l.address == router || l.address == INBOX).collect()
}
