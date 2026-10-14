//! L1 transaction signing.
//!
//! Vendored from op-succinct's `utils/signer/src/lib.rs` (@ 13716c2c) with
//! the GCP Cloud HSM arm removed (it alone drags gcloud-sdk +
//! alloy-signer-gcp; the monorepo remote-signing path is op-signer, which
//! speaks the `Web3Signer` `eth_signTransaction` protocol kept here).

use std::{str::FromStr, sync::Arc};

use alloy_consensus::TxEnvelope;
use alloy_eips::Decodable2718;
use alloy_network::{Ethereum, EthereumWallet, NetworkTransactionBuilder};
use alloy_primitives::{Address, Bytes};
use alloy_provider::{Provider, ProviderBuilder, RootProvider};
use alloy_rpc_types_eth::{TransactionReceipt, TransactionRequest};
use alloy_signer_local::PrivateKeySigner;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result};
use kona_sources::ReloadingRpcClient;
use kona_sp1_host_utils::tls::ClientTls;
use tokio::{sync::Mutex, time::Duration};

use crate::{
    env_var,
    tx_sender::{self, FeeCaps},
};

/// The type of signer to use for signing transactions.
#[derive(Clone, Debug)]
pub enum Signer {
    /// The op-signer endpoint, address it signs for, and optional mTLS client.
    Web3Signer {
        /// op-signer JSON-RPC endpoint.
        url: Url,
        /// Address op-signer uses for L1 transactions.
        address: Address,
        /// op-signer client over mutual TLS; see [`ReloadingRpcClient`] for reloading.
        mtls: Option<ReloadingRpcClient>,
    },
    /// The local signer.
    LocalSigner(PrivateKeySigner),
}

impl Signer {
    /// Returns the L1 address transactions are signed with.
    pub const fn address(&self) -> Address {
        match self {
            Self::Web3Signer { address, .. } => *address,
            Self::LocalSigner(signer) => signer.address(),
        }
    }

    /// Creates a new Web3 signer with the given URL, address, and optional mTLS material.
    pub fn new_web3_signer(url: Url, address: Address, tls: Option<ClientTls>) -> Result<Self> {
        anyhow::ensure!(
            tls.is_none() || url.scheme() == "https",
            "Web3Signer URL must use HTTPS when TLS material is configured"
        );
        let mtls = tls.map(|tls| tls.rpc_client(url.clone())).transpose()?;
        Ok(Self::Web3Signer { url, address, mtls })
    }

    /// Creates a new local signer from a private key string.
    pub fn new_local_signer(private_key_str: &str) -> Result<Self> {
        let private_key =
            PrivateKeySigner::from_str(private_key_str).context("Failed to parse private key")?;
        Ok(Self::LocalSigner(private_key))
    }

    /// Builds a signer from the environment. `OP_ZK_PROPOSER_SIGNER_URL` and
    /// `OP_ZK_PROPOSER_SIGNER_ADDRESS` select [`Signer::Web3Signer`]; optional signer
    /// TLS variables configure mutual TLS. Otherwise, `OP_ZK_PROPOSER_PRIVATE_KEY`
    /// selects [`Signer::LocalSigner`]. Setting only one `Web3Signer` variable, or setting
    /// the local key alongside the `Web3Signer`, is an error.
    pub async fn from_env() -> Result<Self> {
        let signer_url_name = env_var("SIGNER_URL");
        let signer_address_name = env_var("SIGNER_ADDRESS");
        let private_key_name = env_var("PRIVATE_KEY");
        let signer_url = std::env::var(&signer_url_name).ok();
        let signer_address = std::env::var(&signer_address_name).ok();
        match (signer_url, signer_address) {
            (Some(url), Some(address)) => {
                anyhow::ensure!(
                    std::env::var_os(&private_key_name).is_none(),
                    "{private_key_name} and {signer_url_name} are mutually exclusive; configure \
                     either the local key or the Web3Signer"
                );
                let signer_url = Url::parse(&url)
                    .with_context(|| format!("Failed to parse {signer_url_name}"))?;
                let signer_address = Address::from_str(&address)
                    .with_context(|| format!("Failed to parse {signer_address_name}"))?;
                let tls = ClientTls::from_env(crate::ENV_VAR_PREFIX, "SIGNER")?;
                tracing::info!(
                    url = %kona_sources::redacted_url(&signer_url),
                    address = %signer_address,
                    mtls = tls.is_some(),
                    "Using Web3Signer ({signer_url_name} + {signer_address_name})"
                );
                Self::new_web3_signer(signer_url, signer_address, tls)
            }
            (Some(_), None) => {
                anyhow::bail!(
                    "{signer_url_name} is set but {signer_address_name} is not; set both to use \
                     the Web3Signer"
                )
            }
            (None, Some(_)) => {
                anyhow::bail!(
                    "{signer_address_name} is set but {signer_url_name} is not; set both to use \
                     the Web3Signer"
                )
            }
            (None, None) => {
                let tls = ClientTls::from_env(crate::ENV_VAR_PREFIX, "SIGNER")?;
                anyhow::ensure!(
                    tls.is_none(),
                    "{signer_url_name} and {signer_address_name} must be set when signer TLS \
                     material is configured"
                );
                let private_key_str = std::env::var(&private_key_name).map_err(|_| {
                    anyhow::anyhow!(
                        "None of the required signer configurations are set in environment:\n\
                        - For Web3Signer: {signer_url_name} and {signer_address_name}\n\
                        - For Local: {private_key_name}"
                    )
                })?;
                let signer = Self::new_local_signer(&private_key_str)
                    .with_context(|| format!("Failed to parse {private_key_name}"))?;
                tracing::info!(
                    address = %signer.address(),
                    "Using local private-key signer ({private_key_name})"
                );
                Ok(signer)
            }
        }
    }

