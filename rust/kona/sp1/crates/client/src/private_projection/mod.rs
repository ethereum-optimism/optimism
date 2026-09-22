//! Private execution-to-projection relation, shared by native tests and the SP1 guest.
//!
//! The caller authenticates `PublicInputs` against canonical derivation. In particular,
//! deposit-only payload attributes are PUBLIC inputs, not prover-selected private witness.
//! This program does not replace L1 derivation or validate interop dependencies.

use crate::precompiles::ZkvmOpEvmFactory;
use alloy_consensus::{Header, Transaction, TxEnvelope};
use alloy_eips::{BlockNumHash, Decodable2718, Encodable2718};
use alloy_op_evm::{block::OpAlloyReceiptBuilder, post_exec::PostExecEvmFactoryAdapter};
use alloy_primitives::{Address, B256, Bytes, Log, Sealable, address, keccak256};
use alloy_rlp::Decodable;
use alloy_sol_types::{SolCall, SolEvent, sol};
use anyhow::{Result, anyhow, ensure};
use kona_executor::{StatelessL2Builder, TrieDBProvider};
use kona_genesis::RollupConfig;
use kona_interop::ExecutingMessage;
use kona_mpt::{NoopTrieHinter, TrieNode, TrieProvider, ordered_trie_with_encoder};
use kona_protocol::{
    Batch, Frame, L1BlockInfoTx, OutputRoot, SpanBatch, SpanBatchElement, projection,
};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

const MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const INBOX: Address = address!("4200000000000000000000000000000000000022");
const MAX_PRIVATE_DATA: usize = 100_000_000;

sol! {
    /// The standard messenger initiating event, with its original indexed fields.
    event SentMessage(uint256 indexed destination, address indexed target, uint256 indexed nonce, address sender, bytes message);
}

/// One submitted public block, excluding protocol-derived deposits.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProjectionBlock {
    /// L2 timestamp.
    pub timestamp: u64,
    /// L1 origin number.
    pub epoch: u64,
    /// Canonical signed claim/output/replay transactions.
    pub transactions: Vec<Bytes>,
}

/// Expected PUBLIC inputs. A verifier must compute this context from its own
/// canonical derivation/configuration, not accept the publisher's context hash.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublicInputs {
    /// Private rollup config JSON, independently authenticated by the verifier.
    pub private_config: Bytes,
    /// Exact public projection config JSON committed by the claim, including the private genesis
    /// root.
    pub projection_config: Bytes,
    /// Exact dependency-set JSON committed by the claim.
    pub dependency_set: Bytes,
    /// Parent hash of the complete submitted span, including any overlap.
    pub parent_hash: B256,
    /// Surviving canonical projection checkpoint.
    pub anchor: BlockNumHash,
    /// Authenticated private output at that checkpoint.
    pub anchor_output: B256,
    /// Canonical deposit-only replacements, ascending from anchor + 1 to parent.
    pub recovery: Vec<OpBlock>,
    /// Canonical replacement attributes followed by PRIVATE-config-derived attributes for the new
    /// span. Ordinary private attributes retain the private chain's fees; projection fee
    /// policy differs. These include the public execution environment (time, gas, fees,
    /// randomness).
    pub attributes: Vec<OpPayloadAttributes>,
    /// Full submitted span; proof bytes are normalized before committing public inputs.
    pub blocks: Vec<ProjectionBlock>,
}

/// Private witness. Its contents never enter the guest's public journal.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Witness {
    /// Private header corresponding to the authenticated checkpoint.
    pub anchor_header: Header,
    /// Stock framed/compressed private span, committed by `privateDataHash`.
    pub private_data: Bytes,
    /// Content-addressed trie nodes, bytecode and historical headers.
    pub preimages: BTreeMap<B256, Bytes>,
}

