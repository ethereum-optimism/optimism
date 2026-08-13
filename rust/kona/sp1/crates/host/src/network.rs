//! Utilities for network configuration and signer retrieval.

use std::env;

use anyhow::{Context, Result, anyhow, bail};
use sp1_sdk::{
    NetworkProver, ProverClient,
    network::{
        FulfillmentStrategy, NetworkMode, get_default_rpc_url_for_mode, signer::NetworkSigner,
    },
};

use crate::prefixed_env_var;

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

enum NetworkSignerConfig {
    AwsKms(String),
    Local(String),
}

fn network_signer_config(prefix: &str, use_kms_requester: bool) -> Result<NetworkSignerConfig> {
    let private_key_name = prefixed_env_var(prefix, "NETWORK_PRIVATE_KEY");
    let use_kms_name = prefixed_env_var(prefix, "USE_KMS_REQUESTER");
    if use_kms_requester {
        let key_arn = env::var(&private_key_name).with_context(|| {
            format!("{private_key_name} must be set when {use_kms_name} is true")
        })?;
        Ok(NetworkSignerConfig::AwsKms(key_arn))
    } else {
        let private_key = env::var(&private_key_name)
            .ok()
            .filter(|key| !key.trim().is_empty())
            .with_context(|| {
                format!(
                    "{private_key_name} must be set for network proving (or set {use_kms_name}=true \
                     to sign requests with AWS KMS)"
                )
            })?;
        Ok(NetworkSignerConfig::Local(private_key))
    }
}

/// Computes the network signer from `<PREFIX>_NETWORK_PRIVATE_KEY`.
///
/// When `use_kms_requester` is `true`, the value must be an AWS KMS key ARN. Otherwise,
/// it must be a local private key. The proposer supplies `KONA_SP1_PROPOSER` as the prefix.
pub async fn get_network_signer(prefix: &str, use_kms_requester: bool) -> Result<NetworkSigner> {
    let private_key_name = prefixed_env_var(prefix, "NETWORK_PRIVATE_KEY");
    let network_signer = match network_signer_config(prefix, use_kms_requester)? {
        NetworkSignerConfig::AwsKms(key_arn) => {
            let signer = NetworkSigner::aws_kms(&key_arn)
                .await
                .with_context(|| format!("failed to create requester from {private_key_name}"))?;
            tracing::info!("Using KMS requester with address: {:?}", signer.address());
            signer
        }
        NetworkSignerConfig::Local(private_key) => {
            let signer = NetworkSigner::local(&private_key)
                .with_context(|| format!("failed to create requester from {private_key_name}"))?;
            tracing::info!("Using local requester with address: {:?}", signer.address());
            signer
        }
    };

    Ok(network_signer)
}

struct NetworkProverConfig {
    use_kms_requester: bool,
    network_mode: NetworkMode,
    rpc_url: String,
}

fn network_prover_config(
    prefix: &str,
    strategy: FulfillmentStrategy,
) -> Result<NetworkProverConfig> {
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

    let rpc_url_name = prefixed_env_var(prefix, "NETWORK_RPC_URL");
    let rpc_url = env::var(rpc_url_name)
        .ok()
        .filter(|url| !url.trim().is_empty())
        .unwrap_or_else(|| get_default_rpc_url_for_mode(network_mode));

    Ok(NetworkProverConfig { use_kms_requester, network_mode, rpc_url })
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
    let config = network_prover_config(prefix, strategy)?;
    let network_signer = get_network_signer(prefix, config.use_kms_requester).await?;

    let prover = ProverClient::builder()
        .network_for(config.network_mode)
        .signer(network_signer)
        .rpc_url(&config.rpc_url)
        .build()
        .await;

    Ok(prover)
}

#[cfg(test)]
mod tests {
    use super::*;

    const PREFIX: &str = "KONA_SP1_PROPOSER";

    fn set_env(suffix: &str, value: &str) {
        unsafe { env::set_var(prefixed_env_var(PREFIX, suffix), value) };
    }

    fn remove_env(suffix: &str) {
        unsafe { env::remove_var(prefixed_env_var(PREFIX, suffix)) };
    }

    #[test]
    fn requester_selection_uses_configured_network_private_key() {
        set_env("NETWORK_PRIVATE_KEY", "key");
        unsafe { env::set_var("NETWORK_PRIVATE_KEY", "other-key") };

        match network_signer_config(PREFIX, false).unwrap() {
            NetworkSignerConfig::Local(key) => assert_eq!(key, "key"),
            NetworkSignerConfig::AwsKms(_) => panic!("expected local requester"),
        }
        match network_signer_config(PREFIX, true).unwrap() {
            NetworkSignerConfig::AwsKms(key) => assert_eq!(key, "key"),
            NetworkSignerConfig::Local(_) => panic!("expected KMS requester"),
        }

        remove_env("NETWORK_PRIVATE_KEY");
        let err = network_signer_config(PREFIX, false)
            .err()
            .expect("expected missing requester key")
            .to_string();
        assert!(err.contains("KONA_SP1_PROPOSER_NETWORK_PRIVATE_KEY"), "unexpected: {err}");
    }

    #[test]
    fn network_rpc_override_and_mode_defaults() {
        set_env("USE_KMS_REQUESTER", "true");
        set_env("NETWORK_RPC_URL", "https://rpc.example");
        unsafe { env::set_var("NETWORK_RPC_URL", "https://other.example") };
        let config = network_prover_config(PREFIX, FulfillmentStrategy::Auction).unwrap();
        assert!(config.use_kms_requester);
        assert_eq!(config.network_mode, NetworkMode::Mainnet);
        assert_eq!(config.rpc_url, "https://rpc.example");

        remove_env("USE_KMS_REQUESTER");
        remove_env("NETWORK_RPC_URL");
        let config = network_prover_config(PREFIX, FulfillmentStrategy::Auction).unwrap();
        assert!(!config.use_kms_requester);
        assert_eq!(config.network_mode, NetworkMode::Mainnet);
        assert_eq!(config.rpc_url, get_default_rpc_url_for_mode(NetworkMode::Mainnet));

        let config = network_prover_config(PREFIX, FulfillmentStrategy::Hosted).unwrap();
        assert_eq!(config.network_mode, NetworkMode::Reserved);
        assert_eq!(config.rpc_url, get_default_rpc_url_for_mode(NetworkMode::Reserved));

        set_env("NETWORK_RPC_URL", "   ");
        let config = network_prover_config(PREFIX, FulfillmentStrategy::Reserved).unwrap();
        assert_eq!(config.network_mode, NetworkMode::Reserved);
        assert_eq!(config.rpc_url, get_default_rpc_url_for_mode(NetworkMode::Reserved));
    }
}
