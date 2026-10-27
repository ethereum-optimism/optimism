//! The executing messages of a span, read before witness collection.
//!
//! The scan uses the consolidation's message parser, below-head rule, and stateless checks, so it
//! finds a superset of the blocks that the consolidation checks against log commitments. It reads
//! a head that the claimed super root replaced through kona-host, not the L2 node, which may have
//! pruned it.

use std::{
    collections::{BTreeMap, BTreeSet, HashMap},
    sync::Arc,
};

use alloy_consensus::Header;
use alloy_primitives::B256;
use anyhow::{Context, Result};
use kona_interop::{
    DependencySet, ExecutingMessage, MessageGraphError, MessageRules, extract_executing_messages,
};
use kona_registry::HashMap as RegistryHashMap;
use kona_sp1_ethereum_client_utils::chain_config::ChainConfigs;
use kona_sp1_super_range_executor::{OptimisticBlockReceipts, SuperRootAtTimestampResponse};
use op_alloy_consensus::OpReceiptEnvelope;

use crate::ports::{L2BlockSource, ReplacedBlockSource};

/// A block below a local-safe head that an executing message of the span references.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ReferencedBlock {
    pub(crate) chain_id: u64,
    pub(crate) number: u64,
    pub(crate) hash: B256,
    pub(crate) receipts_root: B256,
}

/// One transition timestamp of a span.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ScannedTimestamp {
    pub(crate) timestamp: u64,
    pub(crate) chains: u64,
    /// Chains whose claimed output root differs from the optimistic one.
    pub(crate) replaced_chains: u64,
    /// Deposit gas of the replaced heads, which the deposit-only blocks execute again.
    pub(crate) replaced_deposit_gas: u64,
    /// Executing messages in the heads.
    pub(crate) messages: u64,
    /// `(chain, number)` of each distinct referenced block below its chain's head.
    pub(crate) references: BTreeSet<(u64, u64)>,
}

/// The scan of a span: its timestamps in order and the blocks they reference.
#[derive(Debug, Default)]
pub(crate) struct ScannedSpan {
    pub(crate) timestamps: Vec<ScannedTimestamp>,
    pub(crate) blocks: BTreeMap<(u64, u64), ReferencedBlock>,
}

/// A head with what the scan needs from its receipts.
#[derive(Debug)]
struct ScannedHead {
    header: Header,
    messages: Vec<ExecutingMessage>,
    deposit_gas: u64,
}

impl ScannedHead {
    fn new(header: Header, receipts: &[OpReceiptEnvelope]) -> Self {
        let deposit_gas = receipts
            .iter()
            .take_while(|receipt| matches!(receipt, OpReceiptEnvelope::Deposit(_)))
            .last()
            // Receipts give the gas after refunds, which return at most 1/5 of the gas spent.
            .map_or(0, |receipt| receipt.cumulative_gas_used() * 5 / 4);
        Self { messages: extract_executing_messages(receipts), deposit_gas, header }
    }
}