/// Exact journal checked in every mode, including native and mock execution.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PublicOutputs {
    /// Explicit relation version; not a network verifier identifier.
    pub version: u32,
    /// Full normalized public context, including derived deposits and configuration.
    pub context_hash: B256,
    /// Merkle commitment produced by the production structural validator.
    pub projection_hash: B256,
    /// Private terminal output computed by executing the complete interval.
    pub terminal_output: B256,
}

#[derive(Debug)]
struct Store<'a>(&'a BTreeMap<B256, Bytes>);
impl Store<'_> {
    fn get(&self, hash: B256) -> Result<&[u8]> {
        let bytes = self.0.get(&hash).ok_or_else(|| anyhow!("missing witness preimage {hash}"))?;
        ensure!(keccak256(bytes) == hash, "incorrect witness preimage");
        Ok(bytes)
    }
}
impl TrieProvider for Store<'_> {
    type Error = anyhow::Error;
    fn trie_node_by_hash(&self, hash: B256) -> Result<TrieNode> {
        let mut bytes = self.get(hash)?;
        let node = TrieNode::decode(&mut bytes)?;
        ensure!(bytes.is_empty(), "trailing trie bytes");
        Ok(node)
    }
}
impl TrieDBProvider for Store<'_> {
    fn bytecode_by_hash(&self, hash: B256) -> Result<Bytes> {
        Ok(self.get(hash)?.to_vec().into())
    }
    fn header_by_hash(&self, hash: B256) -> Result<Header> {
        let mut bytes = self.get(hash)?;
        let header = Header::decode(&mut bytes)?;
        ensure!(bytes.is_empty(), "trailing header bytes");
        Ok(header)
    }
}

/// Decode the guest's JSON witness transport and run the same pure relation.
/// JSON is used because rollup configs contain serde-flattened fields that are
/// not compatible with bincode's non-self-describing deserializer.
pub fn execute_encoded(bytes: &[u8]) -> Result<PublicOutputs> {
    let (inputs, witness): (PublicInputs, Witness) = serde_json::from_slice(bytes)?;
    execute(&inputs, &witness)
}

