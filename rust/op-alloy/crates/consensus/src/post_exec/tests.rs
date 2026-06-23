//! Tests for post-exec transaction types and the shared post-exec payload structure check.

use super::*;

#[test]
fn post_exec_payload_rlp_roundtrip_preserves_block_number() {
    let payload = PostExecPayload {
        version: 1,
        block_number: 42,
        gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
    };

    let encoded = payload.to_rlp_bytes();
    let decoded = PostExecPayload::from_rlp_bytes(encoded.as_ref()).expect("decode payload");

    assert_eq!(decoded, payload);
}

#[test]
fn post_exec_payload_rlp_decode_rejects_unknown_version() {
    let payload = PostExecPayload {
        version: POST_EXEC_PAYLOAD_VERSION + 1,
        block_number: 42,
        gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
    };

    let encoded = payload.to_rlp_bytes();
    let err =
        PostExecPayload::from_rlp_bytes(encoded.as_ref()).expect_err("reject unknown version");
    assert_eq!(err, alloy_rlp::Error::Custom("unsupported post-exec payload version"));
}

#[test]
fn post_exec_payload_rlp_decode_rejects_trailing_bytes() {
    let payload = PostExecPayload {
        version: 1,
        block_number: 42,
        gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
    };

    let mut encoded = payload.to_rlp_bytes().to_vec();
    encoded.push(0);

    let err = PostExecPayload::from_rlp_bytes(&encoded).expect_err("reject trailing bytes");
    assert_eq!(err, alloy_rlp::Error::UnexpectedLength);
}

#[test]
fn post_exec_tx_hash_depends_on_block_number() {
    let entries = vec![SDMGasEntry { index: 3, gas_refund: 7 }];
    let tx_a = build_post_exec_tx(42, entries.clone());
    let tx_b = build_post_exec_tx(43, entries);

    assert_ne!(tx_a.tx_hash(), tx_b.tx_hash());
}

#[test]
fn post_exec_tx_eip2718_roundtrip() {
    let tx = build_post_exec_tx(
        99,
        vec![SDMGasEntry { index: 0, gas_refund: 100 }, SDMGasEntry { index: 5, gas_refund: 200 }],
    );

    let mut buf = Vec::new();
    tx.encode_2718(&mut buf);

    let decoded = TxPostExec::decode_2718(&mut buf.as_slice()).expect("decode 2718");
    assert_eq!(decoded, tx);
    assert_eq!(decoded.tx_hash(), tx.tx_hash());
}

#[test]
fn post_exec_tx_eip2718_decode_rejects_unknown_version() {
    let payload = PostExecPayload {
        version: POST_EXEC_PAYLOAD_VERSION + 1,
        block_number: 42,
        gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
    };

    let mut buf = Vec::new();
    buf.put_u8(POST_EXEC_TX_TYPE_ID);
    payload.encode(&mut buf);

    let err = TxPostExec::decode_2718(&mut buf.as_slice())
        .expect_err("2718 decode must reject unknown version");
    assert!(
        matches!(
            err,
            Eip2718Error::RlpError(alloy_rlp::Error::Custom(
                "unsupported post-exec payload version"
            ))
        ),
        "unexpected error: {err:?}"
    );
}

#[test]
fn post_exec_tx_rlp_decode_rejects_unknown_version() {
    let payload = PostExecPayload {
        version: POST_EXEC_PAYLOAD_VERSION + 1,
        block_number: 42,
        gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
    };
    let mut buf = Vec::new();
    payload.encode(&mut buf);

    let err = TxPostExec::decode(&mut buf.as_slice())
        .expect_err("rlp decode must reject unknown version");
    assert_eq!(err, alloy_rlp::Error::Custom("unsupported post-exec payload version"));
}

#[test]
fn post_exec_tx_eip2718_roundtrip_empty_refunds() {
    let tx = build_post_exec_tx(1, vec![]);

    let mut buf = Vec::new();
    tx.encode_2718(&mut buf);

    let decoded = TxPostExec::decode_2718(&mut buf.as_slice()).expect("decode 2718");
    assert_eq!(decoded, tx);
}

#[cfg(feature = "serde")]
#[test]
fn post_exec_tx_serde_serializes_as_payload() {
    let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
    let value = serde_json::to_value(&tx).expect("serialize tx");

    assert_eq!(value, serde_json::to_value(&tx.payload).expect("serialize payload"));
}

#[cfg(feature = "serde")]
#[test]
fn post_exec_tx_serde_roundtrip_preserves_cached_input() {
    let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
    let value = serde_json::to_value(&tx).expect("serialize tx");

    let decoded: TxPostExec = serde_json::from_value(value).expect("deserialize tx");
    assert_eq!(decoded, tx);
    assert_eq!(decoded.input, decoded.payload.to_rlp_bytes());
}

#[cfg(feature = "serde")]
#[test]
fn post_exec_envelope_rpc_serde_roundtrip() {
    use crate::OpTxEnvelope;

    let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
    let envelope = OpTxEnvelope::PostExec(tx.seal_slow());

    // Serializes via `serde_post_exec_tx_rpc` (the RPC `{hash,type,gas,value,input}` shape)
    // and must deserialize back through `serde_post_exec_tx_rpc_de`.
    let value = serde_json::to_value(&envelope).expect("serialize envelope");
    let decoded: OpTxEnvelope = serde_json::from_value(value).expect("deserialize envelope");

    assert_eq!(decoded, envelope);
}

