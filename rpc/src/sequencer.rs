//! Helpers for optimism specific RPC implementations.

use std::sync::Arc;

use alloy_json_rpc::{RpcRecv, RpcSend};
use alloy_primitives::{hex, B256};
use alloy_rpc_client::{ClientBuilder, RpcClient as Client};
use alloy_rpc_types_eth::erc4337::TransactionConditional;
use alloy_transport_http::Http;
use tracing::warn;

use crate::SequencerClientError;

/// A client to interact with a Sequencer
#[derive(Debug, Clone)]
pub struct SequencerClient {
    inner: Arc<SequencerClientInner>,
}

impl SequencerClient {
    /// Creates a new [`SequencerClient`].
    pub fn new(sequencer_endpoint: impl Into<String>) -> Self {
        let client = reqwest::Client::builder().use_rustls_tls().build().unwrap();
        Self::with_client(sequencer_endpoint, client)
    }

    /// Creates a new [`SequencerClient`].
    pub fn with_client(sequencer_endpoint: impl Into<String>, client: reqwest::Client) -> Self {
        let sequencer_endpoint: String = sequencer_endpoint.into();

        let http_client =
            Http::with_client(client, reqwest::Url::parse(&sequencer_endpoint).unwrap());
        let is_local = http_client.guess_local();
        let http_client = ClientBuilder::default().transport(http_client, is_local);

        let inner = SequencerClientInner { sequencer_endpoint, http_client };
        Self { inner: Arc::new(inner) }
    }

    /// Returns the network of the client
    pub fn endpoint(&self) -> &str {
        &self.inner.sequencer_endpoint
    }

    /// Returns the client
    pub fn http_client(&self) -> &Client {
        &self.inner.http_client
    }

    /// Sends a [`alloy_rpc_client::RpcCall`] request to the sequencer endpoint.
    async fn send_rpc_call<Params: RpcSend, Resp: RpcRecv>(
        &self,
        method: &str,
        params: Params,
    ) -> Result<Resp, SequencerClientError> {
        let resp = self
            .http_client()
            .request::<Params, Resp>(method.to_string(), params)
            .await
            .inspect_err(|err| {
                warn!(
                    target: "rpc::sequencer",
                    %err,
                    "HTTP request to sequencer failed",
                );
            })?;
        Ok(resp)
    }

    /// Forwards a transaction to the sequencer endpoint.
    pub async fn forward_raw_transaction(&self, tx: &[u8]) -> Result<B256, SequencerClientError> {
        let rlp_hex = hex::encode_prefixed(tx);
        let tx_hash =
            self.send_rpc_call("eth_sendRawTransaction", (rlp_hex,)).await.inspect_err(|err| {
                warn!(
                    target: "rpc::eth",
                    %err,
                    "Failed to forward transaction to sequencer",
                );
            })?;

        Ok(tx_hash)
    }

    /// Forwards a transaction conditional to the sequencer endpoint.
    pub async fn forward_raw_transaction_conditional(
        &self,
        tx: &[u8],
        condition: TransactionConditional,
    ) -> Result<B256, SequencerClientError> {
        let rlp_hex = hex::encode_prefixed(tx);
        let tx_hash = self
            .send_rpc_call("eth_sendRawTransactionConditional", (rlp_hex, condition))
            .await
            .inspect_err(|err| {
                warn!(
                    target: "rpc::eth",
                    %err,
                    "Failed to forward transaction conditional for sequencer",
                );
            })?;
        Ok(tx_hash)
    }
}

#[derive(Debug)]
struct SequencerClientInner {
    /// The endpoint of the sequencer
    sequencer_endpoint: String,
    /// The HTTP client
    http_client: Client,
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::U64;

    #[test]
    fn test_body_str() {
        let client = SequencerClient::new("http://localhost:8545");

        let request = client
            .http_client()
            .make_request("eth_getBlockByNumber", (U64::from(10),))
            .serialize()
            .unwrap()
            .take_request();
        let body = request.get();

        assert_eq!(
            body,
            r#"{"method":"eth_getBlockByNumber","params":["0xa"],"id":0,"jsonrpc":"2.0"}"#
        );

        let condition = TransactionConditional::default();

        let request = client
            .http_client()
            .make_request(
                "eth_sendRawTransactionConditional",
                (format!("0x{}", hex::encode("abcd")), condition),
            )
            .serialize()
            .unwrap()
            .take_request();
        let body = request.get();

        assert_eq!(
            body,
            r#"{"method":"eth_sendRawTransactionConditional","params":["0x61626364",{"knownAccounts":{}}],"id":1,"jsonrpc":"2.0"}"#
        );
    }
}
