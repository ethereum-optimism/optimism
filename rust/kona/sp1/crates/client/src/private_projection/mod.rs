//! Private execution-to-projection relation of the `sp1-private-projection-v1` profile, shared
//! by native tests, the operator host and the SP1 guest (spec-sound-profile §C, §D).
//!
//! Everything the prover supplies is untrusted. The only verified output is the 672-byte
//! `PublicValuesV1` (§C.2): admission recomputes every word from its own derivation state and
//! consensus config, and the relation authenticates everything else against those words. L1 data
//! is authenticated by `l1Head` (word 15), recovery blocks by `recoveryHash` (word 9), the anchor
//! by `anchorOutputRoot` (word 8), and the configs by words 2–4. Private payload attributes,
//! forced deposits and fee parameters are derived in-guest with Kona's own
//! `StatefulAttributesBuilder`; nothing about them is chosen by the prover.
//!
//! Interop validity of the private chain's imports is out of scope: public interop enforces it on
//! the rendered executing messages, and `messagesRoot` (word 19) makes the rendering complete.

mod l1;
mod providers;

pub use kona_protocol::projection::PUBLIC_VALUES_LEN;
pub use l1::ordered_list_preimages;

use crate::precompiles::ZkvmOpEvmFactory;
use alloy_consensus::{BlockBody, Header, Transaction, TxEnvelope};
use alloy_eips::{BlockNumHash, Decodable2718, Encodable2718};
use alloy_op_evm::{block::OpAlloyReceiptBuilder, post_exec::PostExecEvmFactoryAdapter};
use alloy_primitives::{B256, Bytes, Sealable, U256, keccak256};
use alloy_rlp::Decodable;
use alloy_sol_types::SolCall;
use anyhow::{Context as _, Result, anyhow, ensure};
use core::{
    future::Future,
    task::{Context, Poll, Waker},
};
use kona_derive::{AttributesBuilder, StatefulAttributesBuilder};
use kona_executor::StatelessL2Builder;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_mpt::{NoopTrieHinter, ordered_trie_with_encoder};
use kona_protocol::{
    Batch, Frame, L1BlockInfoTx, L2BlockInfo, OutputRoot, SpanBatch, SpanBatchElement, projection,
};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use providers::{RelationL2Provider, Store, WitnessL1Provider};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, sync::Arc};

const MAX_PRIVATE_DATA: usize = 100_000_000;

/// `PublicValuesV1`, the relation's only output (§C.2): 21 big-endian 32-byte words.
pub type PublicValues = [u8; PUBLIC_VALUES_LEN];

/// One submitted public block, excluding protocol-derived deposits.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ProjectionBlock {
    /// L2 timestamp.
    pub timestamp: u64,
    /// L1 origin number.
    pub epoch: u64,
    /// Canonical signed claim/output/replay transactions.
    pub transactions: Vec<Bytes>,
}

/// Untrusted relation input (§D). Every field is either a public-values word that the verifier
/// recomputes, or is authenticated in-guest by one. There is no prover-supplied attribute path:
/// unknown fields, including the retired `attributes`, are rejected.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RelationInput {
    /// Exact deployed private `rollup.json` bytes (hashed into word 3, then parsed).
    pub private_config: Bytes,
    /// Exact L1 chain-config JSON bytes (hashed into word 3, then parsed).
    pub l1_config: Bytes,
    /// Projection rollup-config JSON (parsed; hashed canonically into word 2).
    pub projection_config: Bytes,
    /// Private dependency-set JSON (parsed; hashed canonically into word 4).
    pub dependency_set: Bytes,
    /// Parent hash of the complete submitted span, including any overlap (word 5).
    pub parent_hash: B256,
    /// Surviving canonical projection checkpoint (words 6–7).
    pub anchor: BlockNumHash,
    /// Private output at that checkpoint (word 8).
    pub anchor_output: B256,
    /// Canonical deposit-only public blocks, ascending from anchor + 1 to the span parent
    /// (committed by word 9).
    pub recovery: Vec<OpBlock>,
    /// Full submitted public span (words 10–19).
    pub blocks: Vec<ProjectionBlock>,
    /// Must equal `claim.l1Head` (word 15); anchors the L1 witness.
    pub l1_head: B256,
}

