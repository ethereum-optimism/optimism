//! Remote SPN requester backed by op-signer.

use alloy_primitives::{Address, B256, Bytes, ChainId, Signature};
use alloy_transport_http::reqwest::Url;
use anyhow::{Result, ensure};
use async_trait::async_trait;
use kona_sources::ReloadingRpcClient;
use serde::Serialize;
use sp1_alloy_signer::{Error, Signer, UnsupportedSignerOperation};

use crate::tls::ClientTls;

/// An SPN requester that delegates EIP-191 message signing to op-signer over mTLS.
///
/// Certificates are reloaded when they rotate on disk; see [`ReloadingRpcClient`].
#[derive(Clone, Debug)]
pub struct OpSignerRequester {
    client: ReloadingRpcClient,
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
        Ok(Self { client: tls.rpc_client(endpoint)?, address })
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
        let response: Bytes = self
            .client
            .client()
            .request("opsigner_signMessage", (args,))
            .await
            .map_err(Error::message)?;
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

    /// Returns no chain ID because SP1 does not use one for chain-independent EIP-191 signing.
    fn chain_id(&self) -> Option<ChainId> {
        tracing::error!("chain_id called on the SPN EIP-191 message-only signer");
        None
    }

    /// Logs and ignores chain-ID updates because SP1 does not use them for EIP-191 signing.
    fn set_chain_id(&mut self, chain_id: Option<ChainId>) {
        tracing::error!(?chain_id, "set_chain_id called on the SPN EIP-191 message-only signer");
    }
}