    /// Signs a fully populated EIP-1559 request with the local key or through op-signer's
    /// `eth_signTransaction`.
    pub(crate) async fn sign_transaction(
        &self,
        mut request: TransactionRequest,
    ) -> Result<TxEnvelope> {
        match self {
            Self::Web3Signer { url, mtls, .. } => {
                request.normalize_data();
                let provider: RootProvider = mtls.as_ref().map_or_else(
                    || ProviderBuilder::default().connect_http(url.clone()),
                    |mtls| ProviderBuilder::default().connect_client(mtls.client()),
                );
                let raw: Bytes = provider
                    .client()
                    .request("eth_signTransaction", (request,))
                    .await
                    .context("eth_signTransaction failed")?;
                TxEnvelope::decode_2718(&mut raw.as_ref()).context(
                    "eth_signTransaction returned bytes that do not decode as a signed envelope",
                )
            }
            Self::LocalSigner(private_key) => {
                let wallet = EthereumWallet::new(private_key.clone());
                Ok(<TransactionRequest as NetworkTransactionBuilder<Ethereum>>::build(
                    request, &wallet,
                )
                .await?)
            }
        }
    }
}

/// Wrapper around Signer that provides thread-safe transaction sending.
/// Transactions are serialized via a Mutex to prevent nonce conflicts.
#[derive(Clone, Debug)]
pub struct SignerLock {
    inner: Arc<Mutex<Signer>>,
    cached_address: Address,
}

impl SignerLock {
    /// Creates a new `SignerLock` wrapping the given Signer.
    pub fn new(signer: Signer) -> Self {
        let cached_address = signer.address();
        Self { inner: Arc::new(Mutex::new(signer)), cached_address }
    }

    /// Returns the address of the signer without acquiring a lock.
    pub const fn address(&self) -> Address {
        self.cached_address
    }

    /// Sends `request` through [`tx_sender`] while holding the signer lock.
    pub async fn send_transaction<P: Provider>(
        &self,
        provider: &P,
        request: TransactionRequest,
        timeout: Duration,
        fee_caps: FeeCaps,
    ) -> Result<TransactionReceipt> {
        let signer = self.inner.lock().await;
        tx_sender::send_transaction(provider, &signer, request, timeout, fee_caps).await
    }
}

#[cfg(test)]
mod tests {
    use std::{env, path::Path};

    use alloy_eips::Encodable2718;
    use alloy_network::{Ethereum, EthereumWallet, NetworkTransactionBuilder};
    use alloy_primitives::{Bytes, TxKind, U256};
    use alloy_rpc_types_eth::{TransactionInput, TransactionRequest};
    use alloy_signer_local::PrivateKeySigner;
    use httpmock::{MockServer, prelude::POST};
    use serde_json::json;
    use serial_test::serial;

    use super::{Address, Signer};

    /// op-signer receives every field it needs to sign, with calldata in `data`, and its
    /// envelope comes back decoded.
    #[tokio::test]
    async fn web3_signer_requests_a_complete_transaction() {
        let key = PrivateKeySigner::random();
        let to = Address::repeat_byte(0x11);
        let request = TransactionRequest {
            from: Some(key.address()),
            to: Some(TxKind::Call(to)),
            value: Some(U256::from(5)),
            input: TransactionInput::new(Bytes::from_static(&[0xab, 0xcd])),
            nonce: Some(7),
            gas: Some(100_000),
            chain_id: Some(1),
            max_fee_per_gas: Some(21_000_000_000),
            max_priority_fee_per_gas: Some(1_000_000_000),
            ..Default::default()
        };
        let signed = <TransactionRequest as NetworkTransactionBuilder<Ethereum>>::build(
            request.clone(),
            &EthereumWallet::new(key.clone()),
        )
        .await
        .unwrap();

        let server = MockServer::start_async().await;
        let op_signer = server
            .mock_async(|when, then| {
                when.method(POST).json_body_includes(
                    json!({
                        "method": "eth_signTransaction",
                        "params": [{
                            "from": key.address(),
                            "to": to,
                            "value": "0x5",
                            "data": "0xabcd",
                            "nonce": "0x7",
                            "gas": "0x186a0",
                            "chainId": "0x1",
                            "maxFeePerGas": "0x4e3b29200",
                            "maxPriorityFeePerGas": "0x3b9aca00",
                        }],
                    })
                    .to_string(),
                );
                let raw = Bytes::from(signed.encoded_2718());
                then.status(200)
                    .header("content-type", "application/json")
                    .body(json!({"jsonrpc": "2.0", "id": 0, "result": raw}).to_string());
            })
            .await;

        let signer =
            Signer::new_web3_signer(server.url("/").parse().unwrap(), key.address(), None).unwrap();
        let envelope = signer.sign_transaction(request).await.unwrap();
        op_signer.assert_async().await;
        assert_eq!(envelope, signed);
    }

