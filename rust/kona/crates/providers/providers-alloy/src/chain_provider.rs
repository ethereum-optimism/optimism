//! L1 derivation adapter over a cached consensus-data source.
use crate::{Cached, ReceiptSource, RpcSource, SourceError};
use alloy_consensus::{Header, Receipt, ReceiptEnvelope, TxEnvelope};
use alloy_primitives::B256;
use alloy_provider::{RootProvider, network::Ethereum};
use alloy_transport_http::reqwest;
use async_trait::async_trait;
use kona_derive::ChainProvider;
use kona_protocol::BlockInfo;

/// An RPC-backed L1 derivation provider. Caching happens after the source accepts each response.
#[derive(Debug, Clone)]
pub struct AlloyChainProvider<S: ReceiptSource = Cached<RpcSource<Ethereum>>> {
    source: S,
}

impl<S: ReceiptSource<Transaction = TxEnvelope>> AlloyChainProvider<S> {
    /// Creates an L1 provider over the selected source and wrappers.
    pub const fn new(source: S) -> Self {
        Self { source }
    }

    /// Returns the latest block number.
    pub async fn latest_block_number(&mut self) -> Result<u64, SourceError> {
        crate::BlockSource::latest_block_number(&mut self.source).await
    }

    /// Returns the chain ID.
    pub async fn chain_id(&mut self) -> Result<u64, SourceError> {
        crate::BlockSource::chain_id(&mut self.source).await
    }
}

impl AlloyChainProvider {
    /// Creates a trusting provider from a URL.
    ///
    /// # Panics
    /// Panics if `cache_size` is zero.
    pub fn new_http(url: reqwest::Url, cache_size: usize) -> Self {
        Self::new(Cached::new(RpcSource::new(RootProvider::new_http(url)), cache_size))
    }
}