/// Private witness. Its contents never enter the public values.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Witness {
    /// Private header corresponding to the authenticated checkpoint.
    pub anchor_header: Header,
    /// The private anchor block's full transaction list (checked against its transactions root).
    /// Empty if and only if the anchor is the private genesis.
    pub anchor_transactions: Vec<Bytes>,
    /// Stock framed/compressed private span, committed by `privateDataHash`.
    pub private_data: Bytes,
    /// Content-addressed preimages: private trie nodes, bytecode and historical headers, plus L1
    /// headers and L1 receipts-trie nodes.
    pub preimages: BTreeMap<B256, Bytes>,
}

/// Word `index` of `pv`.
pub fn public_value_word(pv: &PublicValues, index: usize) -> B256 {
    B256::from_slice(&pv[index * 32..(index + 1) * 32])
}

/// The `execution-mock-v1` admission digest of the statement these public values describe. Every
/// input of `projection::admission_digest` is a public-values word (1, 5–9, 17).
pub fn execution_mock_digest(pv: &PublicValues) -> B256 {
    let w = |i| public_value_word(pv, i);
    let number = u64::try_from(U256::from_be_bytes(w(6).0)).unwrap_or(u64::MAX);
    projection::admission_digest(&projection::Statement {
        chain_id: w(1),
        parent_hash: w(5),
        projection_hash: w(17),
        continuation: projection::Continuation {
            anchor: BlockNumHash { number, hash: w(7) },
            output_root: w(8),
            recovery_hash: w(9),
        },
        claim: projection::RangeClaim {
            version: 0,
            firstBlock: 0,
            lastBlock: 0,
            privateTerminalBlockHash: B256::ZERO,
            privateTerminalParentHash: B256::ZERO,
            anchorBlock: 0,
            anchorOutputRoot: B256::ZERO,
            recoveryHash: B256::ZERO,
            parentOutputRoot: B256::ZERO,
            l1Head: B256::ZERO,
            rollupConfigHash: B256::ZERO,
            depSetHash: B256::ZERO,
            privateDataHash: B256::ZERO,
            proof: Bytes::new(),
        },
        projection_config_hash: w(2),
        private_config_hash: w(3),
        outputs_root: w(18),
        messages_root: w(19),
        terminal_output: w(20),
    })
}

/// Decode the guest's JSON witness transport and run the same pure relation.
/// JSON is used because rollup configs contain serde-flattened fields that are
/// not compatible with bincode's non-self-describing deserializer.
pub fn execute_encoded(bytes: &[u8]) -> Result<PublicValues> {
    let (input, witness): (RelationInput, Witness) = serde_json::from_slice(bytes)?;
    execute(&input, &witness)
}

/// Poll a future whose providers are all synchronous. The relation never waits on I/O, so a
/// suspended future is a bug, not a retry.
fn run_ready<F: Future>(f: F) -> Result<F::Output> {
    let mut f = core::pin::pin!(f);
    match f.as_mut().poll(&mut Context::from_waker(Waker::noop())) {
        Poll::Ready(v) => Ok(v),
        Poll::Pending => Err(anyhow!("relation provider suspended")),
    }
}

fn encode_all(block: &OpBlock) -> Vec<Bytes> {
    block.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect()
}

fn tx_root(txs: &[Bytes]) -> B256 {
    ordered_trie_with_encoder(txs, |tx, out| out.put_slice(tx)).root()
}

/// A block holding only what `L2BlockInfo::from_block_and_genesis` and `to_system_config` read:
/// the header and the first (L1-info) transaction.
fn info_block(header: Header, first_tx: Option<&Bytes>) -> Result<OpBlock> {
    let transactions = match first_tx {
        Some(raw) => vec![OpTxEnvelope::decode_2718_exact(raw)?],
        None => vec![],
    };
    Ok(OpBlock { header, body: BlockBody { transactions, ..Default::default() } })
}

fn parent_context(
    block: &OpBlock,
    cfg: &RollupConfig,
) -> Result<(L2BlockInfo, kona_genesis::SystemConfig)> {
    let info = L2BlockInfo::from_block_and_genesis(block, &cfg.genesis)
        .map_err(|e| anyhow!("private block info: {e}"))?;
    let sys = kona_protocol::to_system_config(block, cfg)
        .map_err(|e| anyhow!("private system config: {e}"))?;
    Ok((info, sys))
}

