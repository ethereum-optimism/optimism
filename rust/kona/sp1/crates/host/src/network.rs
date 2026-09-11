//! Utilities for network configuration and signer retrieval.

use std::{env, sync::Arc};

use alloy_primitives::Address;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, anyhow, bail};
use sp1_alloy_signer::Signer as _;
use sp1_sdk::{
    NetworkProver, ProverClient,
    network::{
        FulfillmentStrategy, NetworkMode, get_default_rpc_url_for_mode, signer::NetworkSigner,
    },
};

use crate::{prefixed_env_var, spn_signer::OpSignerRequester, tls::ClientTls};

/// Parse a fulfillment strategy from a string.
pub fn parse_fulfillment_strategy(value: String) -> Result<FulfillmentStrategy> {
    match value.to_ascii_lowercase().as_str() {
        "reserved" => Ok(FulfillmentStrategy::Reserved),
        "hosted" => Ok(FulfillmentStrategy::Hosted),
        "auction" => Ok(FulfillmentStrategy::Auction),
        _ => bail!(
            "Invalid fulfillment strategy '{value}': must be 'reserved', 'hosted', or 'auction'"
        ),
    }
}

/// Try to determine the network mode from the provided fulfillment strategies.
pub fn determine_network_mode(
    range_proof_strategy: FulfillmentStrategy,
    agg_proof_strategy: FulfillmentStrategy,
) -> Result<NetworkMode> {
    match (range_proof_strategy, agg_proof_strategy) {
        (FulfillmentStrategy::Auction, FulfillmentStrategy::Auction) => Ok(NetworkMode::Mainnet),
        (
            FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved,
            FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved,
        ) => Ok(NetworkMode::Reserved),
        (FulfillmentStrategy::UnspecifiedFulfillmentStrategy, _) |
        (_, FulfillmentStrategy::UnspecifiedFulfillmentStrategy) => {
            Err(anyhow!("The range and agg fulfillment Strategies must be specified"))
        }
        _ => Err(anyhow!(
            "The range fulfillment Strategy '{}' and agg fulfillment Strategy '{}' are incompatible",
            range_proof_strategy.as_str_name().to_ascii_lowercase(),
            agg_proof_strategy.as_str_name().to_ascii_lowercase()
        )),
    }
}

/// Computes the SPN network signer from environment variables.
///
/// A remote op-signer takes precedence when `<PREFIX>_SPN_SIGNER_URL` and
/// `<PREFIX>_SPN_SIGNER_ADDRESS` are set. Its `<PREFIX>_SPN_SIGNER_TLS_CA`, `_CERT`, and `_KEY`
/// variables are required. Otherwise, `<PREFIX>_NETWORK_PRIVATE_KEY` selects AWS KMS or local
/// signing according to `use_kms_requester`.
pub async fn get_network_signer(prefix: &str, use_kms_requester: bool) -> Result<NetworkSigner> {
    let remote_url_name = prefixed_env_var(prefix, "SPN_SIGNER_URL");
    let remote_address_name = prefixed_env_var(prefix, "SPN_SIGNER_ADDRESS");
    let env_value = |name: &str| env::var(name).ok().filter(|value| !value.trim().is_empty());

    let remote_tls = ClientTls::from_env(prefix, "SPN_SIGNER")?;
    match (env_value(&remote_url_name), env_value(&remote_address_name), remote_tls) {
        (Some(endpoint), Some(address), Some(tls)) => {
            let endpoint = endpoint
                .parse::<Url>()
                .with_context(|| format!("{remote_url_name} must be a valid URL"))?;
            let address = address
                .parse::<Address>()
                .with_context(|| format!("{remote_address_name} must be a valid address"))?;
            let signer = OpSignerRequester::new(endpoint, address, tls)
                .context("failed to create remote SPN requester")?;
            tracing::info!("Using remote SPN requester with address: {:?}", signer.address());
            return Ok(NetworkSigner::dynamic(Arc::new(signer)));
        }
        (None, None, None) => {}
        _ => bail!(
            "{remote_url_name}, {remote_address_name}, {prefix}_SPN_SIGNER_TLS_CA, \
             {prefix}_SPN_SIGNER_TLS_CERT, and {prefix}_SPN_SIGNER_TLS_KEY must be set together"
        ),
    }

    let private_key_name = prefixed_env_var(prefix, "NETWORK_PRIVATE_KEY");
    let use_kms_name = prefixed_env_var(prefix, "USE_KMS_REQUESTER");
    let network_signer = if use_kms_requester {
        let key_arn = env::var(&private_key_name).with_context(|| {
            format!("{private_key_name} must be set when {use_kms_name} is true")
        })?;
        let signer = NetworkSigner::aws_kms(&key_arn)
            .await
            .with_context(|| format!("failed to create requester from {private_key_name}"))?;
        tracing::info!("Using KMS requester with address: {:?}", signer.address());
        signer
    } else {
        let private_key = env_value(&private_key_name).with_context(|| {
            format!(
                "{private_key_name} must be set for network proving (or set {use_kms_name}=true \
                 to sign requests with AWS KMS)"
            )
        })?;
        let signer = NetworkSigner::local(&private_key)
            .with_context(|| format!("failed to create requester from {private_key_name}"))?;
        tracing::info!("Using local requester with address: {:?}", signer.address());
        signer
    };

    Ok(network_signer)
}