#[async_trait]
impl<S: ReceiptSource<Transaction = TxEnvelope>> ChainProvider for AlloyChainProvider<S> {
    type Error = SourceError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        crate::BlockSource::header_by_hash(&mut self.source, hash).await
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        let (_, header) = crate::BlockSource::header_by_number(&mut self.source, number).await?;
        Ok(BlockInfo {
            hash: header.hash_slow(),
            number,
            parent_hash: header.parent_hash,
            timestamp: header.timestamp,
        })
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        Ok(self
            .source
            .receipts_by_hash(hash)
            .await?
            .into_iter()
            .map(ReceiptEnvelope::into_receipt)
            .collect())
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        let block = crate::BlockSource::block_by_hash(&mut self.source, hash).await?;
        Ok((
            BlockInfo {
                hash,
                number: block.header.number,
                parent_hash: block.header.parent_hash,
                timestamp: block.header.timestamp,
            },
            block.body.transactions.clone(),
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::AlloyChainProviderError;
    use alloy_eips::eip2718::Eip2718Error;
    use alloy_transport::{RpcError, TransportErrorKind};
    use kona_derive::{PipelineErrorKind, ResetError};
    use std::sync::atomic::Ordering;

    fn trusted_provider(inner: RootProvider, cache_size: usize) -> AlloyChainProvider {
        AlloyChainProvider::new(Cached::new(RpcSource::new(inner), cache_size))
    }

    use alloy_consensus::{Eip658Value, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, Bloom, Bytes, Log};
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use alloy_transport::mock::Asserter;
    use httpmock::prelude::*;

    type EthereumBlock =
        <alloy_provider::network::Ethereum as alloy_provider::Network>::BlockResponse;

    /// Builds an RPC block with the requested transaction metadata.
    fn rpc_block(
        hash: B256,
        transaction_hashes: Vec<B256>,
        transactions_root: B256,
    ) -> EthereumBlock {
        let header = Header { transactions_root, ..Default::default() };
        let mut rpc_header = RpcHeader::new(header);
        // Tests use a trusted provider, so the reported hash may be set independently of the test
        // header's computed hash.
        rpc_header.hash = hash;
        RpcBlock::new(rpc_header, BlockTransactions::Hashes(transaction_hashes))
    }

    /// Creates a chain provider backed by a FIFO mock RPC transport.
    fn mocked_provider(asserter: Asserter) -> AlloyChainProvider {
        trusted_provider(RootProvider::new(RpcClient::mocked(asserter)), 8)
    }

    #[test]
    fn test_from_alloy_chain_provider_error() {
        // Transport errors are transient — retry makes sense.
        let transport_err =
            AlloyChainProviderError::Transport(alloy_transport::RpcError::Transport(
                alloy_transport::TransportErrorKind::Custom("timeout".into()),
            ));
        let kind: PipelineErrorKind = transport_err.into();
        assert!(matches!(kind, PipelineErrorKind::Temporary(_)));

        // Hash-based BlockNotFound: the block was reorged out. Retrying will never succeed
        // — the pipeline must reset. Without this, the safe head stalls on L1 reorgs.
        let kind: PipelineErrorKind =
            AlloyChainProviderError::BlockNotFound(B256::default().into()).into();
        assert!(
            matches!(kind, PipelineErrorKind::Reset(_)),
            "hash-based BlockNotFound must map to Reset (block reorged out)"
        );

        // Number-based BlockNotFound: the next L1 block hasn't been mined yet. This is
        // transient — the pipeline must wait, not reset.
        let kind: PipelineErrorKind = AlloyChainProviderError::BlockNotFound(0u64.into()).into();
        assert!(
            matches!(kind, PipelineErrorKind::Temporary(_)),
            "number-based BlockNotFound must stay Temporary (block not yet produced)"
        );
    }

    #[test]
    fn receipt_fallback_determines_derivation_impact() {
        let missing_hash = B256::repeat_byte(0xaa);
        let temporary = AlloyChainProviderError::ReceiptMethodsFailed {
            raw: Box::new(AlloyChainProviderError::BlockNotFound(missing_hash.into())),
            fallback: Box::new(AlloyChainProviderError::Transport(RpcError::Transport(
                TransportErrorKind::Custom("fallback timeout".into()),
            ))),
        };
        let message = temporary.to_string();
        assert!(message.contains("Block not found"));
        assert!(message.contains("fallback timeout"));
        assert!(matches!(PipelineErrorKind::from(temporary), PipelineErrorKind::Temporary(_)));

        let reset = AlloyChainProviderError::ReceiptMethodsFailed {
            raw: Box::new(AlloyChainProviderError::Transport(RpcError::Transport(
                TransportErrorKind::Custom("raw timeout".into()),
            ))),
            fallback: Box::new(AlloyChainProviderError::BlockNotFound(missing_hash.into())),
        };
        assert!(matches!(
            PipelineErrorKind::from(reset),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == missing_hash.into()
        ));
    }

    #[tokio::test]
    async fn receipts_by_hash_prefers_raw_receipts() {
        let expected = Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: Vec::new(),
        };
        let envelope =
            ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(expected.clone(), Bloom::ZERO));
        let raw_receipt = Bytes::from(envelope.encoded_2718());
        let response = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 0,
            "result": [raw_receipt],
        });

        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(response);
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(500);
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert_eq!(provider.receipts_by_hash(B256::repeat_byte(0xaa)).await.unwrap(), [expected]);
        raw_mock.assert();
        assert_eq!(fallback_mock.calls(), 0);
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_unknown_hash_reset_without_fallback() {
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Option::<EthereumBlock>::None);
        let mut provider = mocked_provider(asserter.clone());
        let hash = B256::repeat_byte(0xab);

        let error = provider.receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(
            &error,
            AlloyChainProviderError::BlockNotFound(id) if *id == hash.into()
        ));
        assert!(matches!(
            PipelineErrorKind::from(error),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == hash.into()
        ));
        assert!(
            asserter.read_q().is_empty(),
            "a confirmed missing block must reset without requesting fallback receipts"
        );
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_existing_empty_block_are_accepted() {
        let hash = B256::repeat_byte(0xac);
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Some(rpc_block(hash, Vec::new(), alloy_consensus::EMPTY_ROOT_HASH)));
        let mut provider = mocked_provider(asserter.clone());

        assert!(provider.receipts_by_hash(hash).await.unwrap().is_empty());
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_nonempty_block_are_rejected() {
        let hash = B256::repeat_byte(0xad);
        let transactions_root = B256::repeat_byte(0x11);
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Some(rpc_block(
            hash,
            vec![B256::repeat_byte(0x22)],
            transactions_root,
        )));
        let provider = mocked_provider(asserter.clone());

        let error = provider.source.inner.raw_receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(
            error,
            AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                hash: error_hash,
                transaction_count: 1,
                transactions_root: error_root,
            } if error_hash == hash && error_root == transactions_root
        ));
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_nonempty_block_preserve_json_fallback() {
        let hash = B256::repeat_byte(0xaf);
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Some(rpc_block(
            hash,
            vec![B256::repeat_byte(0x22)],
            B256::repeat_byte(0x11),
        )));
        asserter.push_success(&Vec::<serde_json::Value>::new());
        let mut provider = mocked_provider(asserter.clone());
        assert!(provider.receipts_by_hash(hash).await.unwrap().is_empty());
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn raw_receipt_decode_error_preserves_source_and_index() {
        let asserter = Asserter::new();
        asserter.push_success(&vec![Bytes::from_static(&[0x05])]);
        let provider = mocked_provider(asserter.clone());
        let hash = B256::repeat_byte(0xae);

        let error = provider.source.inner.raw_receipts_by_hash(hash).await.unwrap_err();
        match error {
            AlloyChainProviderError::RawReceiptDecode { hash: error_hash, index, source } => {
                assert_eq!(error_hash, hash);
                assert_eq!(index, 0);
                assert!(matches!(source, Eip2718Error::RlpError(_)));
            }
            other => panic!("expected typed raw receipt decode error, got {other:?}"),
        }
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn method_not_found_makes_fallback_sticky_across_clones() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32601, "message": "Method not found" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);
        let mut provider_clone = provider.clone();

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xba)).await.unwrap().is_empty());
        assert!(provider_clone.receipts_by_hash(B256::repeat_byte(0xbb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1, "the clone must share the sticky fallback state");
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
        assert!(!provider_clone.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn method_not_found_in_http_error_body_makes_fallback_sticky() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(403).header("content-type", "application/json").body(
                r#"{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}"#,
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn method_not_found_stays_sticky_when_fallback_fails() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32601, "message": "Method not found" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": { "code": -32000, "message": "fallback unavailable" },
                }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        let first_error = provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap_err();
        assert!(matches!(first_error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(matches!(PipelineErrorKind::from(first_error), PipelineErrorKind::Temporary(_)));

        let second_error = provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap_err();
        assert!(matches!(second_error, AlloyChainProviderError::Transport(_)));
        assert!(matches!(PipelineErrorKind::from(second_error), PipelineErrorKind::Temporary(_)));
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn other_raw_errors_fall_back_without_becoming_sticky() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "temporary raw receipt failure" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xca)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xcb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 2);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(provider.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn raw_not_found_uses_successful_fallback() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 0, "result": null }));
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xda)).await.unwrap().is_empty());
        raw_mock.assert();
        fallback_mock.assert();
        assert!(provider.source.inner.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn both_method_errors_are_temporary_and_preserve_both_causes() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": { "code": -32001, "message": "JSON receipts unavailable" },
                }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        let error = provider.receipts_by_hash(B256::repeat_byte(0xea)).await.unwrap_err();
        assert!(matches!(error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(error.to_string().contains("raw receipts unavailable"));
        assert!(error.to_string().contains("JSON receipts unavailable"));

        let pipeline_error = PipelineErrorKind::from(error);
        assert!(matches!(pipeline_error, PipelineErrorKind::Temporary(_)));
        assert!(pipeline_error.to_string().contains("raw receipts unavailable"));
        assert!(pipeline_error.to_string().contains("JSON receipts unavailable"));
        raw_mock.assert();
        fallback_mock.assert();
    }

    #[tokio::test]
    async fn fallback_not_found_preserves_reorg_signal() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": null }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);
        let hash = B256::repeat_byte(0xfa);

        let error = provider.receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(matches!(
            PipelineErrorKind::from(error),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == hash.into()
        ));
        raw_mock.assert();
        fallback_mock.assert();
    }

    #[tokio::test]
    async fn fallback_converts_json_receipts_in_order() {
        let block_hash = B256::repeat_byte(0xdb);
        let tx_hashes = [B256::repeat_byte(0x01), B256::repeat_byte(0x02), B256::repeat_byte(0x03)];
        let logs = [
            Log::new_unchecked(
                Address::repeat_byte(0x10),
                vec![B256::repeat_byte(0x20), B256::repeat_byte(0x21)],
                Bytes::from_static(&[0x01, 0x02]),
            ),
            Log::new_unchecked(
                Address::repeat_byte(0x11),
                vec![B256::repeat_byte(0x22)],
                Bytes::new(),
            ),
            Log::new_unchecked(Address::repeat_byte(0x12), Vec::new(), Bytes::from_static(&[0xff])),
        ];
        let expected = vec![
            Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 46_000,
                logs: logs[..2].to_vec(),
            },
            Receipt {
                status: Eip658Value::Eip658(false),
                cumulative_gas_used: 67_000,
                logs: Vec::new(),
            },
            Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 112_000,
                logs: logs[2..].to_vec(),
            },
        ];

        let mut log_index = 0;
        let json_receipts: Vec<_> = expected
            .iter()
            .zip(["0x2", "0x0", "0x2"])
            .enumerate()
            .map(|(tx_index, (receipt, tx_type))| {
                let json_logs: Vec<_> = receipt
                    .logs
                    .iter()
                    .map(|log| {
                        log_index += 1;
                        serde_json::json!({
                            "address": log.address,
                            "topics": log.topics(),
                            "data": log.data.data,
                            "blockHash": block_hash,
                            "blockNumber": "0x10",
                            "transactionHash": tx_hashes[tx_index],
                            "transactionIndex": format!("{tx_index:#x}"),
                            "logIndex": format!("{:#x}", log_index - 1),
                            "removed": false,
                        })
                    })
                    .collect();
                serde_json::json!({
                    "type": tx_type,
                    "status": if receipt.status.coerce_status() { "0x1" } else { "0x0" },
                    "cumulativeGasUsed": format!("{:#x}", receipt.cumulative_gas_used),
                    "logs": json_logs,
                    "logsBloom": Bloom::ZERO,
                    "transactionHash": tx_hashes[tx_index],
                    "transactionIndex": format!("{tx_index:#x}"),
                    "blockHash": block_hash,
                    "blockNumber": "0x10",
                    "gasUsed": "0x5208",
                    "effectiveGasPrice": "0x3b9aca00",
                    "from": Address::repeat_byte(0x30),
                    "to": Address::repeat_byte(0x31),
                    "contractAddress": null,
                })
            })
            .collect();

        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": json_receipts }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert_eq!(provider.receipts_by_hash(block_hash).await.unwrap(), expected);
        raw_mock.assert();
        fallback_mock.assert();
    }
    #[tokio::test]
    async fn headers_are_checked_before_entering_the_cache() {
        let header = Header::default();
        let hash = header.hash_slow();
        let valid: EthereumBlock =
            RpcBlock::new(RpcHeader::new(header.clone()), BlockTransactions::Hashes(vec![]));
        let mut invalid = valid.clone();
        invalid.header.inner.number = 1;
        let asserter = Asserter::new();
        asserter.push_success(&invalid);
        asserter.push_success(&valid);
        let mut provider = AlloyChainProvider::new(Cached::new(
            RpcSource::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter.clone())),
                false,
            ),
            8,
        ));
        assert!(provider.header_by_hash(hash).await.is_err());
        assert!(provider.source.headers.is_empty());
        assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
        assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
        assert!(asserter.read_q().is_empty());
    }
}