#[cfg(feature = "serde")]
#[test]
fn post_exec_envelope_rpc_deserialize_ignores_extra_fields() {
    use crate::OpTxEnvelope;

    // Mirrors the object op-geth emits in `eth_getBlockByHash` full-transaction responses:
    // the canonical data lives in `input`, with derived placeholder/metadata fields alongside.
    let tx = build_post_exec_tx(27, vec![SDMGasEntry { index: 1, gas_refund: 9 }]);
    let sealed = tx.clone().seal_slow();
    let json = serde_json::json!({
        "type": "0x7d",
        "hash": sealed.hash(),
        "gas": "0x0",
        "value": "0x0",
        "input": tx.input,
        "from": "0x0000000000000000000000000000000000000000",
        "gasPrice": "0x358a0487",
        "blockHash": "0xab6dfcf5ad132602e939db5f84f56945b1be4e136daab002f9bf9ff0c7f9f7a5",
        "blockNumber": "0x1b",
        "transactionIndex": "0x11",
        "blockTimestamp": "0x6a19f379",
    });

    let decoded: OpTxEnvelope = serde_json::from_value(json).expect("deserialize rpc tx");
    let OpTxEnvelope::PostExec(decoded) = decoded else {
        panic!("expected post-exec envelope, got {decoded:?}");
    };
    assert_eq!(decoded.inner(), &tx);
    assert_eq!(decoded.hash(), sealed.hash());
}

// Coverage for `parse_post_exec_payload_from_transactions`, the shared consensus structure
// check used by op-reth's payload builder, the kona executor and the post-exec replay path.
// Each test pins one of the four structural rules (and the two accepting cases).
const PARSE_BLOCK: u64 = 100;

/// A non-post-exec filler tx. A deposit is the cheapest envelope to construct and exercises the
/// `as_post_exec() -> None` branch the same way any normal tx would.
fn filler_tx() -> crate::OpTxEnvelope {
    crate::OpTxEnvelope::Deposit(crate::TxDeposit::default().seal_slow())
}

fn post_exec_tx(block_number: u64) -> crate::OpTxEnvelope {
    crate::OpTxEnvelope::PostExec(
        build_post_exec_tx(block_number, vec![SDMGasEntry { index: 1, gas_refund: 9 }]).seal_slow(),
    )
}

#[test]
fn parse_accepts_trailing_post_exec_tx() {
    let txs = vec![filler_tx(), filler_tx(), post_exec_tx(PARSE_BLOCK)];

    let parsed = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, true)
        .expect("a trailing post-exec tx is valid")
        .expect("a post-exec tx is present");

    assert_eq!(parsed.tx_index, 2);
    assert_eq!(parsed.payload.block_number, PARSE_BLOCK);
    assert_eq!(parsed.payload.gas_refund_entries, vec![SDMGasEntry { index: 1, gas_refund: 9 }]);
}

#[test]
fn parse_returns_none_without_post_exec_tx() {
    let txs = vec![filler_tx(), filler_tx()];

    // Absence of a post-exec tx is not an error in any activation state.
    for sdm_active in [false, true] {
        let parsed = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, sdm_active)
            .expect("no post-exec tx is always valid");
        assert_eq!(parsed, None, "sdm_active={sdm_active}");
    }

    // An empty block likewise yields no payload.
    let empty: Vec<crate::OpTxEnvelope> = vec![];
    assert_eq!(
        parse_post_exec_payload_from_transactions(&empty, PARSE_BLOCK, true)
            .expect("an empty block is valid"),
        None,
    );
}

#[test]
fn parse_rejects_post_exec_tx_when_sdm_inactive() {
    let txs = vec![filler_tx(), post_exec_tx(PARSE_BLOCK)];

    let err = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, false)
        .expect_err("a post-exec tx before SDM activation must be rejected");
    assert_eq!(err, PostExecPayloadValidationError::UnexpectedPostExecTx { tx_index: 1 });
}

#[test]
fn parse_rejects_multiple_post_exec_txs() {
    let txs = vec![filler_tx(), post_exec_tx(PARSE_BLOCK), post_exec_tx(PARSE_BLOCK)];

    let err = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, true)
        .expect_err("a block with two post-exec txs must be rejected");
    assert_eq!(
        err,
        PostExecPayloadValidationError::MultiplePostExecTxs { first_index: 1, duplicate_index: 2 },
    );
}

#[test]
fn parse_rejects_post_exec_tx_not_last() {
    // The post-exec tx must be the final transaction; a normal tx after it is invalid.
    let txs = vec![filler_tx(), post_exec_tx(PARSE_BLOCK), filler_tx()];

    let err = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, true)
        .expect_err("a mid-block post-exec tx must be rejected");
    assert_eq!(
        err,
        PostExecPayloadValidationError::PostExecTxNotLast { tx_index: 1, last_index: 2 },
    );
}

#[test]
fn parse_rejects_payload_anchored_to_wrong_block() {
    let txs = vec![filler_tx(), post_exec_tx(PARSE_BLOCK + 1)];

    let err = parse_post_exec_payload_from_transactions(&txs, PARSE_BLOCK, true)
        .expect_err("a payload anchored to another block must be rejected");
    assert_eq!(
        err,
        PostExecPayloadValidationError::BlockNumberMismatch {
            payload_block_number: PARSE_BLOCK + 1,
            block_number: PARSE_BLOCK,
        },
    );
}
