//! Minimal client for the Succinct Prover Network RPC.
//!
//! Reuses the proto types and generated gRPC client from `sp1-sdk`; the SDK itself only exposes
//! balance/nonce for its own signer and has no transfer/withdraw helpers, so those requests are
//! built and signed here the same way the SDK signs proof requests (EIP-191 over the
//! protobuf-encoded body, `MessageFormat::Binary`).

use std::{str::FromStr, time::Duration};

use alloy_primitives::{Address, B256, U256};
use alloy_signer::Signer;
use alloy_signer_local::PrivateKeySigner;
use anyhow::{Context, Result, ensure};
use prost::Message;
use sp1_sdk::network::proto::{
    network::prover_network_client::ProverNetworkClient,
    types::{
        GetBalanceRequest, GetNonceRequest, GetProofRequestParamsRequest, GetTransferParamsRequest,
        GetWithdrawParamsRequest, MessageFormat, ProofMode, TransactionVariant, TransferRequest,
        TransferRequestBody, WithdrawRequest, WithdrawRequestBody,
    },
};
use tonic::transport::{Channel, ClientTlsConfig, Endpoint};

/// Default SPN mainnet RPC (`MAINNET_RPC_URL` in sp1-sdk).
pub(crate) const MAINNET_SPN_RPC: &str = "https://rpc.mainnet.succinct.xyz";

/// Auctioneer and fee charged by the network for a transfer or withdraw.
#[derive(Debug)]
pub(crate) struct TxParams {
    pub(crate) auctioneer: Address,
    pub(crate) fee: U256,
}

/// Connection to the SPN RPC.
#[derive(Debug)]
pub(crate) struct Spn {
    client: ProverNetworkClient<Channel>,
}

impl Spn {
    pub(crate) async fn connect(url: &str) -> Result<Self> {
        // tonic's TLS needs a process-wide rustls provider; sp1-sdk installs ring the same way.
        let _ = rustls::crypto::ring::default_provider().install_default();
        let mut endpoint = Endpoint::new(url.to_string())
            .with_context(|| format!("invalid SPN RPC URL {url}"))?
            .timeout(Duration::from_secs(60))
            .connect_timeout(Duration::from_secs(15));
        if url.starts_with("https://") {
            endpoint = endpoint.tls_config(ClientTlsConfig::new().with_enabled_roots())?;
        }
        let channel =
            endpoint.connect().await.with_context(|| format!("connecting to SPN RPC {url}"))?;
        Ok(Self { client: ProverNetworkClient::new(channel) })
    }

    /// Spendable SPN balance of any address, in wei of PROVE.
    pub(crate) async fn balance(&self, address: Address) -> Result<U256> {
        let res = self
            .client
            .clone()
            .get_balance(GetBalanceRequest { address: address.to_vec() })
            .await
            .context("SPN GetBalance")?
            .into_inner();
        U256::from_str(&res.amount)
            .with_context(|| format!("SPN GetBalance returned invalid amount {:?}", res.amount))
    }

    pub(crate) async fn transfer_params(&self) -> Result<TxParams> {
        let res = self
            .client
            .clone()
            .get_transfer_params(GetTransferParamsRequest {})
            .await
            .context("SPN GetTransferParams")?
            .into_inner();
        tx_params(&res.auctioneer, &res.fee)
    }

    pub(crate) async fn withdraw_params(&self) -> Result<TxParams> {
        let res = self
            .client
            .clone()
            .get_withdraw_params(GetWithdrawParamsRequest {})
            .await
            .context("SPN GetWithdrawParams")?
            .into_inner();
        tx_params(&res.auctioneer, &res.fee)
    }

    /// Moves `amount` from the signer's SPN account to `to`. The recipient does not sign.
    pub(crate) async fn transfer(
        &self,
        signer: &PrivateKeySigner,
        to: Address,
        amount: U256,
        params: &TxParams,
    ) -> Result<B256> {
        let body = TransferRequestBody {
            nonce: self.nonce(signer.address()).await?,
            to: to.to_vec(),
            amount: amount.to_string(),
            domain: self.domain().await?,
            variant: TransactionVariant::TransferVariant.into(),
            auctioneer: params.auctioneer.to_vec(),
            fee: params.fee.to_string(),
        };
        let signature = sign(signer, &body).await?;
        let res = self
            .client
            .clone()
            .transfer(TransferRequest {
                format: MessageFormat::Binary.into(),
                signature,
                body: Some(body),
            })
            .await
            .context("SPN Transfer")?
            .into_inner();
        tx_hash(&res.tx_hash)
    }

    /// Withdraws `amount` from the signer's SPN account to the same address on L1.
    pub(crate) async fn withdraw(
        &self,
        signer: &PrivateKeySigner,
        amount: U256,
        params: &TxParams,
    ) -> Result<B256> {
        let body = WithdrawRequestBody {
            nonce: self.nonce(signer.address()).await?,
            account: signer.address().to_vec(),
            amount: amount.to_string(),
            domain: self.domain().await?,
            variant: TransactionVariant::WithdrawVariant.into(),
            auctioneer: params.auctioneer.to_vec(),
            fee: params.fee.to_string(),
        };
        let signature = sign(signer, &body).await?;
        let res = self
            .client
            .clone()
            .withdraw(WithdrawRequest {
                format: MessageFormat::Binary.into(),
                signature,
                body: Some(body),
            })
            .await
            .context("SPN Withdraw")?
            .into_inner();
        tx_hash(&res.tx_hash)
    }

    async fn nonce(&self, address: Address) -> Result<u64> {
        Ok(self
            .client
            .clone()
            .get_nonce(GetNonceRequest { address: address.to_vec() })
            .await
            .context("SPN GetNonce")?
            .into_inner()
            .nonce)
    }

    /// The network-global EIP-712 domain separator, as sp1-sdk obtains it for proof requests.
    async fn domain(&self) -> Result<Vec<u8>> {
        let domain = self
            .client
            .clone()
            .get_proof_request_params(GetProofRequestParamsRequest {
                mode: ProofMode::Compressed.into(),
            })
            .await
            .context("SPN GetProofRequestParams")?
            .into_inner()
            .domain;
        ensure!(domain.len() == 32, "SPN returned a {}-byte domain, expected 32", domain.len());
        Ok(domain)
    }
}

async fn sign<M: Message>(signer: &PrivateKeySigner, body: &M) -> Result<Vec<u8>> {
    let signature = signer.sign_message(&body.encode_to_vec()).await?;
    Ok(signature.as_bytes().to_vec())
}

fn tx_params(auctioneer: &[u8], fee: &str) -> Result<TxParams> {
    Ok(TxParams {
        auctioneer: Address::try_from(auctioneer).context("SPN returned an invalid auctioneer")?,
        fee: U256::from_str(fee).with_context(|| format!("SPN returned an invalid fee {fee:?}"))?,
    })
}

fn tx_hash(bytes: &[u8]) -> Result<B256> {
    B256::try_from(bytes).context("SPN returned an invalid tx hash")
}
