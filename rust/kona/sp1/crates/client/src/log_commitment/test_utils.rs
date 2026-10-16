//! Receipt fixtures shared by the native and SP1 log-commitment tests.

use alloy_consensus::{Receipt, ReceiptWithBloom, proofs::ordered_trie_root_with_encoder};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, Log, LogData, keccak256};
use kona_preimage::PreimageKey;
use op_alloy_consensus::{OpDepositReceipt, OpDepositReceiptWithBloom, OpReceiptEnvelope};

use super::BlockReceipts;
use crate::witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore};

/// `count` logs with distinct addresses, 0 to 4 topics, and 0 to 63 data bytes.
pub fn logs(first_seed: u64, count: u64) -> Vec<Log> {
    (first_seed..first_seed + count)
        .map(|seed| Log {
            address: Address::left_padding_from(&seed.to_be_bytes()),
            data: LogData::new_unchecked(
                (0..seed % 5).map(|topic| B256::repeat_byte(topic as u8 + 1)).collect(),
                Bytes::from(vec![seed as u8; (seed * 7 % 70) as usize]),
            ),
        })
        .collect()
}

fn with_bloom(logs: Vec<Log>) -> ReceiptWithBloom<Receipt> {
    ReceiptWithBloom {
        receipt: Receipt { status: true.into(), cumulative_gas_used: 21_000, logs },
        logs_bloom: Default::default(),
    }
}

/// A deposit receipt with the given nonce and receipt version, or with neither.
pub fn deposit(logs: Vec<Log>, nonce_and_version: Option<(u64, u64)>) -> OpReceiptEnvelope {
    let receipt = with_bloom(logs);
    OpReceiptEnvelope::Deposit(OpDepositReceiptWithBloom {
        receipt: OpDepositReceipt {
            inner: receipt.receipt,
            deposit_nonce: nonce_and_version.map(|(nonce, _)| nonce),
            deposit_receipt_version: nonce_and_version.map(|(_, version)| version),
        },
        logs_bloom: receipt.logs_bloom,
    })
}

/// An EIP-1559 receipt.
pub fn eip1559(logs: Vec<Log>) -> OpReceiptEnvelope {
    OpReceiptEnvelope::Eip1559(with_bloom(logs))
}

/// The receipts root of `receipts`.
pub fn receipts_root(receipts: &[OpReceiptEnvelope]) -> B256 {
    ordered_trie_root_with_encoder(receipts, |receipt, out| receipt.encode_2718(out))
}

/// Every receipt type, zero-log receipts at each position, a block without logs, the empty
/// receipts root, a single log, and more than `0x80` receipts (item 0 moves in the trie).
pub fn mixed_blocks() -> Vec<BlockReceipts> {
    let blocks = vec![
        vec![
            deposit(vec![], Some((1, 1))),
            OpReceiptEnvelope::Legacy(with_bloom(logs(1, 2))),
            OpReceiptEnvelope::Eip2930(with_bloom(vec![])),
            eip1559(logs(3, 3)),
            OpReceiptEnvelope::Eip7702(with_bloom(logs(6, 1))),
            OpReceiptEnvelope::PostExec(with_bloom(logs(7, 1))),
            deposit(logs(8, 2), None),
            eip1559(vec![]),
        ],
        vec![deposit(vec![], Some((2, 1))), OpReceiptEnvelope::Legacy(with_bloom(vec![]))],
        vec![],
        vec![eip1559(logs(10, 1))],
        (0..130).map(|i| eip1559(logs(1_000 + 3 * i, i % 3))).collect(),
    ];
    blocks
        .into_iter()
        .map(|receipts| BlockReceipts { receipts_root: receipts_root(&receipts), receipts })
        .collect()
}

/// The receipts root of a 20-receipt block, and a witness of its trie without one non-root node.
pub fn missing_trie_node_witness() -> (B256, DefaultWitnessData) {
    let receipts: Vec<_> = (0..20).map(|i| eip1559(logs(i, 1))).collect();
    let mut trie = kona_mpt::ordered_trie_with_encoder(&receipts, |receipt, out| {
        receipt.encode_2718(out);
    });
    let root = trie.root();
    let mut store = PreimageStore::default();
    let mut omitted = false;
    for (path, node) in trie.take_proof_nodes().into_inner() {
        if !omitted && !path.is_empty() {
            omitted = true;
            continue;
        }
        store.save_preimage(PreimageKey::new_keccak256(*keccak256(&node)), node.to_vec()).unwrap();
    }
    assert!(omitted);
    (root, DefaultWitnessData::from_parts(store, BlobData::default()))
}