/// Builds a network prover using the provided fulfillment strategy and
/// `<PREFIX>_USE_KMS_REQUESTER`.
///
/// `<PREFIX>_NETWORK_RPC_URL` overrides the SP1 endpoint. If it is absent or empty, the
/// SDK default for the fulfillment strategy's network mode is used.
pub async fn build_network_prover_from_env(
    prefix: &str,
    strategy: FulfillmentStrategy,
) -> Result<NetworkProver> {
    let use_kms_name = prefixed_env_var(prefix, "USE_KMS_REQUESTER");
    let use_kms_requester = env::var(&use_kms_name)
        .unwrap_or_else(|_| "false".to_string())
        .parse::<bool>()
        .with_context(|| format!("{use_kms_name} must be true or false"))?;

    let network_mode = match strategy {
        FulfillmentStrategy::Auction => NetworkMode::Mainnet,
        FulfillmentStrategy::Hosted | FulfillmentStrategy::Reserved => NetworkMode::Reserved,
        _ => bail!("Fulfillment strategy must be 'reserved', 'hosted', or 'auction'"),
    };

    let rpc_url = env::var(prefixed_env_var(prefix, "NETWORK_RPC_URL"))
        .ok()
        .filter(|url| !url.trim().is_empty())
        .unwrap_or_else(|| get_default_rpc_url_for_mode(network_mode));
    let network_signer = get_network_signer(prefix, use_kms_requester).await?;

    let prover = ProverClient::builder()
        .network_for(network_mode)
        .signer(network_signer)
        .rpc_url(&rpc_url)
        .build()
        .await;

    Ok(prover)
}

#[cfg(test)]
mod tests {
    use std::path::Path;

    use super::*;

    fn set_env(prefix: &str, suffix: &str, value: impl AsRef<std::ffi::OsStr>) {
        // SAFETY: Each test uses a unique prefix, so concurrent tests do not share variables.
        unsafe { env::set_var(prefixed_env_var(prefix, suffix), value) };
    }

    fn set_tls_env(prefix: &str) {
        let fixture =
            |name: &str| Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata/mtls").join(name);
        set_env(prefix, "SPN_SIGNER_TLS_CA", fixture("ca.crt"));
        set_env(prefix, "SPN_SIGNER_TLS_CERT", fixture("client.crt"));
        set_env(prefix, "SPN_SIGNER_TLS_KEY", fixture("client.key"));
    }

    #[tokio::test]
    async fn remote_signer_takes_precedence_over_local_key() {
        let prefix = "KONA_SP1_HOST_NETWORK_TEST_REMOTE";
        let address = "0x1111111111111111111111111111111111111111";
        set_env(prefix, "SPN_SIGNER_URL", "https://signer.example");
        set_env(prefix, "SPN_SIGNER_ADDRESS", address);
        set_env(prefix, "NETWORK_PRIVATE_KEY", "not-a-private-key");
        set_tls_env(prefix);

        let signer = get_network_signer(prefix, false).await.unwrap();

        assert_eq!(signer.address(), address.parse::<Address>().unwrap());
    }

    #[tokio::test]
    async fn remote_signer_requires_url_and_address_together() {
        let prefix = "KONA_SP1_HOST_NETWORK_TEST_PARTIAL";
        set_env(prefix, "SPN_SIGNER_URL", "https://signer.example");

        let error = get_network_signer(prefix, false).await.unwrap_err().to_string();

        assert!(error.contains("SPN_SIGNER_URL"), "{error}");
        assert!(error.contains("SPN_SIGNER_ADDRESS"), "{error}");
    }

    #[tokio::test]
    async fn remote_signer_rejects_tls_only_configuration() {
        let prefix = "KONA_SP1_HOST_NETWORK_TEST_TLS_ONLY";
        set_env(
            prefix,
            "NETWORK_PRIVATE_KEY",
            "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        );
        set_tls_env(prefix);

        let error = get_network_signer(prefix, false).await.unwrap_err().to_string();

        assert!(error.contains("SPN_SIGNER_URL"), "{error}");
        assert!(error.contains("SPN_SIGNER_TLS_CA"), "{error}");
    }

    #[tokio::test]
    async fn remote_signer_requires_https() {
        let prefix = "KONA_SP1_HOST_NETWORK_TEST_HTTP";
        set_env(prefix, "SPN_SIGNER_URL", "http://signer.example");
        set_env(prefix, "SPN_SIGNER_ADDRESS", "0x1111111111111111111111111111111111111111");
        set_tls_env(prefix);

        let error = get_network_signer(prefix, false).await.unwrap_err().to_string();

        assert!(error.contains("failed to create remote SPN requester"), "{error}");
    }
}