/// Execute and check the entire relation. Pure: no RPC, files, clock or ambient
/// configuration; the result depends only on the supplied inputs. Failure emits no partial journal.
pub fn execute(inputs: &PublicInputs, witness: &Witness) -> Result<PublicOutputs> {
    let private_cfg: RollupConfig = serde_json::from_slice(&inputs.private_config)?;
    let projection_config: RollupConfig = serde_json::from_slice(&inputs.projection_config)?;
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
    let span = SpanBatch {
        batches: inputs
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
        inputs
            .anchor
            .number
            .checked_add(inputs.recovery.len() as u64)
            .and_then(|n| n.checked_add(1)) ==
            Some(first),
        "recovery interval length"
    );
    ensure!(
        inputs.attributes.len() == inputs.recovery.len() + inputs.blocks.len(),
        "attribute count"
    );
    ensure!(witness.anchor_header.number == inputs.anchor.number, "private anchor height");
    let anchor_time = cfg
        .genesis
        .l2_time
        .checked_add(
            inputs
                .anchor
                .number
                .checked_sub(cfg.genesis.l2.number)
                .ok_or_else(|| anyhow!("anchor before genesis"))?
                .checked_mul(cfg.block_time)
                .ok_or_else(|| anyhow!("time overflow"))?,
        )
        .ok_or_else(|| anyhow!("time overflow"))?;
    ensure!(witness.anchor_header.timestamp == anchor_time, "private anchor timestamp");
    ensure!(output(&witness.anchor_header)? == inputs.anchor_output, "private anchor output");

    let mut public_parent = inputs.anchor.hash;
    for (i, block) in inputs.recovery.iter().enumerate() {
        ensure!(
            block.header.parent_hash == public_parent &&
                block.header.number == inputs.anchor.number + i as u64 + 1,
            "recovery ancestry"
        );
        let txs: Vec<Bytes> =
            block.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect();
        ensure!(
            ordered_trie_with_encoder(&txs, |tx, out| out.put_slice(tx)).root() ==
                block.header.transactions_root,
            "recovery transaction root"
        );
        let attrs = &inputs.attributes[i];
        ensure!(
            attrs.transactions.as_ref() == Some(&txs),
            "replacement inputs differ from canonical block"
        );
        ensure!(
            attrs.payload_attributes.timestamp == block.header.timestamp &&
                attrs.payload_attributes.prev_randao == block.header.mix_hash &&
                attrs.payload_attributes.suggested_fee_recipient == block.header.beneficiary &&
                attrs.payload_attributes.parent_beacon_block_root ==
                    block.header.parent_beacon_block_root &&
                attrs.gas_limit == Some(block.header.gas_limit),
            "replacement execution environment"
        );
        public_parent = block.header.hash_slow();
    }
    ensure!(public_parent == inputs.parent_hash, "canonical public parent");
    let recovery_hash = inputs.recovery.iter().rev().fold(B256::ZERO, |hash, b| {
        let txs = b.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect::<Vec<_>>();
        projection::recovery_step(hash, b, &txs)
    });
    let statement = projection::validate_projection_range(
        cfg,
        inputs.parent_hash,
        projection::Continuation {
            anchor: inputs.anchor,
            output_root: inputs.anchor_output,
            recovery_hash,
        },
        &span,
        &projection::StubVerifier,
    )?;
    let claim = &statement.claim;
    ensure!(
        claim.rollupConfigHash == keccak256(&inputs.projection_config),
        "projection configuration commitment"
    );
    ensure!(claim.depSetHash == keccak256(&inputs.dependency_set), "dependency-set commitment");
    ensure!(claim.privateDataHash == keccak256(&witness.private_data), "private data commitment");
    let private_span = decode_private_span(&witness.private_data, &private_cfg)?;
    ensure!(private_span.batches.len() == inputs.blocks.len(), "private span length");

    let mut executor = StatelessL2Builder::new(
        &private_cfg,
        PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory),
        OpAlloyReceiptBuilder::default(),
        Store(&witness.preimages),
        NoopTrieHinter,
        witness.anchor_header.clone().seal_slow(),
    );
    ensure!(executor.compute_output_root()? == inputs.anchor_output, "anchor storage witness");
    let mut private_parent = witness.anchor_header.hash_slow();
    let mut terminal_output = inputs.anchor_output;
    for (i, base_attrs) in inputs.attributes.iter().enumerate() {
        let number = inputs.anchor.number + i as u64 + 1;
        let expected_time = anchor_time
            .checked_add(
                (i as u64 + 1)
                    .checked_mul(cfg.block_time)
                    .ok_or_else(|| anyhow!("time overflow"))?,
            )
            .ok_or_else(|| anyhow!("time overflow"))?;
        ensure!(
            base_attrs.payload_attributes.timestamp == expected_time &&
                base_attrs.no_tx_pool == Some(true),
            "derived attribute schedule"
        );
        let deposits =
            base_attrs.transactions.as_ref().ok_or_else(|| anyhow!("missing derived deposits"))?;
        for (position, tx) in deposits.iter().enumerate() {
            match OpTxEnvelope::decode_2718_exact(tx)? {
                OpTxEnvelope::Deposit(_) => {}
                OpTxEnvelope::PostExec(tx) if number < first && position + 1 == deposits.len() => {
                    ensure!(
                        private_cfg.is_sdm_active(expected_time) &&
                            tx.payload.block_number == number &&
                            tx.payload.gas_refund_entries.is_empty(),
                        "invalid replacement post-exec"
                    );
                }
                _ => {
                    return Err(anyhow!("derived input is not a deposit or replacement post-exec"));
                }
            }
        }
        let first_tx = OpTxEnvelope::decode_2718_exact(
            deposits.first().ok_or_else(|| anyhow!("missing L1 info"))?,
        )?;
        let OpTxEnvelope::Deposit(ref info_tx) = first_tx else { unreachable!() };
        ensure!(
            !kona_protocol::is_projection_user_deposit(info_tx.inner(), 0),
            "unauthenticated L1 info deposit"
        );
        let info = L1BlockInfoTx::decode_calldata(first_tx.input())?;
        let mut attrs = base_attrs.clone();
        let published = number >= first;
        if published {
            let index = (number - first) as usize;
            let b = &private_span.batches[index];
            ensure!(
                b.timestamp == expected_time &&
                    b.epoch_num == info.id().number &&
                    b.epoch_num == inputs.blocks[index].epoch,
                "private/public origin or timestamp"
            );
            if index == 0 {
                ensure!(terminal_output == claim.parentOutputRoot, "private parent after recovery");
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
            attrs
                .transactions
                .as_mut()
                .expect("checked above")
                .extend(b.transactions.iter().cloned());
        }
        let result = executor.build_block(attrs)?;
        terminal_output = executor.compute_output_root()?;
        if published {
            let index = (number - first) as usize;
            let public = &inputs.blocks[index];
            let record_index = usize::from(index == 0);
            let record = TxEnvelope::decode_2718_exact(&public.transactions[record_index])?;
            let root =
                projection::recordOutputCall::abi_decode_validate(record.input())?.outputRoot;
            ensure!(root == terminal_output, "published checkpoint differs from executed output");
            let expected = render(result.execution_result.receipts.iter().flat_map(|r| r.logs()))?;
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
                info.block_hash() == claim.l1Head &&
                    private_span.l1_origin_check.as_slice() ==
                        &info.block_hash().as_slice()[..20],
                "terminal L1 origin"
            );
        }
        private_parent = result.header.hash();
    }
    let mut normalized = inputs.clone();
    // Bind decoded claim data without depending on proof bytes or the signature of its carrier.
    // All carrier envelope fields remain bound by the structural projection hash.
    normalized.blocks[0].transactions[0] = Bytes::new();
    let mut transcript = b"optimism.private-execution.v1\0".to_vec();
    transcript.extend_from_slice(statement.projection_hash.as_slice());
    transcript.extend_from_slice(&serde_json::to_vec(&normalized)?);
    Ok(PublicOutputs {
        version: 1,
        context_hash: keccak256(transcript),
        projection_hash: statement.projection_hash,
        terminal_output,
    })
}

