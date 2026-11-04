//! Consensus fixtures served through the mock RPC transports.

use alloy_consensus::{
    Header, Receipt, ReceiptEnvelope, SignableTransaction, TxEip1559, TxEip2930, TxEip4844,
    TxEip7702, TxEnvelope, TxLegacy,
};
use alloy_primitives::{Address, B256, Bytes, Log, Sealed, Signature};
use alloy_rpc_types_eth::{Block, BlockTransactions, Header as RpcHeader};
use op_alloy_consensus::{OpTxEnvelope, TxDeposit, build_post_exec_tx};
use serde::Serialize;
use serde_json::Value;

pub(crate) fn ethereum_transactions() -> Vec<TxEnvelope> {
    vec![
        TxEip4844 { nonce: 5, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        TxLegacy { nonce: 1, ..Default::default() }.into_signed(Signature::test_signature()).into(),
        TxEip1559 { nonce: 2, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        TxEip2930 { nonce: 3, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        TxEip7702 { nonce: 4, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
    ]
}

pub(crate) fn optimism_transactions() -> Vec<OpTxEnvelope> {
    vec![
        TxLegacy { nonce: 1, ..Default::default() }.into_signed(Signature::test_signature()).into(),
        TxEip1559 { nonce: 2, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        TxEip2930 { nonce: 3, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        TxEip7702 { nonce: 4, ..Default::default() }
            .into_signed(Signature::test_signature())
            .into(),
        OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: Bytes::from_static(&[1, 2, 3]),
            ..Default::default()
        })),
        OpTxEnvelope::PostExec(Sealed::new(build_post_exec_tx(42, vec![]))),
    ]
}

pub(crate) fn rpc_block<T: Serialize>(header: Header, transactions: &[T]) -> Value {
    let block: <alloy_provider::network::Ethereum as alloy_provider::Network>::BlockResponse =
        Block::new(RpcHeader::new(header), BlockTransactions::Full(vec![]));
    let mut value = serde_json::to_value(block).unwrap();
    value["transactions"] = transactions
        .iter()
        .map(|tx| {
            let mut tx = serde_json::to_value(tx).unwrap();
            tx["from"] = serde_json::to_value(Address::ZERO).unwrap();
            tx
        })
        .collect();
    value
}

pub(crate) fn receipts() -> Vec<ReceiptEnvelope> {
    let receipt = Receipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![Log::new_unchecked(
            Address::repeat_byte(1),
            vec![B256::repeat_byte(2)],
            Bytes::from_static(&[3]),
        )],
    };
    vec![
        ReceiptEnvelope::Legacy(receipt.with_bloom()),
        ReceiptEnvelope::Eip1559(
            Receipt { status: false.into(), cumulative_gas_used: 42_000, logs: vec![] }
                .with_bloom(),
        ),
    ]
}

pub(crate) fn json_receipts(receipts: &[ReceiptEnvelope]) -> Vec<Value> {
    receipts
        .iter()
        .enumerate()
        .map(|(index, receipt)| {
            let mut value = serde_json::to_value(receipt).unwrap();
            for (log_index, log) in value["logs"].as_array_mut().unwrap().iter_mut().enumerate() {
                log["blockHash"] = serde_json::to_value(B256::repeat_byte(0xaa)).unwrap();
                log["blockNumber"] = serde_json::json!("0x2a");
                log["transactionHash"] =
                    serde_json::to_value(B256::repeat_byte(index as u8)).unwrap();
                log["transactionIndex"] = serde_json::json!(format!("{index:#x}"));
                log["logIndex"] = serde_json::json!(format!("{log_index:#x}"));
                log["removed"] = serde_json::json!(false);
            }
            for (key, field) in serde_json::json!({
                "blockNumber": "0x2a",
                "blockHash": B256::repeat_byte(0xaa),
                "transactionHash": B256::repeat_byte(index as u8),
                "transactionIndex": format!("{index:#x}"),
                "gasUsed": "0x5208",
                "effectiveGasPrice": "0x1",
                "from": Address::ZERO,
                "to": Address::ZERO,
                "contractAddress": null,
            })
            .as_object()
            .unwrap()
            {
                value[key] = field.clone();
            }
            value
        })
        .collect()
}