/// Scans the transitions of `responses`, the verified responses for the agreed timestamp and each
/// transition timestamp after it.
pub(crate) async fn scan_span(
    source: &dyn L2BlockSource,
    replaced_blocks: &dyn ReplacedBlockSource,
    responses: &[SuperRootAtTimestampResponse],
    configs: &ChainConfigs,
) -> Result<ScannedSpan> {
    let rollup_configs: RegistryHashMap<_, _> =
        configs.rollup_configs.clone().into_iter().collect();
    let rules =
        MessageRules::new(&rollup_configs, configs.dependency_set.get_message_expiry_window());
    let mut heads: HashMap<B256, Arc<ScannedHead>> = HashMap::new();
    let mut initiating_timestamps = BTreeMap::new();
    let mut timestamps = Vec::with_capacity(responses.len().saturating_sub(1));
    for response in responses.iter().skip(1) {
        let data = response.data.as_ref().context("verified span response lost its data")?;
        let claimed: BTreeMap<_, _> =
            data.super_v1.chains.iter().map(|chain| (chain.chain_id, chain.output)).collect();
        let mut scanned =
            ScannedTimestamp { timestamp: data.super_v1.timestamp, ..Default::default() };
        let mut chain_heads = Vec::with_capacity(response.optimistic_at_timestamp.len());
        for (chain_id, optimistic) in &response.optimistic_at_timestamp {
            let chain = u64::try_from(chain_id.0).context("chain ID does not fit in u64")?;
            let hash =
                optimistic.output.as_ref().context("optimistic output has no preimage")?.block_hash;
            let replaced = claimed.get(chain_id) != Some(&optimistic.output_root);
            let head = match heads.get(&hash) {
                Some(head) => head.clone(),
                None => {
                    let head = Arc::new(if replaced {
                        let timestamp = data.super_v1.timestamp;
                        scan_replaced_head(source, replaced_blocks, configs, chain, timestamp, hash)
                            .await?
                    } else {
                        scan_head(source, chain, hash).await?
                    });
                    heads.insert(hash, head.clone());
                    head
                }
            };
            scanned.chains += 1;
            scanned.messages += head.messages.len() as u64;
            if replaced {
                scanned.replaced_chains += 1;
                scanned.replaced_deposit_gas += head.deposit_gas;
            }
            chain_heads.push((chain, head));
        }
        let head_numbers: BTreeMap<_, _> =
            chain_heads.iter().map(|(chain, head)| (*chain, head.header.number)).collect();
        for (chain, head) in &chain_heads {
            for message in &head.messages {
                let Some((reference, timestamp)) = below_head_reference(
                    &rules,
                    &configs.dependency_set,
                    message,
                    *chain,
                    head.header.timestamp,
                    &head_numbers,
                ) else {
                    continue;
                };
                scanned.references.insert(reference);
                initiating_timestamps.insert(reference, timestamp);
            }
        }
        timestamps.push(scanned);
    }

    let mut blocks = BTreeMap::new();
    for (&(chain_id, number), &timestamp) in &initiating_timestamps {
        let header = source.header_by_number(chain_id, number).await?;
        // The run rejects the message at the header's timestamp check and reads no log.
        if header.timestamp != timestamp {
            continue;
        }
        let block = ReferencedBlock {
            chain_id,
            number,
            hash: header.hash(),
            receipts_root: header.receipts_root,
        };
        blocks.insert((chain_id, number), block);
    }
    for scanned in &mut timestamps {
        scanned.references.retain(|reference| blocks.contains_key(reference));
    }
    Ok(ScannedSpan { timestamps, blocks })
}

async fn scan_head(source: &dyn L2BlockSource, chain_id: u64, hash: B256) -> Result<ScannedHead> {
    let header = source.header_by_hash(chain_id, hash).await?;
    let receipts = source.receipts(chain_id, hash, header.receipts_root).await?;
    Ok(ScannedHead::new(header, &receipts))
}

/// A head that the claimed super root replaced, read the way the consolidation reads it: through
/// kona-host, which re-derives and re-executes it on its canonical parent. The supernode reports
/// a denied block only at its own timestamp, so it is the block at `timestamp`.
async fn scan_replaced_head(
    source: &dyn L2BlockSource,
    replaced_blocks: &dyn ReplacedBlockSource,
    configs: &ChainConfigs,
    chain_id: u64,
    timestamp: u64,
    hash: B256,
) -> Result<ScannedHead> {
    let number = configs
        .rollup_configs
        .get(&chain_id)
        .with_context(|| format!("no rollup config for chain {chain_id}"))?
        .block_number_from_timestamp(timestamp);
    let parent_number = number
        .checked_sub(1)
        .with_context(|| format!("replaced block {hash} of chain {chain_id} has no parent"))?;
    let parent = source.header_by_number(chain_id, parent_number).await?;
    let OptimisticBlockReceipts { header, receipts } =
        replaced_blocks.replaced_block(chain_id, parent.hash(), hash).await.with_context(|| {
            format!("failed to re-derive replaced block {hash} of chain {chain_id}")
        })?;
    Ok(ScannedHead::new(header, &receipts))
}