fn output(header: &Header) -> Result<B256> {
    Ok(OutputRoot::from_parts(
        header.state_root,
        header.withdrawals_root.ok_or_else(|| anyhow!("missing private message-passer root"))?,
        header.hash_slow(),
    )
    .hash())
}

fn render<'a>(logs: impl Iterator<Item = &'a Log>) -> Result<Vec<(Address, Bytes)>> {
    let mut out = Vec::new();
    for log in logs {
        if log.address == MESSENGER && log.topics().first() == Some(&SentMessage::SIGNATURE_HASH) {
            let m = SentMessage::decode_log_validate(log)?.data;
            out.push((
                MESSENGER,
                projection::replaySentMessageCall {
                    destination: m.destination,
                    nonce: m.nonce,
                    sender: m.sender,
                    target: m.target,
                    message: m.message,
                }
                .abi_encode()
                .into(),
            ));
        } else if log.address == INBOX &&
            log.topics().first() == Some(&ExecutingMessage::SIGNATURE_HASH)
        {
            let m = ExecutingMessage::decode_log_validate(log)?.data;
            out.push((
                INBOX,
                projection::validateMessageCall {
                    identifier: projection::Identifier {
                        origin: m.identifier.origin,
                        blockNumber: m.identifier.blockNumber,
                        logIndex: m.identifier.logIndex,
                        timestamp: m.identifier.timestamp,
                        chainId: m.identifier.chainId,
                    },
                    payloadHash: m.payloadHash,
                }
                .abi_encode()
                .into(),
            ));
        }
    }
    Ok(out)
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
