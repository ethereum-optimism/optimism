//! L1 transaction signing.
//!
//! Vendored from op-succinct's `utils/signer/src/lib.rs` (@ 13716c2c) with
//! the GCP Cloud HSM arm removed (it alone drags gcloud-sdk +
//! alloy-signer-gcp; the monorepo remote-signing path is op-signer, which
//! speaks the `Web3Signer` `eth_signTransaction` protocol kept here).

use std::{str::FromStr, sync::Arc};

use alloy_consensus::TxEnvelope;
use alloy_eips::Decodable2718;
use alloy_network::{Ethereum, EthereumWallet, TransactionBuilder};
use alloy_primitives::{Address, Bytes};
use alloy_provider::{Provider, ProviderBuilder, Web3Signer};
use alloy_rpc_types_eth::{TransactionReceipt, TransactionRequest};
use alloy_signer_local::PrivateKeySigner;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result};
use tokio::{sync::Mutex, time::Duration};

/// Number of L1 confirmations required before a transaction is considered included.
pub const NUM_CONFIRMATIONS: u64 = 3;

/// Optional operator caps on EIP-1559 fee fields, applied after fee
/// estimation (see [`clamp_fee_caps`]). Unset caps leave the estimate.
#[derive(Clone, Copy, Debug, Default)]
pub struct FeeCaps {
    /// Cap on max fee per gas, in wei.
    pub max_fee_per_gas: Option<u128>,
    /// Cap on max priority fee per gas, in wei.
    pub max_priority_fee_per_gas: Option<u128>,
}

/// Clamps a filled request's EIP-1559 fee fields to the configured caps,
/// then restores the `priority <= max_fee` invariant. Fields the filler
/// left unset are untouched (legacy/pre-1559 requests pass through).
pub const fn clamp_fee_caps(request: &mut TransactionRequest, caps: FeeCaps) {
    if let (Some(cap), Some(fee)) = (caps.max_fee_per_gas, request.max_fee_per_gas) &&
        fee > cap
    {
        request.max_fee_per_gas = Some(cap);
    }
    if let (Some(cap), Some(prio)) =
        (caps.max_priority_fee_per_gas, request.max_priority_fee_per_gas) &&
        prio > cap
    {
        request.max_priority_fee_per_gas = Some(cap);
    }
    if let (Some(fee), Some(prio)) = (request.max_fee_per_gas, request.max_priority_fee_per_gas) &&
        prio > fee
    {
        request.max_priority_fee_per_gas = Some(fee);
    }
}

#[derive(Clone, Debug)]
/// The type of signer to use for signing transactions.
pub enum Signer {
    /// The signer URL and address.
    Web3Signer(Url, Address),
    /// The local signer.
    LocalSigner(PrivateKeySigner),
}

impl Signer {
    /// Returns the L1 address transactions are signed with.
    pub const fn address(&self) -> Address {
        match self {
            Self::Web3Signer(_, address) => *address,
            Self::LocalSigner(signer) => signer.address(),
        }
    }

    /// Creates a new Web3 signer with the given URL and address.
    pub const fn new_web3_signer(url: Url, address: Address) -> Self {
        Self::Web3Signer(url, address)
    }

    /// Creates a new local signer from a private key string.
    pub fn new_local_signer(private_key_str: &str) -> Result<Self> {
        let private_key =
            PrivateKeySigner::from_str(private_key_str).context("Failed to parse private key")?;
        Ok(Self::LocalSigner(private_key))
    }