/// The `(chain, number)` of the block below a head that the run reads for `message`, with the
/// message's initiating timestamp. `None` when the run rejects the message by a stateless rule
/// (in `MessageGraph::check_single_dependency` order) or reads a head.
fn below_head_reference(
    rules: &MessageRules<'_>,
    dependency_set: &DependencySet,
    message: &ExecutingMessage,
    executing_chain: u64,
    executing_timestamp: u64,
    head_numbers: &BTreeMap<u64, u64>,
) -> Option<((u64, u64), u64)> {
    type Rejected = MessageGraphError<()>;
    let identifier = &message.identifier;
    let initiating_chain: u64 = identifier.chainId.saturating_to();
    let initiating_timestamp: u64 = identifier.timestamp.saturating_to();
    if !dependency_set.dependencies.contains_key(&executing_chain) ||
        !dependency_set.dependencies.contains_key(&initiating_chain)
    {
        return None;
    }
    let executing_config = rules.rollup_config_for::<()>(executing_chain).ok()?;
    MessageRules::check_executing_activation::<()>(executing_config, executing_timestamp).ok()?;
    let initiating_config = rules.rollup_config_for::<()>(initiating_chain).ok()?;
    let checks: [Result<(), Rejected>; 4] = [
        MessageRules::check_message_ordering(initiating_timestamp, executing_timestamp),
        MessageRules::check_initiating_activation(initiating_config, initiating_timestamp),
        rules.check_message_expiry(initiating_timestamp, executing_timestamp),
        MessageRules::check_initiating_block_number(
            initiating_config,
            initiating_timestamp,
            identifier.blockNumber,
        ),
    ];
    checks.into_iter().collect::<Result<(), _>>().ok()?;
    let number: u64 = identifier.blockNumber.saturating_to();
    (number < *head_numbers.get(&initiating_chain)?)
        .then_some(((initiating_chain, number), initiating_timestamp))
}

#[cfg(test)]
mod tests {
    use std::sync::{
        Mutex,
        atomic::{AtomicUsize, Ordering},
    };

