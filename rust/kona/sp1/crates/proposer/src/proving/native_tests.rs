//! Native synthesis, derivation, consolidation, and bisection over multi-second chains.

use std::io::Write;

use alloy_consensus::{
    Header, Sealed, SignableTransaction, TxEip1559, TxEnvelope, transaction::SignerRecoverable,
};
use alloy_eips::Encodable2718;
use alloy_op_evm::{OpEvmFactory, block::OpAlloyReceiptBuilder};
use alloy_primitives::{Address, Bytes, Signature, address, keccak256};
use alloy_trie::{EMPTY_ROOT_HASH, Nibbles, TrieAccount};
use kona_executor::StatelessL2Builder;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_interop::{ChainBuilder, ChainDependency, DependencySet, ExecutingMessageBuilder};
use kona_mpt::{NoopTrieProvider, TrieNode};
use kona_proof::l2::OracleL2ChainProvider;
use kona_protocol::{Batch, Frame, L1BlockInfoTx, Predeploys, SingleBatch};
use kona_sp1_client_utils::{
    log_commitment::{
        VerifiedLogCommitments,
        test_utils::{save, save_receipts},
    },
    super_root::hash_super_root_proof,
    witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_ethereum_client_utils::{
    super_consolidation::build_consolidation_outputs,
    super_range::{build_range_outputs, fetch_output_block_hash},
};
use kona_sp1_super_range_executor::{
    BlockId, ChainId, ChainIdAndOutput, OutputV0, OutputWithRequiredL1, SuperRootResponseData,
    SuperV1,
};
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::OpPayloadAttributes;

use super::*;

const GENESIS: u64 = 100;
const CHAIN: u64 = 1_000_001;

struct Fixture {
    configs: ChainConfigs,
    witness: PreimageStore,
    responses: Vec<SuperRootAtTimestampResponse>,
    l1_head: B256,
}

fn save_node(oracle: &mut PreimageStore, node: &TrieNode) {
    save(oracle, alloy_rlp::encode(node));
    match node {
        TrieNode::Extension { node, .. } => save_node(oracle, node),
        TrieNode::Branch { stack } => {
            for child in stack {
                save_node(oracle, child);
            }
        }
        _ => {}
    }
}

fn save_state(oracle: &mut PreimageStore, nonce: u64, accounts: &[(Address, TrieAccount)]) -> B256 {
    let mut root = TrieNode::Empty;
    for (address, nonce) in [
        (Predeploys::L2_TO_L1_MESSAGE_PASSER, 1),
        (address!("deaddeaddeaddeaddeaddeaddeaddeaddead0001"), nonce),
    ] {
        if nonce != 0 {
            root.insert(
                &Nibbles::unpack(keccak256(address)),
                alloy_rlp::encode(TrieAccount { nonce, ..Default::default() }).into(),
                &NoopTrieProvider,
            )
            .unwrap();
        }
    }
    for (address, account) in accounts {
        root.insert(
            &Nibbles::unpack(keccak256(address)),
            alloy_rlp::encode(account).into(),
            &NoopTrieProvider,
        )
        .unwrap();
    }
    save_node(oracle, &root);
    root.blind()
}

fn save_transactions(oracle: &mut PreimageStore, txs: &[Bytes]) -> B256 {
    let mut trie = kona_mpt::ordered_trie_with_encoder(txs, |tx, out| out.put_slice(tx));
    let root = trie.root();
    for node in trie.take_proof_nodes().into_inner().into_values() {
        save(oracle, node.to_vec());
    }
    root
}

fn output(oracle: &mut PreimageStore, header: &Header) -> OutputWithRequiredL1 {
    let output = OutputV0 {
        state_root: header.state_root,
        message_passer_storage_root: EMPTY_ROOT_HASH,
        block_hash: header.hash_slow(),
    };
    let mut preimage = vec![0; 128];
    preimage[32..64].copy_from_slice(output.state_root.as_slice());
    preimage[64..96].copy_from_slice(output.message_passer_storage_root.as_slice());
    preimage[96..128].copy_from_slice(output.block_hash.as_slice());
    OutputWithRequiredL1 {
        output: Some(output),
        output_root: save(oracle, preimage),
        required_l1: BlockId::default(),
    }
}

fn invalid_executing_transaction(chain_id: u64) -> (Address, Bytes, Vec<u8>) {
    let mut chain = ChainBuilder::default();
    chain.add_executing_message(ExecutingMessageBuilder::default().with_origin_chain_id(u64::MAX));
    let log = &chain.receipts[0].logs()[0];
    // Copy the identifier calldata to memory and emit the executing-message event.
    let mut code = vec![0x36, 0x60, 0, 0x60, 0, 0x37];
    for topic in log.topics().iter().rev() {
        code.push(0x7f);
        code.extend_from_slice(topic.as_slice());
    }
    code.extend_from_slice(&[0x60, 160, 0x60, 0, 0xa2, 0]);
    let tx = TxEip1559 {
        chain_id,
        nonce: 0,
        gas_limit: 200_000,
        max_fee_per_gas: 1,
        to: Predeploys::CROSS_L2_INBOX.into(),
        input: log.data.data.clone(),
        ..Default::default()
    }
    .into_signed(Signature::test_signature());
    let sender = tx.recover_signer().unwrap();
    (sender, OpTxEnvelope::Eip1559(tx).encoded_2718().into(), code)
}

fn batch_transaction(config: &RollupConfig, parent: B256, tx: Bytes) -> TxEnvelope {
    let mut batch = Vec::new();
    Batch::Single(SingleBatch {
        parent_hash: parent,
        epoch_num: config.genesis.l1.number,
        epoch_hash: config.genesis.l1.hash,
        timestamp: GENESIS + config.block_time,
        transactions: vec![tx],
    })
    .encode(&mut batch)
    .unwrap();
    let rlp = alloy_rlp::encode(Bytes::from(batch));
    let mut compressed = flate2::write::ZlibEncoder::new(Vec::new(), flate2::Compression::fast());
    compressed.write_all(&rlp).unwrap();
    let frame = Frame::new([0; 16], 0, compressed.finish().unwrap(), true);
    let mut calldata = vec![kona_protocol::DERIVATION_VERSION_0];
    calldata.extend(frame.encode());
    TxEip1559 {
        chain_id: 1,
        to: config.batch_inbox_address.into(),
        gas_limit: 500_000,
        max_fee_per_gas: 1,
        input: calldata.into(),
        ..Default::default()
    }
    .into_signed(Signature::test_signature())
    .into()
}

impl Fixture {
    fn new(periods: &[u64]) -> Self {
        Self::build(periods, false)
    }

    fn with_replacement(periods: &[u64]) -> Self {
        Self::build(periods, true)
    }

    #[allow(clippy::zero_sized_map_values)]
    fn build(periods: &[u64], replace: bool) -> Self {
        let end = GENESIS + 2 * periods.iter().max().unwrap() + 1;
        let mut witness = PreimageStore::default();
        save(&mut witness, vec![0x80]);
        let l1_genesis = Header {
            number: 1,
            timestamp: GENESIS,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            base_fee_per_gas: Some(1),
            ..Default::default()
        };
        let l1_genesis_hash = save(&mut witness, alloy_rlp::encode(&l1_genesis));
        let mut configs = ChainConfigs {
            dependency_set: DependencySet {
                dependencies: BTreeMap::new(),
                override_message_expiry_window: None,
            },
            rollup_configs: BTreeMap::new(),
            l1_config: kona_registry::L1_CONFIGS[&1].clone(),
        };
        let mut chains = Vec::new();
        let mut l1_transactions = Vec::new();
        for (index, &period) in periods.iter().enumerate() {
            let chain_id = CHAIN + index as u64;
            let invalid = replace.then(|| invalid_executing_transaction(chain_id));
            let mut accounts = Vec::new();
            if let Some((sender, _, code)) = &invalid {
                accounts.push((
                    *sender,
                    TrieAccount {
                        balance: U256::from(1_000_000_000_000_000_000u64),
                        ..Default::default()
                    },
                ));
                accounts.push((
                    Predeploys::CROSS_L2_INBOX,
                    TrieAccount {
                        nonce: 1,
                        code_hash: save(&mut witness, code.clone()),
                        ..Default::default()
                    },
                ));
            }
            let genesis = Header {
                timestamp: GENESIS,
                state_root: save_state(&mut witness, 0, &accounts),
                transactions_root: EMPTY_ROOT_HASH,
                receipts_root: EMPTY_ROOT_HASH,
                gas_limit: 30_000_000,
                base_fee_per_gas: Some(1),
                ..Default::default()
            };
            let genesis_hash = save(&mut witness, alloy_rlp::encode(&genesis));
            let mut config = RollupConfig {
                block_time: period,
                seq_window_size: 1,
                max_sequencer_drift: 600,
                l1_chain_id: 1,
                l2_chain_id: chain_id.into(),
                batch_inbox_address: Address::with_last_byte(index as u8 + 1),
                ..Default::default()
            };
            config.genesis.l1 = alloy_eips::BlockNumHash { number: 1, hash: l1_genesis_hash };
            config.genesis.l2 = alloy_eips::BlockNumHash { number: 0, hash: genesis_hash };
            config.genesis.l2_time = GENESIS;
            let batcher_address = if let Some((_, tx, _)) = &invalid {
                let batch = batch_transaction(&config, genesis_hash, tx.clone());
                let sender = batch.recover_signer().unwrap();
                l1_transactions.push(batch.encoded_2718().into());
                sender
            } else {
                Address::ZERO
            };
            config.genesis.system_config = Some(SystemConfig {
                gas_limit: genesis.gas_limit,
                batcher_address,
                ..Default::default()
            });
            let mut headers = vec![genesis];
            let mut original = None;
            for number in 1..=(end - GENESIS) / period {
                let timestamp = GENESIS + period * number;
                let (_, tx) = L1BlockInfoTx::try_new_with_deposit_tx(
                    &config,
                    &configs.l1_config,
                    &config.genesis.system_config.unwrap(),
                    number,
                    &l1_genesis,
                    timestamp,
                )
                .unwrap();
                let tx = tx.encoded_2718();
                let mut attrs = OpPayloadAttributes::default();
                attrs.payload_attributes.timestamp = timestamp;
                attrs.payload_attributes.prev_randao = l1_genesis.mix_hash;
                attrs.payload_attributes.suggested_fee_recipient = Predeploys::SEQUENCER_FEE_VAULT;
                attrs.transactions = Some(vec![tx.clone().into()]);
                attrs.no_tx_pool = Some(true);
                attrs.gas_limit = Some(headers[0].gas_limit);
                let provider = OracleL2ChainProvider::new(
                    headers.last().unwrap().hash_slow(),
                    Arc::new(config.clone()),
                    Arc::new(witness.clone()),
                );
                let mut builder = StatelessL2Builder::new(
                    &config,
                    OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                    OpAlloyReceiptBuilder::default(),
                    provider.clone(),
                    provider,
                    Sealed::new(headers.last().unwrap().clone()),
                );
                if number == 1 &&
                    let Some((_, invalid_tx, _)) = &invalid
                {
                    let mut original_attrs = attrs.clone();
                    original_attrs.transactions.as_mut().unwrap().push(invalid_tx.clone());
                    let outcome = builder.build_block(original_attrs).unwrap();
                    assert!(
                        outcome
                            .execution_result
                            .receipts
                            .iter()
                            .any(|receipt| !receipt.logs().is_empty())
                    );
                    save_receipts(&mut witness, &outcome.execution_result.receipts);
                    save_transactions(&mut witness, &[Bytes::from(tx.clone()), invalid_tx.clone()]);
                    save(&mut witness, alloy_rlp::encode(outcome.header.inner()));
                    original = Some(output(&mut witness, outcome.header.inner()));
                    let provider = OracleL2ChainProvider::new(
                        headers.last().unwrap().hash_slow(),
                        Arc::new(config.clone()),
                        Arc::new(witness.clone()),
                    );
                    builder = StatelessL2Builder::new(
                        &config,
                        OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                        OpAlloyReceiptBuilder::default(),
                        provider.clone(),
                        provider,
                        Sealed::new(headers.last().unwrap().clone()),
                    );
                }
                let outcome = builder.build_block(attrs).unwrap();
                assert_eq!(outcome.header.state_root, save_state(&mut witness, number, &accounts));
                save_receipts(&mut witness, &outcome.execution_result.receipts);
                save_transactions(&mut witness, &[tx.into()]);
                save(&mut witness, alloy_rlp::encode(outcome.header.inner()));
                headers.push(outcome.header.into_inner());
            }
            let outputs =
                headers.iter().map(|header| output(&mut witness, header)).collect::<Vec<_>>();
            chains.push((chain_id, period, outputs, original));
            configs.dependency_set.dependencies.insert(chain_id, ChainDependency {});
            configs.rollup_configs.insert(chain_id, config);
        }
        let l1_transactions_root = save_transactions(&mut witness, &l1_transactions);
        let l1_head = save(
            &mut witness,
            alloy_rlp::encode(&Header {
                parent_hash: l1_genesis_hash,
                number: 2,
                timestamp: GENESIS + 12,
                transactions_root: l1_transactions_root,
                receipts_root: EMPTY_ROOT_HASH,
                ..Default::default()
            }),
        );
        for (_, _, outputs, original) in &mut chains {
            for output in outputs.iter_mut().chain(original.iter_mut()) {
                output.required_l1 = BlockId { hash: l1_head, number: 2 };
            }
        }
        let responses = (GENESIS..=end)
            .map(|timestamp| {
                let optimistic_at_timestamp: BTreeMap<_, _> = chains
                    .iter()
                    .map(|(chain, period, outputs, original)| {
                        let output = if timestamp == GENESIS + period {
                            original.as_ref().unwrap_or(&outputs[1])
                        } else {
                            &outputs[((timestamp - GENESIS) / period) as usize]
                        };
                        (ChainId(U256::from(*chain)), output.clone())
                    })
                    .collect();
                let super_v1 = SuperV1 {
                    timestamp,
                    chains: chains
                        .iter()
                        .map(|(chain, period, outputs, _)| ChainIdAndOutput {
                            chain_id: ChainId(U256::from(*chain)),
                            output: outputs[((timestamp - GENESIS) / period) as usize].output_root,
                        })
                        .collect(),
                };
                let proof = proof_from_super_v1(&super_v1).unwrap();
                let super_root = hash_super_root_proof(&proof).unwrap();
                save(
                    &mut witness,
                    kona_sp1_client_utils::super_root::encode_super_root_proof(&proof).unwrap(),
                );
                SuperRootAtTimestampResponse {
                    current_l1: BlockId { hash: l1_head, number: 3 },
                    current_safe_timestamp: timestamp,
                    current_local_safe_timestamp: timestamp,
                    current_finalized_timestamp: timestamp,
                    chain_ids: optimistic_at_timestamp.keys().copied().collect(),
                    optimistic_at_timestamp,
                    data: Some(SuperRootResponseData {
                        verified_required_l1: BlockId { hash: l1_head, number: 2 },
                        super_v1,
                        super_root,
                    }),
                }
            })
            .collect();
        Self { configs, witness, responses, l1_head }
    }

    fn end(&self) -> u64 {
        self.responses.last().unwrap().data.as_ref().unwrap().super_v1.timestamp
    }

    fn synthesized(&self, span: ChunkSpan) -> SynthesizedExecution {
        synthesize_execution(
            TimestampSpan::new(span.agreed + 1, span.claimed).unwrap(),
            self.l1_head,
            2,
            &self.responses[(span.agreed - GENESIS) as usize..=(span.claimed - GENESIS) as usize],
        )
        .unwrap()
    }

    fn stdin(&self, span: ChunkSpan) -> Result<SP1Stdin> {
        let synthesized = self.synthesized(span);
        build_super_range_stdin(
            &synthesized.range_inputs,
            DefaultWitnessData::from_parts(self.witness.clone(), BlobData::default()),
        )
    }

    async fn replay(&self, span: ChunkSpan) -> Result<SpanResult> {
        let synthesized = self.synthesized(span);
        let witness = DefaultWitnessData::from_parts(self.witness.clone(), BlobData::default());
        let (oracle, beacon) = witness.get_oracle_and_blob_provider().await?;
        let range_outputs = build_range_outputs(
            synthesized.range_inputs,
            oracle.clone(),
            beacon,
            Some(&self.configs),
        )
        .await?;
        let log_commitments = VerifiedLogCommitments::new(Default::default(), &[0; 8], |_, _| {});
        let consolidation_outputs = build_consolidation_outputs(
            synthesized.consolidation_inputs,
            oracle,
            Some(&self.configs),
            Arc::new(log_commitments),
        )
        .await?;
        Ok(SpanResult { range_outputs, consolidation_outputs, proofs: None })
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn every_multisecond_prefix_replays_range_and_consolidation() {
    for periods in [&[2][..], &[3], &[4], &[2, 3]] {
        let fixture = Fixture::new(periods);
        for end in GENESIS + 1..=fixture.end() {
            let outputs = fixture
                .replay(ChunkSpan { agreed: GENESIS, claimed: end })
                .await
                .unwrap_or_else(|err| panic!("periods {periods:?}, endpoint {end}: {err:#}"));
            assert_eq!(
                outputs.range_outputs.transitions.len(),
                (end - GENESIS) as usize * periods.len()
            );
            assert_eq!(outputs.consolidation_outputs.transitions.len(), (end - GENESIS) as usize);
            for transition in &outputs.range_outputs.transitions {
                let chain: u64 = transition.optimistic_block.chain_id.to();
                let period = periods[(chain - CHAIN) as usize];
                let response = &fixture.responses[(transition.timestamp - GENESIS) as usize];
                let expected = &response.optimistic_at_timestamp[&ChainId(U256::from(chain))];
                assert_eq!(transition.optimistic_block.output_root, expected.output_root);
                assert_eq!(
                    transition.optimistic_block.block_hash,
                    expected.output.as_ref().unwrap().block_hash
                );
                if !(transition.timestamp - GENESIS).is_multiple_of(period) {
                    let previous = &fixture.responses
                        [(transition.timestamp - GENESIS - 1) as usize]
                        .optimistic_at_timestamp[&ChainId(U256::from(chain))];
                    assert_eq!(
                        expected, previous,
                        "chain {chain} changed on idle timestamp {}",
                        transition.timestamp
                    );
                }
            }
            for transition in outputs.consolidation_outputs.transitions {
                assert_eq!(
                    transition.super_root,
                    fixture.responses[(transition.timestamp - GENESIS) as usize]
                        .data
                        .as_ref()
                        .unwrap()
                        .super_root
                );
            }
        }
    }
}

fn assert_composes(fixture: &Fixture, whole: &SpanResult, chunks: &[SpanResult]) {
    let ranges: Vec<_> =
        chunks.iter().flat_map(|chunk| chunk.range_outputs.transitions.iter()).collect();
    assert_eq!(ranges, whole.range_outputs.transitions.iter().collect::<Vec<_>>());
    let consolidations: Vec<_> =
        chunks.iter().flat_map(|chunk| chunk.consolidation_outputs.transitions.iter()).collect();
    assert_eq!(consolidations, whole.consolidation_outputs.transitions.iter().collect::<Vec<_>>());
    for pair in chunks.windows(2) {
        assert_eq!(pair[0].range_outputs.span.end + 1, pair[1].range_outputs.span.start);
        assert_eq!(
            pair[0].consolidation_outputs.transitions.last().unwrap().super_root,
            pair[1].consolidation_outputs.previous_super_root
        );
    }
    assert_eq!(
        chunks.last().unwrap().consolidation_outputs.transitions.last().unwrap().super_root,
        fixture.responses.last().unwrap().data.as_ref().unwrap().super_root
    );
    let starting = fixture.responses.first().unwrap().data.as_ref().unwrap();
    let mut aggregation = SuperAggregationInputs {
        l1_head: fixture.l1_head,
        starting_root_hash: starting.super_root,
        starting_super_root_proof: proof_from_super_v1(&starting.super_v1).unwrap(),
        root_claim: fixture.responses.last().unwrap().data.as_ref().unwrap().super_root,
        l2_sequence_number: fixture.end(),
        prover: Address::ZERO,
        range_outputs: chunks.iter().map(|chunk| chunk.range_outputs.clone()).collect(),
        consolidation_outputs: chunks
            .iter()
            .map(|chunk| chunk.consolidation_outputs.clone())
            .collect(),
    };
    aggregation.validate().unwrap();
    let first = aggregation.range_outputs[0].span.start;
    let timestamp = aggregation.range_outputs[0]
        .transitions
        .iter()
        .take_while(|transition| transition.timestamp == first)
        .copied()
        .collect::<Vec<_>>();
    aggregation.range_outputs[0].transitions.drain(..timestamp.len());
    assert!(aggregation.validate().is_err(), "a dropped native timestamp must be rejected");
    aggregation.range_outputs[0].transitions.splice(0..0, timestamp.iter().copied());
    aggregation.range_outputs[0].transitions.splice(0..0, timestamp);
    assert!(aggregation.validate().is_err(), "a duplicated native timestamp must be rejected");
}

#[tokio::test(flavor = "multi_thread")]
async fn independently_replayed_multisecond_chunks_compose() {
    let fixture = Fixture::new(&[2, 3]);
    let whole =
        fixture.replay(ChunkSpan { agreed: GENESIS, claimed: fixture.end() }).await.unwrap();
    for boundary in GENESIS + 1..fixture.end() {
        let mut chunks = Vec::new();
        for span in [
            ChunkSpan { agreed: GENESIS, claimed: boundary },
            ChunkSpan { agreed: boundary, claimed: fixture.end() },
        ] {
            chunks.push(fixture.replay(span).await.unwrap());
        }
        assert_composes(&fixture, &whole, &chunks);
    }
}

struct NativeProver {
    fixture: Fixture,
    limit: StdinByteLimit,
}

impl NativeProver {
    fn new(fixture: Fixture) -> Self {
        let reference = fixture.stdin(ChunkSpan { agreed: GENESIS, claimed: GENESIS + 2 }).unwrap();
        let limit = StdinByteLimit::new(stdin_encoded_len(&reference).unwrap()).unwrap();
        Self { fixture, limit }
    }
}

impl SpanProver for NativeProver {
    type Checked = SpanResult;

    fn check_leaf(
        &self,
        span: ChunkSpan,
        _: Option<SynthesizedExecution>,
    ) -> BoxFuture<'_, Result<SpanResult>> {
        async move {
            let replayed = self.fixture.replay(span).await?;
            ensure_stdin_fits("range", &self.fixture.stdin(span)?, self.limit)?;
            Ok(replayed)
        }
        .boxed()
    }

    fn prove_leaf<'s>(
        &'s self,
        checked: SpanResult,
        _: &'s ChunkProgress,
    ) -> BoxFuture<'s, Result<SpanResult>> {
        async move { Ok(checked) }.boxed()
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn automatic_bisection_replays_idle_gap_chunks() {
    for fixture in [Fixture::new(&[4]), Fixture::with_replacement(&[4])] {
        let prover = NativeProver::new(fixture);
        let span = ChunkSpan { agreed: GENESIS, claimed: prover.fixture.end() };
        let whole = prover.fixture.replay(span).await.unwrap();
        let progress = ChunkProgress::default();
        let chunks = prove_span(&prover, 0, span, None, &progress).await.unwrap();
        assert!(progress.split.get().is_some());
        assert!(chunks.spans.iter().any(|chunk| {
            let agreed = chunk.range_outputs.span.start - 1;
            agreed > GENESIS + 4 && agreed < GENESIS + 8
        }));
        assert_composes(&prover.fixture, &whole, &chunks.spans);
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn multisecond_range_rejects_incorrect_idle_output() {
    let fixture = Fixture::new(&[4]);
    let mut synthesized =
        fixture.synthesized(ChunkSpan { agreed: GENESIS + 4, claimed: GENESIS + 5 });
    let mut witness = fixture.witness.clone();
    let transition = &mut synthesized.range_inputs.claimed_transitions[0];
    let mut preimage = vec![0; 128];
    preimage[32..64].copy_from_slice(B256::repeat_byte(0xff).as_slice());
    preimage[64..96].copy_from_slice(EMPTY_ROOT_HASH.as_slice());
    preimage[96..128].copy_from_slice(transition.optimistic_block.block_hash.as_slice());
    transition.optimistic_block.output_root = save(&mut witness, preimage);
    let (oracle, beacon) = DefaultWitnessData::from_parts(witness, BlobData::default())
        .get_oracle_and_blob_provider()
        .await
        .unwrap();
    let error =
        build_range_outputs(synthesized.range_inputs, oracle, beacon, Some(&fixture.configs))
            .await
            .unwrap_err();
    assert!(
        error.to_string().contains("no-op super-range transition changed output root"),
        "{error:#}"
    );
}

#[tokio::test(flavor = "multi_thread")]
async fn replacement_prefixes_derive_original_and_consolidate_replacement() {
    for periods in [&[2][..], &[3], &[4], &[2, 3]] {
        let fixture = Fixture::with_replacement(periods);
        for end in GENESIS + 1..=fixture.end() {
            let outputs =
                fixture.replay(ChunkSpan { agreed: GENESIS, claimed: end }).await.unwrap_or_else(
                    |err| panic!("replacement periods {periods:?}, endpoint {end}: {err:#}"),
                );
            for transition in &outputs.range_outputs.transitions {
                let chain: u64 = transition.optimistic_block.chain_id.to();
                let timestamp = transition.timestamp;
                let response = &fixture.responses[(timestamp - GENESIS) as usize];
                let optimistic = &response.optimistic_at_timestamp[&ChainId(U256::from(chain))];
                assert_eq!(transition.optimistic_block.output_root, optimistic.output_root);
                let verified = response.data.as_ref().unwrap().super_v1.chains
                    [(chain - CHAIN) as usize]
                    .output;
                if timestamp == GENESIS + periods[(chain - CHAIN) as usize] {
                    assert_ne!(optimistic.output_root, verified);
                } else {
                    assert_eq!(optimistic.output_root, verified);
                }
                let period = periods[(chain - CHAIN) as usize];
                if !(timestamp - GENESIS).is_multiple_of(period) {
                    let previous = &fixture.responses[(timestamp - GENESIS - 1) as usize]
                        .data
                        .as_ref()
                        .unwrap()
                        .super_v1
                        .chains[(chain - CHAIN) as usize];
                    assert_eq!(transition.optimistic_block.output_root, previous.output);
                    assert_eq!(
                        transition.optimistic_block.block_hash,
                        fetch_output_block_hash(&fixture.witness, previous.output).await.unwrap()
                    );
                }
            }
            for transition in outputs.consolidation_outputs.transitions {
                assert_eq!(
                    transition.super_root,
                    fixture.responses[(transition.timestamp - GENESIS) as usize]
                        .data
                        .as_ref()
                        .unwrap()
                        .super_root
                );
            }
        }
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn independently_replayed_replacement_chunks_compose() {
    for periods in [&[2][..], &[3], &[4], &[2, 3]] {
        let fixture = Fixture::with_replacement(periods);
        let whole =
            fixture.replay(ChunkSpan { agreed: GENESIS, claimed: fixture.end() }).await.unwrap();
        for boundary in GENESIS + 1..fixture.end() {
            let mut chunks = Vec::new();
            for span in [
                ChunkSpan { agreed: GENESIS, claimed: boundary },
                ChunkSpan { agreed: boundary, claimed: fixture.end() },
            ] {
                chunks.push(fixture.replay(span).await.unwrap_or_else(|err| {
                    panic!("periods {periods:?}, boundary {boundary}: {err:#}")
                }));
            }
            assert_composes(&fixture, &whole, &chunks);
        }
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn replacement_range_rejects_stale_optimistic_head_at_idle_timestamp() {
    let mut fixture = Fixture::with_replacement(&[4]);
    let original =
        fixture.responses[4].optimistic_at_timestamp[&ChainId(U256::from(CHAIN))].clone();
    fixture.responses[5].optimistic_at_timestamp.insert(ChainId(U256::from(CHAIN)), original);
    let synthesized = fixture.synthesized(ChunkSpan { agreed: GENESIS + 4, claimed: GENESIS + 5 });
    let (oracle, beacon) = DefaultWitnessData::from_parts(fixture.witness, BlobData::default())
        .get_oracle_and_blob_provider()
        .await
        .unwrap();
    let error =
        build_range_outputs(synthesized.range_inputs, oracle, beacon, Some(&fixture.configs))
            .await
            .unwrap_err();
    assert!(
        error.to_string().contains("no-op super-range transition changed output root"),
        "{error:#}"
    );
}