    fn clear_signer_env() {
        for suffix in [
            "PRIVATE_KEY",
            "SIGNER_URL",
            "SIGNER_ADDRESS",
            "SIGNER_TLS_CA",
            "SIGNER_TLS_CERT",
            "SIGNER_TLS_KEY",
        ] {
            // SAFETY: The environment-mutating tests in this module are serialized.
            unsafe { env::remove_var(crate::env_var(suffix)) };
        }
    }

    fn set_signer_env(suffix: &str, value: impl AsRef<std::ffi::OsStr>) {
        // SAFETY: The environment-mutating tests in this module are serialized.
        unsafe { env::set_var(crate::env_var(suffix), value) };
    }

    #[tokio::test]
    #[serial]
    async fn loads_web3_signer_with_tls_material() {
        clear_signer_env();
        let fixtures = Path::new(env!("CARGO_MANIFEST_DIR")).join("../host/testdata/mtls");
        set_signer_env("SIGNER_URL", "https://localhost:8545");
        set_signer_env("SIGNER_ADDRESS", Address::ZERO.to_string());
        set_signer_env("SIGNER_TLS_CA", fixtures.join("ca.crt"));
        set_signer_env("SIGNER_TLS_CERT", fixtures.join("client.crt"));
        set_signer_env("SIGNER_TLS_KEY", fixtures.join("client.key"));

        let signer = Signer::from_env().await.unwrap();
        let Signer::Web3Signer { url, address, mtls } = signer else {
            panic!("expected Web3Signer")
        };
        assert_eq!(url.as_str(), "https://localhost:8545/");
        assert_eq!(address, Address::ZERO);
        assert!(mtls.is_some());
        clear_signer_env();
    }

    #[tokio::test]
    #[serial]
    async fn rejects_http_web3_signer_with_tls_material() {
        clear_signer_env();
        let fixtures = Path::new(env!("CARGO_MANIFEST_DIR")).join("../host/testdata/mtls");
        set_signer_env("SIGNER_URL", "http://localhost:8545");
        set_signer_env("SIGNER_ADDRESS", Address::ZERO.to_string());
        set_signer_env("SIGNER_TLS_CA", fixtures.join("ca.crt"));
        set_signer_env("SIGNER_TLS_CERT", fixtures.join("client.crt"));
        set_signer_env("SIGNER_TLS_KEY", fixtures.join("client.key"));

        let error = Signer::from_env().await.unwrap_err().to_string();
        assert!(error.contains("must use HTTPS"), "{error}");
        clear_signer_env();
    }

    #[tokio::test]
    #[serial]
    async fn rejects_tls_material_without_web3_signer() {
        clear_signer_env();
        let fixtures = Path::new(env!("CARGO_MANIFEST_DIR")).join("../host/testdata/mtls");
        set_signer_env("PRIVATE_KEY", "unused");
        set_signer_env("SIGNER_TLS_CA", fixtures.join("ca.crt"));
        set_signer_env("SIGNER_TLS_CERT", fixtures.join("client.crt"));
        set_signer_env("SIGNER_TLS_KEY", fixtures.join("client.key"));

        let error = Signer::from_env().await.unwrap_err().to_string();
        assert!(error.contains(&crate::env_var("SIGNER_URL")), "{error}");
        assert!(error.contains(&crate::env_var("SIGNER_ADDRESS")), "{error}");
        clear_signer_env();
    }

    #[tokio::test]
    #[serial]
    async fn rejects_local_key_with_web3_signer() {
        clear_signer_env();
        set_signer_env("PRIVATE_KEY", "unused");
        set_signer_env("SIGNER_URL", "https://localhost:8545");
        set_signer_env("SIGNER_ADDRESS", Address::ZERO.to_string());

        let error = Signer::from_env().await.unwrap_err().to_string();
        assert!(error.contains(&crate::env_var("PRIVATE_KEY")), "{error}");
        assert!(error.contains(&crate::env_var("SIGNER_URL")), "{error}");
        assert!(error.contains("mutually exclusive"), "{error}");
        clear_signer_env();
    }
}
