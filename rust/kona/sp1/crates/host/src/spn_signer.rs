//! Remote SPN requester backed by op-signer.

use alloy_primitives::{Address, B256, Bytes, ChainId, Signature};
use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_transport_http::{Http, reqwest::Url};
use anyhow::{Result, ensure};
use async_trait::async_trait;
use serde::Serialize;
use sp1_alloy_signer::{Error, Signer, UnsupportedSignerOperation};

use crate::tls::ClientTls;

/// An SPN requester that delegates EIP-191 message signing to op-signer over mTLS.
#[derive(Clone, Debug)]
pub struct OpSignerRequester {
    client: RpcClient,
    address: Address,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SignMessageArgs {
    message: Bytes,
    sender_address: Address,
}

impl OpSignerRequester {
    /// Builds a requester using the configured client certificate and server CA.
    pub fn new(endpoint: Url, address: Address, tls: ClientTls) -> Result<Self> {
        ensure!(endpoint.scheme() == "https", "SPN op-signer endpoint must use https");
        let transport = Http::with_client(tls.http_client()?, endpoint);
        let client = ClientBuilder::default().transport(transport, false);
        Ok(Self { client, address })
    }
}

#[async_trait]
impl Signer for OpSignerRequester {
    async fn sign_hash(&self, _hash: &B256) -> sp1_alloy_signer::Result<Signature> {
        Err(Error::UnsupportedOperation(UnsupportedSignerOperation::SignHash))
    }

    async fn sign_message(&self, message: &[u8]) -> sp1_alloy_signer::Result<Signature> {
        let args = SignMessageArgs {
            message: Bytes::copy_from_slice(message),
            sender_address: self.address,
        };
        let response: Bytes =
            self.client.request("opsigner_signMessage", (args,)).await.map_err(Error::message)?;
        if response.len() != 65 {
            return Err(Error::message(format!(
                "op-signer returned a {}-byte signature",
                response.len()
            )));
        }
        Signature::from_raw(response.as_ref()).map_err(Error::from)
    }

    fn address(&self) -> Address {
        self.address
    }

    fn chain_id(&self) -> Option<ChainId> {
        None
    }

    fn set_chain_id(&mut self, _chain_id: Option<ChainId>) {}
}