    /// Builds a signer from the environment: `SIGNER_URL` + `SIGNER_ADDRESS` selects
    /// [`Signer::Web3Signer`], otherwise `PRIVATE_KEY` selects [`Signer::LocalSigner`].
    /// Setting only one of `SIGNER_URL` / `SIGNER_ADDRESS` is an error rather than a
    /// silent fallback to the local key.
    pub async fn from_env() -> Result<Self> {
        let signer_url = std::env::var("SIGNER_URL").ok();
        let signer_address = std::env::var("SIGNER_ADDRESS").ok();
        match (signer_url, signer_address) {
            (Some(url), Some(address)) => {
                let signer_url = Url::parse(&url).context("Failed to parse SIGNER_URL")?;
                let signer_address =
                    Address::from_str(&address).context("Failed to parse SIGNER_ADDRESS")?;
                tracing::info!(
                    url = %crate::config::redacted_url(&signer_url),
                    address = %signer_address,
                    "Using Web3Signer (SIGNER_URL + SIGNER_ADDRESS)"
                );
                Ok(Self::new_web3_signer(signer_url, signer_address))
            }
            (Some(_), None) => {
                anyhow::bail!(
                    "SIGNER_URL is set but SIGNER_ADDRESS is not; set both to use the Web3Signer"
                )
            }
            (None, Some(_)) => {
                anyhow::bail!(
                    "SIGNER_ADDRESS is set but SIGNER_URL is not; set both to use the Web3Signer"
                )
            }
            (None, None) => {
                let private_key_str = std::env::var("PRIVATE_KEY").map_err(|_| {
                    anyhow::anyhow!(
                        "None of the required signer configurations are set in environment:\n\
                        - For Web3Signer: SIGNER_URL and SIGNER_ADDRESS\n\
                        - For Local: PRIVATE_KEY"
                    )
                })?;
                let signer = Self::new_local_signer(&private_key_str)?;
                tracing::info!(
                    address = %signer.address(),
                    "Using local private-key signer (PRIVATE_KEY)"
                );
                Ok(signer)
            }
        }
    }