/// Execute and check the entire relation. Pure: no RPC, files, clock or ambient configuration;
/// the result depends only on the supplied inputs. Failure emits no partial public values.
pub fn execute(input: &RelationInput, witness: &Witness) -> Result<PublicValues> {
    // §D.1: configs. Words 2–4 are the hashes of exactly what is parsed here.
    let private_cfg: RollupConfig =
        serde_json::from_slice(&input.private_config).context("private rollup config")?;
    let l1_cfg: L1ChainConfig =
        serde_json::from_slice(&input.l1_config).context("L1 chain config")?;
    let projection_config: RollupConfig =
        serde_json::from_slice(&input.projection_config).context("projection rollup config")?;
    let dep_set: DependencySet =
        serde_json::from_slice(&input.dependency_set).context("dependency set")?;
    let cfg = &projection_config;
    ensure!(
        private_cfg.private_projection.is_none(),
        "private execution must not use projection deposit policy"
    );
    ensure!(
        private_cfg.l2_chain_id == cfg.l2_chain_id &&
            private_cfg.block_time == cfg.block_time &&
            private_cfg.genesis.l2.number == cfg.genesis.l2.number &&
            private_cfg.genesis.l2_time == cfg.genesis.l2_time,
        "private/projection geometry mismatch"
    );
    ensure!(
        private_cfg.is_isthmus_active(private_cfg.genesis.l2_time),
        "private outputs require Isthmus at genesis"
    );
    ensure!(
        !cfg.private_projection
            .as_ref()
            .ok_or_else(|| anyhow!("missing projection config"))?
            .allow_events,
        "generic extra emitters are not supported by relation v1"
    );
    let private_config_hash =
        projection::private_config_hash(&input.private_config, &input.l1_config);
    let projection_config_hash = projection::config_hash(cfg)?;
    let dep_set_hash =
        projection::dependency_set_hash(dep_set.dependencies.keys().map(|id| U256::from(*id)));

    let span = SpanBatch {
        batches: input
            .blocks
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b.timestamp,
                epoch_num: b.epoch,
                transactions: b.transactions.clone(),
            })
            .collect(),
        ..Default::default()
    };
    let (first, last) = projection::range_bounds(cfg, &span)?;
    ensure!(
        input.anchor.number.checked_add(input.recovery.len() as u64).and_then(|n| n.checked_add(1)) ==
            Some(first),
        "recovery interval length"
    );

    // §D.3: the private anchor header carries `anchor_output` (word 8).
    ensure!(witness.anchor_header.number == input.anchor.number, "private anchor height");
    let anchor_time = cfg
        .genesis
        .l2_time
        .checked_add(
            input
                .anchor
                .number
                .checked_sub(cfg.genesis.l2.number)
                .ok_or_else(|| anyhow!("anchor before genesis"))?
                .checked_mul(cfg.block_time)
                .ok_or_else(|| anyhow!("time overflow"))?,
        )
        .ok_or_else(|| anyhow!("time overflow"))?;
    ensure!(witness.anchor_header.timestamp == anchor_time, "private anchor timestamp");
    ensure!(output(&witness.anchor_header)? == input.anchor_output, "private anchor output");

    // §D.2.3: private parent context of the first derived block.
    let anchor_block = if input.anchor.number == private_cfg.genesis.l2.number {
        ensure!(witness.anchor_transactions.is_empty(), "private genesis anchor transactions");
        info_block(witness.anchor_header.clone(), None)?
    } else {
        ensure!(
            !witness.anchor_transactions.is_empty() &&
                tx_root(&witness.anchor_transactions) == witness.anchor_header.transactions_root,
            "private anchor transactions"
        );
        info_block(witness.anchor_header.clone(), witness.anchor_transactions.first())?
    };
    let (mut parent_info, mut sys_config) = parent_context(&anchor_block, &private_cfg)?;

    // Canonical recovery blocks: ancestry, transaction roots and their L1 epochs (§D.2.2). Their
    // authenticity is word 9, which admission recomputes from the canonical public chain.
    let mut public_parent = input.anchor.hash;
    let mut recovery_txs = Vec::with_capacity(input.recovery.len());
    let mut recovery_epochs = Vec::with_capacity(input.recovery.len());
    for (i, block) in input.recovery.iter().enumerate() {
        ensure!(
            block.header.parent_hash == public_parent &&
                block.header.number == input.anchor.number + i as u64 + 1,
            "recovery ancestry"
        );
        let txs = encode_all(block);
        ensure!(tx_root(&txs) == block.header.transactions_root, "recovery transaction root");
        let info_tx = block.body.transactions.first().ok_or_else(|| anyhow!("recovery L1 info"))?;
        let info = L1BlockInfoTx::decode_calldata(info_tx.input())
            .map_err(|e| anyhow!("recovery L1 info: {e}"))?;
        recovery_epochs.push(info.id());
        recovery_txs.push(txs);
        public_parent = block.header.hash_slow();
    }
    ensure!(public_parent == input.parent_hash, "canonical public parent");
    let recovery_hash = input
        .recovery
        .iter()
        .zip(&recovery_txs)
        .rev()
        .fold(B256::ZERO, |hash, (b, txs)| projection::recovery_step(hash, b, txs));

    // The in-guest structural validator: the same admission rules over the same bytes, which
    // also builds words 17–20 from the published span. The relation cannot check its own proof,
    // so the claim's proof bytes are ignored here (they are not part of the statement).
    let mut statement = projection::validate_projection_range(
        cfg,
        projection::ProjectionContext {
            parent_hash: input.parent_hash,
            l1_head: input.l1_head,
            continuation: projection::Continuation {
                anchor: input.anchor,
                output_root: input.anchor_output,
                recovery_hash,
            },
        },
        &span,
        &projection::StubVerifier,
    )?;
    let claim = &statement.claim;
    ensure!(input.l1_head == claim.l1Head, "l1Head binding");
    ensure!(
        claim.rollupConfigHash == projection_config_hash,
        "projection configuration commitment"
    );
    ensure!(claim.depSetHash == dep_set_hash, "dependency-set commitment");
    ensure!(claim.privateDataHash == keccak256(&witness.private_data), "private data commitment");
    let private_span = decode_private_span(&witness.private_data, &private_cfg)?;
    ensure!(private_span.batches.len() == input.blocks.len(), "private span length");

    // §D.2.1: every new-block epoch hash is a consequence of `l1Head`.
    let store = Store(&witness.preimages);
    let min_epoch = input.blocks.iter().map(|b| b.epoch).min().expect("range_bounds: nonempty");
    let epoch_hashes = l1::epoch_hashes(store, input.l1_head, min_epoch)?;

    let private_cfg = Arc::new(private_cfg);
    let l1_cfg = Arc::new(l1_cfg);
    let dep_set = Arc::new(dep_set);
    let mut executor = StatelessL2Builder::new(
        &private_cfg,
        PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory),
        OpAlloyReceiptBuilder::default(),
        store,
        NoopTrieHinter,
        witness.anchor_header.clone().seal_slow(),
    );
    ensure!(executor.compute_output_root()? == input.anchor_output, "anchor storage witness");
    let mut private_parent = witness.anchor_header.hash_slow();
    let mut terminal_output = input.anchor_output;
    let mut output_leaves = Vec::with_capacity(input.blocks.len());
    let mut message_leaves = Vec::new();
    for idx in 0..input.recovery.len() + input.blocks.len() {
        let number = input.anchor.number + idx as u64 + 1;
        let expected_time = anchor_time
            .checked_add(
                (idx as u64 + 1)
                    .checked_mul(cfg.block_time)
                    .ok_or_else(|| anyhow!("time overflow"))?,
            )
            .ok_or_else(|| anyhow!("time overflow"))?;
        let published = idx.checked_sub(input.recovery.len());
        let epoch = match published {
            None => recovery_epochs[idx],
            Some(k) => {
                let n = input.blocks[k].epoch;
                BlockNumHash {
                    number: n,
                    hash: *epoch_hashes.get(&n).ok_or_else(|| anyhow!("missing L1 epoch {n}"))?,
                }
            }
        };

        // §D.2.4: Kona's own attributes derivation over the authenticated L1 witness.
        let mut builder = StatefulAttributesBuilder::new(
            Arc::clone(&private_cfg),
            Arc::clone(&l1_cfg),
            RelationL2Provider::new(parent_info.block_info.hash, sys_config),
            WitnessL1Provider(store),
            Some(Arc::clone(&dep_set)),
        );
        let mut attrs = run_ready(builder.prepare_payload_attributes(parent_info, epoch))?
            .map_err(|e| anyhow!("private attributes of block {number}: {e}"))?;
        ensure!(
            attrs.payload_attributes.timestamp == expected_time && attrs.no_tx_pool == Some(true),
            "derived attribute schedule"
        );
        let mut txs = attrs.transactions.take().ok_or_else(|| anyhow!("missing derived inputs"))?;
        let l1_info = L1BlockInfoTx::decode_calldata(
            OpTxEnvelope::decode_2718_exact(
                txs.first().ok_or_else(|| anyhow!("missing L1 info"))?,
            )?
            .input(),
        )?;

        match published {
            None => {
                // §D.2.5: a recovery block executes the derived private inputs; the canonical
                // public block must be their projection.
                let public = &input.recovery[idx];
                let public_txs = &recovery_txs[idx];
                if let Some(OpTxEnvelope::PostExec(tx)) = public.body.transactions.last() {
                    ensure!(
                        private_cfg.is_sdm_active(expected_time) &&
                            tx.payload.block_number == number &&
                            tx.payload.gas_refund_entries.is_empty(),
                        "invalid replacement post-exec"
                    );
                    txs.push(public_txs.last().expect("nonempty").clone());
                }
                ensure!(txs.len() == public_txs.len(), "recovery input count");
                ensure!(
                    project_l1_info(&txs[0])? == public_txs[0],
                    "recovery L1 info correspondence"
                );
                ensure!(
                    txs[1..] == public_txs[1..],
                    "recovery forced inputs differ from canonical block"
                );
                ensure!(
                    public.header.timestamp == expected_time &&
                        public.header.mix_hash == attrs.payload_attributes.prev_randao &&
                        public.header.beneficiary ==
                            attrs.payload_attributes.suggested_fee_recipient &&
                        public.header.parent_beacon_block_root ==
                            attrs.payload_attributes.parent_beacon_block_root,
                    "replacement execution environment"
                );
            }
            Some(k) => {
                // §D.2.6: a new block executes the derived inputs followed by the committed
                // private sequencer transactions.
                let b = &private_span.batches[k];
                ensure!(
                    b.timestamp == expected_time && b.epoch_num == epoch.number,
                    "private/public origin or timestamp"
                );
                if k == 0 {
                    ensure!(
                        terminal_output == claim.parentOutputRoot,
                        "private parent after recovery"
                    );
                    ensure!(
                        private_span.parent_check.as_slice() == &private_parent.as_slice()[..20],
                        "private span parent"
                    );
                }
                for tx in &b.transactions {
                    ensure!(
                        !matches!(OpTxEnvelope::decode_2718_exact(tx)?, OpTxEnvelope::Deposit(_)),
                        "private span injects deposit"
                    );
                }
                txs.extend(b.transactions.iter().cloned());
            }
        }

        let first_tx = txs[0].clone();
        attrs.transactions = Some(txs);
        let result = executor
            .build_block(attrs)
            .with_context(|| format!("private execution block {number}"))?;
        terminal_output = executor.compute_output_root()?;

        if let Some(k) = published {
            let public = &input.blocks[k];
            let record_index = usize::from(k == 0);
            let record = TxEnvelope::decode_2718_exact(&public.transactions[record_index])?;
            let root =
                projection::recordOutputCall::abi_decode_validate(record.input())?.outputRoot;
            ensure!(root == terminal_output, "published checkpoint differs from executed output");
            output_leaves.push(projection::output_leaf(number, terminal_output));

            // §D.5: the published replays are exactly the rendered private logs, in order.
            let rendered = projection::rendered_logs(
                result.execution_result.receipts.iter().flat_map(|r| r.logs()),
            )?;
            let expected =
                rendered.iter().map(projection::replay_calldata).collect::<Result<Vec<_>, _>>()?;
            message_leaves.extend(projection::message_leaves(number, &rendered)?);
            let actual = public.transactions[record_index + 1..]
                .iter()
                .map(|raw| {
                    let tx = TxEnvelope::decode_2718_exact(raw)?;
                    Ok((tx.to().ok_or_else(|| anyhow!("replay creation"))?, tx.input().clone()))
                })
                .collect::<Result<Vec<_>>>()?;
            ensure!(actual == expected, "projection messages differ from executed receipts");
        }
        if number == last {
            ensure!(
                result.header.hash() == claim.privateTerminalBlockHash &&
                    private_parent == claim.privateTerminalParentHash,
                "private terminal headers"
            );
            ensure!(
                l1_info.block_hash() == claim.l1Head &&
                    private_span.l1_origin_check.as_slice() ==
                        &l1_info.block_hash().as_slice()[..20],
                "terminal L1 origin"
            );
        }
        // §D.2.7: advance the parent context from the block just built.
        private_parent = result.header.hash();
        let built = info_block(result.header.clone().unseal(), Some(&first_tx))?;
        (parent_info, sys_config) = parent_context(&built, &private_cfg)?;
    }

    // §C.3.6/§C.3.7: the executed commitments must equal the ones admission builds from the
    // published span. Word 19 equality is the completeness proof.
    let outputs_root = projection::commitment_root(projection::OUTPUTS_DOMAIN, &output_leaves);
    let messages_root = projection::commitment_root(projection::MESSAGES_DOMAIN, &message_leaves);
    ensure!(outputs_root == statement.outputs_root, "outputs root differs from published records");
    ensure!(
        messages_root == statement.messages_root,
        "messages root differs from published replays"
    );
    ensure!(terminal_output == statement.terminal_output, "terminal output is not the last record");
    ensure!(
        statement.projection_config_hash == projection_config_hash,
        "projection configuration commitment"
    );
    // Word 3 is the hash of the bytes this relation parsed, not the config field: admission
    // supplies its consensus constant, so the proof verifies only if they are equal (§B.2).
    statement.private_config_hash = private_config_hash;
    Ok(projection::public_values(&statement))
}

