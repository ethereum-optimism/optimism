//! Content-addressed witness store and the two providers the in-guest
//! `StatefulAttributesBuilder` reads through (spec-sound-profile §D.2.4).
//!
//! Nothing here trusts the prover: every preimage is checked against its keccak key, L1 headers
//! are addressed by hash, and L1 receipts are reconstructed from the header's receipts root.

use alloy_consensus::{Header, Receipt, ReceiptEnvelope, TxEnvelope};
use alloy_eips::Decodable2718;
use alloy_primitives::{B256, Bytes, keccak256};
use alloy_rlp::Decodable;
use anyhow::{Result, anyhow, ensure};
use async_trait::async_trait;
use core::fmt;
use kona_derive::{ChainProvider, L2ChainProvider, PipelineError, PipelineErrorKind};
use kona_executor::TrieDBProvider;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_mpt::{OrderedListWalker, TrieNode, TrieProvider};
use kona_protocol::{BatchValidationProvider, BlockInfo, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use std::{collections::BTreeMap, sync::Arc};

/// Content-addressed preimages. `get` enforces `keccak256(bytes) == key`.
#[derive(Debug, Clone, Copy)]
pub(crate) struct Store<'a>(pub(crate) &'a BTreeMap<B256, Bytes>);

impl Store<'_> {
    pub(crate) fn get(&self, hash: B256) -> Result<&[u8]> {
        let bytes = self.0.get(&hash).ok_or_else(|| anyhow!("missing witness preimage {hash}"))?;
        ensure!(keccak256(bytes) == hash, "incorrect witness preimage");
        Ok(bytes)
    }

    /// Header by its hash (the RLP preimage of the hash).
    pub(crate) fn header(&self, hash: B256) -> Result<Header> {
        let mut bytes = self.get(hash)?;
        let header = Header::decode(&mut bytes)?;
        ensure!(bytes.is_empty(), "trailing header bytes");
        Ok(header)
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
        self.header(hash)
    }
}

/// Provider error: a missing or inconsistent witness is never retryable inside the relation.
#[derive(Debug)]
pub(crate) struct WitnessError(pub(crate) String);

impl fmt::Display for WitnessError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "relation witness: {}", self.0)
    }
}

impl From<anyhow::Error> for WitnessError {
    fn from(e: anyhow::Error) -> Self {
        Self(format!("{e:#}"))
    }
}

impl From<WitnessError> for PipelineErrorKind {
    fn from(e: WitnessError) -> Self {
        PipelineError::Provider(e.to_string()).crit()
    }
}

/// L1 data authenticated through `l1Head` (headers by hash, receipts by receipts root).
#[derive(Debug, Clone, Copy)]
pub(crate) struct WitnessL1Provider<'a>(pub(crate) Store<'a>);

#[async_trait]
impl ChainProvider for WitnessL1Provider<'_> {
    type Error = WitnessError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, WitnessError> {
        Ok(self.0.header(hash)?)
    }

    async fn block_info_by_number(&mut self, _: u64) -> Result<BlockInfo, WitnessError> {
        Err(WitnessError("L1 lookup by number is not authenticated".into()))
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, WitnessError> {
        let header = self.0.header(hash)?;
        let walker = OrderedListWalker::try_new_hydrated(header.receipts_root, &self.0)
            .map_err(|e| WitnessError(format!("L1 receipts trie of {hash}: {e}")))?;
        walker
            .into_iter()
            .map(|(_, raw)| {
                let envelope = ReceiptEnvelope::decode_2718_exact(raw.as_ref())
                    .map_err(|e| WitnessError(format!("L1 receipt encoding: {e}")))?;
                Ok(envelope
                    .as_receipt()
                    .ok_or_else(|| WitnessError("L1 receipt type".into()))?
                    .clone())
            })
            .collect()
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        _: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), WitnessError> {
        Err(WitnessError("L1 transactions are not part of the relation witness".into()))
    }
}

/// The private L2 side: only the system config of the block currently being extended. The
/// relation builds one per block, from the parent it just executed.
#[derive(Debug, Clone)]
pub(crate) struct RelationL2Provider {
    parent: B256,
    config: SystemConfig,
}

impl RelationL2Provider {
    pub(crate) const fn new(parent: B256, config: SystemConfig) -> Self {
        Self { parent, config }
    }
}

#[async_trait]
impl BatchValidationProvider for RelationL2Provider {
    type Error = WitnessError;

    async fn l2_block_info_by_number(&mut self, _: u64) -> Result<L2BlockInfo, WitnessError> {
        Err(WitnessError("private L2 lookup by number".into()))
    }

    async fn l2_block_info_by_hash(&mut self, _: B256) -> Result<L2BlockInfo, WitnessError> {
        Err(WitnessError("private L2 lookup by hash".into()))
    }

    async fn block_by_number(&mut self, _: u64) -> Result<Arc<OpBlock>, WitnessError> {
        Err(WitnessError("private L2 block lookup".into()))
    }
}

#[async_trait]
impl L2ChainProvider for RelationL2Provider {
    type Error = WitnessError;

    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
        _: Arc<RollupConfig>,
    ) -> Result<SystemConfig, WitnessError> {
        if hash != self.parent {
            return Err(WitnessError(format!("system config requested for non-parent {hash}")));
        }
        Ok(self.config)
    }
}
