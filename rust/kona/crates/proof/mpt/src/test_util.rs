//! Testing utilities for `kona-mpt`

use crate::{TrieNode, TrieProvider, ordered_trie_with_encoder};
use alloc::{collections::BTreeMap, vec::Vec};
use alloy_consensus::{Receipt, ReceiptEnvelope, ReceiptWithBloom, TxEnvelope, TxType};
use alloy_primitives::{B256, Bytes, Log, keccak256};
use alloy_provider::{Provider, ProviderBuilder, network::eip2718::Encodable2718};
use alloy_rlp::Decodable;
use alloy_rpc_types::BlockTransactions;
use reqwest::Url;

const RPC_URL: &str = "https://docs-demo.quiknode.pro/";

#[derive(thiserror::Error, Debug, Eq, PartialEq)]
#[error("TestTrieProviderError: {0}")]
pub(crate) struct TestTrieProviderError(&'static str);

/// Grabs a live merkleized receipts list within a block header.
pub(crate) async fn get_live_derivable_receipts_list()
-> Result<(B256, BTreeMap<B256, Bytes>, Vec<ReceiptEnvelope>), TestTrieProviderError> {
    // Initialize the provider.
    let provider =
        ProviderBuilder::new().connect_http(Url::parse(RPC_URL).expect("invalid rpc url"));

    let block_number = 19005266;
    let block = provider
        .get_block(block_number.into())
        .full()
        .await
        .map_err(|_| TestTrieProviderError("Missing block"))?
        .ok_or(TestTrieProviderError("Missing block"))?;
    let receipts = provider
        .get_block_receipts(block_number.into())
        .await
        .map_err(|_| TestTrieProviderError("Missing receipts"))?
        .ok_or(TestTrieProviderError("Missing receipts"))?;

    let consensus_receipts = receipts
        .into_iter()
        .map(|r| {
            let rpc_receipt = r.inner.as_receipt_with_bloom().expect("Infallible");
            let consensus_receipt = ReceiptWithBloom::new(
                Receipt {
                    status: rpc_receipt.receipt.status,
                    cumulative_gas_used: rpc_receipt.receipt.cumulative_gas_used,
                    logs: rpc_receipt
                        .receipt
                        .logs
                        .iter()
                        .map(|l| Log { address: l.address(), data: l.data().clone() })
                        .collect(),
                },
                rpc_receipt.logs_bloom,
            );

            match r.transaction_type() {
                TxType::Legacy => ReceiptEnvelope::Legacy(consensus_receipt),
                TxType::Eip2930 => ReceiptEnvelope::Eip2930(consensus_receipt),
                TxType::Eip1559 => ReceiptEnvelope::Eip1559(consensus_receipt),
                TxType::Eip4844 => ReceiptEnvelope::Eip4844(consensus_receipt),
                TxType::Eip7702 => ReceiptEnvelope::Eip7702(consensus_receipt),
            }
        })
        .collect::<Vec<_>>();

    // Compute the derivable list
    let mut list =
        ordered_trie_with_encoder(consensus_receipts.as_ref(), |rlp: &ReceiptEnvelope, buf| {
            rlp.encode_2718(buf)
        });
    let root = list.root();

    // Sanity check receipts root is correct
    assert_eq!(block.header.receipts_root, root);

    // Construct the mapping of hashed intermediates -> raw intermediates
    let preimages = list.take_proof_nodes().into_inner().into_iter().fold(
        BTreeMap::default(),
        |mut acc, (_, value)| {
            acc.insert(keccak256(value.as_ref()), value);
            acc
        },
    );

    Ok((root, preimages, consensus_receipts))
}

/// Grabs a live merkleized transactions list within a block header.
pub(crate) async fn get_live_derivable_transactions_list()
-> Result<(B256, BTreeMap<B256, Bytes>, Vec<TxEnvelope>), TestTrieProviderError> {
    // Initialize the provider.
    let provider =
        ProviderBuilder::new().connect_http(Url::parse(RPC_URL).expect("invalid rpc url"));

    let block_number = 19005266;
    let block = provider
        .get_block(block_number.into())
        .full()
        .await
        .map_err(|_| TestTrieProviderError("Missing block"))?
        .ok_or(TestTrieProviderError("Missing block"))?;

    let BlockTransactions::Full(txs) = block.transactions else {
        return Err(TestTrieProviderError("Did not fetch full block"));
    };
    let consensus_txs = txs.into_iter().map(TxEnvelope::from).collect::<Vec<_>>();

    // Compute the derivable list
    let mut list = ordered_trie_with_encoder(consensus_txs.as_ref(), |rlp: &TxEnvelope, buf| {
        rlp.encode_2718(buf)
    });
    let root = list.root();

    // Sanity check transaction root is correct
    assert_eq!(block.header.transactions_root, root);

    // Construct the mapping of hashed intermediates -> raw intermediates
    let preimages = list.take_proof_nodes().into_inner().into_iter().fold(
        BTreeMap::default(),
        |mut acc, (_, value)| {
            acc.insert(keccak256(value.as_ref()), value);
            acc
        },
    );

    Ok((root, preimages, consensus_txs))
}

/// A mock [`TrieProvider`] for testing that serves in-memory preimages.
pub(crate) struct TrieNodeProvider {
    preimages: BTreeMap<B256, Bytes>,
}

impl TrieNodeProvider {
    pub(crate) const fn new(preimages: BTreeMap<B256, Bytes>) -> Self {
        Self { preimages }
    }
}

impl TrieProvider for TrieNodeProvider {
    type Error = TestTrieProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, TestTrieProviderError> {
        TrieNode::decode(
            &mut self
                .preimages
                .get(&key)
                .cloned()
                .ok_or(TestTrieProviderError("key not found in trie"))?
                .as_ref(),
        )
        .map_err(|_| TestTrieProviderError("failed to decode trie node"))
    }
}