    use alloy_consensus::{Sealed, proofs::ordered_trie_root_with_encoder};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, U256};
    use async_trait::async_trait;
    use kona_genesis::RollupConfig;
    use kona_interop::{ChainBuilder, ChainDependency, ExecutingMessageBuilder};
    use kona_sp1_super_range_executor::{
        BlockId, ChainId, ChainIdAndOutput, OutputV0, OutputWithRequiredL1, SuperRootResponseData,
        SuperV1,
    };

    use super::*;

    /// Chain A has 1 s blocks and chain B 4 s blocks, both from genesis time 0 and interop from
    /// genesis.
    const A: u64 = 1_000_001;
    const B: u64 = 1_000_002;

    /// The span's transition timestamps are 12 to 15. A's head at each is the block of that
    /// number; B's head is block 3 (timestamp 12) at all four.
    const SPAN: [u64; 5] = [11, 12, 13, 14, 15];

    #[derive(Default)]
    struct FakeL2 {
        headers: BTreeMap<(u64, B256), Header>,
        by_number: BTreeMap<(u64, u64), Header>,
        receipts: BTreeMap<B256, Vec<OpReceiptEnvelope>>,
        hash_reads: Mutex<Vec<(u64, B256)>>,
        number_reads: Mutex<Vec<(u64, u64)>>,
        receipt_reads: Mutex<Vec<(u64, B256)>>,
    }

    /// The header of block `number` at `timestamp` with `receipts`.
    fn header(number: u64, timestamp: u64, receipts: &[OpReceiptEnvelope]) -> Header {
        let receipts_root =
            ordered_trie_root_with_encoder(receipts, |receipt, out| receipt.encode_2718(out));
        Header { number, timestamp, receipts_root, gas_used: number, ..Default::default() }
    }

    impl FakeL2 {
        /// Adds block `number` of `chain` at `timestamp` with `receipts`, and returns its hash.
        fn block(
            &mut self,
            chain: u64,
            number: u64,
            timestamp: u64,
            receipts: Vec<OpReceiptEnvelope>,
        ) -> B256 {
            let header = header(number, timestamp, &receipts);
            let hash = header.hash_slow();
            self.headers.insert((chain, hash), header.clone());
            self.by_number.insert((chain, number), header);
            self.receipts.insert(hash, receipts);
            hash
        }
    }

    /// A replaced block and the parent it must be asked for.
    struct Replaced {
        parent: B256,
        block: OptimisticBlockReceipts,
    }

    /// Replaced blocks by chain and hash.
    #[derive(Default)]
    struct FakeReplaced {
        blocks: BTreeMap<(u64, B256), Replaced>,
        reads: AtomicUsize,
    }

    #[async_trait]
    impl ReplacedBlockSource for FakeReplaced {
        async fn replaced_block(
            &self,
            chain_id: u64,
            parent: B256,
            hash: B256,
        ) -> Result<OptimisticBlockReceipts> {
            self.reads.fetch_add(1, Ordering::Relaxed);
            match self.blocks.get(&(chain_id, hash)) {
                Some(replaced) if replaced.parent == parent => Ok(replaced.block.clone()),
                _ => anyhow::bail!("unknown replaced block"),
            }
        }
    }

    #[async_trait]
    impl L2BlockSource for FakeL2 {
        async fn header_by_hash(&self, chain_id: u64, hash: B256) -> Result<Header> {
            self.hash_reads.lock().unwrap().push((chain_id, hash));
            self.headers.get(&(chain_id, hash)).cloned().context("unknown header")
        }

        async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Sealed<Header>> {
            self.number_reads.lock().unwrap().push((chain_id, number));
            let header = self.by_number.get(&(chain_id, number)).context("unknown number")?;
            Ok(Sealed::new_unchecked(header.clone(), header.hash_slow()))
        }

        async fn receipts(
            &self,
            chain_id: u64,
            hash: B256,
            _receipts_root: B256,
        ) -> Result<Vec<OpReceiptEnvelope>> {
            self.receipt_reads.lock().unwrap().push((chain_id, hash));
            self.receipts.get(&hash).cloned().context("unknown receipts")
        }
    }

    fn configs() -> ChainConfigs {
        let config = |chain_id: u64, block_time: u64| {
            let mut config = RollupConfig { block_time, ..Default::default() };
            config.l2_chain_id = chain_id.into();
            config.hardforks.lagoon_time = Some(0);
            (chain_id, config)
        };
        ChainConfigs {
            dependency_set: DependencySet {
                dependencies: [(A, ChainDependency {}), (B, ChainDependency {})].into(),
                override_message_expiry_window: None,
            },
            rollup_configs: [config(A, 1), config(B, 4)].into(),
            l1_config: kona_registry::L1_CONFIGS[&1].clone(),
        }
    }

    /// A message that B's block 3 executes into a log of `chain` block `number` at `timestamp`.
    fn message(chain: u64, number: u64, timestamp: u64) -> ExecutingMessageBuilder {
        ExecutingMessageBuilder::default()
            .with_origin_chain_id(chain)
            .with_origin_block_number(number)
            .with_origin_timestamp(timestamp)
            .with_origin_address(Address::repeat_byte(1))
    }

    /// The span's responses over the fake chains, with B's head executing `messages`.
    fn span(
        l2: &mut FakeL2,
        messages: Vec<ExecutingMessageBuilder>,
    ) -> Vec<SuperRootAtTimestampResponse> {
        for number in 9..=15 {
            l2.block(A, number, number, Vec::new());
        }
        let mut executing = ChainBuilder::default();
        for message in messages {
            executing.add_executing_message(message);
        }
        let b3 = l2.block(B, 3, 12, executing.receipts);
        let b2 = l2.block(B, 2, 8, Vec::new());
        SPAN.iter()
            .map(|&timestamp| {
                let a = l2.by_number[&(A, timestamp)].hash_slow();
                let b = if timestamp < 12 { b2 } else { b3 };
                response(timestamp, &[(A, a), (B, b)])
            })
            .collect()
    }

    fn response(timestamp: u64, heads: &[(u64, B256)]) -> SuperRootAtTimestampResponse {
        let output = |block_hash| OutputWithRequiredL1 {
            output: Some(OutputV0 {
                state_root: B256::ZERO,
                message_passer_storage_root: B256::ZERO,
                block_hash,
            }),
            output_root: block_hash,
            required_l1: BlockId::default(),
        };
        SuperRootAtTimestampResponse {
            current_l1: BlockId::default(),
            current_safe_timestamp: timestamp,
            current_local_safe_timestamp: timestamp,
            current_finalized_timestamp: timestamp,
            optimistic_at_timestamp: heads
                .iter()
                .map(|(chain, hash)| (ChainId(U256::from(*chain)), output(*hash)))
                .collect(),
            chain_ids: heads.iter().map(|(chain, _)| ChainId(U256::from(*chain))).collect(),
            data: Some(SuperRootResponseData {
                verified_required_l1: BlockId::default(),
                super_v1: SuperV1 {
                    timestamp,
                    chains: heads
                        .iter()
                        .map(|(chain, hash)| ChainIdAndOutput {
                            chain_id: ChainId(U256::from(*chain)),
                            output: *hash,
                        })
                        .collect(),
                },
                super_root: B256::ZERO,
            }),
        }
    }

    /// Pins references below the initiating chain's head at each timestamp, and none to a head.
    #[tokio::test]
    async fn scan_keeps_only_below_head_references() {
        let mut l2 = FakeL2::default();
        let responses =
            span(&mut l2, vec![message(A, 10, 10), message(A, 12, 12), message(A, 10, 10)]);
        let scan = scan_span(&l2, &FakeReplaced::default(), &responses, &configs()).await.unwrap();
        let references: Vec<Vec<_>> =
            scan.timestamps.iter().map(|t| t.references.iter().copied().collect()).collect();
        let both = vec![(A, 10), (A, 12)];
        assert_eq!(references, [vec![(A, 10)], both.clone(), both.clone(), both]);
        assert_eq!(scan.blocks.keys().copied().collect::<Vec<_>>(), [(A, 10), (A, 12)]);
        assert_eq!(scan.blocks[&(A, 12)].hash, l2.by_number[&(A, 12)].hash_slow());
        assert!(scan.timestamps.iter().all(|t| t.messages == 3 && t.chains == 2));
    }

    /// Pins no reference for a message that the run rejects by a stateless rule.
    #[tokio::test]
    async fn scan_drops_references_that_fail_stateless_rules() {
        type Edit = fn(&mut ChainConfigs, &mut FakeL2);
        let unchanged: Edit = |_, _| {};
        let cases: [(&str, ExecutingMessageBuilder, Edit); 7] = [
            ("chain outside the dependency set", message(A, 10, 10), |configs, _| {
                configs.dependency_set.dependencies.remove(&A);
            }),
            ("executing chain activation", message(A, 10, 10), |configs, _| {
                configs.rollup_configs.get_mut(&B).unwrap().hardforks.lagoon_time = Some(12);
            }),
            ("initiating chain activation", message(A, 10, 10), |configs, _| {
                configs.rollup_configs.get_mut(&A).unwrap().hardforks.lagoon_time = Some(10);
            }),
            // Block 13 is below A's heads at timestamps 14 and 15.
            ("timestamp order", message(A, 13, 13), unchanged),
            ("expiry", message(A, 10, 10), |configs, _| {
                configs.dependency_set.override_message_expiry_window = Some(1.try_into().unwrap());
            }),
            ("block number off the grid", message(A, 9, 10), unchanged),
            ("header timestamp", message(A, 10, 10), |_, l2| {
                l2.by_number.get_mut(&(A, 10)).unwrap().timestamp = 9;
            }),
        ];
        for (name, message, edit) in cases {
            let mut l2 = FakeL2::default();
            let responses = span(&mut l2, vec![message]);
            let mut configs = configs();
            edit(&mut configs, &mut l2);
            let scan =
                scan_span(&l2, &FakeReplaced::default(), &responses, &configs).await.unwrap();
            assert!(scan.blocks.is_empty(), "{name}: {:?}", scan.blocks);
            assert!(scan.timestamps.iter().all(|t| t.references.is_empty()), "{name}");
        }
    }

    /// The span of `span`, except that B's block at timestamp 12 is denied: it executes a message
    /// into A's block 10, and the claimed super root replaces it with a block without messages.
    /// The node holds only the replacement, and `replaced` holds the denied block. Returns the
    /// responses and the denied block's hash.
    fn replaced_span(
        l2: &mut FakeL2,
        replaced: &mut FakeReplaced,
    ) -> (Vec<SuperRootAtTimestampResponse>, B256) {
        for number in 9..=15 {
            l2.block(A, number, number, Vec::new());
        }
        let b2 = l2.block(B, 2, 8, Vec::new());
        let replacement = l2.block(B, 3, 12, Vec::new());
        let mut executing = ChainBuilder::default();
        executing.add_executing_message(message(A, 10, 10));
        let denied_header = header(3, 12, &executing.receipts);
        let denied = denied_header.hash_slow();
        let block = OptimisticBlockReceipts { header: denied_header, receipts: executing.receipts };
        replaced.blocks.insert((B, denied), Replaced { parent: b2, block });
        let responses = SPAN
            .iter()
            .map(|&timestamp| {
                let a = l2.by_number[&(A, timestamp)].hash_slow();
                let b = match timestamp {
                    ..12 => b2,
                    12 => denied,
                    _ => replacement,
                };
                let mut response = response(timestamp, &[(A, a), (B, b)]);
                if timestamp == 12 {
                    let data = response.data.as_mut().unwrap();
                    data.super_v1.chains[1].output = replacement;
                }
                response
            })
            .collect();
        (responses, denied)
    }

    /// Pins a replaced head read without the node, which may have pruned it, and the references of
    /// its messages, which the consolidation checks before it replaces the block.
    #[tokio::test]
    async fn scan_reads_a_replaced_head_without_the_node() {
        let mut l2 = FakeL2::default();
        let mut replaced = FakeReplaced::default();
        let (responses, _) = replaced_span(&mut l2, &mut replaced);
        let scan = scan_span(&l2, &replaced, &responses, &configs()).await.unwrap();
        let per_timestamp = |field: fn(&ScannedTimestamp) -> u64| -> Vec<u64> {
            scan.timestamps.iter().map(field).collect()
        };
        assert_eq!(per_timestamp(|t| t.replaced_chains), [1, 0, 0, 0]);
        assert_eq!(per_timestamp(|t| t.messages), [1, 0, 0, 0]);
        let references: Vec<Vec<_>> =
            scan.timestamps.iter().map(|t| t.references.iter().copied().collect()).collect();
        assert_eq!(references, [vec![(A, 10)], vec![], vec![], vec![]]);
        assert_eq!(scan.blocks.keys().copied().collect::<Vec<_>>(), [(A, 10)]);
    }

    #[tokio::test]
    async fn scan_retains_persisting_replaced_head_references_at_each_timestamp() {
        for agreed in 0..4 {
            let mut l2 = FakeL2::default();
            let mut replaced = FakeReplaced::default();
            let (mut responses, denied) = replaced_span(&mut l2, &mut replaced);
            let optimistic = responses[1].optimistic_at_timestamp[&ChainId(U256::from(B))].clone();
            for response in responses.iter_mut().skip(2) {
                response.optimistic_at_timestamp.insert(ChainId(U256::from(B)), optimistic.clone());
            }
            let scan = scan_span(&l2, &replaced, &responses[agreed..], &configs()).await.unwrap();
            assert_eq!(
                scan.timestamps.iter().map(|step| step.timestamp).collect::<Vec<_>>(),
                SPAN[agreed + 1..],
            );
            for step in &scan.timestamps {
                assert_eq!(step.replaced_chains, 1);
                assert_eq!(step.messages, 1);
                assert_eq!(step.references, BTreeSet::from([(A, 10)]));
            }
            let initiating = &scan.blocks[&(A, 10)];
            assert_eq!(scan.blocks.len(), 1);
            assert_eq!(initiating.hash, l2.by_number[&(A, 10)].hash_slow());
            assert_eq!(initiating.receipts_root, l2.by_number[&(A, 10)].receipts_root);
            assert_eq!(replaced.reads.load(Ordering::Relaxed), 1);
            assert_eq!(*l2.number_reads.lock().unwrap(), [(B, 2), (A, 10)]);
            assert!(
                l2.hash_reads
                    .lock()
                    .unwrap()
                    .iter()
                    .all(|(chain, hash)| *chain == A && *hash != denied)
            );
            assert!(l2.receipt_reads.lock().unwrap().iter().all(|(chain, _)| *chain == A));
        }
    }

    /// Pins an error, not a scan without the replaced head's references, when the replaced-block
    /// source fails. kona-host waits on a block it cannot serve rather than failing, so with it
    /// this covers the errors it does return, such as a host that fails to start or a malformed
    /// header or receipts trie.
    #[tokio::test]
    async fn scan_fails_when_a_replaced_head_cannot_be_read() {
        let mut l2 = FakeL2::default();
        let (responses, denied) = replaced_span(&mut l2, &mut FakeReplaced::default());
        let err = scan_span(&l2, &FakeReplaced::default(), &responses, &configs())
            .await
            .expect_err("a replaced head that cannot be read must fail the scan");
        let err = format!("{err:#}");
        assert!(err.contains("failed to re-derive replaced block"), "{err}");
        assert!(err.contains(&denied.to_string()), "{err}");
    }
}