    /// Sends a transaction request, signed by the configured `signer`, with a caller-supplied
    /// confirmation timeout (in seconds). The filled request's fee fields are
    /// clamped to `fee_caps` before signing (see [`clamp_fee_caps`]).
    pub async fn send_transaction_request_with_timeout(
        &self,
        l1_rpc: Url,
        mut transaction_request: TransactionRequest,
        timeout_secs: u64,
        fee_caps: FeeCaps,
    ) -> Result<TransactionReceipt> {
        match self {
            Self::Web3Signer(signer_url, signer_address) => {
                // Set the from address to the signer address.
                transaction_request.set_from(*signer_address);

                // Fill the transaction request with all of the relevant gas and nonce information.
                let provider = ProviderBuilder::new().network::<Ethereum>().connect_http(l1_rpc);
                let filled_tx = provider.fill(transaction_request).await?;

                // Sign the transaction request using the Web3Signer.
                let web3_provider =
                    ProviderBuilder::new().network::<Ethereum>().connect_http(signer_url.clone());
                let signer = Web3Signer::new(web3_provider.clone(), *signer_address);

                let mut tx = filled_tx.as_builder().unwrap().clone();
                tx.normalize_data();
                clamp_fee_caps(&mut tx, fee_caps);

                let raw: Bytes =
                    signer.provider().client().request("eth_signTransaction", (tx,)).await?;

                let tx_envelope = TxEnvelope::decode_2718(&mut raw.as_ref()).unwrap();

                let receipt = provider
                    .send_tx_envelope(tx_envelope)
                    .await
                    .context("Failed to send transaction")?
                    .with_required_confirmations(NUM_CONFIRMATIONS)
                    .with_timeout(Some(Duration::from_secs(timeout_secs)))
                    .get_receipt()
                    .await?;

                Ok(receipt)
            }
            Self::LocalSigner(private_key) => {
                let provider = ProviderBuilder::new()
                    .network::<Ethereum>()
                    .wallet(EthereumWallet::new(private_key.clone()))
                    .connect_http(l1_rpc.clone());

                // Ensure the request has a `from` address so the wallet filler can sign it.
                transaction_request.set_from(private_key.address());
                // The proposer never deploys contracts; a request without a
                // `to` address is a bug upstream of the signer.
                anyhow::ensure!(
                    transaction_request.to.is_some(),
                    "transaction request has no `to` address"
                );

                // Fill on a separate wallet-less provider so the fee estimates
                // can be clamped: the wallet provider signs during fill,
                // returning an envelope, and a signed envelope cannot be
                // clamped. The wallet provider then re-fills the already-set
                // fields as no-ops and signs.
                let fill_provider =
                    ProviderBuilder::new().network::<Ethereum>().connect_http(l1_rpc);
                let filled = fill_provider.fill(transaction_request).await?;
                let mut request = filled.as_builder().expect("wallet-less fill").clone();
                clamp_fee_caps(&mut request, fee_caps);

                let receipt = provider
                    .send_transaction(request)
                    .await
                    .context("Failed to send transaction")?
                    .with_required_confirmations(NUM_CONFIRMATIONS)
                    .with_timeout(Some(Duration::from_secs(timeout_secs)))
                    .get_receipt()
                    .await?;

                Ok(receipt)
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

    /// Creates a `SignerLock` from environment variables.
    pub async fn from_env() -> Result<Self> {
        Ok(Self::new(Signer::from_env().await?))
    }

    /// Returns the address of the signer without acquiring a lock.
    pub const fn address(&self) -> Address {
        self.cached_address
    }

    /// Sends a transaction request with a caller-supplied confirmation timeout (in seconds).
    /// Transactions are serialized via a Mutex to prevent nonce conflicts.
    pub async fn send_transaction_request_with_timeout(
        &self,
        l1_rpc: Url,
        transaction_request: TransactionRequest,
        timeout_secs: u64,
        fee_caps: FeeCaps,
    ) -> Result<TransactionReceipt> {
        let signer = self.inner.lock().await;
        signer
            .send_transaction_request_with_timeout(
                l1_rpc,
                transaction_request,
                timeout_secs,
                fee_caps,
            )
            .await
    }
}

#[cfg(test)]
mod tests {
    use super::{FeeCaps, clamp_fee_caps};
    use alloy_rpc_types_eth::TransactionRequest;

    fn filled_request(max_fee: u128, max_priority: u128) -> TransactionRequest {
        TransactionRequest {
            max_fee_per_gas: Some(max_fee),
            max_priority_fee_per_gas: Some(max_priority),
            ..Default::default()
        }
    }

    #[test]
    fn clamps_both_fees_to_caps() {
        let mut request = filled_request(100, 50);
        clamp_fee_caps(
            &mut request,
            FeeCaps { max_fee_per_gas: Some(60), max_priority_fee_per_gas: Some(10) },
        );
        assert_eq!(request.max_fee_per_gas, Some(60));
        assert_eq!(request.max_priority_fee_per_gas, Some(10));
    }

    #[test]
    fn unset_caps_leave_estimates() {
        let mut request = filled_request(100, 50);
        clamp_fee_caps(&mut request, FeeCaps::default());
        assert_eq!(request.max_fee_per_gas, Some(100));
        assert_eq!(request.max_priority_fee_per_gas, Some(50));
    }

    #[test]
    fn caps_above_estimates_change_nothing() {
        let mut request = filled_request(100, 50);
        clamp_fee_caps(
            &mut request,
            FeeCaps { max_fee_per_gas: Some(200), max_priority_fee_per_gas: Some(80) },
        );
        assert_eq!(request.max_fee_per_gas, Some(100));
        assert_eq!(request.max_priority_fee_per_gas, Some(50));
    }

    #[test]
    fn priority_reclamped_when_fee_cap_undercuts_it() {
        // Only the max fee is capped, below the estimated priority fee: the
        // priority fee must follow it down to keep priority <= max_fee.
        let mut request = filled_request(100, 50);
        clamp_fee_caps(
            &mut request,
            FeeCaps { max_fee_per_gas: Some(30), max_priority_fee_per_gas: None },
        );
        assert_eq!(request.max_fee_per_gas, Some(30));
        assert_eq!(request.max_priority_fee_per_gas, Some(30));
    }

    #[test]
    fn unfilled_fields_untouched() {
        let mut request = TransactionRequest::default();
        clamp_fee_caps(
            &mut request,
            FeeCaps { max_fee_per_gas: Some(1), max_priority_fee_per_gas: Some(1) },
        );
        assert_eq!(request.max_fee_per_gas, None);
        assert_eq!(request.max_priority_fee_per_gas, None);
    }
}