/// Apply only the projection's fee policy to a canonical Isthmus-or-later L1-info
/// deposit. Authentication of private fee settings still belongs to the caller's
/// canonical context, not to this normalization.
fn project_l1_info(raw: &[u8]) -> Result<Bytes> {
    let OpTxEnvelope::Deposit(tx) = OpTxEnvelope::decode_2718_exact(raw)? else {
        return Err(anyhow!("recovery L1 info is not a deposit"));
    };
    ensure!(!kona_protocol::is_projection_user_deposit(tx.inner(), 0), "invalid recovery L1 info");
    let info = L1BlockInfoTx::decode_calldata(&tx.inner().input)?;
    ensure!(
        matches!(info, L1BlockInfoTx::Isthmus(_) | L1BlockInfoTx::Jovian(_)),
        "unsupported recovery L1 info"
    );
    ensure!(info.encode_calldata() == tx.inner().input, "noncanonical recovery L1 info");
    let mut normalized = tx.inner().clone();
    let mut input = normalized.input.to_vec();
    input[4..12].fill(0); // base and blob fee scalars
    input[164..176].fill(0); // operator fee scalar and constant
    normalized.input = input.into();
    Ok(OpTxEnvelope::from(normalized).encoded_2718().into())
}

fn output(header: &Header) -> Result<B256> {
    Ok(OutputRoot::from_parts(
        header.state_root,
        header.withdrawals_root.ok_or_else(|| anyhow!("missing private message-passer root"))?,
        header.hash_slow(),
    )
    .hash())
}

fn decode_private_span(data: &[u8], cfg: &RollupConfig) -> Result<SpanBatch> {
    ensure!(data.len() <= MAX_PRIVATE_DATA, "private data too large");
    let frames = Frame::parse_frames(data)?;
    ensure!(!frames.is_empty(), "empty private channel");
    let mut compressed = Vec::new();
    for (i, frame) in frames.iter().enumerate() {
        ensure!(
            usize::from(frame.number) == i &&
                frame.id == frames[0].id &&
                frame.is_last == (i + 1 == frames.len()),
            "private frame sequence"
        );
        compressed.extend_from_slice(&frame.data);
    }
    // Strict decompression: unlike ordinary channel streaming, a partial prefix is
    // never sufficient for a committed full-span witness.
    let decoded = if compressed.first() == Some(&1) {
        ensure!(cfg.is_fjord_active(cfg.genesis.l2_time), "brotli before Fjord");
        kona_protocol::decompress_brotli(&compressed[1..], MAX_PRIVATE_DATA)?
    } else {
        miniz_oxide::inflate::decompress_to_vec_zlib_with_limit(&compressed, MAX_PRIVATE_DATA)
            .map_err(|_| anyhow!("invalid or oversized private channel"))?
    };
    let mut rlp = decoded.as_slice();
    let bytes = Bytes::decode(&mut rlp)?;
    ensure!(rlp.is_empty(), "extra private batches");
    let mut raw = bytes.as_ref();
    let Batch::Span(span) = Batch::decode(&mut raw, cfg)? else {
        return Err(anyhow!("private input must be a span"));
    };
    ensure!(raw.is_empty(), "trailing private batch bytes");
    projection::range_bounds(cfg, &span)?;
    Ok(span)
}

#[cfg(test)]
mod tests;
#[cfg(test)]
mod tests_negative;
